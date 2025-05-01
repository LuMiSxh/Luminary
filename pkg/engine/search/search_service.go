package search

import (
	"Luminary/pkg/engine/core"
	"Luminary/pkg/engine/logger"
	"Luminary/pkg/engine/network"
	"Luminary/pkg/engine/parser"
	"Luminary/pkg/provider"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Service provides centralized search capabilities
type Service struct {
	Logger      *logger.Service
	API         *network.APIService
	Extractor   *parser.ExtractorService
	Pagination  *PaginationService
	RateLimiter *network.RateLimiterService
}

// NewSearchService creates a new SearchService with the necessary dependencies
func NewSearchService(
	logger *logger.Service,
	api *network.APIService,
	extractor *parser.ExtractorService,
	pagination *PaginationService,
	rateLimiter *network.RateLimiterService,
) *Service {
	return &Service{
		Logger:      logger,
		API:         api,
		Extractor:   extractor,
		Pagination:  pagination,
		RateLimiter: rateLimiter,
	}
}

// ExecuteSearch handles the common pattern for searching by delegating to the SearchService
func (s *Service) ExecuteSearch(
	ctx context.Context,
	providerID string,
	query string,
	options *core.SearchOptions,
	apiConfig network.APIConfig,
	paginationConfig PaginationConfig,
	extractorSet parser.ExtractorSet,
) ([]core.Manga, error) {
	// ... (keep existing implementation)
	s.Logger.Info("[%s] Searching for: %s (limit: %d, pages: %d)", providerID, query, options.Limit, options.Pages)

	// Apply rate limiting
	domain := network.ExtractDomain(apiConfig.BaseURL)
	s.RateLimiter.Wait(domain)

	// Determine maximum pages to fetch
	maxPages := options.Pages
	if maxPages == 0 {
		// If pages is 0, fetch all pages by setting maxPages to 0
		s.Logger.Info("[%s] Unlimited page fetching enabled. This may take some time...", providerID)
	}

	// Use pagination service to fetch results
	params := PaginatedRequestParams{
		Config:       paginationConfig,
		APIConfig:    apiConfig,
		EndpointName: "search",
		BaseParams:   options,
		PathParams:   []string{},
		ExtractorSet: extractorSet,
		MaxPages:     maxPages,
		ThrottleTime: 500 * time.Millisecond,
	}

	// If a query is provided, modify options to include it
	if query != "" {
		searchOpts := options
		searchOpts.Query = query
		params.BaseParams = searchOpts
	}

	resultsInterface, err := s.Pagination.FetchAllPages(ctx, params)
	if err != nil {
		s.Logger.Error("[%s] Search error: %v", providerID, err)
		return nil, fmt.Errorf("search error: %w", err)
	}

	// Convert to Manga type
	results := make([]core.Manga, 0, len(resultsInterface))
	for _, item := range resultsInterface {
		if manga, ok := item.(*core.Manga); ok {
			results = append(results, *manga)
		}
	}

	// Apply filters if they exist
	if len(options.Filters) > 0 {
		results = s.FilterResults(results, options.Filters)
	}

	// Apply sorting if specified
	if options.Sort != "" && options.Sort != "relevance" {
		results = s.SortResults(results, options.Sort)
	}

	// Apply final limit if specified
	// Note: This is different from the per-page limit handled by pagination
	// This applies to the total number of results after all pages are fetched
	if options.Limit > 0 && options.Pages != 1 && len(results) > options.Limit*options.Pages {
		results = results[:options.Limit*options.Pages]
	}

	s.Logger.Info("[%s] Found %d results for: %s", providerID, len(results), query)
	return results, nil
}

// SearchAcrossProviders performs a search across a given list of providers
func (s *Service) SearchAcrossProviders(
	ctx context.Context,
	// Pass the specific providers needed, not the whole engine
	providersToSearch []provider.Provider,
	query string,
	options core.SearchOptions,
) (map[string][]core.Manga, error) {
	results := make(map[string][]core.Manga)
	var mu sync.Mutex
	var wg sync.WaitGroup

	if len(providersToSearch) == 0 {
		s.Logger.Info("No providers specified or found for search.")
		return results, nil
	}

	s.Logger.Info("Performing search across %d providers for: %s", len(providersToSearch), query)

	errorChan := make(chan error, len(providersToSearch))
	searchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	maxConcurrent := core.GetConcurrency(searchCtx, 5) // Use searchCtx here
	s.Logger.Debug("Using concurrency limit of %d from context", maxConcurrent)
	semaphore := make(chan struct{}, maxConcurrent)

	for _, prov := range providersToSearch {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(p provider.Provider) {
			defer wg.Done()
			defer func() { <-semaphore }()

			select {
			case <-searchCtx.Done():
				s.Logger.Warn("[%s] Search cancelled or timed out before starting: %v", p.ID(), searchCtx.Err())
				return
			default:
			}

			// Call Initialize directly. All providers MUST implement this.
			// If a provider needs no initialization, its Initialize method should simply return nil.
			s.Logger.Debug("[%s] Initializing provider...", p.ID())
			if err := p.Initialize(searchCtx); err != nil {
				// Check if the error is due to context cancellation
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					s.Logger.Warn("[%s] Initialization cancelled or timed out: %v", p.ID(), err)
				} else {
					errorChan <- fmt.Errorf("failed to initialize provider %s: %w", p.ID(), err)
				}
				return
			}
			s.Logger.Debug("[%s] Initialization complete.", p.ID())

			// Check context again after potentially long initialization
			select {
			case <-searchCtx.Done():
				s.Logger.Warn("[%s] Search cancelled or timed out after initialization: %v", p.ID(), searchCtx.Err())
				return
			default:
			}

			// Call Search directly. All providers MUST implement this.
			s.Logger.Debug("[%s] Executing search...", p.ID())
			providerResults, err := p.Search(searchCtx, query, options)
			if err != nil {
				// Check if the error is due to context cancellation/timeout
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					s.Logger.Warn("[%s] Search cancelled or timed out: %v", p.ID(), err)
				} else {
					errorChan <- fmt.Errorf("search error for provider %s: %w", p.ID(), err)
				}
				return
			}
			s.Logger.Debug("[%s] Search returned %d results.", p.ID(), len(providerResults))

			// Add results to the map
			mu.Lock()
			results[p.ID()] = providerResults
			mu.Unlock()
		}(prov)
	}

	wg.Wait()
	close(errorChan)

	var collectedErrors []error
	for err := range errorChan {
		collectedErrors = append(collectedErrors, err)
	}

	if len(collectedErrors) > 0 {
		s.Logger.Warn("Search completed with errors from %d providers:", len(collectedErrors))
		for _, err := range collectedErrors {
			s.Logger.Warn("- %v", err)
		}
		// Optionally return a combined error, e.g., using errors.Join (Go 1.20+)
		// return results, errors.Join(collectedErrors...)
		// For now, just logging and returning partial results
	} else {
		s.Logger.Info("Search across providers completed successfully.")
	}

	return results, nil // Return partial results even if some providers failed
}

