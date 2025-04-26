package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HTTPService struct {
	DefaultClient      *http.Client
	RequestOptions     RequestOptions
	DefaultRetries     int
	DefaultTimeout     time.Duration
	ThrottleTimeAPI    time.Duration
	ThrottleTimeImages time.Duration
}

type RequestOptions struct {
	Headers         http.Header
	UserAgent       string
	Cookies         []*http.Cookie
	Referer         string
	Method          string
	FollowRedirects bool
}

func (h *HTTPService) FetchWithRetries(ctx context.Context, url string, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, h.RequestOptions.Method, url, nil)
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
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Set("User-Agent", h.RequestOptions.UserAgent)
	}

	// Perform request with retries
	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= h.DefaultRetries; attempt++ {
		resp, err = h.DefaultClient.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		lastErr = err
		if resp != nil && resp.Body != nil {
			err := resp.Body.Close()
			if err != nil {
				return nil, err
			}
		}

		// Don't sleep on the last attempt
		if attempt < h.DefaultRetries {
			// Exponential backoff
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
				// Continue with retry
			}
		}
	}

	if lastErr != nil {
		return nil, fmt.Errorf("failed after %d attempts: %w", h.DefaultRetries+1, lastErr)
	}

	return nil, fmt.Errorf("failed after %d attempts with status %d", h.DefaultRetries+1, resp.StatusCode)
}

// FetchJSON fetches and parses JSON
func (h *HTTPService) FetchJSON(ctx context.Context, url string, result interface{}, headers http.Header) error {
	resp, err := h.FetchWithRetries(ctx, url, headers)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	return json.NewDecoder(resp.Body).Decode(result)
}

// FetchString fetches content as string
func (h *HTTPService) FetchString(ctx context.Context, url string, headers http.Header) (string, error) {
	resp, err := h.FetchWithRetries(ctx, url, headers)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("failed to close response body: %v", err)
		}
	}(resp.Body)

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(data), nil
}
