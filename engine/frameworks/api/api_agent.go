package api

import (
	"Luminary/engine"
	"Luminary/engine/frameworks/common"
	"Luminary/errors"
	"context"
	"fmt"
	"net/url"
	"reflect"
	"time"
)

// APIAgentConfig defines configuration for an API-based agent
type APIAgentConfig struct {
	// Basic identity
	ID          string
	Name        string
	Description string
	SiteURL     string

	// API configuration
	BaseURL        string
	RateLimitKey   string
	DefaultHeaders map[string]string
	RetryCount     int
	ThrottleTime   time.Duration

	// Endpoints and extractors
	Endpoints     map[string]EndpointConfig
	ExtractorSets map[string]engine.ExtractorSet

	// Custom functions
	QueryFormatters    map[string]QueryFormatter
	ResponseProcessors map[string]ResponseProcessor

	// Pagination configuration
	PaginationConfig *engine.PaginationConfig

	// Chapter fetching configuration
	ChapterConfig ChapterFetchConfig
}

// ChapterFetchConfig provides detailed configuration for chapter fetching
type ChapterFetchConfig struct {
	// Endpoint information
	EndpointName string // Name of the endpoint for fetching chapters

	// Response paths
	ResponseItemsPath []string // Path to the chapter items in the response
	TotalCountPath    []string // Path to the total count in the response

	// Pagination parameters
	PageParamName   string // Name of the page parameter
	LimitParamName  string // Name of the limit parameter
	OffsetParamName string // Name of the offset parameter
	DefaultPageSize int    // Default number of chapters per page
	MaxPageSize     int    // Maximum number of chapters per page

	// Custom processing function for chapters
	ProcessChapters ProcessChaptersFunc // Custom function to process chapter response
}

// ProcessChaptersFunc is a function that processes chapter responses to extract chapter info
type ProcessChaptersFunc func(ctx context.Context, agent *APIAgent, response interface{}, mangaID string) ([]engine.ChapterInfo, bool, error)

// EndpointConfig defines a specific API endpoint
type EndpointConfig struct {
	Path         string
	Method       string
	ResponseType interface{}
	RequiresAuth bool
}

// QueryFormatter formats query parameters for an endpoint
type QueryFormatter func(params interface{}) url.Values

// ResponseProcessor processes a response before extraction
type ResponseProcessor func(response interface{}, id string) (interface{}, error)

// APIAgent implements the engine.Agent interface for API-based sources
type APIAgent struct {
	config    APIAgentConfig
	engine    *engine.Engine
	apiConfig engine.APIConfig
}

// NewAPIAgent creates a new API-based agent
func NewAPIAgent(e *engine.Engine, config APIAgentConfig) engine.Agent {
	agent := &APIAgent{
		config: config,
		engine: e,
	}

	// Convert the config into engine.APIConfig format
	agent.apiConfig = engine.APIConfig{
		BaseURL:        config.BaseURL,
		RateLimitKey:   config.RateLimitKey,
		RetryCount:     config.RetryCount,
		ThrottleTime:   config.ThrottleTime,
		DefaultHeaders: config.DefaultHeaders,
		Endpoints:      make(map[string]engine.APIEndpoint),
	}

	// Convert each endpoint config to engine.APIEndpoint
	for name, endpointConfig := range config.Endpoints {
		agent.apiConfig.Endpoints[name] = engine.APIEndpoint{
			Path:           endpointConfig.Path,
			Method:         endpointConfig.Method,
			ResponseType:   endpointConfig.ResponseType,
			RequiresAuth:   endpointConfig.RequiresAuth,
			QueryFormatter: agent.getQueryFormatter(name),
			PathFormatter:  engine.DefaultPathFormatter,
		}
	}

	return agent
}

// Config returns the current configuration
func (a *APIAgent) Config() APIAgentConfig {
	return a.config
}

func (a *APIAgent) ID() string          { return a.config.ID }
func (a *APIAgent) Name() string        { return a.config.Name }
func (a *APIAgent) Description() string { return a.config.Description }
func (a *APIAgent) SiteURL() string     { return a.config.SiteURL }

func (a *APIAgent) Initialize(ctx context.Context) error {
	return common.ExecuteInitialize(ctx, a.engine, a.ID(), a.Name(), func(ctx context.Context) error {
		// Custom initialization logic could go here
		return nil
	})
}

