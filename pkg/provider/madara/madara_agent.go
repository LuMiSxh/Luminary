package madara

import (
	"Luminary/pkg/engine"
	"Luminary/pkg/engine/core"
	"Luminary/pkg/engine/network"
	"Luminary/pkg/engine/parser"
	"Luminary/pkg/errors"
	"Luminary/pkg/provider"
	"Luminary/pkg/provider/common"
	"Luminary/pkg/provider/web"
	"Luminary/pkg/util"
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Provider implements the provider.Provider interface for Madara-based sites
type Provider struct {
	config       Config
	htmlProvider *web.Provider
	engine       *engine.Engine
	webScraper   *network.WebScraperService
}

// NewProvider creates a new Madara-based provider
func NewProvider(e *engine.Engine, config Config) provider.Provider {
	// Create HTML provider config
	htmlConfig := web.Config{
		ID:          config.ID,
		Name:        config.Name,
		SiteURL:     config.SiteURL,
		Description: config.Description,
		Headers:     config.Headers,
	}

	htmlProvider := web.NewProvider(e, htmlConfig)

	return &Provider{
		config:       config,
		htmlProvider: htmlProvider,
		engine:       e,
		webScraper:   e.WebScraper,
	}
}

// CreateProvider is a convenience function to create a Madara provider with default configuration
func CreateProvider(e *engine.Engine, id, name, siteURL, description string) provider.Provider {
	return NewProvider(e, DefaultConfig(id, name, siteURL, description))
}

// ID returns the provider's identifier
func (p *Provider) ID() string {
	return p.htmlProvider.ID()
}

// Name returns the provider's display name
func (p *Provider) Name() string {
	return p.htmlProvider.Name()
}

// Description returns the provider's description
func (p *Provider) Description() string {
	return p.htmlProvider.Description()
}

// SiteURL returns the provider's website URL
func (p *Provider) SiteURL() string {
	return p.htmlProvider.SiteURL()
}

// Initialize initializes the provider
func (p *Provider) Initialize(ctx context.Context) error {
	return p.htmlProvider.Initialize(ctx)
}

// FetchMainPage fetches the main page
func (p *Provider) FetchMainPage(ctx context.Context) (*network.WebPage, error) {
	// Create request for the main page
	req := network.NewScraperRequest(p.config.SiteURL)
	for k, v := range p.config.Headers {
		req.SetHeader(k, v)
	}
	return p.webScraper.FetchPage(ctx, req)
}

// Search implements manga search for Madara sites
func (p *Provider) Search(ctx context.Context, query string, options core.SearchOptions) ([]core.Manga, error) {
	p.engine.Logger.Info("[%s] Searching for: %s", p.ID(), query)

	// Track whether we need to use multiple approaches to reach the requested limit
	requestedLimit := options.Limit
	if requestedLimit <= 0 {
		requestedLimit = 50 // Default to 50 if not specified
	}

	// Create a slice to hold all manga results
	var mangaList []core.Manga

	// Use a map to track unique manga IDs to avoid duplicates
	uniqueMangaIDs := make(map[string]bool)

	// Check context for concurrency settings
	concurrency := core.GetConcurrency(ctx, 1) // Default to sequential if not specified
	useParallel := concurrency > 1

	// For the list command, we should prioritize AJAX pagination to get proper results
	// But for search, we might want to try direct page first for better relevance
	if query == "" || requestedLimit > 40 || options.Pages > 1 {
		// For empty query (list command) or when requesting more than typical page size,
		// use AJAX pagination first
		p.engine.Logger.Info("[%s] Using AJAX pagination for limit=%d, pages=%d, concurrency=%d",
			p.ID(), requestedLimit, options.Pages, concurrency)

		var results []core.Manga
		var err error

		// Use parallel search if concurrency > 1, otherwise use sequential
		if useParallel {
			results, err = p.searchWithAjaxParallel(ctx, query, options, concurrency)
		} else {
			results, err = p.searchWithAjax(ctx, query, options)
		}

		if err == nil && len(results) > 0 {
			// Add results to our combined list with deduplication
			for _, manga := range results {
				if !uniqueMangaIDs[manga.ID] {
					uniqueMangaIDs[manga.ID] = true
					mangaList = append(mangaList, manga)
				}
			}

			// If we got enough results, return directly
			if len(mangaList) >= requestedLimit || options.Pages == 1 {
				p.engine.Logger.Info("[%s] Found %d manga with %s AJAX pagination (limit: %d)",
					p.ID(), len(mangaList),
					map[bool]string{true: "parallel", false: "sequential"}[useParallel],
					requestedLimit)

				// Apply the final limit if still needed, but only if not using explicit pagination
				if len(mangaList) > requestedLimit && options.Pages <= 1 {
					mangaList = mangaList[:requestedLimit]
				}

				return mangaList, nil
			}
		}
	}

	// Try direct page scraping as a fallback or additional source
	needMore := len(mangaList) < requestedLimit

	if needMore {
		// 1. Try direct page scraping (more reliable for some sites)
		page, err := p.FetchMainPage(ctx)
		if err == nil {
			// Look for manga listings with different selectors
			selectors := strings.Split(p.config.MangaSelector, ",")
			for _, selector := range selectors {
				selector = strings.TrimSpace(selector)
				if selector == "" {
					continue
				}

				elements, err := page.Find(selector)
				if err == nil && len(elements) > 0 {
					p.engine.Logger.Info("Found manga elements using selector: %s", selector)

					for _, elem := range elements {
						href := elem.Attr("href")
						if href == "" {
							continue
						}

						title := elem.Text()
						if title == "" {
							continue
						}

						// Filter by query if provided
						if query != "" && !strings.Contains(strings.ToLower(title), strings.ToLower(query)) {
							continue
						}

						// Extract manga ID from URL
						id := network.ExtractPathFromURL(href)
						if id == "" {
							continue
						}

						// Skip duplicates
						if uniqueMangaIDs[id] {
							continue
						}
						uniqueMangaIDs[id] = true

						// Add to results
						mangaList = append(mangaList, core.Manga{
							ID:    id,
							Title: title,
						})

						// Stop if we've reached the requested limit
						if len(mangaList) >= requestedLimit {
							break
						}
					}

					if len(mangaList) > 0 {
						break
					}
				}
			}
		}
	}

	// If direct scraping didn't produce results or didn't provide enough,
	// and we haven't tried AJAX yet, try that as a last resort
	needMore = len(mangaList) < requestedLimit && (query != "" && requestedLimit <= 40 && options.Pages == 1)

	if needMore && len(mangaList) == 0 {
		var ajaxResults []core.Manga
		var err error

		// Use parallel search if concurrency > 1, otherwise use sequential
		if useParallel {
			ajaxResults, err = p.searchWithAjaxParallel(ctx, query, options, concurrency)
		} else {
			ajaxResults, err = p.searchWithAjax(ctx, query, options)
		}

		if err == nil {
			// Add results to our combined list with deduplication
			for _, manga := range ajaxResults {
				if !uniqueMangaIDs[manga.ID] {
					uniqueMangaIDs[manga.ID] = true
					mangaList = append(mangaList, manga)
				}

				// Stop if we've reached the requested limit
				if len(mangaList) >= requestedLimit {
					break
				}
			}
		}
	}

	// Apply the final limit if needed, but only if not using explicit pagination
	if len(mangaList) > requestedLimit && options.Pages <= 1 {
		mangaList = mangaList[:requestedLimit]
	}

	p.engine.Logger.Info("[%s] Found total of %d manga", p.ID(), len(mangaList))
	return mangaList, nil
}

// searchWithAjax uses the WordPress AJAX API to search for manga with proper pagination
func (p *Provider) searchWithAjax(ctx context.Context, query string, options core.SearchOptions) ([]core.Manga, error) {
	// Determine requested limit
	requestedLimit := options.Limit
	if requestedLimit <= 0 {
		requestedLimit = 100
	}

	// Determine how many pages to fetch based on options
	pagesToFetch := options.Pages
	if pagesToFetch <= 0 && requestedLimit > 0 {
		// For KissManga, calculate based on 40 items per page
		itemsPerPage := 40
		pagesToFetch = (requestedLimit + itemsPerPage - 1) / itemsPerPage

		// Add 1 extra page for safety
		pagesToFetch++

		p.engine.Logger.Info("[%s] Auto-calculated %d pages needed to satisfy limit of %d (at ~%d items per page)",
			p.ID(), pagesToFetch, requestedLimit, itemsPerPage)
	}

	if pagesToFetch <= 0 {
		// Default to at least 1 page
		pagesToFetch = 1
	}

	// Create a slice to hold all manga results
	var allManga []core.Manga

	// Use a map to track unique manga IDs to avoid duplicates
	uniqueMangaIDs := make(map[string]bool)

	// Fetch each page of results
	for currentPage := 0; currentPage < pagesToFetch; currentPage++ {
		// Only stop on limit if not using explicit pagination
		if requestedLimit > 0 && len(allManga) >= requestedLimit && options.Pages <= 0 {
			p.engine.Logger.Info("[%s] Already collected %d manga (requested: %d), stopping pagination",
				p.ID(), len(allManga), requestedLimit)
			break
		} else if options.Pages > 0 {
			p.engine.Logger.Debug("[%s] Collected %d manga so far, continuing to fetch requested pages (%d)",
				p.ID(), len(allManga), options.Pages)
		}

		p.engine.Logger.Info("[%s] Fetching manga list page %d of %d (collected %d/%d so far)",
			p.ID(), currentPage+1, pagesToFetch, len(allManga), requestedLimit)

		// Create form data for the AJAX request
		formData := url.Values{}

		// Use custom action if provided, otherwise use default
		action := "madara_load_more"
		if p.config.CustomLoadAction != "" {
			action = p.config.CustomLoadAction
		}

		// KissManga uses a different pagination approach
		// The first page is 0, but later pages use a different parameter format
		formData.Set("action", action)
		formData.Set("template", "madara-core/content/content-archive")

		if currentPage == 0 {
			// First page
			formData.Set("page", "0")
			formData.Set("vars[paged]", "1") // KissManga expects paged=1 for first page
		} else {
			// Subsequent pages - KissManga expects page=N and paged=N+1
			formData.Set("page", fmt.Sprintf("%d", currentPage))
			formData.Set("vars[paged]", fmt.Sprintf("%d", currentPage+1))
		}

		// Set other parameters
		formData.Set("vars[post_type]", "wp-manga")
		formData.Set("vars[posts_per_page]", "100") // Request maximum items
		formData.Set("vars[orderby]", "date")       // Sort by date (most Madara sites default to this)
		formData.Set("vars[order]", "DESC")         // Descending order (newest first)

		if query != "" {
			formData.Set("vars[s]", query)
		}

		// Debug log the form data
		p.engine.Logger.Debug("[%s] AJAX request params for page %d: %v",
			p.ID(), currentPage+1, formData)

		// Create request
		req := network.NewScraperRequest(parser.UrlJoin(p.config.SiteURL, "wp-admin/admin-ajax.php"))
		req.SetMethod("POST")
		for k, v := range p.config.Headers {
			req.SetHeader(k, v)
		}
		req.SetHeader("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		req.SetHeader("X-Requested-With", "XMLHttpRequest")
		req.SetHeader("Origin", p.config.SiteURL)
		req.SetHeader("Referer", p.config.SiteURL)
		req.SetFormData(formData)

		// Apply rate limiting between requests (except for first page)
		if currentPage > 0 {
			time.Sleep(1500 * time.Millisecond) // Increase delay to 1.5 seconds
		}

		// Fetch page
		page, err := p.webScraper.FetchPage(ctx, req)
		if err != nil {
			p.engine.Logger.Warn("[%s] AJAX search failed for page %d: %v",
				p.ID(), currentPage+1, err)
			// If first page fails, return error; otherwise just use what we have
			if currentPage == 0 {
				return []core.Manga{}, err
			}
			break
		}

		// Try different selectors to find manga entries
		var pageResults []core.Manga
		selectors := strings.Split(p.config.MangaSelector, ",")

		// Debug - check if page content exists
		pageHTML := page.GetText()
		if pageHTML == "" {
			p.engine.Logger.Warn("[%s] Empty page content received for page %d",
				p.ID(), currentPage+1)
		} else {
			// Check if page content contains manga entries by looking for common patterns
			hasMangaContent := strings.Contains(pageHTML, "post-title") ||
				strings.Contains(pageHTML, "manga-title") ||
				strings.Contains(pageHTML, "page-item-detail")

			if !hasMangaContent {
				p.engine.Logger.Debug("[%s] Page %d doesn't appear to contain manga entries",
					p.ID(), currentPage+1)

				// Add a small excerpt of the content for debugging
				contentPreview := pageHTML
				if len(contentPreview) > 200 {
					contentPreview = contentPreview[:200] + "..."
				}
			}
		}

		for _, selector := range selectors {
			selector = strings.TrimSpace(selector)
			if selector == "" {
				continue
			}

			elements, err := page.Find(selector)
			if err != nil {
				p.engine.Logger.Debug("[%s] Selector '%s' failed: %v",
					p.ID(), selector, err)
				continue
			}

			p.engine.Logger.Debug("[%s] Selector '%s' found %d elements on page %d",
				p.ID(), selector, len(elements), currentPage+1)

			if len(elements) > 0 {
				for _, elem := range elements {
					href := elem.Attr("href")
					if href == "" {
						continue
					}

					title := elem.Text()
					if title == "" {
						continue
					}

					// Filter by query if provided
					if query != "" && !strings.Contains(strings.ToLower(title), strings.ToLower(query)) {
						continue
					}

					// Extract manga ID from URL
					id := network.ExtractPathFromURL(href)
					if id == "" {
						continue
					}

					// Check if we already have this manga (avoid duplicates)
					if uniqueMangaIDs[id] {
						continue
					}
					uniqueMangaIDs[id] = true

					// Add to results
					pageResults = append(pageResults, core.Manga{
						ID:    id,
						Title: title,
					})
				}
			}

			if len(pageResults) > 0 {
				break
			}
		}

		p.engine.Logger.Info("[%s] Found %d manga on page %d",
			p.ID(), len(pageResults), currentPage+1)

		// Add page results to the combined results
		allManga = append(allManga, pageResults...)

		// If we didn't get any results for this page, we're probably at the end
		if len(pageResults) == 0 {
			p.engine.Logger.Info("[%s] No more results found after page %d, stopping pagination",
				p.ID(), currentPage+1)
			break
		}

		// If we're at the requested page limit, stop
		if options.Pages > 0 && currentPage+1 >= options.Pages {
			p.engine.Logger.Info("[%s] Reached requested page limit of %d",
				p.ID(), options.Pages)
			break
		}

		// Check for context cancellation or timeout
		select {
		case <-ctx.Done():
			p.engine.Logger.Warn("[%s] Context cancelled or timed out after fetching %d pages",
				p.ID(), currentPage+1)
			break
		default:
			// Continue processing
		}
	}

	// Apply the final limit if requested, but only if not using explicit pagination
	// When using explicit pagination (options.Pages > 0), don't apply the limit to the combined results
	if requestedLimit > 0 && options.Pages <= 0 && len(allManga) > requestedLimit {
		p.engine.Logger.Info("[%s] Trimming results from %d to requested limit %d",
			p.ID(), len(allManga), requestedLimit)
		allManga = allManga[:requestedLimit]
	} else if options.Pages > 0 {
		p.engine.Logger.Info("[%s] Using explicit pagination (%d pages), returning all %d results",
			p.ID(), options.Pages, len(allManga))
	}

	p.engine.Logger.Info("[%s] Found total of %d manga via AJAX pagination", p.ID(), len(allManga))
	return allManga, nil
}

// searchWithAjaxParallel uses the WordPress AJAX API to search for manga with parallel page fetching
func (p *Provider) searchWithAjaxParallel(ctx context.Context, query string, options core.SearchOptions, concurrency int) ([]core.Manga, error) {
	// Determine requested limit
	requestedLimit := options.Limit
	if requestedLimit <= 0 {
		requestedLimit = 100
	}

	// Determine how many pages to fetch based on options
	pagesToFetch := options.Pages
	if pagesToFetch <= 0 {
		// For Madara sites, calculate based on 40 items per page
		itemsPerPage := 40
		pagesToFetch = (requestedLimit + itemsPerPage - 1) / itemsPerPage

		// Add 1 extra page for safety
		pagesToFetch++

		p.engine.Logger.Info("[%s] Auto-calculated %d pages needed to satisfy limit of %d (at ~%d items per page)",
			p.ID(), pagesToFetch, requestedLimit, itemsPerPage)
	}

	if pagesToFetch <= 0 {
		// Default to at least 1 page
		pagesToFetch = 1
	}

	// Ensure reasonable concurrency
	if concurrency <= 0 {
		concurrency = 1
	} else if concurrency > pagesToFetch {
		concurrency = pagesToFetch
	}

	p.engine.Logger.Info("[%s] Starting parallel search with concurrency %d, fetching %d pages",
		p.ID(), concurrency, pagesToFetch)

	// Create a slice to hold all manga results
	var allManga []core.Manga
	var resultsMutex sync.Mutex

	// Use a map to track unique manga IDs to avoid duplicates
	uniqueMangaIDs := make(map[string]bool)

	// Create a wait group to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Create a channel to limit concurrency
	semaphore := make(chan struct{}, concurrency)

	// Create a channel for errors
	errorChan := make(chan error, pagesToFetch)

	// Create a slice to hold results from each page, to maintain order
	type pageResult struct {
		page    int
		results []core.Manga
	}
	resultsChan := make(chan pageResult, pagesToFetch)

	// Launch a goroutine for each page
	for currentPage := 0; currentPage < pagesToFetch; currentPage++ {
		// Increment wait group counter
		wg.Add(1)

		// Acquire semaphore token (this blocks if we've reached max concurrency)
		semaphore <- struct{}{}

		// Launch goroutine for this page
		go func(page int) {
			// Release semaphore token and decrement wait group counter when done
			defer func() {
				<-semaphore
				wg.Done()
			}()

			p.engine.Logger.Debug("[%s] Fetching page %d of %d in parallel",
				p.ID(), page+1, pagesToFetch)

			// Create form data for the AJAX request
			formData := url.Values{}

			// Use custom action if provided, otherwise use default
			action := "madara_load_more"
			if p.config.CustomLoadAction != "" {
				action = p.config.CustomLoadAction
			}

			// Set form data based on page number
			formData.Set("action", action)
			formData.Set("template", "madara-core/content/content-archive")

			if page == 0 {
				// First page
				formData.Set("page", "0")
				formData.Set("vars[paged]", "1") // KissManga expects paged=1 for first page
			} else {
				// Subsequent pages - KissManga expects page=N and paged=N+1
				formData.Set("page", fmt.Sprintf("%d", page))
				formData.Set("vars[paged]", fmt.Sprintf("%d", page+1))
			}

			// Set other parameters
			formData.Set("vars[post_type]", "wp-manga")
			formData.Set("vars[posts_per_page]", "100") // Request maximum items
			formData.Set("vars[orderby]", "date")       // Sort by date (most Madara sites default to this)
			formData.Set("vars[order]", "DESC")         // Descending order (newest first)

			if query != "" {
				formData.Set("vars[s]", query)
			}

			// Create request
			req := network.NewScraperRequest(parser.UrlJoin(p.config.SiteURL, "wp-admin/admin-ajax.php"))
			req.SetMethod("POST")
			for k, v := range p.config.Headers {
				req.SetHeader(k, v)
			}
			req.SetHeader("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
			req.SetHeader("X-Requested-With", "XMLHttpRequest")
			req.SetHeader("Origin", p.config.SiteURL)
			req.SetHeader("Referer", p.config.SiteURL)
			req.SetFormData(formData)

			// Apply rate limiting using engine's rate limiter service
			domain := network.ExtractDomain(p.config.SiteURL)
			p.engine.RateLimiter.Wait(domain)

			// Fetch page
			fetchedPage, err := p.webScraper.FetchPage(ctx, req)
			if err != nil {
				p.engine.Logger.Warn("[%s] AJAX search failed for page %d: %v",
					p.ID(), page+1, err)
				errorChan <- err
				return
			}

			// Parse the results
			var pageResults []core.Manga
			selectors := strings.Split(p.config.MangaSelector, ",")

			for _, selector := range selectors {
				selector = strings.TrimSpace(selector)
				if selector == "" {
					continue
				}

				elements, err := fetchedPage.Find(selector)
				if err != nil {
					p.engine.Logger.Debug("[%s] Selector '%s' failed on page %d: %v",
						p.ID(), selector, page+1, err)
					continue
				}

				p.engine.Logger.Debug("[%s] Selector '%s' found %d elements on page %d",
					p.ID(), selector, len(elements), page+1)

				if len(elements) > 0 {
					for _, elem := range elements {
						href := elem.Attr("href")
						if href == "" {
							continue
						}

						title := elem.Text()
						if title == "" {
							continue
						}

						// Filter by query if provided
						if query != "" && !strings.Contains(strings.ToLower(title), strings.ToLower(query)) {
							continue
						}

						// Extract manga ID from URL
						id := network.ExtractPathFromURL(href)
						if id == "" {
							continue
						}

						// Use mutex when checking uniqueness
						resultsMutex.Lock()
						isDuplicate := uniqueMangaIDs[id]
						if !isDuplicate {
							uniqueMangaIDs[id] = true
						}
						resultsMutex.Unlock()

						// Skip if duplicate
						if isDuplicate {
							continue
						}

						// Add to page results
						pageResults = append(pageResults, core.Manga{
							ID:    id,
							Title: title,
						})
					}
				}

				if len(pageResults) > 0 {
					break
				}
			}

			p.engine.Logger.Info("[%s] Found %d manga on page %d",
				p.ID(), len(pageResults), page+1)

			// Send results to channel
			resultsChan <- pageResult{
				page:    page,
				results: pageResults,
			}

			// Check if we need to stop (e.g., no more results)
			if len(pageResults) == 0 {
				p.engine.Logger.Info("[%s] No results on page %d, may have reached end",
					p.ID(), page+1)
			}

		}(currentPage)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(semaphore)
	close(errorChan)
	close(resultsChan)

	// Check for errors (we can continue with partial results)
	var errs []error
	for err := range errorChan {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		p.engine.Logger.Warn("[%s] %d pages failed to fetch", p.ID(), len(errs))
	}

	// Process results
	resultsMap := make(map[int][]core.Manga)
	for result := range resultsChan {
		resultsMap[result.page] = result.results
	}

	// Combine results in correct order
	for i := 0; i < pagesToFetch; i++ {
		if results, ok := resultsMap[i]; ok {
			allManga = append(allManga, results...)
		}
	}

	// Apply rhe final limit if requested, but only if not using explicit pagination
	if options.Pages <= 0 && len(allManga) > requestedLimit {
		p.engine.Logger.Info("[%s] Trimming results from %d to requested limit %d",
			p.ID(), len(allManga), requestedLimit)
		allManga = allManga[:requestedLimit]
	}

	p.engine.Logger.Info("[%s] Found total of %d manga via parallel AJAX pagination",
		p.ID(), len(allManga))

	return allManga, nil
}

// cleanMadaraDescription cleans up a manga description from Madara-based sites
// It removes excessive whitespace, "Show more" buttons, and common boilerplate text
func cleanMadaraDescription(description string) string {
	if description == "" {
		return ""
	}

	// Check if it's a KissManga-style description with boilerplate text
	if strings.Contains(description, "is a Manga/Manhwa/Manhua in english language") &&
		strings.Contains(description, "updating comic site. The Summary is") {

		// Extract just the actual description part
		parts := strings.Split(description, "The Summary is")
		if len(parts) > 1 {
			description = parts[1]
		}
	}

	// Remove "Show more" text and surrounding whitespace
	description = strings.ReplaceAll(description, "Show more", "")

	// Replace multiple newlines with a single newline
	re := regexp.MustCompile(`\n\s*\n`)
	description = re.ReplaceAllString(description, "\n")

	// Replace multiple spaces with a single space
	re = regexp.MustCompile(`[ \t]+`)
	description = re.ReplaceAllString(description, " ")

	// Trim whitespace from the beginning and end
	description = strings.TrimSpace(description)

	return description
}

// GetManga retrieves manga details
func (p *Provider) GetManga(ctx context.Context, id string) (*core.MangaInfo, error) {
	p.engine.Logger.Info("[%s] Getting manga details for: %s", p.ID(), id)

	// Create a basic MangaInfo
	mangaInfo := &core.MangaInfo{
		Manga: core.Manga{
			ID:    id,
			Title: "", // Will be filled by the page title
		},
	}

	// Fetch the manga page
	page, err := p.htmlProvider.FetchPage(ctx, parser.UrlJoin(p.config.SiteURL, id))
	if err != nil {
		return nil, &errors.ProviderError{
			ProviderID:   p.ID(),
			ResourceType: "manga",
			ResourceID:   id,
			Message:      "Failed to fetch manga page",
			Err:          err,
		}
	}

	// Get manga title (try multiple selectors)
	titleSelectors := []string{
		".post-title h1",
		".entry-title",
		"h1",
		"title",
	}

	for _, selector := range titleSelectors {
		titleElem, err := page.FindOne(selector)
		if err == nil && titleElem != nil {
			mangaInfo.Title = strings.TrimSpace(titleElem.Text())
			if mangaInfo.Title != "" {
				break
			}
		}
	}

	// If still no title, use page title
	if mangaInfo.Title == "" {
		mangaInfo.Title = page.GetTitle()
	}

	// Get manga description (try multiple selectors)
	descSelectors := []string{
		".description-summary",
		".summary__content",
		".manga-excerpt",
		".post-content_item:contains('Summary') .summary-content",
		".summary-content",
		".summary p",
	}

	for _, selector := range descSelectors {
		descElement, err := page.FindOne(selector)
		if err == nil && descElement != nil {
			rawDescription := strings.TrimSpace(descElement.Text())
			if rawDescription != "" {
				// Clean up the description to remove "Show more" and excessive whitespace
				mangaInfo.Description = cleanMadaraDescription(rawDescription)
				if mangaInfo.Description != "" {
					break
				}
			}
		}
	}

	// If we couldn't find a description with selectors, try to find it in the page content
	if mangaInfo.Description == "" {
		// Extract potential description from page content
		pageText := page.GetText()
		if strings.Contains(pageText, "The Summary is") {
			parts := strings.Split(pageText, "The Summary is")
			if len(parts) > 1 {
				rawDescription := parts[1]
				// Clean up the extracted description
				mangaInfo.Description = cleanMadaraDescription(rawDescription)
			}
		}
	}

	// Get manga authors (try multiple selectors)
	authorSelectors := []string{
		".author-content a",
		".post-content_item:contains('Author') .summary-content",
		".post-content_item:contains('Art') .summary-content",
	}

	var authors []string
	for _, selector := range authorSelectors {
		authorElements, err := page.Find(selector)
		if err == nil && len(authorElements) > 0 {
			for _, authorElement := range authorElements {
				authorName := strings.TrimSpace(authorElement.Text())
				if authorName != "" {
					authors = append(authors, authorName)
				}
			}
		}
	}
	mangaInfo.Authors = authors

	// Get manga tags/genres (try multiple selectors)
	tagSelectors := []string{
		".genres-content a",
		".post-content_item:contains('Genre') .summary-content a",
		".post-content_item:contains('Tag') .summary-content a",
	}

	var tags []string
	for _, selector := range tagSelectors {
		tagElements, err := page.Find(selector)
		if err == nil && len(tagElements) > 0 {
			for _, tagElement := range tagElements {
				tagName := strings.TrimSpace(tagElement.Text())
				if tagName != "" {
					tags = append(tags, tagName)
				}
			}
		}
	}
	mangaInfo.Tags = tags

	// Get manga status (try multiple selectors)
	statusSelectors := []string{
		".post-status .post-content_item:contains('Status') .summary-content",
		".post-content_item:contains('Status') .summary-content",
	}

	for _, selector := range statusSelectors {
		statusElement, err := page.FindOne(selector)
		if err == nil && statusElement != nil {
			mangaInfo.Status = strings.TrimSpace(statusElement.Text())
			if mangaInfo.Status != "" {
				break
			}
		}
	}

	// Get chapters
	chapterSelectors := strings.Split(p.config.ChapterSelector, ",")
	var chapters []core.ChapterInfo

	// Try each chapter selector
	for _, selector := range chapterSelectors {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			continue
		}

		chapterElements, err := page.Find(selector)
		if err == nil && len(chapterElements) > 0 {
			for _, elem := range chapterElements {
				href := elem.Attr("href")
				if href == "" {
					continue
				}

				// Extract chapter ID
				chapterID := network.ExtractPathFromURL(href)
				if chapterID == "" {
					continue
				}

				// Get chapter title
				title := strings.TrimSpace(elem.Text())

				// Extract chapter number from title or URL
				chapterNumber := parser.ExtractChapterNumber(title)
				if chapterNumber == 0 {
					chapterNumber = parser.ExtractChapterNumber(chapterID)
				}

				// Get chapter date
				dateText := ""
				dateElement, err := elem.FindOne("span.chapter-release-date")
				if err == nil && dateElement != nil {
					dateText = strings.TrimSpace(dateElement.Text())
				}

				// Parse date if available
				var date time.Time
				if dateText != "" {
					// Try various date formats
					dateFormats := []string{
						"January 2, 2006",
						"Jan 2, 2006",
						"2006-01-02",
						"01/02/2006",
					}

					for _, format := range dateFormats {
						date, _ = time.Parse(format, dateText)
						if !date.IsZero() {
							break
						}
					}
				}

				// If date is still zero, use current time
				if date.IsZero() {
					date = time.Now()
				}

				chapter := core.ChapterInfo{
					ID:     chapterID,
					Title:  title,
					Number: chapterNumber,
					Date:   date,
				}

				chapters = append(chapters, chapter)
			}

			if len(chapters) > 0 {
				break
			}
		}
	}

	// If no chapters found via direct scraping, try AJAX
	if len(chapters) == 0 {
		// Try to find the chapter placeholder for AJAX loading
		placeholders, err := page.Find("[id^=\"manga-chapters-holder\"][data-id]")
		if err == nil && len(placeholders) > 0 {
			for _, placeholder := range placeholders {
				dataID := placeholder.Attr("data-id")
				if dataID != "" {
					chapters, _ = p.fetchChaptersViaAjax(ctx, id, dataID)
					if len(chapters) > 0 {
						break
					}
				}
			}
		}
	}

	mangaInfo.Chapters = chapters

	// If we still don't have a title, use the ID
	if mangaInfo.Title == "" {
		mangaInfo.Title = id
	}

	return mangaInfo, nil
}

// fetchChaptersViaAjax retrieves chapters via AJAX
func (p *Provider) fetchChaptersViaAjax(ctx context.Context, mangaID, dataID string) ([]core.ChapterInfo, error) {
	// Try both AJAX methods, starting with the newer one

	// 1. Try the new endpoint (mangaID/ajax/chapters/)
	newEndpoint := parser.UrlJoin(p.config.SiteURL, mangaID, "ajax/chapters/")
	req := network.NewScraperRequest(newEndpoint)
	req.SetMethod("POST")
	for k, v := range p.config.Headers {
		req.SetHeader(k, v)
	}
	req.SetHeader("X-Requested-With", "XMLHttpRequest")

	page, err := p.webScraper.FetchPage(ctx, req)
	if err == nil {
		chapters := p.extractChaptersFromPage(page, mangaID)
		if len(chapters) > 0 {
			return chapters, nil
		}
	}

	// 2. Try the old endpoint (wp-admin/admin-ajax.php)
	if !p.config.UseLegacyAjax {
		oldEndpoint := parser.UrlJoin(p.config.SiteURL, "wp-admin/admin-ajax.php")
		formData := url.Values{}
		formData.Set("action", "manga_get_chapters")
		formData.Set("manga", dataID)

		req = network.NewScraperRequest(oldEndpoint)
		req.SetMethod("POST")
		for k, v := range p.config.Headers {
			req.SetHeader(k, v)
		}
		req.SetHeader("Content-Type", "application/x-www-form-urlencoded")
		req.SetHeader("X-Requested-With", "XMLHttpRequest")
		req.SetFormData(formData)

		page, err = p.webScraper.FetchPage(ctx, req)
		if err == nil {
			chapters := p.extractChaptersFromPage(page, mangaID)
			if len(chapters) > 0 {
				return chapters, nil
			}
		}
	}

	return []core.ChapterInfo{}, nil
}

// extractChaptersFromPage extracts chapter information from a page
func (p *Provider) extractChaptersFromPage(page *network.WebPage, mangaID string) []core.ChapterInfo {
	var chapters []core.ChapterInfo

	// Try each chapter selector
	chapterSelectors := strings.Split(p.config.ChapterSelector, ",")
	for _, selector := range chapterSelectors {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			continue
		}

		chapterElements, err := page.Find(selector)
		if err == nil && len(chapterElements) > 0 {
			for _, elem := range chapterElements {
				href := elem.Attr("href")
				if href == "" {
					continue
				}

				// Extract chapter ID
				chapterID := network.ExtractPathFromURL(href)
				if chapterID == "" {
					continue
				}

				// Get chapter title
				title := strings.TrimSpace(elem.Text())

				// Extract chapter number from title or URL
				chapterNumber := parser.ExtractChapterNumber(title)
				if chapterNumber == 0 {
					chapterNumber = parser.ExtractChapterNumber(chapterID)
				}

				chapter := core.ChapterInfo{
					ID:     chapterID,
					Title:  title,
					Number: chapterNumber,
					Date:   time.Now(), // We'd need more parsing to get the actual date
				}

				chapters = append(chapters, chapter)
			}

			if len(chapters) > 0 {
				break
			}
		}
	}

	return chapters
}

// GetChapter retrieves chapter details
func (p *Provider) GetChapter(ctx context.Context, chapterID string) (*core.Chapter, error) {
	p.engine.Logger.Info("[%s] Getting chapter details for: %s", p.ID(), chapterID)

	// Create a basic Chapter
	chapter := &core.Chapter{
		Info: core.ChapterInfo{
			ID:    chapterID,
			Title: "",
		},
	}

	// Extract chapter number from the ID
	chapter.Info.Number = parser.ExtractChapterNumber(chapterID)

	// Fetch the chapter page
	// First try with style=list
	chapterURL := parser.UrlJoin(p.config.SiteURL, chapterID)
	if !strings.Contains(chapterURL, "?") {
		chapterURL += "?style=list"
	} else {
		chapterURL += "&style=list"
	}

	page, err := p.htmlProvider.FetchPage(ctx, chapterURL)

	// If that fails, try without style=list
	if err != nil {
		chapterURL = parser.UrlJoin(p.config.SiteURL, chapterID)
		cleanChapterUrl := util.CleanImageURL(chapterURL)
		page, err = p.htmlProvider.FetchPage(ctx, cleanChapterUrl)
		if err != nil {
			return nil, &errors.ProviderError{
				ProviderID:   p.ID(),
				ResourceType: "chapter",
				ResourceID:   chapterID,
				Message:      "Failed to fetch chapter page",
				Err:          err,
			}
		}
	}

	// Get chapter title
	titleSelectors := []string{
		"h1.wp-manga-title",
		"h1.entry-title",
		"h1",
	}

	for _, selector := range titleSelectors {
		titleElem, err := page.FindOne(selector)
		if err == nil && titleElem != nil {
			chapter.Info.Title = strings.TrimSpace(titleElem.Text())
			if chapter.Info.Title != "" {
				break
			}
		}
	}

	// If still no title, use page title
	if chapter.Info.Title == "" {
		chapter.Info.Title = page.GetTitle()
	}

	// Try multiple selectors for images
	var pages []core.Page
	pageSelectors := strings.Split(p.config.PageSelector, ",")

	for _, selector := range pageSelectors {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			continue
		}

		imageElements, err := page.Find(selector)
		if err == nil && len(imageElements) > 0 {
			for i, elem := range imageElements {
				// Try various attributes for image URL
				imageURL := elem.Attr("src")
				if imageURL == "" {
					imageURL = elem.Attr("data-src")
				}
				if imageURL == "" {
					imageURL = elem.Attr("data-url")
				}
				if imageURL == "" {
					imageURL = elem.Attr("data-lazy-src")
				}

				// Skip if no image URL found
				if imageURL == "" {
					continue
				}

				// trim the URL and remove special characters like \n
				imageURL = strings.TrimSpace(imageURL)
				imageURL = util.CleanImageURL(imageURL)

				// Make absolute URL if needed
				if !strings.HasPrefix(imageURL, "http") {
					imageURL = parser.UrlJoin(p.config.SiteURL, imageURL)
				}

				// Extract filename from URL
				urlParts := strings.Split(imageURL, "/")
				filename := urlParts[len(urlParts)-1]
				if filename == "" {
					filename = fmt.Sprintf("page_%03d.jpg", i+1)
				}

				// Create page
				p := core.Page{
					Index:    i,
					URL:      imageURL,
					Filename: filename,
				}

				pages = append(pages, p)
			}

			if len(pages) > 0 {
				break
			}
		}
	}

	// Set the pages
	chapter.Pages = pages

	// Try to extract the manga ID from the chapter URL
	parts := strings.Split(chapterID, "/")
	if len(parts) > 0 {
		chapter.MangaID = parts[0]
	}

	return chapter, nil
}

// TryGetMangaForChapter attempts to get manga info for a chapter
func (p *Provider) TryGetMangaForChapter(ctx context.Context, chapterID string) (*core.Manga, error) {
	// Get the chapter to extract manga ID
	chapter, err := p.GetChapter(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	// If manga ID is available in chapter
	if chapter.MangaID != "" {
		// Get manga details
		mangaInfo, err := p.GetManga(ctx, chapter.MangaID)
		if err != nil {
			return nil, err
		}
		return &mangaInfo.Manga, nil
	}

	return nil, fmt.Errorf("couldn't determine manga for chapter %s", chapterID)
}

// DownloadChapter downloads a chapter
func (p *Provider) DownloadChapter(ctx context.Context, chapterID, destDir string) error {
	return common.ExecuteDownloadChapter(
		ctx,
		p.engine,
		p.ID(),
		p.Name(),
		chapterID,
		destDir,
		p.GetChapter,
		p.TryGetMangaForChapter,
	)
}
