package engine

import (
	"Luminary/errors"
	"context"
	"fmt"
	"time"
)

// ExecuteInitialize handles common agent initialization logic
func ExecuteInitialize(ctx context.Context, engine *Engine, agentID, agentName string, initFunc func(context.Context) error) error {
	// Log initialization
	engine.Logger.Info("Initializing agent: %s (%s)", agentName, agentID)

	// Call the agent-specific initialization
	err := initFunc(ctx)
	if err != nil {
		engine.Logger.Error("Failed to initialize agent %s: %v", agentID, err)
		return &errors.AgentError{
			AgentID: agentID,
			Message: "Failed to initialize agent",
			Err:     err,
		}
	}

	engine.Logger.Info("Agent initialized: %s", agentID)
	return nil
}

// ExecuteSearch handles the common pattern for searching by delegating to the SearchService
func ExecuteSearch(
	ctx context.Context,
	engine *Engine,
	agentID string,
	query string,
	options *SearchOptions,
	apiConfig APIConfig,
	paginationConfig PaginationConfig,
	extractorSet ExtractorSet,
) ([]Manga, error) {
	// Delegate to the search service
	results, err := engine.Search.ExecuteSearch(
		ctx, agentID, query, options, apiConfig, paginationConfig, extractorSet)

	if err != nil {
		return nil, &errors.AgentError{
			AgentID: agentID,
			Message: fmt.Sprintf("Search failed for query '%s'", query),
			Err:     err,
		}
	}

	return results, nil
}

// ExecuteGetManga handles the common pattern for retrieving manga details
func ExecuteGetManga(
	ctx context.Context,
	engine *Engine,
	agentID string,
	mangaID string,
	apiConfig APIConfig,
	extractorSet ExtractorSet,
	fetchChaptersFunc func(context.Context, string) ([]ChapterInfo, error),
) (*MangaInfo, error) {
	// Log manga retrieval
	engine.Logger.Info("[%s] Fetching manga details for: %s", agentID, mangaID)

	// Apply rate limiting
	domain := extractDomainFromUrl(apiConfig.BaseURL)
	engine.RateLimiter.Wait(domain)

	// Fetch manga details using API service
	response, err := engine.API.FetchFromAPI(
		ctx,
		apiConfig,
		"manga",
		nil,
		mangaID,
	)

	if err != nil {
		// Check for not found error specifically
		if errors.IsNotFound(err) {
			return nil, errors.NewAgentNotFoundError(agentID, "manga", mangaID, err)
		}

		// For other errors, wrap with agent context
		return nil, &errors.AgentError{
			AgentID:      agentID,
			ResourceType: "manga",
			ResourceID:   mangaID,
			Message:      "Failed to fetch manga",
			Err:          err,
		}
	}

	// Extract manga data
	result, err := engine.Extractor.Extract(extractorSet, response)
	if err != nil {
		return nil, &errors.AgentError{
			AgentID:      agentID,
			ResourceType: "manga",
			ResourceID:   mangaID,
			Message:      "Failed to extract manga data",
			Err:          err,
		}
	}

	// Convert to MangaInfo
	mangaInfo, ok := result.(*MangaInfo)
	if !ok {
		return nil, &errors.AgentError{
			AgentID:      agentID,
			ResourceType: "manga",
			ResourceID:   mangaID,
			Message:      fmt.Sprintf("Expected MangaInfo, got %T", result),
			Err:          errors.ErrInvalidInput,
		}
	}

	// Validate the result
	if mangaInfo.ID == "" {
		mangaInfo.ID = mangaID
	}

	if mangaInfo.Title == "" {
		engine.Logger.Warn("[%s] Manga with ID %s has no title", agentID, mangaID)
	}

	// Fetch chapters for this manga
	chapters, err := fetchChaptersFunc(ctx, mangaID)
	if err != nil {
		engine.Logger.Warn("[%s] Failed to fetch chapters for manga %s: %v", agentID, mangaID, err)
		// Continue anyway, just with empty chapters list
		mangaInfo.Chapters = []ChapterInfo{}
	} else {
		mangaInfo.Chapters = chapters
	}

	return mangaInfo, nil
}

// ExecuteGetChapter handles the common pattern for retrieving chapter details
func ExecuteGetChapter(
	ctx context.Context,
	engine *Engine,
	agentID string,
	chapterID string,
	apiConfig APIConfig,
	extractorSet ExtractorSet,
	processFunc func(interface{}, string) (*Chapter, error),
) (*Chapter, error) {
	// Log chapter retrieval
	engine.Logger.Info("[%s] Fetching chapter details for: %s", agentID, chapterID)

	// Apply rate limiting
	domain := extractDomainFromUrl(apiConfig.BaseURL)
	engine.RateLimiter.Wait(domain)

	// Fetch chapter details using API service
	response, err := engine.API.FetchFromAPI(
		ctx,
		apiConfig,
		"chapter",
		nil,
		chapterID,
	)

	if err != nil {
		// Check for not found error specifically
		if errors.IsNotFound(err) {
			return nil, errors.NewAgentNotFoundError(agentID, "chapter", chapterID, err)
		}

		// For other errors, wrap with agent context
		return nil, &errors.AgentError{
			AgentID:      agentID,
			ResourceType: "chapter",
			ResourceID:   chapterID,
			Message:      "Failed to fetch chapter",
			Err:          err,
		}
	}

	// Process the response with agent-specific logic
	if processFunc != nil {
		chapter, err := processFunc(response, chapterID)
		if err != nil {
			return nil, &errors.AgentError{
				AgentID:      agentID,
				ResourceType: "chapter",
				ResourceID:   chapterID,
				Message:      "Failed to process chapter data",
				Err:          err,
			}
		}
		return chapter, nil
	}

	// Or use the general extractor if no custom processing
	result, err := engine.Extractor.Extract(extractorSet, response)
	if err != nil {
		return nil, &errors.AgentError{
			AgentID:      agentID,
			ResourceType: "chapter",
			ResourceID:   chapterID,
			Message:      "Failed to extract chapter data",
			Err:          err,
		}
	}

	// Convert to Chapter
	chapter, ok := result.(*Chapter)
	if !ok {
		return nil, &errors.AgentError{
			AgentID:      agentID,
			ResourceType: "chapter",
			ResourceID:   chapterID,
			Message:      fmt.Sprintf("Expected Chapter, got %T", result),
			Err:          errors.ErrInvalidInput,
		}
	}

	return chapter, nil
}

