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

package errors

import (
	"context"
	"errors"
	"fmt"
)

// ErrorBuilder provides a fluent interface for building tracked errors
type ErrorBuilder struct {
	err *TrackedError
}

// Track wraps any error with automatic tracking and returns a builder
func Track(err error) *ErrorBuilder {
	if err == nil {
		return nil
	}

	// If it's already a TrackedError, wrap it in a builder
	var tracked *TrackedError
	if As(err, &tracked) {
		return &ErrorBuilder{err: tracked}
	}

	// Create new tracked error
	tracked = trackError(err)
	return &ErrorBuilder{err: tracked}
}

// New creates a new error with tracking
func New(message string) *ErrorBuilder {
	return Track(fmt.Errorf(message))
}

// Newf creates a new formatted error with tracking
func Newf(format string, args ...interface{}) *ErrorBuilder {
	return Track(fmt.Errorf(format, args...))
}

// WithContext adds context data to the error
func (b *ErrorBuilder) WithContext(key string, value interface{}) *ErrorBuilder {
	if b == nil || b.err == nil {
		return b
	}

	b.err.Context[key] = value
	return b
}

// WithContextMap adds multiple context values
func (b *ErrorBuilder) WithContextMap(context map[string]interface{}) *ErrorBuilder {
	if b == nil || b.err == nil {
		return b
	}

	for k, v := range context {
		b.err.Context[k] = v
	}
	return b
}

// WithMessage sets a user-friendly message
func (b *ErrorBuilder) WithMessage(message string) *ErrorBuilder {
	if b == nil || b.err == nil {
		return b
	}

	b.err.UserMessage = message
	return b
}

// WithMessagef sets a formatted user-friendly message
func (b *ErrorBuilder) WithMessagef(format string, args ...interface{}) *ErrorBuilder {
	return b.WithMessage(fmt.Sprintf(format, args...))
}

// WithOperation adds operation context
func (b *ErrorBuilder) WithOperation(operation string) *ErrorBuilder {
	if b == nil || b.err == nil {
		return b
	}

	if len(b.err.CallChain) > 0 {
		b.err.CallChain[len(b.err.CallChain)-1].Operation = operation
	}

	return b
}

// AsCategory sets the error category
func (b *ErrorBuilder) AsCategory(category ErrorCategory) *ErrorBuilder {
	if b == nil || b.err == nil {
		return b
	}

	b.err.Category = category
	return b
}

// Category-specific helpers

// AsNetwork marks the error as network-related
func (b *ErrorBuilder) AsNetwork() *ErrorBuilder {
	return b.AsCategory(CategoryNetwork)
}

// AsParser marks the error as parsing-related
func (b *ErrorBuilder) AsParser() *ErrorBuilder {
	return b.AsCategory(CategoryParser)
}

// AsProvider marks the error as provider-related
func (b *ErrorBuilder) AsProvider(providerID string) *ErrorBuilder {
	return b.
		AsCategory(CategoryProvider).
		WithContext("provider_id", providerID)
}

// AsTimeout marks the error as timeout-related
func (b *ErrorBuilder) AsTimeout() *ErrorBuilder {
	return b.AsCategory(CategoryTimeout)
}

// AsNotFound marks the error as not-found
func (b *ErrorBuilder) AsNotFound() *ErrorBuilder {
	return b.AsCategory(CategoryNotFound)
}

// AsAuth marks the error as authentication-related
func (b *ErrorBuilder) AsAuth() *ErrorBuilder {
	return b.AsCategory(CategoryAuth)
}

// AsRateLimit marks the error as rate-limit-related
func (b *ErrorBuilder) AsRateLimit() *ErrorBuilder {
	return b.AsCategory(CategoryRateLimit)
}

// AsFileSystem marks the error as filesystem-related
func (b *ErrorBuilder) AsFileSystem() *ErrorBuilder {
	return b.AsCategory(CategoryFileSystem)
}

// AsDownload marks the error as download-related
func (b *ErrorBuilder) AsDownload() *ErrorBuilder {
	return b.AsCategory(CategoryDownload)
}

