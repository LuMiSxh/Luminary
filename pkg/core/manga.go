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

import "time"

// Manga represents basic manga information
type Manga struct {
	ID                string   `json:"id"`
	Title             string   `json:"title"`
	AlternativeTitles []string `json:"alt_titles,omitempty"`
	Description       string   `json:"description,omitempty"`
	Authors           []string `json:"authors,omitempty"`
	Status            string   `json:"status,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	CoverURL          string   `json:"cover_url,omitempty"`
}

// MangaInfo represents detailed manga information including chapters
type MangaInfo struct {
	Manga
	Chapters           []ChapterInfo `json:"chapters"`
	LastUpdated        *time.Time    `json:"last_updated,omitempty"`
	AvailableLanguages []string      `json:"available_languages,omitempty"`
}

// ChapterInfo represents basic chapter information
type ChapterInfo struct {
	ID       string     `json:"id"`
	Title    string     `json:"title"`
	Number   float64    `json:"number"`
	Volume   string     `json:"volume,omitempty"`
	Language string     `json:"language,omitempty"`
	Date     *time.Time `json:"date,omitempty"`
}

// Chapter represents a full chapter with pages
type Chapter struct {
	Info    ChapterInfo `json:"info"`
	MangaID string      `json:"manga_id"`
	Pages   []Page      `json:"pages"`
}

// Page represents a single page in a chapter
type Page struct {
	Index    int    `json:"index"`
	URL      string `json:"url"`
	Filename string `json:"filename,omitempty"`
}

// SearchOptions configures search behavior
type SearchOptions struct {
	Query            string                 `json:"query"`
	Limit            int                    `json:"limit"`
	Pages            int                    `json:"pages"`
	Sort             string                 `json:"sort,omitempty"`
	Filters          map[string]interface{} `json:"filters,omitempty"`
	IncludeAltTitles bool                   `json:"include_alt_titles,omitempty"`
	Concurrency      int                    `json:"concurrency,omitempty"`
}

// DownloadOptions configures download behavior
type DownloadOptions struct {
	OutputDir    string `json:"output_dir"`
	Format       string `json:"format,omitempty"`
	Quality      int    `json:"quality,omitempty"`
	Concurrent   int    `json:"concurrent,omitempty"`
	SkipExisting bool   `json:"skip_existing,omitempty"`
}
