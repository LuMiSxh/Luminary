package util

import (
	"encoding/json"
	"fmt"
	"os"
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
