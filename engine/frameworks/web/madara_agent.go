package web

import (
	"Luminary/engine"
	"Luminary/engine/frameworks/common"
	"Luminary/errors"
	"Luminary/utils"
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// MadaraAgent implements the engine.Agent interface for Madara-based sites
type MadaraAgent struct {
	config     MadaraConfig
	htmlAgent  *HTMLAgent
	engine     *engine.Engine
	webScraper *engine.WebScraperService
}

// NewMadaraAgent creates a new Madara-based agent
func NewMadaraAgent(e *engine.Engine, config MadaraConfig) engine.Agent {
	// Create HTML agent config
	htmlConfig := HTMLAgentConfig{
		ID:          config.ID,
		Name:        config.Name,
		SiteURL:     config.SiteURL,
		Description: config.Description,
		Headers:     config.Headers,
	}

	htmlAgent := NewHTMLAgent(e, htmlConfig)

	return &MadaraAgent{
		config:     config,
		htmlAgent:  htmlAgent,
		engine:     e,
		webScraper: e.WebScraper,
	}
}

// CreateMadaraAgent is a convenience function to create a Madara agent with default configuration
func CreateMadaraAgent(e *engine.Engine, id, name, siteURL, description string) engine.Agent {
	return NewMadaraAgent(e, DefaultMadaraConfig(id, name, siteURL, description))
}

// ID returns the agent's identifier
func (a *MadaraAgent) ID() string {
	return a.htmlAgent.ID()
}

// Name returns the agent's display name
func (a *MadaraAgent) Name() string {
	return a.htmlAgent.Name()
}

// Description returns the agent's description
func (a *MadaraAgent) Description() string {
	return a.htmlAgent.Description()
}

// SiteURL returns the agent's website URL
func (a *MadaraAgent) SiteURL() string {
	return a.htmlAgent.SiteURL()
}

// Initialize initializes the agent
func (a *MadaraAgent) Initialize(ctx context.Context) error {
	return a.htmlAgent.Initialize(ctx)
}

// FetchMainPage fetches the main page
func (a *MadaraAgent) FetchMainPage(ctx context.Context) (*engine.WebPage, error) {
	// Create request for the main page
	req := engine.NewScraperRequest(a.config.SiteURL)
	for k, v := range a.config.Headers {
		req.SetHeader(k, v)
	}
	return a.webScraper.FetchPage(ctx, req)
}

// Search implements manga search for Madara sites
func (a *MadaraAgent) Search(ctx context.Context, query string, options engine.SearchOptions) ([]engine.Manga, error) {
	a.engine.Logger.Info("[%s] Searching for: %s", a.ID(), query)

	// Try multiple approaches to find manga

	// 1. First try direct page scraping (more reliable for some sites)
	page, err := a.FetchMainPage(ctx)
	if err == nil {
		// Look for manga listings with different selectors
		var mangaList []engine.Manga

		// Split the selector string and try each one
		selectors := strings.Split(a.config.MangaSelector, ",")
		for _, selector := range selectors {
			selector = strings.TrimSpace(selector)
			if selector == "" {
				continue
			}

			elements, err := page.Find(selector)
			if err == nil && len(elements) > 0 {
				a.engine.Logger.Info("Found manga elements using selector: %s", selector)

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
					id := engine.ExtractPathFromURL(href)
					if id == "" {
						continue
					}

					// Add to results
					mangaList = append(mangaList, engine.Manga{
						ID:    id,
						Title: title,
					})
				}

				if len(mangaList) > 0 {
					a.engine.Logger.Info("Found %d manga by direct page scraping", len(mangaList))
					return mangaList, nil
				}
			}
		}
	}

	// 2. If direct scraping failed, try AJAX requests (common in Madara sites)
	return a.searchWithAjax(ctx, query, options)
}

