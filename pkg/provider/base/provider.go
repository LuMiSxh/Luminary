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

package base

import (
	"Luminary/pkg/core"
	"Luminary/pkg/engine"
	"Luminary/pkg/errors"
	"context"
	"fmt"
	"time"
)

// Type represents the provider implementation strategy
type Type string

const (
	TypeAPI    Type = "api"
	TypeWeb    Type = "web"
	TypeMadara Type = "madara"
)

// Provider is the base implementation for all providers
type Provider struct {
	// Configuration
	Config Config

	// Engine reference
	Engine *engine.Engine

	// Overridable operations
	ops Operations
}

// Config holds provider configuration
type Config struct {
	// Identity
	ID          string
	Name        string
	Description string
	SiteURL     string

	// Provider type determines default behavior
	Type Type

	// Configuration based on type
	API    *APIConfig
	Web    *WebConfig
	Madara *MadaraConfig

	// Common settings
	Headers   map[string]string
	RateLimit time.Duration
	Timeout   time.Duration
}

// APIConfig for API-based providers
type APIConfig struct {
	BaseURL   string
	Endpoints map[string]string
	// Response mapping configuration
	ResponseMapping map[string]ResponseMap
}

// WebConfig for web scraping providers
type WebConfig struct {
	SearchPath string
	MangaPath  string
	// CSS selectors for common elements
	Selectors map[string]string
}

// MadaraConfig for Madara WordPress theme sites
type MadaraConfig struct {
	// CSS selectors specific to Madara
	Selectors map[string]string
	// Whether to use AJAX search
	AjaxSearch bool
	// Custom AJAX action if different from default
	CustomLoadAction string
}

// ResponseMap defines how to map API responses to core types
type ResponseMap struct {
	IDField      string
	TitleField   string
	ChaptersPath string
	// Additional field mappings
	Fields map[string]string
}

// Operations that can be overridden
type Operations struct {
	Initialize      func(ctx context.Context) error
	Search          func(ctx context.Context, query string, options core.SearchOptions) ([]core.Manga, error)
	GetManga        func(ctx context.Context, id string) (*core.MangaInfo, error)
	GetChapter      func(ctx context.Context, chapterID string) (*core.Chapter, error)
	GetChapterPages func(ctx context.Context, chapterID string) ([]string, error)
	DownloadChapter func(ctx context.Context, chapterID, destDir string) error
}

// Interface compliance check
var _ engine.Provider = (*Provider)(nil)

// Identity methods

func (p *Provider) ID() string          { return p.Config.ID }
func (p *Provider) Name() string        { return p.Config.Name }
func (p *Provider) Description() string { return p.Config.Description }
func (p *Provider) SiteURL() string     { return p.Config.SiteURL }

// Initialize initializes the provider
func (p *Provider) Initialize(ctx context.Context) error {
	p.Engine.Logger.Info("Initializing provider: %s (%s)", p.Name(), p.ID())

	if p.ops.Initialize != nil {
		if err := p.ops.Initialize(ctx); err != nil {
			return errors.Track(err).
				WithContext("provider", p.ID()).
				AsProvider(p.ID()).
				Error()
		}
	}

	p.Engine.Logger.Info("Provider initialized: %s", p.ID())
	return nil
}

// Search performs a manga search
func (p *Provider) Search(ctx context.Context, query string, options core.SearchOptions) ([]core.Manga, error) {
	if p.ops.Search != nil {
		return p.ops.Search(ctx, query, options)
	}

	// Use default implementation based on type
	switch p.Config.Type {
	case TypeAPI:
		return p.defaultAPISearch(ctx, query, options)
	case TypeWeb, TypeMadara:
		return p.defaultWebSearch(ctx, query, options)
	default:
		return nil, errors.Track(fmt.Errorf("search not implemented for provider type: %s", p.Config.Type)).
			AsProvider(p.ID()).
			Error()
	}
}

// GetManga retrieves detailed manga information
func (p *Provider) GetManga(ctx context.Context, id string) (*core.MangaInfo, error) {
	if p.ops.GetManga != nil {
		return p.ops.GetManga(ctx, id)
	}

	switch p.Config.Type {
	case TypeAPI:
		return p.defaultAPIGetManga(ctx, id)
	case TypeWeb, TypeMadara:
		return p.defaultWebGetManga(ctx, id)
	default:
		return nil, errors.Track(fmt.Errorf("get manga not implemented for provider type: %s", p.Config.Type)).
			AsProvider(p.ID()).
			Error()
	}
}

// GetChapter retrieves chapter information with pages
func (p *Provider) GetChapter(ctx context.Context, chapterID string) (*core.Chapter, error) {
	if p.ops.GetChapter != nil {
		return p.ops.GetChapter(ctx, chapterID)
	}

	// Default implementation
	pages, err := p.getChapterPages(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	chapter := &core.Chapter{
		Info: core.ChapterInfo{
			ID: chapterID,
		},
		Pages: make([]core.Page, len(pages)),
	}

	for i, url := range pages {
		chapter.Pages[i] = core.Page{
			Index: i,
			URL:   url,
		}
	}

	return chapter, nil
}

// TryGetMangaForChapter attempts to retrieve manga info for a chapter
func (p *Provider) TryGetMangaForChapter(ctx context.Context, chapterID string) (*core.Manga, error) {
	// Most providers will need custom implementation
	// This is a fallback that returns an error
	return nil, errors.Track(fmt.Errorf("manga lookup from chapter not supported")).
		AsProvider(p.ID()).
		Error()
}

// DownloadChapter downloads a chapter to the specified directory
func (p *Provider) DownloadChapter(ctx context.Context, chapterID, destDir string) error {
	if p.ops.DownloadChapter != nil {
		return p.ops.DownloadChapter(ctx, chapterID, destDir)
	}

	// Default implementation using engine's download service
	chapter, err := p.GetChapter(ctx, chapterID)
	if err != nil {
		return err
	}

	return p.Engine.Download.DownloadChapter(ctx, chapter, destDir)
}

// Helper to get chapter pages
func (p *Provider) getChapterPages(ctx context.Context, chapterID string) ([]string, error) {
	if p.ops.GetChapterPages != nil {
		return p.ops.GetChapterPages(ctx, chapterID)
	}

	return nil, errors.Track(fmt.Errorf("get chapter pages not implemented")).
		AsProvider(p.ID()).
		Error()
}
