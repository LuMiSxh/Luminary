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
	"Luminary/pkg/engine/network"
	"Luminary/pkg/engine/parser/html"
	"Luminary/pkg/errors"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// defaultAPISearch implements default search for API providers
func (p *Provider) defaultAPISearch(ctx context.Context, query string, options core.SearchOptions) ([]core.Manga, error) {
	if p.Config.API == nil {
		return nil, errors.Track(fmt.Errorf("API configuration not set")).AsProvider(p.ID()).Error()
	}

	endpoint, ok := p.Config.API.Endpoints["search"]
	if !ok {
		return nil, errors.Track(fmt.Errorf("search endpoint not configured")).AsProvider(p.ID()).Error()
	}

	// Build URL with query parameters
	u, err := url.Parse(p.Config.API.BaseURL + endpoint)
	if err != nil {
		return nil, errors.Track(err).AsProvider(p.ID()).Error()
	}

	q := u.Query()
	q.Set("q", query)
	if options.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", options.Limit))
	}

	u.RawQuery = q.Encode()

	// Make request
	resp, err := p.Engine.Network.Request(ctx, &network.Request{
		URL:       u.String(),
		Method:    "GET",
		Headers:   p.Config.Headers,
		RateLimit: p.Config.RateLimit,
	})
	if err != nil {
		return nil, errors.Track(err).
			WithContext("search_url", u.String()).
			WithMessage("Failed to connect to API. Please check your internet connection.").
			AsNetwork().Error()
	}

	// Check if we have a valid response
	if resp == nil || resp.Body == nil {
		return nil, errors.New("received empty response from API").
			WithContext("search_url", u.String()).
			AsNetwork().Error()
	}

	// Parse response based on mapping
	var results []core.Manga
	mapping, hasMapping := p.Config.API.ResponseMapping["search"]

	if hasMapping {
		// Use custom mapping
		var data map[string]interface{}
		if err := json.Unmarshal(resp.Body, &data); err != nil {
			return nil, errors.Track(err).AsProvider(p.ID()).Error()
		}

		// Extract results array
		resultsPath := strings.Split(mapping.Fields["results"], ".")
		resultsData := extractFromPath(data, resultsPath)

		if items, ok := resultsData.([]interface{}); ok {
			for _, item := range items {
				if m, ok := item.(map[string]interface{}); ok {
					manga := core.Manga{
						ID:    extractString(m, mapping.IDField),
						Title: extractString(m, mapping.TitleField),
					}

					// Extract additional fields
					if descField, ok := mapping.Fields["description"]; ok {
						manga.Description = extractString(m, descField)
					}

					results = append(results, manga)
				}
			}
		}
	} else {
		// Try generic parsing
		if err := json.Unmarshal(resp.Body, &results); err != nil {
			// Try wrapped response
			var wrapped struct {
				Data    []core.Manga `json:"data"`
				Results []core.Manga `json:"results"`
			}
			if err := json.Unmarshal(resp.Body, &wrapped); err != nil {
				return nil, errors.Track(err).
					WithContext("response", string(resp.Body)).
					AsProvider(p.ID()).Error()
			}

			if len(wrapped.Data) > 0 {
				results = wrapped.Data
			} else {
				results = wrapped.Results
			}
		}
	}

	return results, nil
}

