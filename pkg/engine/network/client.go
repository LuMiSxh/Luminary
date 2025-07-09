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
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"Luminary/pkg/engine/logger"
	"Luminary/pkg/errors"
)

// Client provides unified HTTP operations with rate limiting and retries
type Client struct {
	http    *http.Client
	limiter *RateLimiter
	logger  logger.Logger

	// Default settings
	defaultRetries int
	defaultTimeout time.Duration
	defaultHeaders map[string]string
}

// NewClient creates a new network client
func NewClient(logger logger.Logger) *Client {
	return &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			},
		},
		limiter:        NewRateLimiter(),
		logger:         logger,
		defaultRetries: 3,
		defaultTimeout: 30 * time.Second,
		defaultHeaders: map[string]string{
			"User-Agent": "Luminary/1.0",
			"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
		},
	}
}

// Do execute an HTTP request with rate limiting and retries
func (c *Client) Do(ctx context.Context, req *Request) (*Response, error) {
	// Apply rate limiting
	if req.RateLimit > 0 {
		if err := c.limiter.Wait(ctx, req.URL, req.RateLimit); err != nil {
			return nil, errors.Track(err).
				WithContext("url", req.URL).
				AsNetwork().
				Error()
		}
	}

	// Set defaults
	if req.Method == "" {
		req.Method = "GET"
	}
	if req.Timeout == 0 {
		req.Timeout = c.defaultTimeout
	}
	if req.MaxRetries == 0 {
		req.MaxRetries = c.defaultRetries
	}

	// Execute with retries
	return c.executeWithRetry(ctx, req)
}

// Request is a convenience method for simple requests
func (c *Client) Request(ctx context.Context, req *Request) (*Response, error) {
	return c.Do(ctx, req)
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, url string) (*Response, error) {
	return c.Do(ctx, &Request{
		URL:    url,
		Method: "GET",
	})
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, url string, body io.Reader) (*Response, error) {
	return c.Do(ctx, &Request{
		URL:    url,
		Method: "POST",
		Body:   body,
	})
}

