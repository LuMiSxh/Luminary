package engine

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// SearchOptions for search customization
type SearchOptions struct {
	Query   string            // The search query string
	Limit   int               // Maximum number of results to return
	Fields  []string          // Fields to search within (e.g., "title", "author", "description")
	Filters map[string]string // Field-specific filters
	Sort    string            // Sort order (e.g., "relevance", "name")
}

// SearchService provides centralized search capabilities
type SearchService struct {
	Logger      *LoggerService
	API         *APIService
	Extractor   *ExtractorService
	Pagination  *PaginationService
	RateLimiter *RateLimiterService
}

// NewSearchService creates a new SearchService with the necessary dependencies
func NewSearchService(
	logger *LoggerService,
	api *APIService,
	extractor *ExtractorService,
	pagination *PaginationService,
	rateLimiter *RateLimiterService,
) *SearchService {
	return &SearchService{
		Logger:      logger,
		API:         api,
		Extractor:   extractor,
		Pagination:  pagination,
		RateLimiter: rateLimiter,
	}
}

// ExecuteSearch performs a search against a single agent
func (s *SearchService) ExecuteSearch(
	ctx context.Context,
	agentID string,
	query string,
	options SearchOptions,
	apiConfig APIConfig,
	paginationConfig PaginationConfig,
	extractorSet ExtractorSet,
) ([]Manga, error) {
	// Log search request
	s.Logger.Info("[%s] Searching for: %s (limit: %d)", agentID, query, options.Limit)

	// Apply rate limiting
	domain := extractDomainFromUrl(apiConfig.BaseURL)
	s.RateLimiter.Wait(domain)

	// Use pagination service to fetch results
	params := PaginatedRequestParams{
		Config:       paginationConfig,
		APIConfig:    apiConfig,
		EndpointName: "search",
		BaseParams:   options,
		PathParams:   []string{},
		ExtractorSet: extractorSet,
		MaxPages:     1, // Typically search results are on one page
		ThrottleTime: 500 * time.Millisecond,
	}

	// If query is provided, modify options to include it
	if query != "" {
		searchOpts := options
		searchOpts.Query = query
		params.BaseParams = searchOpts
	}

	resultsInterface, err := s.Pagination.FetchAllPages(ctx, params)
	if err != nil {
		s.Logger.Error("[%s] Search error: %v", agentID, err)
		return nil, fmt.Errorf("search error: %w", err)
	}

	// Convert to Manga type
	results := make([]Manga, 0, len(resultsInterface))
	for _, item := range resultsInterface {
		if manga, ok := item.(*Manga); ok {
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

	// Apply limit if specified
	if options.Limit > 0 && len(results) > options.Limit {
		results = results[:options.Limit]
	}

	s.Logger.Info("[%s] Found %d results for: %s", agentID, len(results), query)
	return results, nil
}

// SearchAcrossProviders performs a search across multiple or all providers
func (s *SearchService) SearchAcrossProviders(
	ctx context.Context,
	engine *Engine,
	query string,
	options SearchOptions,
	agentIDs []string, // If empty, search all agents
) (map[string][]Manga, error) {
	results := make(map[string][]Manga)
	var mu sync.Mutex
	var wg sync.WaitGroup

	s.Logger.Info("Performing search across providers for: %s", query)

	// Determine which agents to search
	var agentsToSearch []Agent
	if len(agentIDs) == 0 {
		// Search all agents
		agentsToSearch = engine.AllAgents()
	} else {
		// Search only specified agents
		for _, id := range agentIDs {
			if agent, exists := engine.GetAgent(id); exists {
				agentsToSearch = append(agentsToSearch, agent)
			}
		}
	}

	// Set up error collection
	errorChan := make(chan error, len(agentsToSearch))

	// Create a child context with timeout
	searchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Set semaphore for concurrency control
	// Use a sensible number of concurrent searches to avoid overwhelming APIs
	maxConcurrent := 3
	semaphore := make(chan struct{}, maxConcurrent)

	// Search each agent concurrently
	for _, agent := range agentsToSearch {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(a Agent) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			// Initialize agent if needed
			if err := a.Initialize(searchCtx); err != nil {
				errorChan <- fmt.Errorf("failed to initialize agent %s: %w", a.ID(), err)
				return
			}

			// Execute search against this agent
			agentResults, err := a.Search(searchCtx, query, options)
			if err != nil {
				errorChan <- fmt.Errorf("search error for agent %s: %w", a.ID(), err)
				return
			}

			// Add results to the map
			mu.Lock()
			results[a.ID()] = agentResults
			mu.Unlock()
		}(agent)
	}

	// Wait for all searches to complete
	wg.Wait()
	close(errorChan)

	// Collect errors
	var errors []error
	for err := range errorChan {
		errors = append(errors, err)
	}

	// Log errors but still return partial results
	if len(errors) > 0 {
		s.Logger.Warn("Search completed with errors from %d agents", len(errors))
		for _, err := range errors {
			s.Logger.Warn("%v", err)
		}
	}

	return results, nil
}

// FilterResults applies filters to search results
func (s *SearchService) FilterResults(results []Manga, filters map[string]string) []Manga {
	if len(filters) == 0 {
		return results
	}

	filtered := make([]Manga, 0, len(results))

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

// SortResults sorts search results according to the specified criteria
func (s *SearchService) SortResults(results []Manga, sortBy string) []Manga {
	// Make a copy to avoid modifying the original
	sorted := make([]Manga, len(results))
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

// extractDomainFromUrl extracts the domain from a URL string
func extractDomainFromUrl(urlStr string) string {
	// Simple extraction - just return the URL as is
	// In a real implementation, this would use url.Parse to extract the hostname
	return urlStr
}