// defaultWebSearch implements default search for web scraping providers
func (p *Provider) defaultWebSearch(ctx context.Context, query string, options core.SearchOptions) ([]core.Manga, error) {
	var searchURL string

	if p.Config.Web != nil && p.Config.Web.SearchPath != "" {
		searchURL = p.Config.SiteURL + strings.ReplaceAll(p.Config.Web.SearchPath, "{query}", url.QueryEscape(query))
	} else if p.Config.Type == TypeMadara {
		// Madara default search
		searchURL = p.Config.SiteURL + "/?s=" + url.QueryEscape(query) + "&post_type=wp-manga"
	} else {
		searchURL = p.Config.SiteURL + "/search?q=" + url.QueryEscape(query)
	}

	// Make request
	resp, err := p.Engine.Network.Request(ctx, &network.Request{
		URL:       searchURL,
		Headers:   p.Config.Headers,
		RateLimit: p.Config.RateLimit,
	})
	if err != nil {
		return nil, errors.Track(err).
			WithContext("search_url", searchURL).
			WithMessage("Failed to connect to website. Please check your internet connection.").
			AsNetwork().Error()
	}

	// Check if we have a valid response
	if resp == nil || resp.Body == nil {
		return nil, errors.New("received empty response from server").
			AsNetwork().Error()
	}

	// Parse HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, errors.Track(err).
			WithContext("search_url", searchURL).
			AsParser().Error()
	}

	// Extract results using selectors
	var selector string
	if p.Config.Type == TypeMadara && p.Config.Madara != nil {
		selector = p.Config.Madara.Selectors["search"]
	} else if p.Config.Web != nil {
		selector = p.Config.Web.Selectors["search_results"]
	}

	if selector == "" {
		selector = "a.manga-title, .manga-item a, .post-title a" // Common patterns
	}

	elements, err := doc.Select(selector).All()
	if err != nil {
		return nil, errors.Track(err).AsProvider(p.ID()).Error()
	}

	var results []core.Manga
	for _, elem := range elements {
		href := elem.Extract().Href()
		if href == "" {
			continue
		}

		// Extract ID from URL
		id := extractIDFromURL(href, p.Config.SiteURL)

		results = append(results, core.Manga{
			ID:    id,
			Title: elem.Extract().Text(),
		})
	}

	return results, nil
}

// defaultAPIGetManga implements default manga retrieval for API providers
func (p *Provider) defaultAPIGetManga(ctx context.Context, id string) (*core.MangaInfo, error) {
	if p.Config.API == nil {
		return nil, errors.Track(fmt.Errorf("API configuration not set")).AsProvider(p.ID()).Error()
	}

	endpoint, ok := p.Config.API.Endpoints["manga"]
	if !ok {
		return nil, errors.Track(fmt.Errorf("manga endpoint not configured")).AsProvider(p.ID()).Error()
	}

	// Build URL
	buildUrl := p.Config.API.BaseURL + strings.ReplaceAll(endpoint, "{id}", id)

	// Make request
	resp, err := p.Engine.Network.Request(ctx, &network.Request{
		URL:       buildUrl,
		Method:    "GET",
		Headers:   p.Config.Headers,
		RateLimit: p.Config.RateLimit,
	})
	if err != nil {
		return nil, errors.Track(err).
			WithContext("manga_url", buildUrl).
			WithMessage("Failed to fetch manga details. Please check your internet connection.").
			AsNetwork().Error()
	}

	// Check if we have a valid response
	if resp == nil || resp.Body == nil {
		return nil, errors.New("received empty response from server").
			WithContext("manga_url", buildUrl).
			AsNetwork().Error()
	}

	// Parse response
	var info core.MangaInfo
	if err := json.Unmarshal(resp.Body, &info); err != nil {
		return nil, errors.Track(err).
			WithContext("response", string(resp.Body)).
			AsProvider(p.ID()).Error()
	}

	// Fetch chapters if endpoint is configured
	if chaptersEndpoint, ok := p.Config.API.Endpoints["chapters"]; ok {
		chapters, err := p.fetchAPIChapters(ctx, id, chaptersEndpoint)
		if err != nil {
			p.Engine.Logger.Warn("Failed to fetch chapters: %v", err)
		} else {
			info.Chapters = chapters
		}
	}

	return &info, nil
}