// executeWithRetry executes a request with retry logic
func (c *Client) executeWithRetry(ctx context.Context, req *Request) (*Response, error) {
	var allErrors []error // Collect all errors during retries

	for attempt := 0; attempt <= req.MaxRetries; attempt++ {
		resp, err := c.executeRequest(ctx, req)

		// Log response details
		if err != nil {
			c.logger.Debug("[HTTP] Request failed (attempt %d/%d): %v", attempt+1, req.MaxRetries+1, err)
			// Add attempt context to the error
			attemptErr := errors.Track(err).
				WithMessage(fmt.Sprintf("attempt %d/%d failed", attempt+1, req.MaxRetries+1)).
				AsNetwork().Error()
			allErrors = append(allErrors, attemptErr)
		} else if resp != nil {
			c.logger.Debug("[HTTP] Response received (attempt %d/%d): status %d", attempt+1, req.MaxRetries+1, resp.StatusCode)
		}

		// Network or connection error - retry
		if err != nil || resp == nil {
			if attempt < req.MaxRetries {
				// Exponential backoff
				backoff := time.Duration(1<<uint(attempt)) * time.Second
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}

				c.logger.Debug("[HTTP] Retrying in %v...", backoff)

				select {
				case <-ctx.Done():
					ctxErr := errors.Track(ctx.Err()).
						WithMessage("request canceled by context").
						AsNetwork().Error()
					allErrors = append(allErrors, ctxErr)
					return nil, errors.Join(allErrors...)
				case <-time.After(backoff):
					// Continue with retry
				}
			}
			continue
		}

		// At this point, we know resp is not nil

		// Check status code - retry only for 5xx server errors
		if resp.StatusCode >= 500 {
			// Create a server error and add it to the list
			serverErr := errors.Track(
				fmt.Errorf("server returned %d status code", resp.StatusCode),
			).WithMessage(
				fmt.Sprintf("attempt %d/%d: server error %d", attempt+1, req.MaxRetries+1, resp.StatusCode),
			).AsNetwork().Error()
			allErrors = append(allErrors, serverErr)

			if attempt < req.MaxRetries {
				// Exponential backoff
				backoff := time.Duration(1<<uint(attempt)) * time.Second
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}

				c.logger.Debug("[HTTP] Server error %d, retrying in %v...", resp.StatusCode, backoff)

				select {
				case <-ctx.Done():
					allErrors = append(allErrors, errors.Track(ctx.Err()).
						WithMessage("request canceled by context during backoff").
						AsNetwork().Error())
					return nil, errors.Join(allErrors...)
				case <-time.After(backoff):
					// Continue with retry
				}
			}
			continue
		}

		// For 4xx client errors, don't retry but return a specific error
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			// Create specific error types based on status code
			var clientErr error
			switch resp.StatusCode {
			case http.StatusNotFound:
				clientErr = errors.Track(fmt.Errorf("resource not found")).
					WithContext("status_code", resp.StatusCode).
					WithContext("url", req.URL).
					AsNetwork().Error()
			case http.StatusUnauthorized, http.StatusForbidden:
				clientErr = errors.Track(fmt.Errorf("unauthorized access")).
					WithContext("status_code", resp.StatusCode).
					WithContext("url", req.URL).
					AsNetwork().Error()
			case http.StatusBadRequest:
				clientErr = errors.Track(fmt.Errorf("bad request")).
					WithContext("status_code", resp.StatusCode).
					WithContext("url", req.URL).
					AsNetwork().Error()
			case http.StatusTooManyRequests:
				clientErr = errors.Track(fmt.Errorf("rate limit exceeded")).
					WithContext("status_code", resp.StatusCode).
					WithContext("url", req.URL).
					AsRateLimit().Error()
			default:
				clientErr = errors.Track(fmt.Errorf("client error: %d %s", resp.StatusCode, resp.Status)).
					WithContext("status_code", resp.StatusCode).
					WithContext("url", req.URL).
					AsNetwork().Error()
			}
			return nil, clientErr
		}

		// Success - return the response
		return resp, nil
	}

	// All retry attempts failed - use errors.Join to combine all errors
	if len(allErrors) > 0 {
		return nil, errors.Join(allErrors...)
	}

	// This should rarely happen (if allErrors is empty after retries), but for completeness
	fallbackErr := errors.Track(fmt.Errorf("HTTP request failed after %d attempts", req.MaxRetries+1)).
		WithContext("url", req.URL).
		WithContext("method", req.Method).
		AsNetwork().Error()
	return nil, fallbackErr
}

// executeRequest performs a single HTTP request
func (c *Client) executeRequest(ctx context.Context, req *Request) (*Response, error) {
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, req.Body)
	if err != nil {
		return nil, errors.Track(err).
			WithContext("url", req.URL).
			WithContext("method", req.Method).
			AsNetwork().Error()
	}

	// Set headers (defaults first, then request-specific)
	for k, v := range c.defaultHeaders {
		httpReq.Header.Set(k, v)
	}
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	// Set timeout
	if req.Timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, req.Timeout)
		defer cancel()
		httpReq = httpReq.WithContext(ctx)
	}

	// Execute request
	c.logger.Debug("[HTTP] %s request to %s", req.Method, req.URL)
	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, errors.Track(err).
			WithContext("url", req.URL).
			WithContext("method", req.Method).
			AsNetwork().Error()
	}

	// Check for nil response
	if httpResp == nil {
		return nil, errors.Track(fmt.Errorf("nil HTTP response received")).
			WithContext("url", req.URL).
			WithContext("method", req.Method).
			AsNetwork().Error()
	}

	// Create response using the newResponse helper from types.go
	resp, err := newResponse(httpResp)
	if err != nil {
		return nil, err
	}

	c.logger.Debug("[HTTP] %s %s - Status: %d", req.Method, req.URL, resp.StatusCode)
	return resp, nil
}

// SetDefaultHeader sets a default header for all requests
func (c *Client) SetDefaultHeader(key, value string) {
	c.defaultHeaders[key] = value
}

// SetDefaultTimeout sets the default timeout for requests
func (c *Client) SetDefaultTimeout(timeout time.Duration) {
	c.defaultTimeout = timeout
}

// SetDefaultRetries sets the default number of retries
func (c *Client) SetDefaultRetries(retries int) {
	c.defaultRetries = retries
}
