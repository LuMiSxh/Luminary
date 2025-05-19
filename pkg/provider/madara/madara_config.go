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

package madara

// Config holds configuration for Madara-based sites
type Config struct {
	ID          string            // Short identifier
	Name        string            // Display name
	SiteURL     string            // Base URL for the site
	Description string            // Site description
	Headers     map[string]string // Additional HTTP headers

	// Selectors for different content types
	MangaSelector   string // Selector for manga list items
	ChapterSelector string // Selector for chapter list items
	PageSelector    string // Selector for page images

	// Custom options
	UseLegacyAjax    bool   // Use old-style AJAX requests
	CustomLoadAction string // Custom AJAX action for loading more content
}

// DefaultConfig returns a default configuration for Madara sites
func DefaultConfig(id, name, siteURL, description string) Config {
	return Config{
		ID:              id,
		Name:            name,
		SiteURL:         siteURL,
		Description:     description,
		MangaSelector:   "div.post-title h3 a, div.post-title h5 a, div.page-item-detail.manga div.post-title h3 a",
		ChapterSelector: "li.wp-manga-chapter > a",
		PageSelector:    "div.page-break source, div.page-break img, .reading-content img",
		UseLegacyAjax:   false,
		Headers: map[string]string{
			"User-Provider":   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
			"Accept-Language": "en-US,en;q=0.9",
			"Cache-Control":   "max-age=0",
			"Connection":      "keep-alive",
		},
	}
}
