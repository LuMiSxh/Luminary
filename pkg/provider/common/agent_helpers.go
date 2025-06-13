// Luminary: A streamlined CLI tool for searching and downloading manga.
// Copyright (C) 2025 Luca M. Schmidt (LuMiSxh)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"Luminary/pkg/engine"
	"Luminary/pkg/engine/core"
	"Luminary/pkg/engine/downloader"
	"Luminary/pkg/engine/network"
	"Luminary/pkg/engine/parser"
	"Luminary/pkg/engine/search"
	"Luminary/pkg/errors"
	"context"
	"fmt"
	"time"
)

// ExecuteInitialize handles common provider initialization logic
func ExecuteInitialize(ctx context.Context, e *engine.Engine, providerID string, providerName string, initFunc func(context.Context) error) error {
	// Log initialization
	e.Logger.Info("Initializing provider: %s (%s)", providerName, providerID)

	// Call the provider-specific initialization
	err := initFunc(ctx)
	if err != nil {
		e.Logger.Error("Failed to initialize provider %s: %v", providerID, err)
		return errors.TP(err, providerID)
	}

	e.Logger.Info("Provider initialized: %s", providerID)
	return nil
}

// ExecuteSearch handles the common pattern for searching by delegating to the SearchService
func ExecuteSearch(
	ctx context.Context,
	e *engine.Engine,
	providerID string,
	query string,
	options *core.SearchOptions,
	apiConfig network.APIConfig,
	paginationConfig search.PaginationConfig,
	extractorSet parser.ExtractorSet,
) ([]core.Manga, error) {
	// Delegate to the search service
	results, err := e.Search.ExecuteSearch(
		ctx, providerID, query, options, apiConfig, paginationConfig, extractorSet)

	if err != nil {
		return nil, errors.TP(err, providerID)
	}

	return results, nil
}

// ExecuteGetManga handles the common pattern for retrieving manga details
func ExecuteGetManga(
	ctx context.Context,
	e *engine.Engine,
	providerID string,
	mangaID string,
	apiConfig network.APIConfig,
	extractorSet parser.ExtractorSet,
	fetchChaptersFunc func(context.Context, string) ([]core.ChapterInfo, error),
) (*core.MangaInfo, error) {
	// Log manga retrieval
	e.Logger.Info("[%s] Fetching manga details for: %s", providerID, mangaID)

	// Apply rate limiting
	domain := network.ExtractDomain(apiConfig.BaseURL)
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
		return nil, errors.TP(err, providerID)
	}

	// Extract manga data
	result, err := e.Extractor.Extract(extractorSet, response)
	if err != nil {
		return nil, errors.TP(err, providerID)
	}

	// Convert to MangaInfo
	mangaInfo, ok := result.(*core.MangaInfo)
	if !ok {
		return nil, errors.TP(err, providerID)
	}

	// Validate the result
	if mangaInfo.ID == "" {
		mangaInfo.ID = mangaID
	}

	if mangaInfo.Title == "" {
		e.Logger.Warn("[%s] Manga with ID %s has no title", providerID, mangaID)
	}

	// Fetch chapters for this manga
	chapters, err := fetchChaptersFunc(ctx, mangaID)
	if err != nil {
		e.Logger.Warn("[%s] Failed to fetch chapters for manga %s: %v", providerID, mangaID, err)
		// Continue anyway, just with empty chapters list
		mangaInfo.Chapters = []core.ChapterInfo{}
	} else {
		mangaInfo.Chapters = chapters
	}

	return mangaInfo, nil
}

