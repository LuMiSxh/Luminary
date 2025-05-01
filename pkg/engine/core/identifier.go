package core

import (
	"fmt"
	"strings"
)

// FormatMangaID creates a standardized manga ID in the format "provider:id"
func FormatMangaID(providerID, mangaID string) string {
	return fmt.Sprintf("%s:%s", providerID, mangaID)
}

// ParseMangaID parses a manga ID in the format "provider:id" into its components
func ParseMangaID(combinedID string) (providerID string, mangaID string, err error) {
	parts := strings.SplitN(combinedID, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid ID format, must be 'provider:id'")
	}
	return parts[0], parts[1], nil
}