// FilterResults remains unchanged...
func (s *Service) FilterResults(results []core.Manga, filters map[string]string) []core.Manga {
	// ... (keep existing implementation)
	if len(filters) == 0 {
		return results
	}

	filtered := make([]core.Manga, 0, len(results))

	for _, manga := range results {
		matches := true

		// Check each filter against manga properties
		for field, value := range filters {
			lowerValue := strings.ToLower(value)

			switch strings.ToLower(field) {
			case "title":
				if !strings.Contains(strings.ToLower(manga.Title), lowerValue) {
					matches = false
				}
			case "author":
				authorMatch := false
				for _, author := range manga.Authors {
					if strings.Contains(strings.ToLower(author), lowerValue) {
						authorMatch = true
						break
					}
				}
				if !authorMatch {
					matches = false
				}
			case "status":
				if !strings.EqualFold(manga.Status, value) {
					matches = false
				}
			case "tag", "genre":
				tagMatch := false
				for _, tag := range manga.Tags {
					if strings.Contains(strings.ToLower(tag), lowerValue) {
						tagMatch = true
						break
					}
				}
				if !tagMatch {
					matches = false
				}
			}

			if !matches {
				break
			}
		}

		if matches {
			filtered = append(filtered, manga)
		}
	}

	return filtered
}

// SortResults remains unchanged...
func (s *Service) SortResults(results []core.Manga, sortBy string) []core.Manga {
	// ... (keep existing implementation)
	// Make a copy to avoid modifying the original
	sorted := make([]core.Manga, len(results))
	copy(sorted, results)

	switch strings.ToLower(sortBy) {
	case "title", "name":
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Title < sorted[j].Title
		})
	case "author":
		sort.Slice(sorted, func(i, j int) bool {
			// Get first author or empty string
			authorI := ""
			if len(sorted[i].Authors) > 0 {
				authorI = sorted[i].Authors[0]
			}

			authorJ := ""
			if len(sorted[j].Authors) > 0 {
				authorJ = sorted[j].Authors[0]
			}

			return authorI < authorJ
		})
	case "status":
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Status < sorted[j].Status
		})
	}

	return sorted
}
