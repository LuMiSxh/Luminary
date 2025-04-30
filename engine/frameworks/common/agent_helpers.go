package common

import (
	"Luminary/engine"
	"Luminary/errors"
	"context"
	"fmt"
	"time"
)

// ExecuteInitialize handles common agent initialization logic
func ExecuteInitialize(ctx context.Context, e *engine.Engine, agentID, agentName string, initFunc func(context.Context) error) error {
	// Log initialization
	e.Logger.Info("Initializing agent: %s (%s)", agentName, agentID)

	// Call the agent-specific initialization
	err := initFunc(ctx)
	if err != nil {
		e.Logger.Error("Failed to initialize agent %s: %v", agentID, err)
		return &errors.AgentError{
			AgentID: agentID,
			Message: "Failed to initialize agent",
			Err:     err,
		}
	}

	e.Logger.Info("Agent initialized: %s", agentID)
	return nil
}

// ExecuteSearch handles the common pattern for searching by delegating to the SearchService
func ExecuteSearch(
	ctx context.Context,
	e *engine.Engine,
	agentID string,
	query string,
	options *engine.SearchOptions,
	apiConfig engine.APIConfig,
	paginationConfig engine.PaginationConfig,
	extractorSet engine.ExtractorSet,
) ([]engine.Manga, error) {
	// Delegate to the search service
	results, err := e.Search.ExecuteSearch(
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
	e *engine.Engine,
	agentID string,
	mangaID string,
	apiConfig engine.APIConfig,
	extractorSet engine.ExtractorSet,
	fetchChaptersFunc func(context.Context, string) ([]engine.ChapterInfo, error),
) (*engine.MangaInfo, error) {
	// Log manga retrieval
	e.Logger.Info("[%s] Fetching manga details for: %s", agentID, mangaID)

	// Apply rate limiting
	domain := e.ExtractDomain(apiConfig.BaseURL)
	e.RateLimiter.Wait(domain)

	// Fetch manga details using API service
	response, err := e.API.FetchFromAPI(
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
	result, err := e.Extractor.Extract(extractorSet, response)
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
	mangaInfo, ok := result.(*engine.MangaInfo)
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
		e.Logger.Warn("[%s] Manga with ID %s has no title", agentID, mangaID)
	}

	// Fetch chapters for this manga
	chapters, err := fetchChaptersFunc(ctx, mangaID)
	if err != nil {
		e.Logger.Warn("[%s] Failed to fetch chapters for manga %s: %v", agentID, mangaID, err)
		// Continue anyway, just with empty chapters list
		mangaInfo.Chapters = []engine.ChapterInfo{}
	} else {
		mangaInfo.Chapters = chapters
	}

	return mangaInfo, nil
}

// ExecuteGetChapter handles the common pattern for retrieving chapter details
func ExecuteGetChapter(
	ctx context.Context,
	e *engine.Engine,
	agentID string,
	chapterID string,
	apiConfig engine.APIConfig,
	extractorSet engine.ExtractorSet,
	processFunc func(interface{}, string) (*engine.Chapter, error),
) (*engine.Chapter, error) {
	// Log chapter retrieval
	e.Logger.Info("[%s] Fetching chapter details for: %s", agentID, chapterID)

	// Apply rate limiting
	domain := e.ExtractDomain(apiConfig.BaseURL)
	e.RateLimiter.Wait(domain)

	// Fetch chapter details using API service
	response, err := e.API.FetchFromAPI(
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
	result, err := e.Extractor.Extract(extractorSet, response)
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
	chapter, ok := result.(*engine.Chapter)
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
	e *engine.Engine,
	agentID string,
	agentName string,
	chapterID string,
	destDir string,
	getChapterFunc func(context.Context, string) (*engine.Chapter, error),
	getMangaForChapterFunc func(context.Context, string) (*engine.Manga, error),
) error {
	// Log download request
	e.Logger.Info("[%s] Downloading chapter: %s to %s", agentID, chapterID, destDir)

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
		e.Logger.Debug("[%s] Couldn't find manga for chapter %s, using fallback title", agentID, chapterID)
		mangaTitle = fmt.Sprintf("%s-%s", agentName, chapterID)
	}

	// Extract chapter and volume numbers
	chapterNum := &chapter.Info.Number
	if *chapterNum == 0 {
		chapterNum = nil
	}

	// Check for volume override in context
	var volumeNum *int
	if vol, hasOverride := engine.GetVolumeOverride(ctx); hasOverride {
		volumeNum = &vol
	} else if chapter.Info.Title != "" {
		// Try to extract volume from title if not overridden
		_, extractedVol := e.Metadata.ExtractChapterInfo(chapter.Info.Title)
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
	metadata := engine.ChapterMetadata{
		MangaID:      mangaID,
		MangaTitle:   mangaTitle,
		ChapterID:    chapterID,
		ChapterNum:   chapterNum,
		VolumeNum:    volumeNum,
		ChapterTitle: chapter.Info.Title,
		AgentID:      agentID,
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
	concurrency := e.Download.MaxConcurrency
	if contextConcurrency := engine.GetConcurrency(ctx, concurrency); contextConcurrency > 0 {
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
				time.Sleep(e.HTTP.ThrottleTimeAPI)
			} else {
				time.Sleep(e.HTTP.ThrottleTimeImages)
			}
		},
	}

	// Log and start download
	e.Logger.Info("[%s] Downloading %d pages for chapter %s", agentID, len(chapter.Pages), chapterID)

	// Use the engine's download service to download the chapter
	err = e.Download.DownloadChapter(ctx, config)
	if err != nil {
		e.Logger.Error("[%s] Download failed: %v", agentID, err)
		return &errors.AgentError{
			AgentID:      agentID,
			ResourceType: "chapter",
			ResourceID:   chapterID,
			Message:      "Download failed",
			Err:          err,
		}
	}

	e.Logger.Info("[%s] Successfully downloaded chapter %s", agentID, chapterID)
	return nil
}
