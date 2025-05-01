package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DownloadService provides file download capabilities with concurrency control
type DownloadService struct {
	MaxConcurrency int
	Throttle       time.Duration
	OutputFormat   string
	Client         *http.Client
	Logger         *LoggerService
}

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
	Files        []DownloadRequest // Files to download
	WaitDuration func(bool)        // Optional throttling function (bool = isRetry)
}

// DownloadChapter downloads a chapter with the given config
func (d *DownloadService) DownloadChapter(ctx context.Context, config DownloadJobConfig) error {
	// Use service client if not provided in config
	client := d.Client

	// Get concurrency from context or use default from service
	concurrency := GetConcurrency(ctx, d.MaxConcurrency)
	if d.Logger != nil {
		d.Logger.Debug("Using concurrency limit of %d from context", concurrency)
	}

	// Check for volume override in context
	if volumeOverride, hasOverride := GetVolumeOverride(ctx); hasOverride {
		config.Metadata.VolumeNum = &volumeOverride
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
	chapterDirName := formatChapterDirName(config.Metadata.VolumeNum, config.Metadata.ChapterNum)
	chapterDir := filepath.Join(mangaDir, chapterDirName)

	// Create the chapter directory
	if err := os.MkdirAll(chapterDir, 0755); err != nil {
		return fmt.Errorf("failed to create chapter directory: %w", err)
	}

	// Prepare worker pool
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)
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
			resp, err := client.Do(req)
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
			pageFilename := formatPageFilename(file.Index, file.PageCount, ext)

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
		if d.Logger != nil {
			d.Logger.Error("Download errors (%d): %v", len(errs), errs[0])
		}
		return fmt.Errorf("download errors (%d): %v", len(errs), errs[0])
	}

	return nil
}

// Track sequential volume IDs for chapters without volume info
var (
	sequentialVolume     = 1
	sequentialVolumeLock sync.Mutex
)

// getNextSequentialVolume returns the next sequential volume ID and increments the counter
func getNextSequentialVolume() int {
	sequentialVolumeLock.Lock()
	defer sequentialVolumeLock.Unlock()

	vol := sequentialVolume
	sequentialVolume++
	return vol
}

// formatChapterDirName generates a consistently named directory name for a chapter (without full path)
func formatChapterDirName(volume *int, chapter *float64) string {
	if volume != nil && chapter != nil {
		// Both volume and chapter are available
		return fmt.Sprintf("%04d-%s", *volume, formatChapterNumber(*chapter))
	} else if chapter != nil {
		// Only chapter is available - use sequential volume ID
		seqVol := getNextSequentialVolume()
		return fmt.Sprintf("%04d-%s", seqVol, formatChapterNumber(*chapter))
	} else {
		// Neither is available, use a default directory
		seqVol := getNextSequentialVolume()
		return fmt.Sprintf("%04d-unknown", seqVol)
	}
}

// formatPageFilename generates a consistently named filename for a page
// Always uses sequential numbering regardless of volume/chapter info
func formatPageFilename(pageIndex int, totalPages int, extension string) string {
	// Determine maximum page digits for padding
	pageDigits := len(fmt.Sprintf("%d", totalPages))
	if pageDigits < 2 {
		pageDigits = 2 // Minimum 2-digit padding for pages
	}

	// Format page number with leading zeros
	paddedPage := fmt.Sprintf("%0*d", pageDigits, pageIndex)

	// Prepare file extension
	ext := extension
	if ext == "" || ext[0] != '.' {
		ext = "." + ext
	}

	// Always use simple sequential format for page files
	return fmt.Sprintf("%s%s", paddedPage, ext)
}

// formatChapterNumber formats a chapter number with proper padding
// Handles both integer (5 -> "005") and decimal chapters (5.5 -> "005.5")
func formatChapterNumber(chapter float64) string {
	// Check if chapter is a whole number
	if chapter == float64(int(chapter)) {
		// Integer chapter (e.g., 5 -> "005")
		return fmt.Sprintf("%03d", int(chapter))
	} else {
		// Decimal chapter (e.g., 5.5 -> "005.5")
		intPart := int(chapter)
		fracPart := chapter - float64(intPart)

		// Format with 3 digits for integer part and preserve decimal part
		return fmt.Sprintf("%03d%s", intPart, fmt.Sprintf("%.1f", fracPart)[1:])
	}
}

// sanitizeFilename removes invalid characters from a filename
func sanitizeFilename(name string) string {
	// Replace invalid filename characters with underscores
	invalid := []rune{'<', '>', ':', '"', '/', '\\', '|', '?', '*'}
	for _, char := range invalid {
		name = strings.ReplaceAll(name, string(char), "_")
	}

	// Trim spaces and dots at the beginning and end
	name = strings.Trim(name, " .")

	// If the name is empty after sanitization, use a default
	if name == "" {
		name = "Unknown-Manga"
	}

	// Limit the length to avoid file system issues
	maxLength := 100
	if len(name) > maxLength {
		name = name[:maxLength]
	}

	return name
}