// ExecuteDownloadChapter handles the common pattern for downloading a chapter
func ExecuteDownloadChapter(
	ctx context.Context,
	engine *Engine,
	agentID string,
	agentName string,
	chapterID string,
	destDir string,
	getChapterFunc func(context.Context, string) (*Chapter, error),
	getMangaForChapterFunc func(context.Context, string) (*Manga, error),
) error {
	// Log download request
	engine.Logger.Info("[%s] Downloading chapter: %s to %s", agentID, chapterID, destDir)

	// Get chapter information
	chapter, err := getChapterFunc(ctx, chapterID)
	if err != nil {
		return &errors.AgentError{
			AgentID:      agentID,
			ResourceType: "chapter",
			ResourceID:   chapterID,
			Message:      "Failed to get chapter info for download",
			Err:          err,
		}
	}

	// Try to get manga info for proper manga title
	var mangaTitle string
	var mangaID string

	manga, err := getMangaForChapterFunc(ctx, chapterID)
	if err == nil && manga != nil {
		mangaTitle = manga.Title
		mangaID = manga.ID
	} else {
		// Fall back to using chapter title
		engine.Logger.Debug("[%s] Couldn't find manga for chapter %s, using fallback title", agentID, chapterID)
		mangaTitle = fmt.Sprintf("%s-%s", agentName, chapterID)
	}

	// Extract chapter and volume numbers
	chapterNum := &chapter.Info.Number
	if *chapterNum == 0 {
		chapterNum = nil
	}

	// Check for volume override in context
	var volumeNum *int
	if vol, hasOverride := GetVolumeOverride(ctx); hasOverride {
		volumeNum = &vol
	} else if chapter.Info.Title != "" {
		// Try to extract volume from title if not overridden
		_, extractedVol := engine.Metadata.ExtractChapterInfo(chapter.Info.Title)
		volumeNum = extractedVol
	}

	// Make sure there are pages to download
	if len(chapter.Pages) == 0 {
		return &errors.AgentError{
			AgentID:      agentID,
			ResourceType: "chapter",
			ResourceID:   chapterID,
			Message:      "Chapter has no pages to download",
			Err:          errors.ErrInvalidInput,
		}
	}

	// Prepare metadata
	metadata := ChapterMetadata{
		MangaID:      mangaID,
		MangaTitle:   mangaTitle,
		ChapterID:    chapterID,
		ChapterNum:   chapterNum,
		VolumeNum:    volumeNum,
		ChapterTitle: chapter.Info.Title,
		AgentID:      agentID,
	}

	// Convert pages to download requests
	downloadFiles := make([]DownloadRequest, len(chapter.Pages))
	for i, page := range chapter.Pages {
		downloadFiles[i] = DownloadRequest{
			URL:       page.URL,
			Index:     i + 1,
			Filename:  page.Filename,
			PageCount: len(chapter.Pages),
		}
	}

	// Extract concurrency settings from context or use default
	concurrency := engine.Download.MaxConcurrency
	if contextConcurrency := GetConcurrency(ctx, concurrency); contextConcurrency > 0 {
		concurrency = contextConcurrency
	}

	// Set up download configuration
	config := DownloadJobConfig{
		Metadata:    metadata,
		OutputDir:   destDir,
		Concurrency: concurrency,
		Files:       downloadFiles,
		WaitDuration: func(isRetry bool) {
			if isRetry {
				time.Sleep(engine.HTTP.ThrottleTimeAPI)
			} else {
				time.Sleep(engine.HTTP.ThrottleTimeImages)
			}
		},
	}

	// Log and start download
	engine.Logger.Info("[%s] Downloading %d pages for chapter %s", agentID, len(chapter.Pages), chapterID)

	// Use the engine's download service to download the chapter
	err = engine.Download.DownloadChapter(ctx, config)
	if err != nil {
		engine.Logger.Error("[%s] Download failed: %v", agentID, err)
		return &errors.AgentError{
			AgentID:      agentID,
			ResourceType: "chapter",
			ResourceID:   chapterID,
			Message:      "Download failed",
			Err:          err,
		}
	}

	engine.Logger.Info("[%s] Successfully downloaded chapter %s", agentID, chapterID)
	return nil
}
