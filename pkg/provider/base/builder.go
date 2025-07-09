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
	"context"
)

// Builder provides fluent configuration for providers
type Builder struct {
	provider *Provider
}

// New creates a new provider builder
func New(engine *engine.Engine, config Config) *Builder {
	p := &Provider{
		Config: config,
		Engine: engine,
		ops:    Operations{},
	}

	b := &Builder{provider: p}
	b.setDefaults()

	return b
}

// setDefaults sets default implementations based on provider type
func (b *Builder) setDefaults() {
	// Default implementations are set in the Provider methods
	// This ensures they're available even without explicit configuration
}

// WithInitialize sets a custom initialization function
func (b *Builder) WithInitialize(fn func(context.Context) error) *Builder {
	b.provider.ops.Initialize = fn
	return b
}

// WithSearch sets a custom search function
func (b *Builder) WithSearch(fn func(context.Context, string, core.SearchOptions) ([]core.Manga, error)) *Builder {
	// Wrap the function to inject provider reference
	b.provider.ops.Search = func(ctx context.Context, query string, opts core.SearchOptions) ([]core.Manga, error) {
		return fn(ctx, query, opts)
	}
	return b
}

// WithGetManga sets a custom manga retrieval function
func (b *Builder) WithGetManga(fn func(context.Context, string) (*core.MangaInfo, error)) *Builder {
	b.provider.ops.GetManga = fn
	return b
}

// WithGetChapter sets a custom chapter retrieval function
func (b *Builder) WithGetChapter(fn func(context.Context, string) (*core.Chapter, error)) *Builder {
	b.provider.ops.GetChapter = fn
	return b
}

// WithGetChapterPages sets a custom page retrieval function
func (b *Builder) WithGetChapterPages(fn func(context.Context, string) ([]string, error)) *Builder {
	b.provider.ops.GetChapterPages = fn
	return b
}

// WithDownloadChapter sets a custom download function
func (b *Builder) WithDownloadChapter(fn func(context.Context, string, string) error) *Builder {
	b.provider.ops.DownloadChapter = fn
	return b
}

// Build returns the configured provider
func (b *Builder) Build() engine.Provider {
	return b.provider
}

// APIBuilder - Helper builder for API configuration
type APIBuilder struct {
	config APIConfig
}

// NewAPIConfig creates a new API configuration builder
func NewAPIConfig(baseURL string) *APIBuilder {
	return &APIBuilder{
		config: APIConfig{
			BaseURL:         baseURL,
			Endpoints:       make(map[string]string),
			ResponseMapping: make(map[string]ResponseMap),
		},
	}
}

// WithEndpoint adds an API endpoint
func (b *APIBuilder) WithEndpoint(name, path string) *APIBuilder {
	b.config.Endpoints[name] = path
	return b
}

// WithResponseMapping adds response mapping configuration
func (b *APIBuilder) WithResponseMapping(endpoint string, mapping ResponseMap) *APIBuilder {
	b.config.ResponseMapping[endpoint] = mapping
	return b
}

// Build returns the API configuration
func (b *APIBuilder) Build() *APIConfig {
	return &b.config
}
