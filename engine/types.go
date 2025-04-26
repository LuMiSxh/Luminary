package engine

import (
	"context"
	"time"
)

// Agent defines the interface for manga sources
type Agent interface {
	ID() string
	Name() string
	Description() string
	SiteURL() string

	Initialize(ctx context.Context) error

	Search(ctx context.Context, query string, options SearchOptions) ([]Manga, error)
	GetManga(ctx context.Context, id string) (*MangaInfo, error)
	GetChapter(ctx context.Context, chapterID string) (*Chapter, error)
	TryGetMangaForChapter(ctx context.Context, chapterID string) (*Manga, error)
	DownloadChapter(ctx context.Context, chapterID, destDir string) error
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
