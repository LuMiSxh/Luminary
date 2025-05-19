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

package network

import (
	"Luminary/pkg/engine/logger"
	"Luminary/pkg/errors"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"
)

// APIEndpoint defines a specific API endpoint configuration
type APIEndpoint struct {
	Path           string                         // Path template with placeholders like {id}
	Method         string                         // HTTP method (GET, POST, etc.)
	ResponseType   interface{}                    // Type to parse response into
	QueryFormatter func(interface{}) url.Values   // Function to build query params from input
	PathFormatter  func(string, ...string) string // Function to format path with IDs
	RequiresAuth   bool                           // Whether this endpoint requires authentication
}

// APIConfig holds the configuration for an API
type APIConfig struct {
	BaseURL        string                 // Base URL for the API
	Endpoints      map[string]APIEndpoint // Named endpoints
	DefaultHeaders map[string]string      // Headers to include in all requests
	RateLimitKey   string                 // Key to use for rate limiting (usually domain)
	RetryCount     int                    // Number of retries for failed requests
	ThrottleTime   time.Duration          // Time to wait between requests
}

// APIService provides methods for working with APIs
type APIService struct {
	HTTP        *HTTPService
	RateLimiter *RateLimiterService
	Logger      *logger.Service
}

// NewAPIService creates a new API service
func NewAPIService(http *HTTPService, rateLimiter *RateLimiterService, logger *logger.Service) *APIService {
	return &APIService{
		HTTP:        http,
		RateLimiter: rateLimiter,
		Logger:      logger,
	}
}

// FetchFromAPI fetches data from an API endpoint with improved error handling
func (a *APIService) FetchFromAPI(
	ctx context.Context,
	config APIConfig,
	endpointName string,
	params interface{},
	pathParams ...string,
) (interface{}, error) {
	endpoint, exists := config.Endpoints[endpointName]
	if !exists {
		return nil, &errors.APIError{
			Endpoint: endpointName,
			Message:  "Endpoint not found",
			Err:      errors.ErrInvalidInput,
		}
	}

	// Format the path with provided parameters
	path := endpoint.Path
	if endpoint.PathFormatter != nil {
		path = endpoint.PathFormatter(path, pathParams...)
	} else if len(pathParams) > 0 {
		// Simple replacement of {id} with first path parameter
		path = strings.Replace(path, "{id}", pathParams[0], 1)
	}

	// Build full URL
	fullURL := fmt.Sprintf("%s%s", config.BaseURL, path)

	// Apply query parameters if provided
	if params != nil {
		var queryParams url.Values

		// Use endpoint's QueryFormatter if available, otherwise fall back to BuildQueryParams
		if endpoint.QueryFormatter != nil {
			queryParams = endpoint.QueryFormatter(params)
		} else {
			// Use our generic reflection-based function as a fallback
			queryParams = BuildQueryParams(params)
		}

		if len(queryParams) > 0 {
			fullURL = fmt.Sprintf("%s?%s", fullURL, queryParams.Encode())
		}
	}

	// Apply rate limiting
	a.RateLimiter.Wait(config.RateLimitKey)

	// Set up headers
	headers := make(http.Header)
	for k, v := range config.DefaultHeaders {
		headers.Set(k, v)
	}
	headers.Set("Accept", "application/json")

	// Create a new instance of the response type
	var responseData interface{}
	if endpoint.ResponseType != nil {
		// Create a new instance of the response type based on the endpoint config
		responseType := reflect.TypeOf(endpoint.ResponseType)

		// Ensure we're creating a proper pointer instance
		if responseType.Kind() == reflect.Ptr {
			// Create a new instance of the pointed-to type
			responseData = reflect.New(responseType.Elem()).Interface()
		} else {
			// If it's not a pointer type, create a pointer to this type
			responseData = reflect.New(responseType).Interface()
		}
	} else {
		// If no response type specified, use a map to store generic JSON
		responseData = &map[string]interface{}{}
	}

	// Log request details at debug level
	a.Logger.Debug("[API] Requesting: %s %s", endpoint.Method, fullURL)

	// Make the HTTP request and handle errors
	err := a.HTTP.FetchJSON(ctx, fullURL, responseData, headers)
	if err != nil {
		// Check for specific error types and enhance them with API context
		if errors.IsNotFound(err) {
			// Extract resource details
			resourceType := ""
			resourceID := ""

			// Try to determine from the URL path
			pathParts := strings.Split(strings.Trim(path, "/"), "/")
			if len(pathParts) > 0 {
				resourceType = pathParts[0]

				// Try to get the resource ID
				if len(pathParams) > 0 {
					resourceID = pathParams[0]
				}
			}

			return nil, &errors.ResourceNotFoundError{
				APIError: errors.APIError{
					Endpoint: endpointName,
					URL:      fullURL,
					Message:  fmt.Sprintf("%s not found", resourceType),
					Err:      err,
				},
				ResourceType: resourceType,
				ResourceID:   resourceID,
			}
		}

		// For other errors, wrap them with API context
		var apiErr = &errors.APIError{
			Endpoint: endpointName,
			URL:      fullURL,
			Message:  "API request failed",
			Err:      err,
		}

		// Try to extract status code if it's an HTTP error
		var httpErr *errors.HTTPError
		if errors.As(err, &httpErr) {
			apiErr.StatusCode = httpErr.StatusCode
			apiErr.Message = httpErr.Message
		}

		return nil, apiErr
	}

	a.Logger.Debug("[API] Request successful: %s", fullURL)
	return responseData, nil
}

