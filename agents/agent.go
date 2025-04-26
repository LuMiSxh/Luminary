package agents

import (
	"Luminary/engine"
	"context"
	"fmt"
	"sync"
	"time"
)

// BaseAgent provides a simplified implementation of the Agent interface
// that leverages the new engine services
type BaseAgent struct {
	// Basic agent information
	id          string
	name        string
	description string
	siteURL     string
	apiURL      string

	// Configuration for different API endpoints
	APIConfig        engine.APIConfig
	ExtractorSets    map[string]engine.ExtractorSet
	PaginationConfig engine.PaginationConfig

	// Engine services
	Engine *engine.Engine

	// State management
	initialized  bool
	lastInitTime time.Time
	initMutex    sync.Mutex
	mangaCache   map[string]*MangaInfo
	cacheMutex   sync.RWMutex
}

// NewBaseAgent creates a new streamlined agent
func NewBaseAgent(id, name, description string) *BaseAgent {
	return &BaseAgent{
		id:            id,
		name:          name,
		description:   description,
		Engine:        engine.New(),
		mangaCache:    make(map[string]*MangaInfo),
		initialized:   false,
		ExtractorSets: make(map[string]engine.ExtractorSet),
	}
}

// ID returns the agent's identifier
func (a *BaseAgent) ID() string {
	return a.id
}

// Name returns the agent's display name
func (a *BaseAgent) Name() string {
	return a.name
}

// Description returns the agent's description
func (a *BaseAgent) Description() string {
	return a.description
}

// SiteURL returns the agent's website URL
func (a *BaseAgent) SiteURL() string {
	return a.siteURL
}

// SetSiteURL sets the agent's website URL
func (a *BaseAgent) SetSiteURL(url string) {
	a.siteURL = url
}

// APIURL returns the agent's API URL
func (a *BaseAgent) APIURL() string {
	return a.APIConfig.BaseURL
}

// SetAPIURL sets the agent's API URL
func (a *BaseAgent) SetAPIURL(url string) {
	a.apiURL = url
	a.APIConfig.BaseURL = url
}

// GetEngine returns the agent's engine instance
func (a *BaseAgent) GetEngine() *engine.Engine {
	return a.Engine
}

// Initialize ensures the agent is properly initialized
func (a *BaseAgent) Initialize(ctx context.Context) error {
	a.initMutex.Lock()
	defer a.initMutex.Unlock()

	// Skip if already initialized recently (within 30 minutes)
	if a.initialized && time.Since(a.lastInitTime) < 30*time.Minute {
		return nil
	}

	// Log initialization
	a.Engine.Logger.Info("Initializing agent: %s (%s)", a.name, a.id)

	// Call the source-specific initialization
	err := a.OnInitialize(ctx)
	if err != nil {
		a.Engine.Logger.Error("Failed to initialize agent %s: %v", a.id, err)
		return fmt.Errorf("failed to initialize agent: %w", err)
	}

	a.initialized = true
	a.lastInitTime = time.Now()
	a.Engine.Logger.Info("Agent initialized: %s", a.id)
	return nil
}

// OnInitialize is meant to be overridden by specific agents
func (a *BaseAgent) OnInitialize(ctx context.Context) error {
	// Default implementation does nothing
	return nil
}

// Search implements the Agent interface for searching
func (a *BaseAgent) Search(ctx context.Context, query string, options engine.SearchOptions) ([]Manga, error) {
	// Initialize if needed
	if err := a.Initialize(ctx); err != nil {
		return nil, err
	}

	// Log search request
	a.Engine.Logger.Info("[%s] Searching for: %s", a.id, query)

	// Create a cache key
	cacheKey := fmt.Sprintf("search:%s:%s:%d", a.id, query, options.Limit)

	// Check cache
	var cachedResults []Manga
	if a.Engine.Cache.Get(cacheKey, &cachedResults) {
		a.Engine.Logger.Debug("[%s] Using cached search results for: %s", a.id, query)
		return cachedResults, nil
	}

	// Use pagination service to fetch results
	params := engine.PaginatedRequestParams{
		Config:       a.PaginationConfig,
		APIConfig:    a.APIConfig,
		EndpointName: "search",
		BaseParams:   options,
		PathParams:   []string{},
		ExtractorSet: a.ExtractorSets["manga"],
		MaxPages:     1, // Typically search results are on one page
		ThrottleTime: 500 * time.Millisecond,
	}

	// If query is provided, modify options to include it
	if query != "" {
		searchOpts := options
		searchOpts.Query = query // Actually set the query field
		params.BaseParams = searchOpts
	}

	resultsInterface, err := a.Engine.Pagination.FetchAllPages(ctx, params)
	if err != nil {
		a.Engine.Logger.Error("[%s] Search error: %v", a.id, err)
		return nil, err
	}

	// Convert to Manga type
	results := make([]Manga, 0, len(resultsInterface))
	for _, item := range resultsInterface {
		if manga, ok := item.(*Manga); ok {
			results = append(results, *manga)
		}
	}

	// Cache results
	if err := a.Engine.Cache.Set(cacheKey, results); err != nil {
		a.Engine.Logger.Warn("[%s] Failed to cache search results: %v", a.id, err)
	}

	a.Engine.Logger.Info("[%s] Found %d results for: %s", a.id, len(results), query)
	return results, nil
}

