package util

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

func ParseFloat64(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func ParseInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// CleanImageURL handles cleaning the image URLs by removing control characters
// and extracting URLs that might be embedded in the URL
func CleanImageURL(dirtyURL string) string {
	// First handle the case where the actual URL is embedded with control characters
	if strings.Contains(dirtyURL, "http") {
		// Find all URLs in the string
		re := regexp.MustCompile(`https?://[^\s\t\n\r]+`)
		matches := re.FindAllString(dirtyURL, -1)

		if len(matches) > 0 {
			// Take the last match, which is likely the actual image URL
			return matches[len(matches)-1]
		}
	}

	// Remove control characters
	return regexp.MustCompile(`[\t\n\r]+`).ReplaceAllString(dirtyURL, "")
}

// FormatDate formats a time.Time value for display
func FormatDate(date time.Time) string {
	if date.IsZero() {
		return "Unknown"
	}
	return date.Format("2006-01-02")
}
