package utils

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

// Track sequential volume IDs for chapters without volume info
var (
	sequentialVolume     = 1
	sequentialVolumeLock sync.Mutex
)

// GetNextSequentialVolume returns the next sequential volume ID and increments the counter
func GetNextSequentialVolume() int {
	sequentialVolumeLock.Lock()
	defer sequentialVolumeLock.Unlock()

	vol := sequentialVolume
	sequentialVolume++
	return vol
}

// FormatChapterDirName generates a consistently named directory name for a chapter (without full path)
func FormatChapterDirName(volume *int, chapter *float64) string {
	if volume != nil && chapter != nil {
		// Both volume and chapter are available
		return fmt.Sprintf("%04d-%s", *volume, formatChapterNumber(*chapter))
	} else if chapter != nil {
		// Only chapter is available - use sequential volume ID
		seqVol := GetNextSequentialVolume()
		return fmt.Sprintf("%04d-%s", seqVol, formatChapterNumber(*chapter))
	} else {
		// Neither is available, use a default directory
		seqVol := GetNextSequentialVolume()
		return fmt.Sprintf("%04d-unknown", seqVol)
	}
}

// FormatPageFilename generates a consistently named filename for a page
// Always uses sequential numbering regardless of volume/chapter info
func FormatPageFilename(pageIndex int, totalPages int, extension string) string {
	// Determine maximum page digits for padding
	pageDigits := len(strconv.Itoa(totalPages))
	if pageDigits < 2 {
		pageDigits = 2 // Minimum 2-digit padding for pages
	}

	// Format page number with leading zeros
	paddedPage := fmt.Sprintf("%0*d", pageDigits, pageIndex)

	// Prepare file extension
	ext := strings.ToLower(extension)
	if !strings.HasPrefix(ext, ".") {
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
		return fmt.Sprintf("%03d%s", intPart, strings.TrimPrefix(fmt.Sprintf("%.1f", fracPart), "0"))
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