// GetManga implements the Agent interface for retrieving manga details
func (a *BaseAgent) GetManga(ctx context.Context, id string) (*MangaInfo, error) {
	// Initialize if needed
	if err := a.Initialize(ctx); err != nil {
		return nil, err
	}

	// Check memory cache
	a.cacheMutex.RLock()
	manga, found := a.mangaCache[id]
	a.cacheMutex.RUnlock()

	if found {
		a.Engine.Logger.Debug("[%s] Using cached manga info for: %s", a.id, id)
		return manga, nil
	}

	// Try disk cache
	cacheKey := fmt.Sprintf("manga:%s:%s", a.id, id)
	var cachedManga MangaInfo
	if a.Engine.Cache.Get(cacheKey, &cachedManga) {
		a.Engine.Logger.Debug("[%s] Using disk-cached manga info for: %s", a.id, id)

		// Store in memory cache
		a.cacheMutex.Lock()
		a.mangaCache[id] = &cachedManga
		a.cacheMutex.Unlock()

		return &cachedManga, nil
	}

	// Log manga retrieval
	a.Engine.Logger.Info("[%s] Fetching manga details for: %s", a.id, id)

	// Fetch manga details using API service
	response, err := a.Engine.API.FetchFromAPI(
		ctx,
		a.APIConfig,
		"manga",
		nil,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manga: %w", err)
	}

	// Extract manga data
	result, err := a.Engine.Extractor.Extract(a.ExtractorSets["manga"], response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract manga data: %w", err)
	}

	// Convert to MangaInfo
	mangaInfo, ok := result.(*MangaInfo)
	if !ok {
		return nil, fmt.Errorf("expected MangaInfo, got %T", result)
	}

	// Fetch chapters for this manga
	mangaInfo.Chapters, err = a.fetchChaptersForManga(ctx, id)
	if err != nil {
		a.Engine.Logger.Warn("[%s] Failed to fetch chapters for manga %s: %v", a.id, id, err)
		// Continue anyway, just with empty chapters list
	}

	// Store in caches
	a.cacheMutex.Lock()
	a.mangaCache[id] = mangaInfo
	a.cacheMutex.Unlock()

	if err := a.Engine.Cache.Set(cacheKey, mangaInfo); err != nil {
		a.Engine.Logger.Warn("[%s] Failed to cache manga info: %v", a.id, err)
	}

	return mangaInfo, nil
}

