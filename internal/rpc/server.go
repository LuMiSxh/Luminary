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
	"Luminary/pkg/core"
	"Luminary/pkg/engine"
	"Luminary/pkg/engine/logger"
	"Luminary/pkg/errors"
	"context"
	"fmt"
	"net/rpc"
	"runtime"
	"strings"
	"time"
)

// RuntimeInfo contains runtime information
type RuntimeInfo struct {
	GoVersion string
	OS        string
	Arch      string
}

// Server wraps all RPC services
type Server struct {
	engine  *engine.Engine
	version string
}

// NewServer creates a new RPC server with all services registered
func NewServer(e *engine.Engine, version string) *rpc.Server {
	server := rpc.NewServer()

	// Create service container
	services := &Server{
		engine:  e,
		version: version,
	}

	// Register services
	err := server.RegisterName("Version", &VersionService{server: services})
	if err != nil {
		return nil
	}
	err = server.RegisterName("Providers", &ProvidersService{server: services})
	if err != nil {
		return nil
	}
	err = server.RegisterName("Search", &SearchService{server: services})
	if err != nil {
		return nil
	}
	err = server.RegisterName("Info", &InfoService{server: services})
	if err != nil {
		return nil
	}
	err = server.RegisterName("Download", &DownloadService{server: services})
	if err != nil {
		return nil
	}
	err = server.RegisterName("List", &ListService{server: services})
	if err != nil {
		return nil
	}

	return server
}

// --- Version Service ---

type VersionService struct {
	server *Server
}

type VersionRequest struct{}

type VersionResponse struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	LogFile   string `json:"log_file,omitempty"`
}

