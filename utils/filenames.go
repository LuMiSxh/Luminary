package utils

import (
	"fmt"
	"strings"
)

// FormatMangaID creates a standardized manga ID in the format "agent:id"
func FormatMangaID(agentID, mangaID string) string {
	return fmt.Sprintf("%s:%s", agentID, mangaID)
}

// ParseMangaID parses a manga ID in the format "agent:id" into its components
func ParseMangaID(combinedID string) (agentID string, mangaID string, err error) {
	parts := strings.SplitN(combinedID, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid ID format, must be 'agent:id'")
	}
	return parts[0], parts[1], nil
}