// fetchChaptersForManga fetches all chapters for a manga
func (a *BaseAgent) fetchChaptersForManga(ctx context.Context, mangaID string) ([]ChapterInfo, error) {
	// Use pagination service to fetch all chapters
	params := engine.PaginatedRequestParams{
		Config:       a.PaginationConfig,
		APIConfig:    a.APIConfig,
		EndpointName: "chapters",
		BaseParams:   nil, // Default params
		PathParams:   []string{mangaID},
		ExtractorSet: a.ExtractorSets["chapterInfo"],
		MaxPages:     10, // Reasonable limit to prevent excessive requests
		ThrottleTime: 500 * time.Millisecond,
	}

	resultsInterface, err := a.Engine.Pagination.FetchAllPages(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert to ChapterInfo type
	chapters := make([]ChapterInfo, 0, len(resultsInterface))
	for _, item := range resultsInterface {
		if chapter, ok := item.(*ChapterInfo); ok {
			chapters = append(chapters, *chapter)
		}
	}

	return chapters, nil
}

// GetChapter implements the Agent interface for retrieving chapter details
func (a *BaseAgent) GetChapter(ctx context.Context, chapterID string) (*Chapter, error) {
	// Initialize if needed
	if err := a.Initialize(ctx); err != nil {
		return nil, err
	}

	// Try cache first
	cacheKey := fmt.Sprintf("chapter:%s:%s", a.id, chapterID)
	var cachedChapter Chapter
	if a.Engine.Cache.Get(cacheKey, &cachedChapter) {
		a.Engine.Logger.Debug("[%s] Using cached chapter info for: %s", a.id, chapterID)
		return &cachedChapter, nil
	}

	// Log chapter retrieval
	a.Engine.Logger.Info("[%s] Fetching chapter details for: %s", a.id, chapterID)

	// Fetch chapter details using API service
	response, err := a.Engine.API.FetchFromAPI(
		ctx,
		a.APIConfig,
		"chapter",
		nil,
		chapterID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chapter: %w", err)
	}

	// Extract chapter data
	result, err := a.Engine.Extractor.Extract(a.ExtractorSets["chapter"], response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract chapter data: %w", err)
	}

	// Convert to Chapter
	chapter, ok := result.(*Chapter)
	if !ok {
		return nil, fmt.Errorf("expected Chapter, got %T", result)
	}

	// Cache the result
	if err := a.Engine.Cache.Set(cacheKey, chapter); err != nil {
		a.Engine.Logger.Warn("[%s] Failed to cache chapter info: %v", a.id, err)
	}

	return chapter, nil
}

// TryGetMangaForChapter attempts to get manga info for a chapter
func (a *BaseAgent) TryGetMangaForChapter(ctx context.Context, chapterID string) (*Manga, error) {
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

	// Try alternative approach - fetch from API
	response, err := a.Engine.API.FetchFromAPI(
		ctx,
		a.APIConfig,
		"chapterManga",
		nil,
		chapterID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manga for chapter: %w", err)
	}

	// Extract manga ID from the response
	mangaID, err := a.extractMangaIDFromChapterResponse(response)
	if err != nil {
		return nil, err
	}

	// Now fetch the manga details
	mangaInfo, err := a.GetManga(ctx, mangaID)
	if err != nil {
		return nil, err
	}

	return &mangaInfo.Manga, nil
}

// extractMangaIDFromChapterResponse extracts the manga ID from a chapter response
// This is a stub that should be implemented by specific agents
func (a *BaseAgent) extractMangaIDFromChapterResponse(response interface{}) (string, error) {
	// This is a stub - in a real implementation, you would extract the manga ID
	// from the response using the appropriate path
	return "", fmt.Errorf("extractMangaIDFromChapterResponse not implemented by this agent")
}

// DownloadChapter implements the Agent interface for downloading a chapter
func (a *BaseAgent) DownloadChapter(ctx context.Context, chapterID, destDir string) error {
	// Initialize if needed
	if err := a.Initialize(ctx); err != nil {
		return err
	}

	// Log download request
	a.Engine.Logger.Info("[%s] Downloading chapter: %s to %s", a.id, chapterID, destDir)

	// Get chapter information
	chapter, err := a.GetChapter(ctx, chapterID)
	if err != nil {
		return err
	}

	// Try to get manga info for proper manga title
	var mangaTitle string
	var mangaID string

	manga, err := a.TryGetMangaForChapter(ctx, chapterID)
	if err == nil && manga != nil {
		mangaTitle = manga.Title
		mangaID = manga.ID
	} else {
		// Fall back to using chapter title
		a.Engine.Logger.Debug("[%s] Couldn't find manga for chapter %s, using fallback title", a.id, chapterID)
		mangaTitle = fmt.Sprintf("%s-%s", a.Name(), chapterID)
	}

	// Extract chapter and volume numbers
	chapterNum := &chapter.Info.Number
	if *chapterNum == 0 {
		chapterNum = nil
	}

	// Check for volume override in context
	var volumeNum *int
	if val := ctx.Value("volume_override"); val != nil {
		if volNum, ok := val.(int); ok && volNum > 0 {
			volumeNum = &volNum
		}
	} else if chapter.Info.Title != "" {
		// Try to extract volume from title if not overridden
		_, extractedVol := a.Engine.Metadata.ExtractChapterInfo(chapter.Info.Title)
		volumeNum = extractedVol
	}

	// Prepare metadata
	metadata := engine.ChapterMetadata{
		MangaID:      mangaID,
		MangaTitle:   mangaTitle,
		ChapterID:    chapterID,
		ChapterNum:   chapterNum,
		VolumeNum:    volumeNum,
		ChapterTitle: chapter.Info.Title,
		AgentID:      a.ID(),
	}

	// Convert pages to download requests
	downloadFiles := make([]engine.DownloadRequest, len(chapter.Pages))
	for i, page := range chapter.Pages {
		downloadFiles[i] = engine.DownloadRequest{
			URL:       page.URL,
			Index:     i + 1,
			Filename:  page.Filename,
			PageCount: len(chapter.Pages),
		}
	}

	// Extract concurrency settings from context or use default
	concurrency := a.Engine.Download.MaxConcurrency
	if contextConcurrency, ok := ctx.Value("concurrency").(int); ok && contextConcurrency > 0 {
		concurrency = contextConcurrency
	}

	// Set up download configuration
	config := engine.DownloadJobConfig{
		Metadata:    metadata,
		OutputDir:   destDir,
		Concurrency: concurrency,
		Files:       downloadFiles,
		WaitDuration: func(isRetry bool) {
			if isRetry {
				time.Sleep(a.Engine.HTTP.ThrottleTimeAPI)
			} else {
				time.Sleep(a.Engine.HTTP.ThrottleTimeImages)
			}
		},
	}

	// Log and start download
	a.Engine.Logger.Info("[%s] Downloading %d pages for chapter %s", a.id, len(chapter.Pages), chapterID)

	// Use the engine's download service to download the chapter
	err = a.Engine.Download.DownloadChapter(ctx, config)
	if err != nil {
		a.Engine.Logger.Error("[%s] Download failed: %v", a.id, err)
		return err
	}

	a.Engine.Logger.Info("[%s] Successfully downloaded chapter %s", a.id, chapterID)
	return nil
}

// ExtractDomain extracts the domain from a URL
func (a *BaseAgent) ExtractDomain(urlStr string) string {
	// Use the engine's utility method
	return a.Engine.ExtractDomain(urlStr)
}