// AsPanic marks the error as panic-related
func (b *ErrorBuilder) AsPanic() *ErrorBuilder {
	return b.AsCategory(CategoryPanic)
}

// Error returns the tracked error
func (b *ErrorBuilder) Error() error {
	if b == nil || b.err == nil {
		return nil
	}
	return b.err
}

// Unwrap returns the tracked error for error unwrapping
func (b *ErrorBuilder) Unwrap() error {
	return b.Error()
}

// String implements fmt.Stringer
func (b *ErrorBuilder) String() string {
	if b == nil || b.err == nil {
		return ""
	}
	return b.err.Error()
}

// Must panics if the error is not nil
func (b *ErrorBuilder) Must() {
	if b != nil && b.err != nil {
		panic(b.err)
	}
}

// Must is used for functions that return a value and error
func Must[T any](value T, err error) T {
	if err != nil {
		panic(Track(err))
	}
	return value
}

// Handle executes a function if there's an error
func (b *ErrorBuilder) Handle(fn func(error)) *ErrorBuilder {
	if b != nil && b.err != nil {
		fn(b.err)
	}
	return b
}

// Log logs the error using the provided logger function
func (b *ErrorBuilder) Log(logFn func(string, ...interface{})) *ErrorBuilder {
	if b != nil && b.err != nil {
		logFn("Error: %v", b.err)
	}
	return b
}

// Wrap wraps this error with additional context
func (b *ErrorBuilder) Wrap(message string) *ErrorBuilder {
	if b == nil || b.err == nil {
		return b
	}

	wrapped := fmt.Errorf("%s: %w", message, b.err)
	return Track(wrapped)
}

// Wrapf wraps this error with formatted context
func (b *ErrorBuilder) Wrapf(format string, args ...interface{}) *ErrorBuilder {
	message := fmt.Sprintf(format, args...)
	return b.Wrap(message)
}

// Context helpers for common scenarios

// WithHTTPContext adds HTTP-related context
func (b *ErrorBuilder) WithHTTPContext(method, url string, statusCode int) *ErrorBuilder {
	return b.
		WithContext("method", method).
		WithContext("url", url).
		WithContext("status_code", statusCode)
}

// WithFileContext adds file-related context
func (b *ErrorBuilder) WithFileContext(path string, operation string) *ErrorBuilder {
	return b.
		WithContext("file_path", path).
		WithContext("file_operation", operation)
}

// WithRetryContext adds retry-related context
func (b *ErrorBuilder) WithRetryContext(attempt, maxAttempts int) *ErrorBuilder {
	return b.
		WithContext("attempt", attempt).
		WithContext("max_attempts", maxAttempts)
}

// IsCategory checks if the error is of a specific category
func (b *ErrorBuilder) IsCategory(category ErrorCategory) bool {
	if b == nil || b.err == nil {
		return false
	}
	return b.err.Category == category
}

// IsRetryable checks if the error should be retried
func (b *ErrorBuilder) IsRetryable() bool {
	if b == nil || b.err == nil {
		return false
	}

	switch b.err.Category {
	case CategoryNetwork, CategoryTimeout, CategoryRateLimit:
		// Check if it's not a client error
		if statusCode, ok := b.err.Context["status_code"].(int); ok {
			return statusCode >= 500 || statusCode == 429
		}
		return true
	default:
		return false
	}
}

// Chain multiple errors together
func Chain(errs ...error) error {
	var nonNil []error
	for _, err := range errs {
		if err != nil {
			nonNil = append(nonNil, err)
		}
	}

	switch len(nonNil) {
	case 0:
		return nil
	case 1:
		return Track(nonNil[0]).Error()
	default:
		return Join(nonNil...)
	}
}

// FromContext creates an error from a context
func FromContext(ctx context.Context) *ErrorBuilder {
	if err := ctx.Err(); err != nil {
		builder := Track(err)

		if errors.Is(err, context.Canceled) {
			return builder.
				AsCategory(CategoryTimeout).
				WithMessage("Operation was cancelled")
		} else if errors.Is(err, context.DeadlineExceeded) {
			return builder.
				AsCategory(CategoryTimeout).
				WithMessage("Operation timed out")
		}

		return builder
	}

	return nil
}
