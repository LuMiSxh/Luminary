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

package core

import (
	"time"
)

// Manga represents basic manga information
type Manga struct {
	ID          string
	Title       string
	Cover       string
	Description string
	Authors     []string
	Status      string
	Tags        []string
	AltTitles   []string
}

// MangaInfo contains detailed manga information
type MangaInfo struct {
	Manga
	Chapters    []ChapterInfo
	LastUpdated *time.Time // When the manga info was last updated on the source
}

// ChapterInfo represents chapter metadata
type ChapterInfo struct {
	ID       string
	Title    string
	Number   float64
	Date     *time.Time // Nullable date - nil when no date is available
	Language *string    // Nullable language - nil when language is not specified
}

// Chapter contains detailed chapter information
type Chapter struct {
	Info    ChapterInfo
	Pages   []Page
	MangaID string
}

// Page represents a manga page
type Page struct {
	Index    int
	URL      string
	Filename string
}

// SearchOptions for search customization
type SearchOptions struct {
	Query   string            // The search query string
	Limit   int               // Maximum number of results per page
	Pages   int               // Number of pages to fetch (0 for all pages)
	Fields  []string          // Fields to search within (e.g., "title", "author", "description")
	Filters map[string]string // Field-specific filters
	Sort    string            // Sort order (e.g., "relevance", "name")
}