// ExecuteGetChapter handles the common pattern for retrieving chapter details
func ExecuteGetChapter(
	ctx context.Context,
	e *engine.Engine,
	providerID string,
	chapterID string,
	apiConfig network.APIConfig,
	extractorSet parser.ExtractorSet,
	processFunc func(interface{}, string) (*core.Chapter, error),
) (*core.Chapter, error) {
	// Log chapter retrieval
	e.Logger.Info("[%s] Fetching chapter details for: %s", providerID, chapterID)

	// Apply rate limiting
	domain := network.ExtractDomain(apiConfig.BaseURL)
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
		return nil, errors.TP(err, providerID)
	}

	// Process the response with provider-specific logic
	if processFunc != nil {
		chapter, err := processFunc(response, chapterID)
		if err != nil {
			return nil, errors.TP(err, providerID)
		}
		return chapter, nil
	}

	// Or use the general extractor if no custom processing
	result, err := e.Extractor.Extract(extractorSet, response)
	if err != nil {
		return nil, errors.TP(err, providerID)
	}

	// Convert to Chapter
	chapter, ok := result.(*core.Chapter)
	if !ok {
		return nil, errors.TP(fmt.Errorf("expected Chapter, got %T", result), providerID)
	}

	return chapter, nil
}

// ExecuteDownloadChapter handles the common pattern for downloading a chapter
func ExecuteDownloadChapter(
	ctx context.Context,
	e *engine.Engine,
	providerID string,
	providerName string,
	chapterID string,
	destDir string,
	getChapterFunc func(context.Context, string) (*core.Chapter, error),
	getMangaForChapterFunc func(context.Context, string) (*core.Manga, error),
) error {
	// Log download request
	e.Logger.Info("[%s] Downloading chapter: %s to %s", providerID, chapterID, destDir)

	// Get chapter information
	chapter, err := getChapterFunc(ctx, chapterID)
	if err != nil {
		return errors.TP(err, providerID)
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
		e.Logger.Debug("[%s] Couldn't find manga for chapter %s, using fallback title", providerID, chapterID)
		mangaTitle = fmt.Sprintf("%s-%s", providerName, chapterID)
	}

	// Extract chapter and volume numbers
	chapterNum := &chapter.Info.Number
	if *chapterNum == 0 {
		chapterNum = nil
	}

	// Check for volume override in context
	var volumeNum *int
	if vol, hasOverride := core.GetVolumeOverride(ctx); hasOverride {
		volumeNum = &vol
	} else if chapter.Info.Title != "" {
		// Try to extract volume from title if not overridden
		_, extractedVol := e.Metadata.ExtractChapterInfo(chapter.Info.Title)
		volumeNum = extractedVol
	}

	// Make sure there are pages to download
	if len(chapter.Pages) == 0 {
		return errors.TP(err, providerID)
	}

	// Prepare metadata
	metadata := downloader.ChapterMetadata{
		MangaID:      mangaID,
		MangaTitle:   mangaTitle,
		ChapterID:    chapterID,
		ChapterNum:   chapterNum,
		VolumeNum:    volumeNum,
		ChapterTitle: chapter.Info.Title,
		ProviderID:   providerID,
	}

	// Convert pages to download requests
	downloadFiles := make([]downloader.DownloadRequest, len(chapter.Pages))
	for i, page := range chapter.Pages {
		downloadFiles[i] = downloader.DownloadRequest{
			URL:       page.URL,
			Index:     i + 1,
			Filename:  page.Filename,
			PageCount: len(chapter.Pages),
		}
	}

	// Set up download configuration
	// Removed the explicit concurrency setting - will be retrieved from context
	config := downloader.DownloadJobConfig{
		Metadata:  metadata,
		OutputDir: destDir,
		Files:     downloadFiles,
		WaitDuration: func(isRetry bool) {
			if isRetry {
				time.Sleep(e.HTTP.ThrottleTimeAPI)
			} else {
				time.Sleep(e.HTTP.ThrottleTimeImages)
			}
		},
	}

	// Log and start download
	e.Logger.Info("[%s] Downloading %d pages for chapter %s", providerID, len(chapter.Pages), chapterID)

	// Use the engine's download service to download the chapter
	// The context already contains concurrency settings
	err = e.Download.DownloadChapter(ctx, config)
	if err != nil {
		e.Logger.Error("[%s] Download failed: %v", providerID, err)
		return errors.TP(err, providerID)
	}

	e.Logger.Info("[%s] Successfully downloaded chapter %s", providerID, chapterID)
	return nil
}
