package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// ChapterMetadata contains essential metadata for file naming
type ChapterMetadata struct {
	MangaID      string   // ID of the manga
	MangaTitle   string   // Title of the manga
	ChapterID    string   // ID of the chapter
	ChapterNum   *float64 // Chapter number (can be nil)
	VolumeNum    *int     // Volume number (can be nil)
	ChapterTitle string   // Title of the chapter
	AgentID      string   // ID of the agent (for uniqueness)
}

// DownloadRequest represents a single file to download
type DownloadRequest struct {
	URL       string // URL of the file to download
	Index     int    // Index/ordering of the file (1-based)
	Filename  string // Original filename (used for extension)
	PageCount int    // Total number of pages for padding
}

// DownloadJobConfig contains configuration for a download job
type DownloadJobConfig struct {
	Metadata     ChapterMetadata   // Metadata for the chapter
	OutputDir    string            // Base directory to save files
	Concurrency  int               // Number of concurrent downloads
	Files        []DownloadRequest // Files to download
	Client       *http.Client      // HTTP client to use
	WaitDuration func(bool)        // Optional throttling function (bool = isRetry)
}

// GetVolumeOverride checks if a volume override is set in the context
func GetVolumeOverride(ctx context.Context) (*int, bool) {
	if val := ctx.Value("volume_override"); val != nil {
		if volNum, ok := val.(int); ok && volNum > 0 {
			return &volNum, true
		}
	}
	return nil, false
}

// DownloadChapterConcurrently downloads multiple files concurrently with proper naming
func DownloadChapterConcurrently(ctx context.Context, config DownloadJobConfig) error {
	// Check for volume override in context
	if volumeOverride, hasOverride := GetVolumeOverride(ctx); hasOverride {
		// Override volume in metadata
		config.Metadata.VolumeNum = volumeOverride
	}

	// Step 1: Create a sanitized manga title directory
	var mangaTitle string
	if config.Metadata.MangaTitle != "" {
		mangaTitle = sanitizeFilename(config.Metadata.MangaTitle)
	} else {
		// Generate a unique manga directory name if title not available
		mangaTitle = fmt.Sprintf("%s-%s", config.Metadata.AgentID, config.Metadata.MangaID)
		if mangaTitle == "-" {
			mangaTitle = "Unknown-Manga"
		}
	}

	// Create the manga directory
	mangaDir := filepath.Join(config.OutputDir, mangaTitle)
	if err := os.MkdirAll(mangaDir, 0755); err != nil {
		return fmt.Errorf("failed to create manga directory: %w", err)
	}

	// Step 2: Create the chapter directory inside the manga directory
	chapterDirName := FormatChapterDirName(config.Metadata.VolumeNum, config.Metadata.ChapterNum)
	chapterDir := filepath.Join(mangaDir, chapterDirName)

	// Create the chapter directory
	if err := os.MkdirAll(chapterDir, 0755); err != nil {
		return fmt.Errorf("failed to create chapter directory: %w", err)
	}

	// Prepare worker pool
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.Concurrency)
	errorChan := make(chan error, len(config.Files))

	// Process each file
	for _, file := range config.Files {
		wg.Add(1)

		// Acquire semaphore token
		semaphore <- struct{}{}

		go func(file DownloadRequest) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore when done

			// Apply throttling if provided
			if config.WaitDuration != nil {
				config.WaitDuration(false)
			}

			// Create request
			req, err := http.NewRequestWithContext(ctx, "GET", file.URL, nil)
			if err != nil {
				errorChan <- fmt.Errorf("failed to create request for page %d: %w", file.Index, err)
				return
			}

			// Make request
			resp, err := config.Client.Do(req)
			if err != nil {
				errorChan <- fmt.Errorf("failed to download page %d: %w", file.Index, err)
				return
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					errorChan <- fmt.Errorf("failed to close response body for page %d: %w", file.Index, err)
				}
			}(resp.Body)

			// Check response status
			if resp.StatusCode != http.StatusOK {
				errorChan <- fmt.Errorf("server returned %d for page %d", resp.StatusCode, file.Index)
				return
			}

			// Read response body
			data, err := io.ReadAll(resp.Body)
			if err != nil {
				errorChan <- fmt.Errorf("failed to read page %d: %w", file.Index, err)
				return
			}

			// Get file extension from original filename
			ext := filepath.Ext(file.Filename)
			if ext == "" {
				ext = ".jpg" // Default extension if none found
			}

			// Generate page filename with sequential numbering
			pageFilename := FormatPageFilename(file.Index, file.PageCount, ext)

			// Full path to save the file
			fullPath := filepath.Join(chapterDir, pageFilename)

			// Write the file
			if err := os.WriteFile(fullPath, data, 0644); err != nil {
				errorChan <- fmt.Errorf("failed to save page %d: %w", file.Index, err)
				return
			}
		}(file)
	}

	// Wait for all downloads to complete
	wg.Wait()
	close(errorChan)

	// Check for errors
	var errs []error
	for err := range errorChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("download errors (%d): %v", len(errs), errs[0])
	}

	return nil
}
