package provider

import (
	"Luminary/pkg/engine/core"
	"context"
)

// Provider defines the interface for manga sources
type Provider interface {
	ID() string
	Name() string
	Description() string
	SiteURL() string

	Initialize(ctx context.Context) error

	Search(ctx context.Context, query string, options core.SearchOptions) ([]core.Manga, error)
	GetManga(ctx context.Context, id string) (*core.MangaInfo, error)
	GetChapter(ctx context.Context, chapterID string) (*core.Chapter, error)
	TryGetMangaForChapter(ctx context.Context, chapterID string) (*core.Manga, error)
	DownloadChapter(ctx context.Context, chapterID, destDir string) error
}
