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
	Logger             *logger.Service
}

type RequestOptions struct {
	Headers         http.Header
	UserProvider    string
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
		return nil, errors.TN(err)
	}

	// Apply default headers
	for k, v := range h.RequestOptions.Headers {
		req.Header[k] = v
	}

	// Apply additional headers
	for k, v := range headers {
		req.Header[k] = v
	}

	// Set user provider if not already set
	if _, ok := req.Header["User-Provider"]; !ok && h.RequestOptions.UserProvider != "" {
		req.Header.Set("User-Provider", h.RequestOptions.UserProvider)
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
					return nil, errors.TN(err)
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
					return nil, errors.TN(err)
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
				return nil, errors.TN(errors.ErrNotFound)
			case http.StatusUnauthorized, http.StatusForbidden:
				return nil, errors.TN(errors.ErrUnauthorized)
			case http.StatusBadRequest:
				return nil, errors.TN(errors.ErrBadRequest)
			case http.StatusTooManyRequests:
				return nil, errors.TN(errors.ErrRateLimit)
			default:
				return nil, errors.TN(errors.ErrBadRequest)
			}
		}

		// Success - return the response
		return resp, nil
	}

	// Should never reach here, but just in case
	return nil, errors.TN(fmt.Errorf("HTTP request failed - Could be a network problem"))
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
		return errors.TN(err)
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
		return errors.TN(err)
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
		return "", errors.TN(err)
	}

	return string(data), nil
}
