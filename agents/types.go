package agents

import (
	"context"
	"time"
)

// Status represents an agent's status
type Status string

const (
	StatusStable       Status = "stable"
	StatusExperimental Status = "experimental"
	StatusOutdated     Status = "outdated"
)

// Agent defines the interface for manga sources
type Agent interface {
	ID() string
	Name() string
	Description() string
	Status() Status

	Search(ctx context.Context, query string, options SearchOptions) ([]Manga, error)
	GetManga(ctx context.Context, id string) (*MangaInfo, error)
	GetChapter(ctx context.Context, mangaID, chapterID string) (*Chapter, error)
	DownloadChapter(ctx context.Context, mangaID, chapterID, destDir string) error
}

// SearchOptions for search customization
type SearchOptions struct {
	Limit   int
	Fields  []string
	Filters map[string]string
	Sort    string
}

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
	Info  ChapterInfo
	Pages []Page
}

// Page represents a manga page
type Page struct {
	Index    int
	URL      string
	Filename string
}
