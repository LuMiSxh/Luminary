package engine

import (
	"Luminary/errors"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type HTTPService struct {
	DefaultClient      *http.Client
	RequestOptions     RequestOptions
	DefaultRetries     int
	DefaultTimeout     time.Duration
	ThrottleTimeAPI    time.Duration
	ThrottleTimeImages time.Duration
	Logger             *LoggerService
}

type RequestOptions struct {
	Headers         http.Header
	UserAgent       string
	Cookies         []*http.Cookie
	Referer         string
	Method          string
	FollowRedirects bool
}

// ExtractDomain extracts the domain from a URL
func ExtractDomain(urlStr string) string {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		// If parsing fails, return the whole URL as the domain
		return urlStr
	}
	return parsed.Host
}

// FetchWithRetries performs an HTTP request with retry logic
func (h *HTTPService) FetchWithRetries(ctx context.Context, url string, headers http.Header) (*http.Response, error) {
	// Determine the method to use
	method := h.RequestOptions.Method
	if method == "" {
		method = "GET" // Default to GET if not specified
	}

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Apply default headers
	for k, v := range h.RequestOptions.Headers {
		req.Header[k] = v
	}

	// Apply additional headers
	for k, v := range headers {
		req.Header[k] = v
	}

	// Set user agent if not already set
	if _, ok := req.Header["User-Agent"]; !ok && h.RequestOptions.UserAgent != "" {
		req.Header.Set("User-Agent", h.RequestOptions.UserAgent)
	}

	// Apply cookies if any
	for _, cookie := range h.RequestOptions.Cookies {
		req.AddCookie(cookie)
	}

	// Set referer if specified
	if h.RequestOptions.Referer != "" {
		req.Header.Set("Referer", h.RequestOptions.Referer)
	}

	// Log the request details if logger is available
	if h.Logger != nil {
		h.Logger.Debug("[HTTP] %s request to %s", req.Method, req.URL.String())
	}

	// Perform request with retries
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= h.DefaultRetries; attempt++ {
		resp, err = h.DefaultClient.Do(req)

		// Log response details
		if h.Logger != nil {
			if err != nil {
				h.Logger.Debug("[HTTP] Request failed (attempt %d/%d): %v", attempt+1, h.DefaultRetries+1, err)
			} else {
				h.Logger.Debug("[HTTP] Response received (attempt %d/%d): status %d", attempt+1, h.DefaultRetries+1, resp.StatusCode)
			}
		}

		// Network or connection error - retry
		if err != nil {
			lastErr = &errors.HTTPError{
				Message: "Network connection error",
				URL:     url,
				Err:     errors.ErrNetworkIssue,
			}

			if attempt < h.DefaultRetries {
				// Exponential backoff
				backoff := time.Duration(1<<uint(attempt)) * time.Second
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}

				if h.Logger != nil {
					h.Logger.Debug("[HTTP] Retrying in %v...", backoff)
				}

				select {
				case <-ctx.Done():
					return nil, &errors.HTTPError{
						Message: "Request canceled",
						URL:     url,
						Err:     ctx.Err(),
					}
				case <-time.After(backoff):
					// Continue with retry
				}
			}
			continue
		}

		// Check status code - retry only for 5xx server errors
		if resp.StatusCode >= 500 {
			// Read the error response body for debugging
			bodyBytes, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()

			errorBody := ""
			if len(bodyBytes) > 0 {
				errorBody = string(bodyBytes)
				if h.Logger != nil {
					h.Logger.Debug("[HTTP] Server error response body: %s", errorBody)
				}
			}

			lastErr = &errors.HTTPError{
				StatusCode: resp.StatusCode,
				URL:        url,
				Message:    http.StatusText(resp.StatusCode),
				Body:       errorBody,
				Err:        errors.ErrServerError,
			}

			if attempt < h.DefaultRetries {
				// Exponential backoff
				backoff := time.Duration(1<<uint(attempt)) * time.Second
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}

				if h.Logger != nil {
					h.Logger.Debug("[HTTP] Server error %d, retrying in %v...", resp.StatusCode, backoff)
				}

				select {
				case <-ctx.Done():
					return nil, &errors.HTTPError{
						Message: "Request canceled during retry",
						URL:     url,
						Err:     ctx.Err(),
					}
				case <-time.After(backoff):
					// Continue with retry
				}
			}
			continue
		}

		// For 4xx client errors, don't retry but return a specific error
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			// Read the error response body
			bodyBytes, readErr := io.ReadAll(resp.Body)
			errorBody := ""
			if readErr == nil && len(bodyBytes) > 0 {
				errorBody = string(bodyBytes)
				if h.Logger != nil {
					h.Logger.Debug("[HTTP] Error response body: %s", errorBody)
				}
			}
			_ = resp.Body.Close()

			// Create specific error types based on status code
			switch resp.StatusCode {
			case http.StatusNotFound:
				// Extract resource type and ID from the request URL if possible
				resourceType, resourceID := extractResourceInfo(req.URL.Path)
				return nil, &errors.NotFoundError{
					HTTPError: errors.HTTPError{
						StatusCode: resp.StatusCode,
						URL:        url,
						Message:    "Resource not found",
						Body:       errorBody,
						Err:        errors.ErrNotFound,
					},
					ResourceType: resourceType,
					ResourceID:   resourceID,
				}
			case http.StatusUnauthorized, http.StatusForbidden:
				return nil, &errors.HTTPError{
					StatusCode: resp.StatusCode,
					URL:        url,
					Message:    http.StatusText(resp.StatusCode),
					Body:       errorBody,
					Err:        errors.ErrUnauthorized,
				}
			case http.StatusBadRequest:
				return nil, &errors.HTTPError{
					StatusCode: resp.StatusCode,
					URL:        url,
					Message:    "Bad request",
					Body:       errorBody,
					Err:        errors.ErrBadRequest,
				}
			case http.StatusTooManyRequests:
				return nil, &errors.HTTPError{
					StatusCode: resp.StatusCode,
					URL:        url,
					Message:    "Rate limit exceeded",
					Body:       errorBody,
					Err:        errors.ErrRateLimit,
				}
			default:
				return nil, &errors.HTTPError{
					StatusCode: resp.StatusCode,
					URL:        url,
					Message:    http.StatusText(resp.StatusCode),
					Body:       errorBody,
				}
			}
		}

		// Success - return the response
		return resp, nil
	}

	// All retries failed
	if lastErr != nil {
		return nil, lastErr
	}

	// Should never reach here, but just in case
	return nil, &errors.HTTPError{
		URL:     url,
		Message: "Failed after all retries",
		Err:     errors.ErrServerError,
	}
}