func (s *VersionService) Get(req *VersionRequest, resp *VersionResponse) error {
	*resp = VersionResponse{
		Version:   s.server.version,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	if logFile := s.server.engine.Logger.(*logger.Service).LogFile(); logFile != "" {
		resp.LogFile = logFile
	} else {
		resp.LogFile = "disabled"
	}

	return nil
}

// --- Providers Service ---

type ProvidersService struct {
	server *Server
}

type ProvidersRequest struct{}

type ProviderInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ProvidersResponse []ProviderInfo

func (s *ProvidersService) List(req *ProvidersRequest, resp *ProvidersResponse) error {
	providers := s.server.engine.AllProviders()

	result := make([]ProviderInfo, len(providers))
	for i, p := range providers {
		result[i] = ProviderInfo{
			ID:          p.ID(),
			Name:        p.Name(),
			Description: p.Description(),
		}
	}

	*resp = result
	return nil
}

// --- Search Service ---

type SearchService struct {
	server *Server
}

type SearchRequest struct {
	Query            string `json:"query"`
	Provider         string `json:"provider,omitempty"`
	Limit            int    `json:"limit,omitempty"`
	Pages            int    `json:"pages,omitempty"`
	Sort             string `json:"sort,omitempty"`
	IncludeAltTitles bool   `json:"include_alt_titles,omitempty"`
	Concurrency      int    `json:"concurrency,omitempty"`
}

type SearchResultItem struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Provider     string   `json:"provider"`
	ProviderName string   `json:"provider_name"`
	AltTitles    []string `json:"alt_titles,omitempty"`
	Authors      []string `json:"authors,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

type SearchResponse struct {
	Query   string             `json:"query"`
	Results []SearchResultItem `json:"results"`
	Count   int                `json:"count"`
}

func (s *SearchService) Search(req *SearchRequest, resp *SearchResponse) error {
	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Pages <= 0 {
		req.Pages = 1
	}

	// Create search options
	options := core.SearchOptions{
		Query:            req.Query,
		Limit:            req.Limit,
		Pages:            req.Pages,
		Sort:             req.Sort,
		IncludeAltTitles: req.IncludeAltTitles,
		Concurrency:      req.Concurrency,
	}

	var results []SearchResultItem
	ctx := context.Background()

	if req.Provider != "" {
		// Search single provider
		provider, err := s.server.engine.GetProvider(req.Provider)
		if err != nil {
			return errors.Track(err).AsProvider(req.Provider).Error()
		}

		mangas, err := provider.Search(ctx, req.Query, options)
		if err != nil {
			return errors.Track(err).AsProvider(req.Provider).Error()
		}

		for _, manga := range mangas {
			results = append(results, SearchResultItem{
				ID:           fmt.Sprintf("%s:%s", req.Provider, manga.ID),
				Title:        manga.Title,
				Provider:     req.Provider,
				ProviderName: provider.Name(),
				AltTitles:    manga.AlternativeTitles,
				Authors:      manga.Authors,
				Tags:         manga.Tags,
			})
		}
	} else {
		// Search all providers
		for _, provider := range s.server.engine.AllProviders() {
			mangas, err := provider.Search(ctx, req.Query, options)
			if err != nil {
				s.server.engine.Logger.Error("Search failed for %s: %v", provider.ID(), err)
				continue
			}

			for _, manga := range mangas {
				results = append(results, SearchResultItem{
					ID:           fmt.Sprintf("%s:%s", provider.ID(), manga.ID),
					Title:        manga.Title,
					Provider:     provider.ID(),
					ProviderName: provider.Name(),
					AltTitles:    manga.AlternativeTitles,
					Authors:      manga.Authors,
					Tags:         manga.Tags,
				})
			}
		}
	}

	*resp = SearchResponse{
		Query:   req.Query,
		Results: results,
		Count:   len(results),
	}

	return nil
}

// --- Info Service ---

type InfoService struct {
	server *Server
}

type InfoRequest struct {
	MangaID        string `json:"manga_id"`
	LanguageFilter string `json:"language_filter,omitempty"`
	ShowLanguages  bool   `json:"show_languages,omitempty"`
}

type InfoResponse struct {
	ID                   string             `json:"id"`
	Title                string             `json:"title"`
	Provider             string             `json:"provider"`
	ProviderName         string             `json:"provider_name"`
	Description          string             `json:"description"`
	Authors              []string           `json:"authors"`
	Status               string             `json:"status"`
	Tags                 []string           `json:"tags"`
	Chapters             []core.ChapterInfo `json:"chapters"`
	ChapterCount         int                `json:"chapter_count"`
	LastUpdated          *time.Time         `json:"last_updated,omitempty"`
	AvailableLanguages   []string           `json:"available_languages,omitempty"`
	FilteredChapters     bool               `json:"filtered_chapters,omitempty"`
	OriginalChapterCount int                `json:"original_chapter_count,omitempty"`
}

func (s *InfoService) Get(req *InfoRequest, resp *InfoResponse) error {
	// Parse combined ID
	parts := strings.SplitN(req.MangaID, ":", 2)
	if len(parts) != 2 {
		return errors.Newf("invalid manga ID format: %s", req.MangaID).Error()
	}

	providerID, mangaID := parts[0], parts[1]

	// Get provider
	provider, err := s.server.engine.GetProvider(providerID)
	if err != nil {
		return errors.Track(err).AsProvider(providerID).Error()
	}

	// Get manga info
	ctx := context.Background()
	info, err := provider.GetManga(ctx, mangaID)
	if err != nil {
		return errors.Track(err).AsProvider(providerID).Error()
	}

	// Build response
	*resp = InfoResponse{
		ID:           req.MangaID,
		Title:        info.Title,
		Provider:     providerID,
		ProviderName: provider.Name(),
		Description:  info.Description,
		Authors:      info.Authors,
		Status:       info.Status,
		Tags:         info.Tags,
		Chapters:     info.Chapters,
		ChapterCount: len(info.Chapters),
		LastUpdated:  info.LastUpdated,
	}

	// Handle language filtering
	if req.LanguageFilter != "" {
		languages := strings.Split(req.LanguageFilter, ",")
		filteredChapters := filterChaptersByLanguage(info.Chapters, languages)

		resp.Chapters = filteredChapters
		resp.ChapterCount = len(filteredChapters)
		resp.FilteredChapters = true
		resp.OriginalChapterCount = len(info.Chapters)
	}

	// Include available languages if requested
	if req.ShowLanguages {
		resp.AvailableLanguages = getAvailableLanguages(info.Chapters)
	}

	return nil
}

// --- Download Service ---

type DownloadService struct {
	server *Server
}

type DownloadRequest struct {
	ChapterID string `json:"chapter_id"`
	OutputDir string `json:"output_dir"`
}

type DownloadResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Path      string `json:"path,omitempty"`
	PageCount int    `json:"page_count,omitempty"`
}

func (s *DownloadService) Chapter(req *DownloadRequest, resp *DownloadResponse) error {
	// Parse combined ID
	parts := strings.SplitN(req.ChapterID, ":", 2)
	if len(parts) != 2 {
		return errors.Newf("invalid chapter ID format: %s", req.ChapterID).Error()
	}

	providerID, chapterID := parts[0], parts[1]

	// Get provider
	provider, err := s.server.engine.GetProvider(providerID)
	if err != nil {
		return errors.Track(err).AsProvider(providerID).Error()
	}

	// Download chapter
	ctx := context.Background()
	if err := provider.DownloadChapter(ctx, chapterID, req.OutputDir); err != nil {
		return errors.Track(err).AsProvider(providerID).Error()
	}

	*resp = DownloadResponse{
		Success: true,
		Message: "Chapter downloaded successfully",
		Path:    req.OutputDir,
	}

	return nil
}

// --- List Service ---

type ListService struct {
	server *Server
}

type ListRequest struct {
	Provider string `json:"provider,omitempty"`
	Limit    int    `json:"limit,omitempty"`
	Page     int    `json:"page,omitempty"`
}

type ListItem struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Provider     string `json:"provider"`
	ProviderName string `json:"provider_name"`
}

type ListResponse struct {
	Results      []ListItem `json:"results"`
	Count        int        `json:"count"`
	Provider     string     `json:"provider,omitempty"`
	ProviderName string     `json:"provider_name,omitempty"`
}

func (s *ListService) Latest(req *ListRequest, resp *ListResponse) error {
	// This would typically fetch latest manga from providers
	// For now, return empty as this requires provider-specific implementation
	*resp = ListResponse{
		Results: []ListItem{},
		Count:   0,
	}

	if req.Provider != "" {
		if provider := s.server.engine.GetProviderOrNil(req.Provider); provider != nil {
			resp.Provider = req.Provider
			resp.ProviderName = provider.Name()
		}
	} else {
		resp.ProviderName = "Multiple Providers"
	}

	return nil
}

// Helper functions

func filterChaptersByLanguage(chapters []core.ChapterInfo, languages []string) []core.ChapterInfo {
	var filtered []core.ChapterInfo

	langMap := make(map[string]bool)
	for _, lang := range languages {
		langMap[strings.ToLower(lang)] = true
	}

	for _, ch := range chapters {
		if ch.Language == "" || langMap[strings.ToLower(ch.Language)] {
			filtered = append(filtered, ch)
		}
	}

	return filtered
}

func getAvailableLanguages(chapters []core.ChapterInfo) []string {
	langMap := make(map[string]bool)

	for _, ch := range chapters {
		if ch.Language != "" {
			langMap[ch.Language] = true
		}
	}

	var languages []string
	for lang := range langMap {
		languages = append(languages, lang)
	}

	return languages
}
