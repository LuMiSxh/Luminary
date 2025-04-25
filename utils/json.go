package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// APIResponse represents a standardized API response structure
type APIResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// OutputJSON marshals and outputs a standardized API response
func OutputJSON(status string, data interface{}, err error) {
	response := APIResponse{
		Status: status,
	}

	if err != nil {
		response.Status = "error"
		response.Error = err.Error()
	} else if data != nil {
		response.Data = data
	}

	// Marshal the response struct to JSON
	jsonData, jsonErr := json.Marshal(response)
	if jsonErr != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", jsonErr)
		return
	}

	// Print the JSON response
	fmt.Println(string(jsonData))
}

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
