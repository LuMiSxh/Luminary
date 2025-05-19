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