// defaultWebGetManga implements default manga retrieval for web scraping providers
func (p *Provider) defaultWebGetManga(ctx context.Context, id string) (*core.MangaInfo, error) {
	var mangaURL string

	if p.Config.Web != nil && p.Config.Web.MangaPath != "" {
		mangaURL = p.Config.SiteURL + strings.ReplaceAll(p.Config.Web.MangaPath, "{id}", id)
	} else {
		mangaURL = p.Config.SiteURL + "/manga/" + id
	}

	// Make request
	resp, err := p.Engine.Network.Request(ctx, &network.Request{
		URL:       mangaURL,
		Headers:   p.Config.Headers,
		RateLimit: p.Config.RateLimit,
	})
	if err != nil {
		return nil, errors.Track(err).
			WithContext("manga_url", mangaURL).
			WithMessage("Failed to fetch manga information. Please check your internet connection.").
			AsNetwork().Error()
	}

	// Check if we have a valid response
	if resp == nil || resp.Body == nil {
		return nil, errors.New("received empty response from server").
			WithContext("manga_url", mangaURL).
			AsNetwork().Error()
	}

	// Parse HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, errors.Track(err).
			WithContext("manga_url", mangaURL).
			AsParser().Error()
	}

	info := &core.MangaInfo{
		Manga: core.Manga{
			ID: id,
		},
	}

	// Extract title
	titleSelector := p.getSelector("title", "h1.manga-title, h1.post-title, .manga-info h1")
	if elem, err := doc.Select(titleSelector).First(); err == nil {
		info.Title = elem.Extract().Text()
	}

	// Extract description
	descSelector := p.getSelector("description", ".description, .summary, .manga-description")
	if elem, err := doc.Select(descSelector).First(); err == nil {
		info.Description = elem.Extract().Text()
	}

	// Extract chapters
	chapterSelector := p.getSelector("chapters", "li.chapter a, .chapter-list a")
	if chapters, err := doc.Select(chapterSelector).All(); err == nil {
		for i, ch := range chapters {
			info.Chapters = append(info.Chapters, core.ChapterInfo{
				ID:     extractIDFromURL(ch.Extract().Href(), p.Config.SiteURL),
				Title:  ch.Extract().Text(),
				Number: float64(len(chapters) - i), // Reverse order
			})
		}
	}

	return info, nil
}

// Helper functions

func (p *Provider) getSelector(name string, defaultSelector string) string {
	// Check Madara config first
	if p.Config.Type == TypeMadara && p.Config.Madara != nil {
		if sel, ok := p.Config.Madara.Selectors[name]; ok {
			return sel
		}
	}

	// Check web config
	if p.Config.Web != nil {
		if sel, ok := p.Config.Web.Selectors[name]; ok {
			return sel
		}
	}

	return defaultSelector
}

func (p *Provider) fetchAPIChapters(ctx context.Context, mangaID string, endpoint string) ([]core.ChapterInfo, error) {
	buildUrl := p.Config.API.BaseURL + strings.ReplaceAll(endpoint, "{id}", mangaID)

	resp, err := p.Engine.Network.Request(ctx, &network.Request{
		URL:       buildUrl,
		Method:    "GET",
		Headers:   p.Config.Headers,
		RateLimit: p.Config.RateLimit,
	})
	if err != nil {
		return nil, errors.Track(err).
			WithContext("chapters_url", buildUrl).
			AsNetwork().Error()
	}

	// Check if we have a valid response
	if resp == nil || resp.Body == nil {
		return nil, errors.New("received empty response from API").
			WithContext("chapters_url", buildUrl).
			AsNetwork().Error()
	}

	var chapters []core.ChapterInfo
	if err := json.Unmarshal(resp.Body, &chapters); err != nil {
		// Try wrapped response
		var wrapped struct {
			Data     []core.ChapterInfo `json:"data"`
			Chapters []core.ChapterInfo `json:"chapters"`
		}
		if err := json.Unmarshal(resp.Body, &wrapped); err != nil {
			return nil, err
		}

		if len(wrapped.Data) > 0 {
			chapters = wrapped.Data
		} else {
			chapters = wrapped.Chapters
		}
	}

	return chapters, nil
}

func extractFromPath(data map[string]interface{}, path []string) interface{} {
	current := interface{}(data)

	for _, key := range path {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[key]
		} else {
			return nil
		}
	}

	return current
}

func extractString(data map[string]interface{}, field string) string {
	if val, ok := data[field]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func extractIDFromURL(href, siteURL string) string {
	// Remove site URL
	id := strings.TrimPrefix(href, siteURL)
	id = strings.TrimPrefix(id, "/")

	// Common patterns
	patterns := []string{
		"manga/", "chapter/", "read/", "series/",
	}

	for _, pattern := range patterns {
		if idx := strings.Index(id, pattern); idx >= 0 {
			id = id[idx+len(pattern):]
			break
		}
	}

	// Remove trailing slashes and query params
	if idx := strings.Index(id, "?"); idx >= 0 {
		id = id[:idx]
	}
	id = strings.TrimSuffix(id, "/")

	return id
}