// BuildQueryParams converts a struct to URL query parameters using reflection
func BuildQueryParams(options interface{}) url.Values {
	params := url.Values{}

	// If options is nil, return empty params
	if options == nil {
		return params
	}

	// Get the value of the options
	v := reflect.ValueOf(options)

	// If it's a pointer, dereference it
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return params
		}
		v = v.Elem()
	}

	// Ensure we're dealing with a struct
	if v.Kind() != reflect.Struct {
		return params
	}

	// Get the type of the struct
	t := v.Type()

	// Iterate through all fields of the struct
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Get the param name from the json tag or use the field name
		paramName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "-" {
				paramName = parts[0]
			}
		}

		// Convert the field value to a query parameter based on its type
		switch fieldValue.Kind() {
		case reflect.String:
			val := fieldValue.String()
			if val != "" {
				params.Set(paramName, val)
			}

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val := fieldValue.Int()
			if val != 0 {
				params.Set(paramName, fmt.Sprintf("%d", val))
			}

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			val := fieldValue.Uint()
			if val != 0 {
				params.Set(paramName, fmt.Sprintf("%d", val))
			}

		case reflect.Float32, reflect.Float64:
			val := fieldValue.Float()
			if val != 0 {
				params.Set(paramName, fmt.Sprintf("%g", val))
			}

		case reflect.Bool:
			val := fieldValue.Bool()
			params.Set(paramName, fmt.Sprintf("%t", val))

		case reflect.Slice:
			// Handle string slices
			if fieldValue.Len() > 0 {
				if fieldValue.Type().Elem().Kind() == reflect.String {
					for j := 0; j < fieldValue.Len(); j++ {
						val := fieldValue.Index(j).String()
						if val != "" {
							params.Add(paramName+"[]", val)
						}
					}
				}
			}

		case reflect.Map:
			// Handle map[string]string for filters
			if fieldValue.Len() > 0 {
				if fieldValue.Type().Key().Kind() == reflect.String &&
					fieldValue.Type().Elem().Kind() == reflect.String {
					iter := fieldValue.MapRange()
					for iter.Next() {
						key := iter.Key().String()
						val := iter.Value().String()
						if key != "" && val != "" {
							params.Set(key, val)
						}
					}
				}
			}
		default:
			panic("unhandled default case")
		}
	}

	return params
}

// DefaultPathFormatter is a simple path formatter that replaces {id} with the first parameter
func DefaultPathFormatter(path string, params ...string) string {
	if len(params) > 0 {
		return strings.Replace(path, "{id}", params[0], 1)
	}
	return path
}
