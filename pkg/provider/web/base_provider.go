package web

import (
	"Luminary/pkg/engine"
	"Luminary/pkg/engine/core"
	"Luminary/pkg/engine/network"
	"Luminary/pkg/provider/common"
	"context"
	"fmt"
)

// Config holds the configuration for HTML-based providers
type Config struct {
	ID          string            // Short identifier
	Name        string            // Display name
	SiteURL     string            // Base URL
	Description string            // Site description
	Headers     map[string]string // Default HTTP headers
}

// Provider provides a base implementation for HTML-based (non-API) providers
type Provider struct {
	config     Config
	engine     *engine.Engine
	webScraper *network.WebScraperService
}

// NewProvider creates a new HTML-based provider
func NewProvider(e *engine.Engine, config Config) *Provider {
	// Create web scraper service if not created yet
	if e.WebScraper == nil {
		e.WebScraper = network.NewWebScraperService(e.HTTP, e.DOM, e.RateLimiter, e.Logger)
	}

	return &Provider{
		config:     config,
		engine:     e,
		webScraper: e.WebScraper,
	}
}

// ID returns the provider's identifier
func (p *Provider) ID() string {
	return p.config.ID
}

// Name returns the provider's display name
func (p *Provider) Name() string {
	return p.config.Name
}

// Description returns the provider's description
func (p *Provider) Description() string {
	return p.config.Description
}

// SiteURL returns the provider's website URL
func (p *Provider) SiteURL() string {
	return p.config.SiteURL
}

// Config returns the provider's configuration
func (p *Provider) Config() Config {
	return p.config
}

// Initialize initializes the provider
func (p *Provider) Initialize(ctx context.Context) error {
	return common.ExecuteInitialize(ctx, p.engine, p.ID(), p.Name(), func(ctx context.Context) error {
		// No special initialization needed for HTML providers
		return nil
	})
}

// CreateRequest creates a new scraper request with the provider's default headers
func (p *Provider) CreateRequest(url string) *network.ScraperRequest {
	req := network.NewScraperRequest(url)

	// Add default headers
	for key, value := range p.config.Headers {
		req.SetHeader(key, value)
	}

	return req
}

// FetchPage fetches a web page using the provider's configuration
func (p *Provider) FetchPage(ctx context.Context, url string) (*network.WebPage, error) {
	req := p.CreateRequest(url)
	return p.webScraper.FetchPage(ctx, req)
}

// Search implements a basic search for HTML-based providers
// Should be overridden by specific provider types
func (p *Provider) Search(ctx context.Context, query string, options core.SearchOptions) ([]core.Manga, error) {
	p.engine.Logger.Info("[%s] Searching for: %s", p.ID(), query)
	return nil, fmt.Errorf("search not implemented for HTML provider %s", p.ID())
}

// GetManga retrieves manga details
// Should be overridden by specific provider types
func (p *Provider) GetManga(ctx context.Context, id string) (*core.MangaInfo, error) {
	p.engine.Logger.Info("[%s] Getting manga details for: %s", p.ID(), id)
	return nil, fmt.Errorf("GetManga not implemented for HTML provider %s", p.ID())
}

// GetChapter retrieves chapter details
// Should be overridden by specific provider types
func (p *Provider) GetChapter(ctx context.Context, chapterID string) (*core.Chapter, error) {
	p.engine.Logger.Info("[%s] Getting chapter details for: %s", p.ID(), chapterID)
	return nil, fmt.Errorf("GetChapter not implemented for HTML provider %s", p.ID())
}

// TryGetMangaForChapter attempts to get manga info for a chapter
// Could be overridden by specific provider types for optimization
func (p *Provider) TryGetMangaForChapter(ctx context.Context, chapterID string) (*core.Manga, error) {
	// Get the chapter to extract manga ID
	chapter, err := p.GetChapter(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	// If manga ID is available in chapter
	if chapter.MangaID != "" {
		// Get manga details
		mangaInfo, err := p.GetManga(ctx, chapter.MangaID)
		if err != nil {
			return nil, err
		}
		return &mangaInfo.Manga, nil
	}

	return nil, fmt.Errorf("couldn't determine manga for chapter %s", chapterID)
}

// DownloadChapter downloads a chapter
// Could be overridden by specific provider types
func (p *Provider) DownloadChapter(ctx context.Context, chapterID, destDir string) error {
	return common.ExecuteDownloadChapter(
		ctx,
		p.engine,
		p.ID(),
		p.Name(),
		chapterID,
		destDir,
		p.GetChapter,
		p.TryGetMangaForChapter,
	)
}