// searchWithAjax uses the WordPress AJAX API to search for manga
func (a *MadaraAgent) searchWithAjax(ctx context.Context, query string, options engine.SearchOptions) ([]engine.Manga, error) {
	// Determine limit
	limit := options.Limit
	if limit <= 0 {
		limit = 100
	}

	// Create form data for the AJAX request
	formData := url.Values{}

	// Use custom action if provided, otherwise use default
	action := "madara_load_more"
	if a.config.CustomLoadAction != "" {
		action = a.config.CustomLoadAction
	}

	formData.Set("action", action)
	formData.Set("template", "madara-core/content/content-archive")
	formData.Set("page", "0")
	formData.Set("vars[paged]", "0")
	formData.Set("vars[post_type]", "wp-manga")
	formData.Set("vars[posts_per_page]", fmt.Sprintf("%d", limit))

	if query != "" {
		formData.Set("vars[s]", query)
	}

	// Create request
	req := engine.NewScraperRequest(engine.UrlJoin(a.config.SiteURL, "wp-admin/admin-ajax.php"))
	req.SetMethod("POST")
	for k, v := range a.config.Headers {
		req.SetHeader(k, v)
	}
	req.SetHeader("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.SetHeader("X-Requested-With", "XMLHttpRequest")
	req.SetHeader("Origin", a.config.SiteURL)
	req.SetHeader("Referer", a.config.SiteURL)
	req.SetFormData(formData)

	// Fetch page
	page, err := a.webScraper.FetchPage(ctx, req)
	if err != nil {
		a.engine.Logger.Warn("AJAX search failed: %v", err)
		return []engine.Manga{}, nil
	}

	// Try different selectors to find manga entries
	var mangaList []engine.Manga
	selectors := strings.Split(a.config.MangaSelector, ",")

	for _, selector := range selectors {
		selector = strings.TrimSpace(selector)
		if selector == "" {
			continue
		}

		elements, err := page.Find(selector)
		if err == nil && len(elements) > 0 {
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
				id := engine.ExtractPathFromURL(href)
				if id == "" {
					continue
				}

				mangaList = append(mangaList, engine.Manga{
					ID:    id,
					Title: title,
				})
			}

			if len(mangaList) > 0 {
				a.engine.Logger.Info("Found %d manga via AJAX", len(mangaList))
				break
			}
		}
	}

	return mangaList, nil
}

