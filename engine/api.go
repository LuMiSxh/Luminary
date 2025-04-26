package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	Cache       *CacheService
	Logger      *LoggerService
}

// NewAPIService creates a new API service
func NewAPIService(http *HTTPService, rateLimiter *RateLimiterService, cache *CacheService, logger *LoggerService) *APIService {
	return &APIService{
		HTTP:        http,
		RateLimiter: rateLimiter,
		Cache:       cache,
		Logger:      logger,
	}
}

// FetchFromAPI fetches data from an API endpoint
func (a *APIService) FetchFromAPI(
	ctx context.Context,
	config APIConfig,
	endpointName string,
	params interface{},
	pathParams ...string,
) (interface{}, error) {
	endpoint, exists := config.Endpoints[endpointName]
	if !exists {
		return nil, fmt.Errorf("endpoint not found: %s", endpointName)
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
	if params != nil && endpoint.QueryFormatter != nil {
		queryParams := endpoint.QueryFormatter(params)
		if len(queryParams) > 0 {
			fullURL = fmt.Sprintf("%s?%s", fullURL, queryParams.Encode())
		}
	}

	// Check cache before making request
	cacheKey := fmt.Sprintf("api:%s:%s", config.RateLimitKey, fullURL)
	if endpoint.Method == "GET" || endpoint.Method == "" {
		var cachedResponse interface{}
		if a.Cache.Get(cacheKey, &cachedResponse) {
			a.Logger.Debug("Using cached response for: %s", fullURL)
			return cachedResponse, nil
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

	// Make the request
	var responseData interface{}
	if endpoint.ResponseType != nil {
		// Create a new instance of the response type
		responseData = endpoint.ResponseType
	}

	// Perform the request with retries
	retryCount := config.RetryCount
	if retryCount <= 0 {
		retryCount = 3 // Default retry count
	}

	var lastErr error
	for attempt := 0; attempt <= retryCount; attempt++ {
		if attempt > 0 {
			a.Logger.Debug("Retrying request (%d/%d): %s", attempt, retryCount, fullURL)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
				// Exponential backoff
			}
		}

		a.Logger.Debug("Making request to: %s", fullURL)

		var err error
		if responseData != nil {
			err = a.HTTP.FetchJSON(ctx, fullURL, responseData, headers)
		} else {
			var genericResponse map[string]interface{}
			err = a.HTTP.FetchJSON(ctx, fullURL, &genericResponse, headers)
			responseData = genericResponse
		}

		if err == nil {
			break
		}

		lastErr = err
		if attempt == retryCount {
			return nil, fmt.Errorf("API request failed after %d attempts: %w", retryCount+1, lastErr)
		}
	}

	// Cache the response for GET requests
	if endpoint.Method == "GET" || endpoint.Method == "" {
		if err := a.Cache.Set(cacheKey, responseData); err != nil {
			a.Logger.Warn("Failed to cache response: %v", err)
		}
	}

	return responseData, nil
}

// BuildQueryParams converts a search options struct to URL query parameters
func BuildQueryParams(options interface{}) url.Values {
	params := url.Values{}

	// Use reflection to extract fields from the options struct
	// For simplicity, we'll just handle common options manually

	// This could be improved with reflection for a more generic approach
	switch opts := options.(type) {
	case SearchOptions:
		if opts.Limit > 0 {
			params.Set("limit", fmt.Sprintf("%d", opts.Limit))
		}
		if opts.Sort != "" {
			params.Set("sort", opts.Sort)
		}
		// Add other fields as needed
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

// ParseResponseBody parses a JSON response body into the given result type
func ParseResponseBody(data []byte, result interface{}) error {
	return json.Unmarshal(data, result)
}