func (a *APIAgent) Search(ctx context.Context, query string, options engine.SearchOptions) ([]engine.Manga, error) {
	extractorSet, ok := a.config.ExtractorSets["search"]
	if !ok {
		return nil, fmt.Errorf("search extractor not configured for %s", a.Name())
	}

	// Use provided pagination config or default
	paginationConfig := engine.PaginationConfig{
		LimitParam:     "limit",
		OffsetParam:    "offset",
		TotalCountPath: []string{"total"},
		ItemsPath:      []string{"data"},
		DefaultLimit:   20,
		MaxLimit:       100,
	}

	if a.config.PaginationConfig != nil {
		paginationConfig = *a.config.PaginationConfig
	}

	// Implementation using engine.ExecuteSearch with the provided extractors
	return common.ExecuteSearch(
		ctx,
		a.engine,
		a.ID(),
		query,
		&options,
		a.apiConfig,
		paginationConfig,
		extractorSet,
	)
}

func (a *APIAgent) GetManga(ctx context.Context, id string) (*engine.MangaInfo, error) {
	extractorSet, ok := a.config.ExtractorSets["manga"]
	if !ok {
		return nil, fmt.Errorf("manga extractor not configured for %s", a.Name())
	}

	return common.ExecuteGetManga(
		ctx,
		a.engine,
		a.ID(),
		id,
		a.apiConfig,
		extractorSet,
		a.fetchChaptersForManga,
	)
}

// Enhanced chapter fetching with pagination support
func (a *APIAgent) fetchChaptersForManga(ctx context.Context, mangaID string) ([]engine.ChapterInfo, error) {
	// Check if chapter fetching is configured
	if a.config.ChapterConfig.EndpointName == "" {
		a.engine.Logger.Debug("[%s] No chapter endpoint configured, returning empty chapters", a.ID())
		return []engine.ChapterInfo{}, nil
	}

	// Set up default values if not configured
	pageSize := a.config.ChapterConfig.DefaultPageSize
	if pageSize <= 0 {
		pageSize = 100 // Default to 100 if not specified
	}

	// Get the parameters names
	offsetParam := a.config.ChapterConfig.OffsetParamName
	if offsetParam == "" {
		offsetParam = "offset" // Default
	}

	limitParam := a.config.ChapterConfig.LimitParamName
	if limitParam == "" {
		limitParam = "limit" // Default
	}

	a.engine.Logger.Info("[%s] Fetching chapters for manga: %s", a.ID(), mangaID)

	// If we have a custom process function, use it
	if a.config.ChapterConfig.ProcessChapters != nil {
		return a.fetchChaptersWithCustomProcessor(ctx, mangaID)
	}

	// Otherwise, use a standard paginated approach
	return a.fetchChaptersWithPagination(ctx, mangaID)
}

// fetchChaptersWithCustomProcessor uses a custom processor function to handle chapter fetching
func (a *APIAgent) fetchChaptersWithCustomProcessor(ctx context.Context, mangaID string) ([]engine.ChapterInfo, error) {
	endpointName := a.config.ChapterConfig.EndpointName
	pageSize := a.config.ChapterConfig.DefaultPageSize
	if pageSize <= 0 {
		pageSize = 100 // Default
	}

	// Initialize the chapter slice
	var allChapters []engine.ChapterInfo

	// Start with page 0
	page := 0
	hasMore := true

	for hasMore {
		// Create parameters with pagination
		params := struct {
			Offset int
			Limit  int
			Page   int
		}{
			Offset: page * pageSize,
			Limit:  pageSize,
			Page:   page,
		}

		a.engine.Logger.Debug("[%s] Fetching chapters page %d for manga %s", a.ID(), page+1, mangaID)

		// Fetch a page of chapters
		response, err := a.engine.API.FetchFromAPI(
			ctx,
			a.apiConfig,
			endpointName,
			params,
			mangaID,
		)

		if err != nil {
			// Handle not found errors specially
			if errors.IsNotFound(err) {
				// If this is the first page, report the error
				if page == 0 {
					return nil, fmt.Errorf("no chapters found for manga %s", mangaID)
				}
				// Otherwise, we've just reached the end
				break
			}

			return nil, fmt.Errorf("failed to fetch chapters (page %d): %w", page+1, err)
		}

		// Process this page of chapters using the custom processor
		chapterInfoList, morePages, err := a.config.ChapterConfig.ProcessChapters(ctx, a, response, mangaID)
		if err != nil {
			return nil, fmt.Errorf("failed to process chapters: %w", err)
		}

		// Add to overall list
		allChapters = append(allChapters, chapterInfoList...)

		// Update loop control flag and move to next page
		hasMore = morePages
		page++
	}
	a.engine.Logger.Info("[%s] Retrieved %d chapters for manga %s", a.ID(), len(allChapters), mangaID)
	return allChapters, nil
}