// GetManga retrieves manga details
func (a *MadaraAgent) GetManga(ctx context.Context, id string) (*engine.MangaInfo, error) {
	a.engine.Logger.Info("[%s] Getting manga details for: %s", a.ID(), id)

	// Create a basic MangaInfo
	mangaInfo := &engine.MangaInfo{
		Manga: engine.Manga{
			ID:    id,
			Title: "", // Will be filled by the page title
		},
	}

	// Fetch the manga page
	page, err := a.htmlAgent.FetchPage(ctx, engine.UrlJoin(a.config.SiteURL, id))
	if err != nil {
		return nil, &errors.AgentError{
			AgentID:      a.ID(),
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
	}

	for _, selector := range descSelectors {
		descElement, err := page.FindOne(selector)
		if err == nil && descElement != nil {
			mangaInfo.Description = strings.TrimSpace(descElement.Text())
			if mangaInfo.Description != "" {
				break
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
	chapterSelectors := strings.Split(a.config.ChapterSelector, ",")
	var chapters []engine.ChapterInfo

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
				chapterID := engine.ExtractPathFromURL(href)
				if chapterID == "" {
					continue
				}

				// Get chapter title
				title := strings.TrimSpace(elem.Text())

				// Extract chapter number from title or URL
				chapterNumber := engine.ExtractChapterNumber(title)
				if chapterNumber == 0 {
					chapterNumber = engine.ExtractChapterNumber(chapterID)
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

				chapter := engine.ChapterInfo{
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
		// Try to find the chapters placeholder for AJAX loading
		placeholders, err := page.Find("[id^=\"manga-chapters-holder\"][data-id]")
		if err == nil && len(placeholders) > 0 {
			for _, placeholder := range placeholders {
				dataID := placeholder.Attr("data-id")
				if dataID != "" {
					chapters, _ = a.fetchChaptersViaAjax(ctx, id, dataID)
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
func (a *MadaraAgent) fetchChaptersViaAjax(ctx context.Context, mangaID, dataID string) ([]engine.ChapterInfo, error) {
	// Try both AJAX methods, starting with the newer one

	// 1. Try the new endpoint (mangaID/ajax/chapters/)
	newEndpoint := engine.UrlJoin(a.config.SiteURL, mangaID, "ajax/chapters/")
	req := engine.NewScraperRequest(newEndpoint)
	req.SetMethod("POST")
	for k, v := range a.config.Headers {
		req.SetHeader(k, v)
	}
	req.SetHeader("X-Requested-With", "XMLHttpRequest")

	page, err := a.webScraper.FetchPage(ctx, req)
	if err == nil {
		chapters := a.extractChaptersFromPage(page, mangaID)
		if len(chapters) > 0 {
			return chapters, nil
		}
	}

	// 2. Try the old endpoint (wp-admin/admin-ajax.php)
	if !a.config.UseLegacyAjax {
		oldEndpoint := engine.UrlJoin(a.config.SiteURL, "wp-admin/admin-ajax.php")
		formData := url.Values{}
		formData.Set("action", "manga_get_chapters")
		formData.Set("manga", dataID)

		req = engine.NewScraperRequest(oldEndpoint)
		req.SetMethod("POST")
		for k, v := range a.config.Headers {
			req.SetHeader(k, v)
		}
		req.SetHeader("Content-Type", "application/x-www-form-urlencoded")
		req.SetHeader("X-Requested-With", "XMLHttpRequest")
		req.SetFormData(formData)

		page, err = a.webScraper.FetchPage(ctx, req)
		if err == nil {
			chapters := a.extractChaptersFromPage(page, mangaID)
			if len(chapters) > 0 {
				return chapters, nil
			}
		}
	}

	return []engine.ChapterInfo{}, nil
}

// extractChaptersFromPage extracts chapter information from a page
func (a *MadaraAgent) extractChaptersFromPage(page *engine.WebPage, mangaID string) []engine.ChapterInfo {
	var chapters []engine.ChapterInfo

	// Try each chapter selector
	chapterSelectors := strings.Split(a.config.ChapterSelector, ",")
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
				chapterID := engine.ExtractPathFromURL(href)
				if chapterID == "" {
					continue
				}

				// Get chapter title
				title := strings.TrimSpace(elem.Text())

				// Extract chapter number from title or URL
				chapterNumber := engine.ExtractChapterNumber(title)
				if chapterNumber == 0 {
					chapterNumber = engine.ExtractChapterNumber(chapterID)
				}

				chapter := engine.ChapterInfo{
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
func (a *MadaraAgent) GetChapter(ctx context.Context, chapterID string) (*engine.Chapter, error) {
	a.engine.Logger.Info("[%s] Getting chapter details for: %s", a.ID(), chapterID)

	// Create a basic Chapter
	chapter := &engine.Chapter{
		Info: engine.ChapterInfo{
			ID:    chapterID,
			Title: "",
		},
	}

	// Extract chapter number from the ID
	chapter.Info.Number = engine.ExtractChapterNumber(chapterID)

	// Fetch the chapter page
	// First try with style=list
	chapterURL := engine.UrlJoin(a.config.SiteURL, chapterID)
	if !strings.Contains(chapterURL, "?") {
		chapterURL += "?style=list"
	} else {
		chapterURL += "&style=list"
	}

	page, err := a.htmlAgent.FetchPage(ctx, chapterURL)

	// If that fails, try without style=list
	if err != nil {
		chapterURL = engine.UrlJoin(a.config.SiteURL, chapterID)
		cleanChapterUrl := utils.CleanImageURL(chapterURL)
		page, err = a.htmlAgent.FetchPage(ctx, cleanChapterUrl)
		if err != nil {
			return nil, &errors.AgentError{
				AgentID:      a.ID(),
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
	var pages []engine.Page
	pageSelectors := strings.Split(a.config.PageSelector, ",")

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
				imageURL = utils.CleanImageURL(imageURL)

				// Make absolute URL if needed
				if !strings.HasPrefix(imageURL, "http") {
					imageURL = engine.UrlJoin(a.config.SiteURL, imageURL)
				}

				// Extract filename from URL
				urlParts := strings.Split(imageURL, "/")
				filename := urlParts[len(urlParts)-1]
				if filename == "" {
					filename = fmt.Sprintf("page_%03d.jpg", i+1)
				}

				// Create page
				p := engine.Page{
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
func (a *MadaraAgent) TryGetMangaForChapter(ctx context.Context, chapterID string) (*engine.Manga, error) {
	// Get the chapter to extract manga ID
	chapter, err := a.GetChapter(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	// If manga ID is available in chapter
	if chapter.MangaID != "" {
		// Get manga details
		mangaInfo, err := a.GetManga(ctx, chapter.MangaID)
		if err != nil {
			return nil, err
		}
		return &mangaInfo.Manga, nil
	}

	return nil, fmt.Errorf("couldn't determine manga for chapter %s", chapterID)
}

// DownloadChapter downloads a chapter
func (a *MadaraAgent) DownloadChapter(ctx context.Context, chapterID, destDir string) error {
	return common.ExecuteDownloadChapter(
		ctx,
		a.engine,
		a.ID(),
		a.Name(),
		chapterID,
		destDir,
		a.GetChapter,
		a.TryGetMangaForChapter,
	)
}
