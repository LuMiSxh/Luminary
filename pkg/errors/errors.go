package errors

import (
	stderrors "errors" // Import standard errors package
	"fmt"
)

// Export standard functions from the errors package
var (
	As     = stderrors.As
	Is     = stderrors.Is
	Unwrap = stderrors.Unwrap
)

// Standard errors that can be used across the application
var (
	ErrNotFound     = stderrors.New("resource not found")
	ErrUnauthorized = stderrors.New("unauthorized")
	ErrBadRequest   = stderrors.New("bad request")
	ErrServerError  = stderrors.New("server error")
	ErrTimeout      = stderrors.New("operation timed out")
	ErrRateLimit    = stderrors.New("rate limit exceeded")
	ErrInvalidInput = stderrors.New("invalid input")
	ErrNetworkIssue = stderrors.New("network connection issue")
)

// IsNotFound checks if err is or wraps a not found error
func IsNotFound(err error) bool {
	return Is(err, ErrNotFound)
}

// IsUnauthorized checks if err is or wraps an unauthorized error
func IsUnauthorized(err error) bool {
	return Is(err, ErrUnauthorized)
}

// IsServerError checks if err is or wraps a server error
func IsServerError(err error) bool {
	return Is(err, ErrServerError)
}

// HTTPError represents an HTTP-related error
type HTTPError struct {
	StatusCode int
	URL        string
	Message    string
	Body       string
	Err        error
}

func (e *HTTPError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("HTTP %d: %s - %s", e.StatusCode, e.Message, e.Body)
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

func (e *HTTPError) Unwrap() error {
	return e.Err
}

func (e *HTTPError) Is(target error) bool {
	switch e.StatusCode {
	case 404:
		return target == ErrNotFound
	case 401, 403:
		return target == ErrUnauthorized
	case 400, 422:
		return target == ErrBadRequest
	case 429:
		return target == ErrRateLimit
	case 500, 502, 503, 504:
		return target == ErrServerError
	default:
		return false
	}
}

// NotFoundError represents a 404 error specifically
type NotFoundError struct {
	HTTPError
	ResourceType string
	ResourceID   string
}

func (e *NotFoundError) Error() string {
	if e.ResourceType != "" && e.ResourceID != "" {
		return fmt.Sprintf("%s with ID '%s' not found", e.ResourceType, e.ResourceID)
	}
	return fmt.Sprintf("Resource not found: %s", e.URL)
}

func (e *NotFoundError) Is(target error) bool {
	return target == ErrNotFound
}

// APIError represents an error at the API level
type APIError struct {
	Endpoint   string
	URL        string
	StatusCode int
	Message    string
	Err        error
}

func (e *APIError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("API error [%s]: %s - %v", e.Endpoint, e.Message, e.Err)
	}
	return fmt.Sprintf("API error [%s]: %s", e.Endpoint, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

func (e *APIError) Is(target error) bool {
	// If we have an underlying error, let it determine the error type
	if e.Err != nil {
		return Is(e.Err, target)
	}

	// Otherwise, map status codes to error types
	switch e.StatusCode {
	case 404:
		return target == ErrNotFound
	case 401, 403:
		return target == ErrUnauthorized
	case 400, 422:
		return target == ErrBadRequest
	case 429:
		return target == ErrRateLimit
	case 500, 502, 503, 504:
		return target == ErrServerError
	default:
		return false
	}
}

// ResourceNotFoundError specifically for when a resource is not found via API
type ResourceNotFoundError struct {
	APIError
	ResourceType string
	ResourceID   string
}

func (e *ResourceNotFoundError) Error() string {
	return fmt.Sprintf("%s with ID '%s' not found", e.ResourceType, e.ResourceID)
}

func (e *ResourceNotFoundError) Is(target error) bool {
	return target == ErrNotFound
}

// ProviderError represents an error at the provider level
type ProviderError struct {
	ProviderID   string
	ResourceType string
	ResourceID   string
	Message      string
	Err          error
}

func (e *ProviderError) Error() string {
	base := fmt.Sprintf("Agent [%s] error", e.ProviderID)
	if e.ResourceType != "" && e.ResourceID != "" {
		base = fmt.Sprintf("%s: %s with ID '%s'", base, e.ResourceType, e.ResourceID)
	}
	if e.Message != "" {
		base = fmt.Sprintf("%s - %s", base, e.Message)
	}
	if e.Err != nil {
		base = fmt.Sprintf("%s: %v", base, e.Err)
	}
	return base
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

func (e *ProviderError) Is(target error) bool {
	if e.Err != nil {
		return Is(e.Err, target)
	}
	return false
}

// NewAgentNotFoundError creates a new agent error with not found context
func NewAgentNotFoundError(agentID, resourceType, resourceID string, err error) error {
	return &ProviderError{
		ProviderID:   agentID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Message:      "not found",
		Err:          ErrNotFound,
	}
}
