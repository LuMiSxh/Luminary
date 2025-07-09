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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"Luminary/pkg/engine/parser/html"
	"Luminary/pkg/errors"
)

// Request represents an HTTP request configuration
type Request struct {
	URL     string
	Method  string
	Headers map[string]string
	Body    io.Reader

	// Options
	RateLimit  time.Duration
	Timeout    time.Duration
	MaxRetries int

	// Form data (for POST requests)
	FormData url.Values
}

// Response represents an HTTP response with parsed content
type Response struct {
	StatusCode int
	Status     string
	Headers    http.Header
	Body       []byte

	// Request info
	URL    string
	Method string

	// Lazy-parsed content
	json json.RawMessage
	html *html.Parser
}

// newResponse creates a Response from an http.Response
func newResponse(httpResp *http.Response) (*Response, error) {
	if httpResp == nil {
		return nil, errors.Track(fmt.Errorf("cannot create response from nil http.Response")).
			AsNetwork().Error()
	}

	defer func() {
		if httpResp.Body != nil {
			if err := httpResp.Body.Close(); err != nil {
				// Log but don't fail on close errors
			}
		}
	}()

	// Read body safely
	var body []byte
	var err error

	if httpResp.Body != nil {
		body, err = io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, errors.Track(err).
				WithContext("url", httpResp.Request.URL.String()).
				WithMessage("Failed to read response body").
				AsNetwork().Error()
		}
	}

	// Throw an error when the body is nil instead of creating empty slice
	if body == nil {
		return nil, errors.Track(fmt.Errorf("response body is nil")).
			WithContext("url", httpResp.Request.URL.String()).
			WithMessage("Received response with nil body").
			AsParser().Error()
	}

	resp := &Response{
		StatusCode: httpResp.StatusCode,
		Status:     httpResp.Status,
		Headers:    httpResp.Header,
		Body:       body,
		URL:        httpResp.Request.URL.String(),
		Method:     httpResp.Request.Method,
	}

	// Pre-parse JSON if content-type indicates JSON
	contentType := httpResp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		resp.json = body
	}

	return resp, nil
}

// JSON unmarshal the response body as JSON
func (r *Response) JSON(v interface{}) error {
	// Defensive check for nil receiver
	if r == nil {
		return errors.Track(fmt.Errorf("cannot parse JSON: response is nil")).
			AsParser().Error()
	}

	if r.json == nil {
		r.json = r.Body
	}

	// Handle empty body
	if len(r.json) == 0 {
		return errors.Track(fmt.Errorf("cannot parse JSON from empty response body")).
			WithContext("url", r.URL).
			AsParser().Error()
	}

	if err := json.Unmarshal(r.json, v); err != nil {
		// Create preview safely, handling nil Body
		preview := "empty response"
		if r.Body != nil && len(r.Body) > 0 {
			previewLen := min(len(r.Body), 200)
			preview = string(r.Body[:previewLen])
		}

		return errors.Track(err).
			WithContext("url", r.URL).
			WithContext("response_preview", preview).
			WithContext("body_length", len(r.Body)).
			AsParser().Error()
	}

	return nil
}

// HTML returns the parsed HTML document
func (r *Response) HTML() (*html.Parser, error) {
	if r.html == nil {
		var err error
		r.html, err = html.Parse(r.Body)
		if err != nil {
			return nil, errors.Track(err).
				WithContext("url", r.URL).
				AsParser().
				Error()
		}
	}
	return r.html, nil
}

// Text returns the response body as a string
func (r *Response) Text() string {
	return string(r.Body)
}

// IsJSON checks if the response is JSON
func (r *Response) IsJSON() bool {
	contentType := r.Headers.Get("Content-Type")
	return strings.Contains(contentType, "application/json")
}

// IsHTML checks if the response is HTML
func (r *Response) IsHTML() bool {
	contentType := r.Headers.Get("Content-Type")
	return strings.Contains(contentType, "text/html")
}

// IsSuccess checks if the response indicates success
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsError checks if the response indicates an error
func (r *Response) IsError() bool {
	return r.StatusCode >= 400
}

// Save saves the response body to a file
func (r *Response) Save(filepath string) error {
	return os.WriteFile(filepath, r.Body, 0644)
}

// RequestBuilder - Builder for creating requests
type RequestBuilder struct {
	req *Request
}

// NewRequest creates a new request builder
func NewRequest(url string) *RequestBuilder {
	return &RequestBuilder{
		req: &Request{
			URL:     url,
			Method:  "GET",
			Headers: make(map[string]string),
		},
	}
}

// Method sets the HTTP method
func (b *RequestBuilder) Method(method string) *RequestBuilder {
	b.req.Method = method
	return b
}

// Header adds a header
func (b *RequestBuilder) Header(key, value string) *RequestBuilder {
	b.req.Headers[key] = value
	return b
}

// Headers adds multiple headers
func (b *RequestBuilder) Headers(headers map[string]string) *RequestBuilder {
	for k, v := range headers {
		b.req.Headers[k] = v
	}
	return b
}

// Body sets the request body
func (b *RequestBuilder) Body(body io.Reader) *RequestBuilder {
	b.req.Body = body
	return b
}

// Form sets form data for POST requests
func (b *RequestBuilder) Form(data url.Values) *RequestBuilder {
	b.req.FormData = data
	b.req.Method = "POST"
	b.req.Headers["Content-Type"] = "application/x-www-form-urlencoded"
	b.req.Body = strings.NewReader(data.Encode())
	return b
}

// RateLimit sets the rate limit delay
func (b *RequestBuilder) RateLimit(delay time.Duration) *RequestBuilder {
	b.req.RateLimit = delay
	return b
}

// Timeout sets the request timeout
func (b *RequestBuilder) Timeout(timeout time.Duration) *RequestBuilder {
	b.req.Timeout = timeout
	return b
}

// Retries sets the maximum number of retries
func (b *RequestBuilder) Retries(retries int) *RequestBuilder {
	b.req.MaxRetries = retries
	return b
}

// Build returns the constructed request
func (b *RequestBuilder) Build() *Request {
	return b.req
}