// FetchJSON fetches and parses JSON with improved error handling
func (h *HTTPService) FetchJSON(ctx context.Context, url string, result interface{}, headers http.Header) error {
	if h.Logger != nil {
		h.Logger.Debug("[HTTP] Fetching JSON from %s", url)
	}

	resp, err := h.FetchWithRetries(ctx, url, headers)
	if err != nil {
		return err // Already wrapped with specific error types
	}

	defer func() {
		if err := resp.Body.Close(); err != nil && h.Logger != nil {
			h.Logger.Warn("failed to close response body: %v", err)
		}
	}()

	// Read the body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &errors.HTTPError{
			StatusCode: resp.StatusCode,
			URL:        url,
			Message:    "Failed to read response body",
			Err:        err,
		}
	}

	// Log a sample of the response for debugging
	if h.Logger != nil {
		// Only log first 1000 chars to avoid flooding logs
		sampleSize := 1000
		if len(bodyBytes) < sampleSize {
			sampleSize = len(bodyBytes)
		}
		h.Logger.Debug("[HTTP] Response body sample (%d bytes): %s", len(bodyBytes), string(bodyBytes[:sampleSize]))
	}

	// Parse JSON into the result
	if err := json.Unmarshal(bodyBytes, result); err != nil {
		// If JSON parsing fails, include some of the response body in the error
		sampleSize := 200
		if len(bodyBytes) < sampleSize {
			sampleSize = len(bodyBytes)
		}
		return &errors.HTTPError{
			StatusCode: resp.StatusCode,
			URL:        url,
			Message:    fmt.Sprintf("Failed to parse JSON response: %v", err),
			Body:       string(bodyBytes[:sampleSize]) + "...",
			Err:        errors.ErrBadRequest,
		}
	}

	if h.Logger != nil {
		h.Logger.Debug("[HTTP] Successfully parsed JSON response")
	}

	return nil
}

// FetchString fetches content as string
func (h *HTTPService) FetchString(ctx context.Context, url string, headers http.Header) (string, error) {
	resp, err := h.FetchWithRetries(ctx, url, headers)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil && h.Logger != nil {
			h.Logger.Warn("failed to close response body: %v", err)
		}
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &errors.HTTPError{
			StatusCode: resp.StatusCode,
			URL:        url,
			Message:    "Failed to read response",
			Err:        err,
		}
	}

	return string(data), nil
}

// Helper function to extract resource type and ID from URL path
// E.g., "/manga/12345" -> "manga", "12345"
func extractResourceInfo(path string) (string, string) {
	parts := []string{}
	current := ""

	// Split the path into parts
	for _, char := range path {
		if char == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	// We need at least 2 parts for type/ID
	if len(parts) < 2 {
		return "", ""
	}

	// The last part is likely the resource ID
	resourceID := parts[len(parts)-1]

	// The second-to-last part is likely the resource type
	resourceType := parts[len(parts)-2]

	return resourceType, resourceID
}
