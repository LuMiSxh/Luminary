package web

import (
	"Luminary/engine"
	"Luminary/engine/frameworks/common"
	"context"
	"fmt"
)

// HTMLAgentConfig holds the configuration for HTML-based agents
type HTMLAgentConfig struct {
	ID          string            // Short identifier
	Name        string            // Display name
	SiteURL     string            // Base URL
	Description string            // Site description
	Headers     map[string]string // Default HTTP headers
}

// HTMLAgent provides a base implementation for HTML-based (non-API) agents
type HTMLAgent struct {
	config     HTMLAgentConfig
	engine     *engine.Engine
	webScraper *engine.WebScraperService
}

// NewHTMLAgent creates a new HTML-based agent
func NewHTMLAgent(e *engine.Engine, config HTMLAgentConfig) *HTMLAgent {
	// Create web scraper service if not created yet
	if e.WebScraper == nil {
		e.WebScraper = engine.NewWebScraperService(e.HTTP, e.DOM, e.RateLimiter, e.Logger)
	}

	return &HTMLAgent{
		config:     config,
		engine:     e,
		webScraper: e.WebScraper,
	}
}

// ID returns the agent's identifier
func (a *HTMLAgent) ID() string {
	return a.config.ID
}

// Name returns the agent's display name
func (a *HTMLAgent) Name() string {
	return a.config.Name
}

// Description returns the agent's description
func (a *HTMLAgent) Description() string {
	return a.config.Description
}

// SiteURL returns the agent's website URL
func (a *HTMLAgent) SiteURL() string {
	return a.config.SiteURL
}

// Config returns the agent's configuration
func (a *HTMLAgent) Config() HTMLAgentConfig {
	return a.config
}

// Initialize initializes the agent
func (a *HTMLAgent) Initialize(ctx context.Context) error {
	return common.ExecuteInitialize(ctx, a.engine, a.ID(), a.Name(), func(ctx context.Context) error {
		// No special initialization needed for HTML agents
		return nil
	})
}

// CreateRequest creates a new scraper request with the agent's default headers
func (a *HTMLAgent) CreateRequest(url string) *engine.ScraperRequest {
	req := engine.NewScraperRequest(url)

	// Add default headers
	for key, value := range a.config.Headers {
		req.SetHeader(key, value)
	}

	return req
}

// FetchPage fetches a web page using the agent's configuration
func (a *HTMLAgent) FetchPage(ctx context.Context, url string) (*engine.WebPage, error) {
	req := a.CreateRequest(url)
	return a.webScraper.FetchPage(ctx, req)
}

// Search implements a basic search for HTML-based agents
// Should be overridden by specific agent types
func (a *HTMLAgent) Search(ctx context.Context, query string, options engine.SearchOptions) ([]engine.Manga, error) {
	a.engine.Logger.Info("[%s] Searching for: %s", a.ID(), query)
	return nil, fmt.Errorf("search not implemented for HTML agent %s", a.ID())
}

// GetManga retrieves manga details
// Should be overridden by specific agent types
func (a *HTMLAgent) GetManga(ctx context.Context, id string) (*engine.MangaInfo, error) {
	a.engine.Logger.Info("[%s] Getting manga details for: %s", a.ID(), id)
	return nil, fmt.Errorf("GetManga not implemented for HTML agent %s", a.ID())
}

// GetChapter retrieves chapter details
// Should be overridden by specific agent types
func (a *HTMLAgent) GetChapter(ctx context.Context, chapterID string) (*engine.Chapter, error) {
	a.engine.Logger.Info("[%s] Getting chapter details for: %s", a.ID(), chapterID)
	return nil, fmt.Errorf("GetChapter not implemented for HTML agent %s", a.ID())
}

// TryGetMangaForChapter attempts to get manga info for a chapter
// Could be overridden by specific agent types for optimization
func (a *HTMLAgent) TryGetMangaForChapter(ctx context.Context, chapterID string) (*engine.Manga, error) {
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
// Could be overridden by specific agent types
func (a *HTMLAgent) DownloadChapter(ctx context.Context, chapterID, destDir string) error {
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