// fetchChaptersWithPagination uses a standard paginated approach to fetch chapters
func (a *APIAgent) fetchChaptersWithPagination(ctx context.Context, mangaID string) ([]engine.ChapterInfo, error) {
	endpointName := a.config.ChapterConfig.EndpointName
	pageSize := a.config.ChapterConfig.DefaultPageSize
	if pageSize <= 0 {
		pageSize = 100 // Default
	}

	// Get parameter names
	offsetParam := a.config.ChapterConfig.OffsetParamName
	if offsetParam == "" {
		offsetParam = "offset" // Default
	}

	limitParam := a.config.ChapterConfig.LimitParamName
	if limitParam == "" {
		limitParam = "limit" // Default
	}

	// Initialize the chapters slice
	var allChapters []engine.ChapterInfo

	// Create extractor if available
	extractorSet, hasExtractor := a.config.ExtractorSets["chapters"]

	// Start with page 0
	page := 0

	for {
		// Create parameters structure dynamically
		paramsMap := make(map[string]interface{})
		paramsMap[offsetParam] = page * pageSize
		paramsMap[limitParam] = pageSize

		// Convert to a struct using reflection
		params := createParamsStruct(paramsMap)

		a.engine.Logger.Debug("[%s] Fetching chapters page %d for manga %s", a.ID(), page+1, mangaID)

		// Fetch a page of chapters
		response, err := a.engine.API.FetchFromAPI(
			ctx,
			a.apiConfig,
			endpointName,
			params,
			mangaID,
		)

		if err != nil {
			// Handle not found errors specially
			if errors.IsNotFound(err) {
				// If this is the first page, report the error
				if page == 0 {
					return nil, fmt.Errorf("no chapters found for manga %s", mangaID)
				}
				// Otherwise, we've just reached the end
				break
			}

			return nil, fmt.Errorf("failed to fetch chapters (page %d): %w", page+1, err)
		}

		// Extract items from the response
		itemsPath := a.config.ChapterConfig.ResponseItemsPath
		if len(itemsPath) == 0 {
			itemsPath = []string{"data"} // Default
		}

		items, err := getValueFromPath(response, itemsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to extract chapters from response: %w", err)
		}

		// Convert to a slice
		itemsSlice, ok := convertToSlice(items)
		if !ok || len(itemsSlice) == 0 {
			// No more items, we're done
			break
		}

		// Extract chapter information
		var pageChapters []engine.ChapterInfo

		if hasExtractor {
			// Use the extractor
			extractedItems, err := a.engine.Extractor.ExtractList(extractorSet, response, itemsPath)
			if err != nil {
				return nil, fmt.Errorf("failed to extract chapters: %w", err)
			}

			// Convert extracted items to ChapterInfo
			for _, item := range extractedItems {
				if chapterInfo, ok := item.(*engine.ChapterInfo); ok && chapterInfo != nil {
					pageChapters = append(pageChapters, *chapterInfo)
				}
			}
		} else {
			// Fallback: simple extraction without extractor
			// This is a simplistic approach - specific agents should implement a custom processor
			for i, item := range itemsSlice {
				if itemMap, ok := item.(map[string]interface{}); ok {
					chapterInfo := engine.ChapterInfo{
						ID: fmt.Sprintf("%v", i), // Fallback ID
					}

					// Extract ID if possible
					if id, ok := getStringValue(itemMap, "id"); ok {
						chapterInfo.ID = id
					}

					// Extract title if possible
					if title, ok := getStringValue(itemMap, "title"); ok {
						chapterInfo.Title = title
					}

					pageChapters = append(pageChapters, chapterInfo)
				}
			}
		}

		// Add chapters to the overall list
		allChapters = append(allChapters, pageChapters...)

		// Check if we've reached the end
		if len(itemsSlice) < pageSize {
			break // No more items
		}

		// Move to next page
		page++
	}

	a.engine.Logger.Info("[%s] Retrieved %d chapters for manga %s", a.ID(), len(allChapters), mangaID)
	return allChapters, nil
}

