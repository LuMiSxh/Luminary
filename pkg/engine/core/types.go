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
	Chapters []ChapterInfo
}

// ChapterInfo represents chapter metadata
type ChapterInfo struct {
	ID     string
	Title  string
	Number float64
	Date   time.Time
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
