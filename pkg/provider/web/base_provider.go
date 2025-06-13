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

package web

import (
	"Luminary/pkg/engine"
	"Luminary/pkg/engine/core"
	"Luminary/pkg/engine/network"
	"Luminary/pkg/errors"
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
	return nil, errors.TP(fmt.Errorf("search not implemented for HTML provider %s", p.ID()), p.ID())
}

// GetManga retrieves manga details
// Should be overridden by specific provider types
func (p *Provider) GetManga(ctx context.Context, id string) (*core.MangaInfo, error) {
	p.engine.Logger.Info("[%s] Getting manga details for: %s", p.ID(), id)
	return nil, errors.TP(fmt.Errorf("GetManga not implemented for HTML provider %s", p.ID()), p.ID())
}

// GetChapter retrieves chapter details
// Should be overridden by specific provider types
func (p *Provider) GetChapter(ctx context.Context, chapterID string) (*core.Chapter, error) {
	p.engine.Logger.Info("[%s] Getting chapter details for: %s", p.ID(), chapterID)
	return nil, errors.TP(fmt.Errorf("GetChapter not implemented for HTML provider %s", p.ID()), p.ID())
}

// TryGetMangaForChapter attempts to get manga info for a chapter
// Could be overridden by specific provider types for optimization
func (p *Provider) TryGetMangaForChapter(ctx context.Context, chapterID string) (*core.Manga, error) {
	// Get the chapter to extract manga ID
	chapter, err := p.GetChapter(ctx, chapterID)
	if err != nil {
		return nil, errors.TP(err, p.ID())
	}

	// If manga ID is available in chapter
	if chapter.MangaID != "" {
		// Get manga details
		mangaInfo, err := p.GetManga(ctx, chapter.MangaID)
		if err != nil {
			return nil, errors.TP(err, p.ID())
		}
		return &mangaInfo.Manga, nil
	}

	return nil, errors.TP(fmt.Errorf("couldn't determine manga for chapter %s", chapterID), p.ID())
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
