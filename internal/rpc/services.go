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

package rpc

import (
	"Luminary/pkg/engine"
	"Luminary/pkg/engine/core"
	"Luminary/pkg/errors"
	"Luminary/pkg/provider"
	"Luminary/pkg/util"
	"context"
	"fmt"
	"sync"
	"time"
)

// Services is a container for all RPC services, holding shared resources.
type Services struct {
	engine  *engine.Engine
	version string
}

// NewServices creates a new Services container.
func NewServices(e *engine.Engine, version string) *Services {
	return &Services{
		engine:  e,
		version: version,
	}
}

// --- Version Service ---

// VersionService handles version-related RPC calls.
type VersionService struct {
	Services *Services
}

// VersionInfo holds the data for the version response.
type VersionInfo struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	LogFile   string `json:"log_file,omitempty"`
}

// Get returns the version information of the Luminary application.
func (s *VersionService) Get(args *struct{}, reply *VersionInfo) error {
	*reply = VersionInfo{
		Version: s.Services.version,
	}

	runtimeInfo := GetRuntimeInfo()
	reply.GoVersion = runtimeInfo.GoVersion
	reply.OS = runtimeInfo.OS
	reply.Arch = runtimeInfo.Arch

	if s.Services.engine != nil && s.Services.engine.Logger != nil {
		logFile := s.Services.engine.Logger.LogFile
		if logFile != "" {
			reply.LogFile = logFile
		} else {
			reply.LogFile = "disabled"
		}
	}
	return nil
}

// --- Providers Service ---

// ProvidersService handles provider-listing RPC calls.
type ProvidersService struct {
	Services *Services
}

// ProviderInfo holds information about a single provider.
type ProviderInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// List returns a list of all available manga providers.
func (s *ProvidersService) List(args *struct{}, reply *[]ProviderInfo) error {
	providers := s.Services.engine.AllProvider()
	result := make([]ProviderInfo, 0, len(providers))

	for _, p := range providers {
		result = append(result, ProviderInfo{
			ID:          p.ID(),
			Name:        p.Name(),
			Description: p.Description(),
		})
	}
	*reply = result
	return nil
}

// --- Search Service ---

// SearchService handles manga search RPC calls.
type SearchService struct {
	Services *Services
}

// SearchRequest defines the parameters for a search operation.
type SearchRequest struct {
	Query            string            `json:"query"`
	Provider         string            `json:"provider,omitempty"` // Specific provider ID, or empty for all
	Limit            int               `json:"limit,omitempty"`
	Pages            int               `json:"pages,omitempty"`
	Sort             string            `json:"sort,omitempty"`
	Fields           []string          `json:"fields,omitempty"`
	Filters          map[string]string `json:"filters,omitempty"`
	IncludeAltTitles bool              `json:"include_alt_titles,omitempty"`
	Concurrency      int               `json:"concurrency,omitempty"`
}

