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

package provider

import (
	"Luminary/pkg/core"
	"context"
)

// Provider defines the interface all manga providers must implement
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