func (a *APIAgent) GetChapter(ctx context.Context, chapterID string) (*engine.Chapter, error) {
	extractorSet, ok := a.config.ExtractorSets["chapter"]
	if !ok {
		return nil, fmt.Errorf("chapter extractor not configured for %s", a.Name())
	}

	// Get the response processor if configured
	var processorFn func(interface{}, string) (*engine.Chapter, error)
	if rawFn := a.getResponseProcessor("chapter"); rawFn != nil {
		// We need to adapt the function to return the correct type
		processorFn = func(response interface{}, id string) (*engine.Chapter, error) {
			processed, err := rawFn(response, id)
			if err != nil {
				return nil, err
			}

			// Check if the processed result is already the correct type
			if chapter, ok := processed.(*engine.Chapter); ok {
				return chapter, nil
			}

			// Otherwise, log a warning and return nil
			a.engine.Logger.Warn("Response processor did not return *engine.Chapter")
			return nil, fmt.Errorf("invalid processor return type")
		}
	}

	return common.ExecuteGetChapter(
		ctx,
		a.engine,
		a.ID(),
		chapterID,
		a.apiConfig,
		extractorSet,
		processorFn,
	)
}

func (a *APIAgent) TryGetMangaForChapter(ctx context.Context, chapterID string) (*engine.Manga, error) {
	// Fetch chapter details first to get manga ID
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

func (a *APIAgent) DownloadChapter(ctx context.Context, chapterID, destDir string) error {
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

// Helper methods
func (a *APIAgent) getQueryFormatter(endpointName string) func(interface{}) url.Values {
	if formatter, exists := a.config.QueryFormatters[endpointName]; exists {
		return formatter
	}
	return engine.BuildQueryParams
}

func (a *APIAgent) getResponseProcessor(endpointName string) func(interface{}, string) (interface{}, error) {
	if processor, exists := a.config.ResponseProcessors[endpointName]; exists {
		return processor
	}
	return nil
}

// Helper functions for processing responses

// createParamsStruct creates a struct from a map for API requests
func createParamsStruct(paramsMap map[string]interface{}) interface{} {
	// For simplicity, we'll create a struct dynamically
	// This is a simple approach - for production use, reflection would be more complex
	type Params struct {
		Offset int `json:"offset"`
		Limit  int `json:"limit"`
		Page   int `json:"page"`
	}

	params := Params{}

	if val, ok := paramsMap["offset"].(int); ok {
		params.Offset = val
	}
	if val, ok := paramsMap["limit"].(int); ok {
		params.Limit = val
	}
	if val, ok := paramsMap["page"].(int); ok {
		params.Page = val
	}

	return params
}

// getValueFromPath extracts a value from a nested structure using a path
func getValueFromPath(data interface{}, path []string) (interface{}, error) {
	if len(path) == 0 {
		return data, nil
	}

	if data == nil {
		return nil, fmt.Errorf("data is nil")
	}

	current := path[0]
	restPath := path[1:]

	// Handle maps
	if dataMap, ok := data.(map[string]interface{}); ok {
		value, exists := dataMap[current]
		if !exists {
			return nil, fmt.Errorf("key '%s' not found in map", current)
		}

		if len(restPath) == 0 {
			return value, nil
		}

		return getValueFromPath(value, restPath)
	}

	// Handle structs using reflection
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() == reflect.Struct {
		field := v.FieldByName(current)
		if !field.IsValid() {
			return nil, fmt.Errorf("field '%s' not found in struct", current)
		}

		value := field.Interface()
		if len(restPath) == 0 {
			return value, nil
		}

		return getValueFromPath(value, restPath)
	}

	return nil, fmt.Errorf("cannot traverse path in %T", data)
}

// convertToSlice attempts to convert a value to a slice of interface{}
func convertToSlice(value interface{}) ([]interface{}, bool) {
	// Handle nil
	if value == nil {
		return nil, false
	}

	// Handle directly provided slices
	if slice, ok := value.([]interface{}); ok {
		return slice, true
	}

	// Use reflection for other types
	v := reflect.ValueOf(value)

	// Dereference pointers
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Handle arrays and slices
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
		// Convert to []interface{}
		result := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			result[i] = v.Index(i).Interface()
		}
		return result, true
	}

	return nil, false
}

// getStringValue extracts a string value from a map
func getStringValue(data map[string]interface{}, key string) (string, bool) {
	if val, ok := data[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal, true
		}
	}
	return "", false
}