// SearchResultItem represents a single manga item in search results.
type SearchResultItem struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Provider     string   `json:"provider"`
	ProviderName string   `json:"provider_name,omitempty"`
	AltTitles    []string `json:"alt_titles,omitempty"`
	Authors      []string `json:"authors,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

// SearchResponse defines the structure of a search operation's response.
type SearchResponse struct {
	Query   string             `json:"query"`
	Results []SearchResultItem `json:"results"`
	Count   int                `json:"count"`
}

// Search performs a manga search based on the provided request.
func (s *SearchService) Search(args *SearchRequest, reply *SearchResponse) error {
	// Input validation (unchanged)
	if args.Limit <= 0 {
		args.Limit = 10
	}
	if args.Pages <= 0 {
		args.Pages = 1
	}
	if args.Sort == "" {
		args.Sort = "relevance"
	}
	if args.Concurrency <= 0 {
		args.Concurrency = 5
	}

	options := core.SearchOptions{
		Limit: args.Limit, Pages: args.Pages, Fields: args.Fields,
		Filters: args.Filters, Sort: args.Sort,
	}

	multipleProviders := args.Provider == ""
	timeoutDuration := calculateSearchTimeout(args.Limit, args.Pages, multipleProviders)
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()
	ctx = core.WithConcurrency(ctx, args.Concurrency)

	var providersToSearch []provider.Provider
	if args.Provider != "" {
		p, exists := s.Services.engine.GetProvider(args.Provider)
		if !exists {
			return ProviderNotFound(args.Provider) // ← SIMPLE!
		}
		providersToSearch = append(providersToSearch, p)
	} else {
		providersToSearch = s.Services.engine.AllProvider()
	}

	results, err := s.Services.engine.Search.SearchAcrossProviders(ctx, providersToSearch, args.Query, options)
	if err != nil {
		return SearchFailed(err, args.Query) // ← AUTOMATIC FUNCTION TRACKING!
	}

	// Process results (unchanged)
	var allResults []SearchResultItem
	for providerID, mangaList := range results {
		prov, _ := s.Services.engine.GetProvider(providerID)
		for _, manga := range mangaList {
			item := SearchResultItem{
				ID:    core.FormatMangaID(providerID, manga.ID),
				Title: manga.Title, Provider: providerID, ProviderName: prov.Name(),
			}
			if args.IncludeAltTitles {
				item.AltTitles = manga.AltTitles
			}
			item.Authors = manga.Authors
			item.Tags = manga.Tags
			allResults = append(allResults, item)
		}
	}

	*reply = SearchResponse{Query: args.Query, Results: allResults, Count: len(allResults)}
	return nil
}

// --- Info Service ---

// InfoService handles RPC calls for fetching detailed manga information.
type InfoService struct {
	Services *Services
}

// InfoRequest defines the parameters for fetching manga info.
type InfoRequest struct {
	MangaID        string `json:"manga_id"`                  // Expected format: "provider:id"
	LanguageFilter string `json:"language_filter,omitempty"` // Comma-separated language codes/names
	ShowLanguages  bool   `json:"show_languages,omitempty"`  // Whether to include available languages in response
}

// MangaInfo holds detailed information about a manga.
type MangaInfo struct {
	ID                   string             `json:"id"`
	Title                string             `json:"title"`
	Provider             string             `json:"provider"`
	ProviderName         string             `json:"provider_name"`
	Description          string             `json:"description,omitempty"`
	Authors              []string           `json:"authors,omitempty"`
	Status               string             `json:"status,omitempty"`
	Tags                 []string           `json:"tags,omitempty"`
	Chapters             []core.ChapterInfo `json:"chapters"`
	ChapterCount         int                `json:"chapter_count"`
	LastUpdated          *string            `json:"last_updated,omitempty"`           // Nullable date in RFC3339 format
	AvailableLanguages   []string           `json:"available_languages,omitempty"`    // Available languages if requested
	FilteredChapters     bool               `json:"filtered_chapters,omitempty"`      // Whether chapters were filtered
	OriginalChapterCount int                `json:"original_chapter_count,omitempty"` // Original count before filtering
}

// Get retrieves detailed information for a specific manga.
func (s *InfoService) Get(args *InfoRequest, reply *MangaInfo) error {
	providerID, mangaID, err := core.ParseMangaID(args.MangaID)
	if err != nil {
		return InvalidInput("manga_id", args.MangaID) // ← USING HELPER!
	}

	prov, exists := s.Services.engine.GetProvider(providerID)
	if !exists {
		return ProviderNotFound(providerID) // ← USING HELPER!
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	manga, err := prov.GetManga(ctx, mangaID)
	if err != nil {
		if errors.IsNotFound(err) {
			return ResourceNotFound("manga", mangaID) // ← USING HELPER!
		}
		return FetchInfoFailed(err, args.MangaID) // ← NEW HELPER!
	}
	if manga == nil || manga.Title == "" {
		return InvalidMangaData(args.MangaID, "empty or missing title") // ← NEW HELPER!
	}
	if manga.Chapters == nil {
		manga.Chapters = []core.ChapterInfo{}
	}

	// Store original chapter information
	originalChapters := manga.Chapters
	originalChapterCount := len(originalChapters)
	filteredChapters := false

	// Apply language filtering if requested
	if args.LanguageFilter != "" {
		languageFilter := util.NewLanguageFilter(args.LanguageFilter)
		if languageFilter != nil {
			manga.Chapters = languageFilter.FilterChapters(manga.Chapters)
			filteredChapters = len(manga.Chapters) < originalChapterCount
		}
	}

	chapters := make([]core.ChapterInfo, len(manga.Chapters))
	for i, ch := range manga.Chapters {
		chapters[i] = core.ChapterInfo{
			ID:       core.FormatMangaID(providerID, ch.ID), // Chapter ID needs provider prefix too
			Title:    ch.Title,
			Number:   ch.Number,
			Date:     nil, // Initialize to nil
			Language: nil, // Initialize to nil
		}

		// Handle nullable date
		if ch.Date != nil {
			chapters[i].Date = ch.Date
		}

		// Handle nullable language
		if ch.Language != nil {
			chapters[i].Language = ch.Language
		}
	}

	var lastUpdatedStr *string
	if manga.LastUpdated != nil {
		str := manga.LastUpdated.Format(time.RFC3339)
		lastUpdatedStr = &str
	}

	// Prepare available languages if requested
	var availableLanguages []string
	if args.ShowLanguages {
		availableLanguages = util.GetAvailableLanguages(originalChapters)
	}

	*reply = MangaInfo{
		ID:                   core.FormatMangaID(providerID, manga.ID),
		Title:                manga.Title,
		Provider:             providerID,
		ProviderName:         prov.Name(),
		Description:          manga.Description,
		Authors:              manga.Authors,
		Status:               manga.Status,
		Tags:                 manga.Tags,
		Chapters:             chapters,
		ChapterCount:         len(chapters),
		LastUpdated:          lastUpdatedStr,
		AvailableLanguages:   availableLanguages,
		FilteredChapters:     filteredChapters,
		OriginalChapterCount: originalChapterCount,
	}
	return nil
}

// --- Download Service ---

// DownloadService handles manga chapter download RPC calls.
type DownloadService struct {
	Services *Services
}

// DownloadRequest defines parameters for a chapter download operation.
type DownloadRequest struct {
	ChapterID   string `json:"chapter_id"` // Expected format: "provider:id"
	OutputDir   string `json:"output_dir,omitempty"`
	Volume      *int   `json:"volume,omitempty"` // Pointer to allow distinguishing between 0 and not set
	Concurrency int    `json:"concurrency,omitempty"`
}

// DownloadResponse defines the structure of a download operation's response.
type DownloadResponse struct {
	ChapterID    string `json:"chapter_id"` // The original chapter ID part (without provider)
	Provider     string `json:"provider"`
	ProviderName string `json:"provider_name"`
	OutputDir    string `json:"output_dir"`
	Success      bool   `json:"success"`
	Message      string `json:"message,omitempty"`
}

// Download initiates the download of a manga chapter.
func (s *DownloadService) Download(args *DownloadRequest, reply *DownloadResponse) error {
	providerID, chapterIDPart, err := core.ParseMangaID(args.ChapterID)
	if err != nil {
		return InvalidInput("chapter_id", args.ChapterID) // ← SIMPLE!
	}

	prov, exists := s.Services.engine.GetProvider(providerID)
	if !exists {
		return ProviderNotFound(providerID) // ← SIMPLE!
	}

	outputDir := args.OutputDir
	if outputDir == "" {
		outputDir = "./downloads" // Default output directory
	}

	concurrency := args.Concurrency
	if concurrency <= 0 {
		concurrency = 5 // Default concurrency
	}

	baseCtx := context.Background()
	ctx := core.WithConcurrency(baseCtx, concurrency)

	if args.Volume != nil {
		ctx = core.WithVolumeOverride(ctx, *args.Volume)
	}

	downloadCtx, cancel := context.WithTimeout(ctx, 10*time.Minute) // 10-minute timeout for download
	defer cancel()

	err = prov.DownloadChapter(downloadCtx, chapterIDPart, outputDir)
	if err != nil {
		*reply = DownloadResponse{
			ChapterID: chapterIDPart, Provider: providerID, ProviderName: prov.Name(),
			OutputDir: outputDir, Success: false,
			Message: fmt.Sprintf("Failed to download chapter: %v", err),
		}

		if errors.IsNotFound(err) {
			return ResourceNotFound("chapter", chapterIDPart)
		}

		return DownloadFailed(err, args.ChapterID) // ← AUTOMATIC TRACKING!
	}

	*reply = DownloadResponse{
		ChapterID: chapterIDPart, Provider: providerID, ProviderName: prov.Name(),
		OutputDir: outputDir, Success: true,
		Message: fmt.Sprintf("Successfully downloaded chapter %s to %s", args.ChapterID, outputDir),
	}
	return nil
}

// --- List Service ---

// ListService handles RPC calls for listing manga.
type ListService struct {
	Services *Services
}

// ListRequest defines parameters for a manga listing operation.
type ListRequest struct {
	Provider    string `json:"provider,omitempty"` // Specific provider ID, or empty for all
	Limit       int    `json:"limit,omitempty"`
	Pages       int    `json:"pages,omitempty"`
	Concurrency int    `json:"concurrency,omitempty"`
}

// ListResponse defines the structure of a list operation's response.
// It reuses SearchResultItem for individual manga.
type ListResponse struct {
	Mangas       []SearchResultItem `json:"mangas"`
	Count        int                `json:"count"`
	Provider     string             `json:"provider,omitempty"` // Included if a specific provider was requested
	ProviderName string             `json:"provider_name,omitempty"`
}

// List retrieves a list of manga, potentially filtered by provider.
func (s *ListService) List(args *ListRequest, reply *ListResponse) error {
	if args.Limit <= 0 {
		args.Limit = 50 // Default limit
	}
	// Pages = 0 can mean all, so we respect that if passed. Default to 1 if not set.
	if args.Pages == 0 && args.Limit != 0 { // If limit is set but pages is 0, assume 1 page.
		args.Pages = 1
	}

	if args.Concurrency <= 0 {
		args.Concurrency = 5
	}

	options := core.SearchOptions{
		Limit: args.Limit,
		Pages: args.Pages,
		// Empty query for listing
	}

	multipleProviders := args.Provider == ""
	timeoutDuration := calculateListTimeout(args.Limit, args.Pages, multipleProviders)

	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()
	ctx = core.WithConcurrency(ctx, args.Concurrency)

	var allMangas []SearchResultItem

	if !multipleProviders {
		p, exists := s.Services.engine.GetProvider(args.Provider)
		if !exists {
			return ProviderNotFound(args.Provider) // ← USING HELPER!
		}
		mangas, err := p.Search(ctx, "", options) // Empty query for listing
		if err != nil {
			return ListMangaFailed(err, args.Provider) // ← NEW HELPER!
		}
		for _, manga := range mangas {
			allMangas = append(allMangas, SearchResultItem{
				ID:           core.FormatMangaID(p.ID(), manga.ID),
				Title:        manga.Title,
				Provider:     p.ID(),
				ProviderName: p.Name(),
				Authors:      manga.Authors,
				Tags:         manga.Tags,
			})
		}
		reply.Provider = p.ID()
		reply.ProviderName = p.Name()
	} else {
		// Concurrent fetching from all providers
		providersList := s.Services.engine.AllProvider()
		if len(providersList) == 0 {
			reply.Mangas = []SearchResultItem{}
			reply.Count = 0
			return nil
		}

		var wg sync.WaitGroup
		resultBatchesChan := make(chan []SearchResultItem, len(providersList))
		errorChan := make(chan error, len(providersList))

		for _, p := range providersList {
			wg.Add(1)
			go func(currentProvider provider.Provider) {
				defer wg.Done()
				mangas, err := currentProvider.Search(ctx, "", options)
				if err != nil {
					if ctx.Err() != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, ctx.Err())) {
						s.Services.engine.Logger.Debug("Provider %s search cancelled/timed out: %v", currentProvider.ID(), err)
						return
					}
					errorChan <- fmt.Errorf("provider %s list error: %w", currentProvider.ID(), err)
					return
				}

				providerResults := make([]SearchResultItem, 0, len(mangas))
				for _, manga := range mangas {
					providerResults = append(providerResults, SearchResultItem{
						ID:           core.FormatMangaID(currentProvider.ID(), manga.ID),
						Title:        manga.Title,
						Provider:     currentProvider.ID(),
						ProviderName: currentProvider.Name(),
						Authors:      manga.Authors,
						Tags:         manga.Tags,
					})
				}
				if len(providerResults) > 0 {
					select {
					case resultBatchesChan <- providerResults:
					case <-ctx.Done():
						s.Services.engine.Logger.Debug("Context done, not sending results from provider %s", currentProvider.ID())
						return
					}
				}
			}(p)
		}

		// Goroutine to close channels once all workers are done
		waitAndCloseDone := make(chan struct{})
		go func() {
			wg.Wait()
			close(resultBatchesChan)
			close(errorChan)
			close(waitAndCloseDone)
		}()

		// Collect results and errors
		collecting := true
		var returnErr error

	collectLoop:
		for collecting {
			select {
			case batch, ok := <-resultBatchesChan:
				if ok {
					allMangas = append(allMangas, batch...)
				} else {
					resultBatchesChan = nil
				}
			case err, ok := <-errorChan:
				if ok {
					if err != nil {
						s.Services.engine.Logger.Warn("Error during manga list sub-operation: %v", err)
					}
				} else {
					errorChan = nil
				}
			case <-ctx.Done():
				s.Services.engine.Logger.Error("Listing operation timed out globally: %v", ctx.Err())
				returnErr = Timeout("list", timeoutDuration) // ← USING HELPER!
				select {
				case <-waitAndCloseDone:
				case <-time.After(2 * time.Second):
					s.Services.engine.Logger.Warn("Grace period for channel cleanup exceeded")
				}
				resultBatchesChan = nil
				errorChan = nil
				break collectLoop
			}

			if resultBatchesChan == nil && errorChan == nil {
				collecting = false
			}
		}

		reply.Provider = ""
		reply.ProviderName = "Multiple Providers"
		if returnErr != nil {
			reply.Mangas = allMangas
			reply.Count = len(allMangas)
			return returnErr
		}
	}

	reply.Mangas = allMangas
	reply.Count = len(allMangas)
	if len(allMangas) == 0 && (args.Provider != "" || multipleProviders) {
		s.Services.engine.Logger.Warn("No manga found for the given criteria")
	} else if len(allMangas) > 0 {
		s.Services.engine.Logger.Info("Listed %d mangas successfully", len(allMangas))
	}
	return nil
}

// --- Timeout Calculation Helpers (similar to commands) ---

func calculateSearchTimeout(limit, pages int, multipleProviders bool) time.Duration {
	timeoutDuration := 30 * time.Second
	if limit == 0 || pages == 0 {
		timeoutDuration = 15 * time.Minute
		if limit == 0 && pages == 0 {
			timeoutDuration = 30 * time.Minute
		}
	} else if pages > 3 || limit > 50 {
		timeoutDuration = 5 * time.Minute
		if pages > 5 {
			pageTimeoutFactor := time.Duration(pages) / 5
			if pageTimeoutFactor > 1 {
				extraTimeout := pageTimeoutFactor * 2 * time.Minute
				if extraTimeout > 10*time.Minute {
					extraTimeout = 10 * time.Minute
				}
				timeoutDuration += extraTimeout
			}
		}
	}
	if multipleProviders {
		timeoutDuration += 5 * time.Minute
	}
	return timeoutDuration
}

func calculateListTimeout(limit, pages int, multipleProviders bool) time.Duration {
	timeoutDuration := 60 * time.Second
	if limit == 0 || pages == 0 {
		timeoutDuration = 5 * time.Minute
		if limit == 0 && pages == 0 {
			timeoutDuration = 10 * time.Minute
		}
	} else if pages > 3 || limit > 50 {
		timeoutDuration = 3 * time.Minute
		if pages > 5 {
			pageTimeoutFactor := time.Duration(pages) / 5
			if pageTimeoutFactor > 1 {
				extraTimeout := pageTimeoutFactor * time.Minute
				if extraTimeout > 5*time.Minute {
					extraTimeout = 5 * time.Minute
				}
				timeoutDuration += extraTimeout
			}
		}
	}
	if multipleProviders {
		timeoutDuration += 2 * time.Minute
	}
	return timeoutDuration
}
