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

package download

import (
	"Luminary/pkg/core"
	"Luminary/pkg/engine/logger"
	"Luminary/pkg/engine/network"
	"Luminary/pkg/errors"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Service handles file downloads
type Service struct {
	client       *network.Client
	logger       logger.Logger
	concurrency  int
	outputFormat string
	throttle     time.Duration
}

// NewService creates a new download service
func NewService(client *network.Client, logger logger.Logger) *Service {
	return &Service{
		client:       client,
		logger:       logger,
		concurrency:  3,
		outputFormat: "png",
		throttle:     500 * time.Millisecond,
	}
}

// DownloadChapter downloads all pages of a chapter
func (s *Service) DownloadChapter(ctx context.Context, chapter *core.Chapter, destDir string) error {
	if len(chapter.Pages) == 0 {
		return errors.New("chapter has no pages").AsProvider("").Error()
	}

	// Create chapter directory
	chapterDir := filepath.Join(destDir, s.sanitizeFilename(fmt.Sprintf("Chapter_%g", chapter.Info.Number)))
	if err := os.MkdirAll(chapterDir, 0755); err != nil {
		return errors.Track(err).
			WithContext("directory", chapterDir).
			AsFileSystem().
			Error()
	}

	s.logger.Info("Downloading chapter %.1f to %s (%d pages)",
		chapter.Info.Number, chapterDir, len(chapter.Pages))

	// Download pages concurrently
	return s.downloadPages(ctx, chapter.Pages, chapterDir)
}

// DownloadFile downloads a single file
func (s *Service) DownloadFile(ctx context.Context, url, destPath string) error {
	// Check if file already exists
	if _, err := os.Stat(destPath); err == nil {
		s.logger.Debug("File already exists: %s", destPath)
		return nil
	}

	// Create temporary file
	tempPath := destPath + ".tmp"

	// Download to temporary file
	if err := s.downloadToFile(ctx, url, tempPath); err != nil {
		err := os.Remove(tempPath)
		if err != nil {
			return errors.Track(err).
				WithContext("file", tempPath).
				AsFileSystem().
				Error()
		}
		return errors.Track(err).Error()
	}

	// Rename to final path
	if err := os.Rename(tempPath, destPath); err != nil {
		err := os.Remove(tempPath)
		if err != nil {
			return errors.Track(err).
				WithContext("file", tempPath).
				AsFileSystem().
				Error()
		}

		return errors.Track(err).
			WithContext("file", destPath).
			AsFileSystem().
			Error()
	}

	return nil
}

// downloadPages downloads multiple pages concurrently
func (s *Service) downloadPages(ctx context.Context, pages []core.Page, destDir string) error {
	// Create work channel
	type job struct {
		page  core.Page
		index int
	}

	jobs := make(chan job, len(pages))
	errorChan := make(chan error, len(pages))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < s.concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				select {
				case <-ctx.Done():
					errorChan <- ctx.Err()
					return
				default:
					if err := s.downloadPage(ctx, j.page, j.index, destDir); err != nil {
						errorChan <- err
					}
				}
			}
		}()
	}

	// Queue jobs
	for i, page := range pages {
		jobs <- job{page: page, index: i}
	}
	close(jobs)

	// Wait for completion
	wg.Wait()
	close(errorChan)

	// Check for errors
	var errs []error
	for err := range errorChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// downloadPage downloads a single page
func (s *Service) downloadPage(ctx context.Context, page core.Page, index int, destDir string) error {
	// Determine filename
	filename := page.Filename
	if filename == "" {
		ext := s.extractExtension(page.URL)
		if ext == "" {
			ext = s.outputFormat
		}
		filename = fmt.Sprintf("page_%03d.%s", index+1, ext)
	}

	destPath := filepath.Join(destDir, filename)

	// Apply throttling
	if s.throttle > 0 {
		time.Sleep(s.throttle)
	}

	s.logger.Debug("Downloading page %d: %s", index+1, page.URL)

	return s.DownloadFile(ctx, page.URL, destPath)
}

// downloadToFile downloads content to a file
func (s *Service) downloadToFile(ctx context.Context, url, destPath string) error {
	// Create request
	resp, err := s.client.Request(ctx, &network.Request{
		URL:    url,
		Method: "GET",
		Headers: map[string]string{
			"Accept": "image/webp,image/apng,image/*,*/*;q=0.8",
		},
	})
	if err != nil {
		return errors.Track(err).
			WithContext("url", url).
			AsDownload().
			Error()
	}

	// Create output file
	file, err := os.Create(destPath)
	if err != nil {
		return errors.Track(err).
			WithContext("file", destPath).
			AsFileSystem().
			Error()
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			s.logger.Error("Failed to close file %s: %v", destPath, err)
		} else {
			s.logger.Debug("File closed: %s", destPath)
		}
	}(file)

	// Write content
	if _, err := io.Copy(file, strings.NewReader(string(resp.Body))); err != nil {
		return errors.Track(err).
			WithContext("file", destPath).
			AsDownload().
			Error()
	}

	return nil
}

// sanitizeFilename makes a string safe for use as a filename
func (s *Service) sanitizeFilename(name string) string {
	// Replace invalid characters
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)

	name = replacer.Replace(name)
	name = strings.TrimSpace(name)

	// Limit length
	if len(name) > 100 {
		name = name[:100]
	}

	return name
}

// extractExtension extracts file extension from URL
func (s *Service) extractExtension(url string) string {
	// Remove query parameters
	if idx := strings.Index(url, "?"); idx >= 0 {
		url = url[:idx]
	}

	// Get extension
	ext := filepath.Ext(url)
	if ext != "" {
		return strings.TrimPrefix(ext, ".")
	}

	return ""
}

// SetConcurrency sets the number of concurrent downloads
func (s *Service) SetConcurrency(n int) {
	if n > 0 {
		s.concurrency = n
	}
}

// SetThrottle sets the delay between downloads
func (s *Service) SetThrottle(d time.Duration) {
	s.throttle = d
}

// SetOutputFormat sets the default output format
func (s *Service) SetOutputFormat(format string) {
	s.outputFormat = format
}
