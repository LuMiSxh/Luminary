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
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// AutoTracker provides comprehensive error tracking for all types of errors
type AutoTracker struct {
	// Enable panic recovery
	RecoverPanics bool

	// Enable goroutine tracking
	TrackGoroutines bool

	// Custom error classifiers
	Classifiers []ErrorClassifier
}

// ErrorClassifier allows custom error classification
type ErrorClassifier func(error) (ErrorCategory, map[string]interface{})

// NewAutoTracker creates a new comprehensive error tracker
func NewAutoTracker() *AutoTracker {
	return &AutoTracker{
		RecoverPanics:   true,
		TrackGoroutines: true,
		Classifiers:     []ErrorClassifier{},
	}
}

// WrapAny automatically handles any type of error with comprehensive classification
func (at *AutoTracker) WrapAny(err error, context ...map[string]interface{}) error {
	if err == nil {
		return nil
	}

	// Start with basic tracking
	trackedErr := Track(err, context...)

	// Enhanced classification for specific error types
	var te *TrackedError
	if errors.As(trackedErr, &te) {
		category, extraContext := at.classifySpecificError(err)
		if category != CategoryUnknown {
			te.Category = category
		}

		// Add extra context from classification
		for k, v := range extraContext {
			te.Context[k] = v
		}
	}

	return trackedErr
}

// WrapHTTPError specifically handles HTTP-related errors
func (at *AutoTracker) WrapHTTPError(err error, req *http.Request, resp *http.Response) error {
	if err == nil {
		return nil
	}

	ctx := map[string]interface{}{}

	if req != nil {
		ctx["http_method"] = req.Method
		ctx["url"] = req.URL.String()
		ctx["user_agent"] = req.UserAgent()

		if req.ContentLength > 0 {
			ctx["content_length"] = req.ContentLength
		}
	}

	if resp != nil {
		ctx["status_code"] = resp.StatusCode
		ctx["status_text"] = resp.Status
		ctx["content_type"] = resp.Header.Get("Content-Type")
		ctx["content_length"] = resp.ContentLength
	}

	trackedErr := at.WrapAny(err, ctx)
	var te *TrackedError
	if errors.As(trackedErr, &te) {
		te.Category = CategoryNetwork
	}

	return trackedErr
}

// WrapJSONError handles JSON parsing errors with context
func (at *AutoTracker) WrapJSONError(err error, data []byte, target interface{}) error {
	if err == nil {
		return nil
	}

	ctx := map[string]interface{}{
		"data_type":   "json",
		"data_size":   len(data),
		"target_type": fmt.Sprintf("%T", target),
	}

	// Try to identify where in the JSON the error occurred
	var jsonErr *json.SyntaxError
	if errors.As(err, &jsonErr) {
		ctx["json_offset"] = jsonErr.Offset
		ctx["error_type"] = "syntax_error"

		// Extract problematic section
		if jsonErr.Offset < int64(len(data)) && jsonErr.Offset > 0 {
			start := max(0, int(jsonErr.Offset)-20)
			end := min(len(data), int(jsonErr.Offset)+20)
			ctx["error_context"] = string(data[start:end])
		}
	}

	var jsonErr2 *json.UnmarshalTypeError
	if errors.As(err, &jsonErr2) {
		ctx["json_field"] = jsonErr2.Field
		ctx["expected_type"] = jsonErr2.Type.String()
		ctx["actual_value"] = jsonErr2.Value
		ctx["error_type"] = "type_error"
	}

	trackedErr := at.WrapAny(err, ctx)
	var te *TrackedError
	if errors.As(trackedErr, &te) {
		te.Category = CategoryParsing
	}

	return trackedErr
}

// WrapFileSystemError handles file system errors
func (at *AutoTracker) WrapFileSystemError(err error, path string, operation string) error {
	if err == nil {
		return nil
	}

	ctx := map[string]interface{}{
		"file_path":  path,
		"operation":  operation,
		"error_type": at.classifyFileSystemError(err),
	}

	// Add file info if accessible
	if info, statErr := os.Stat(path); statErr == nil {
		ctx["file_size"] = info.Size()
		ctx["file_mode"] = info.Mode().String()
		ctx["is_dir"] = info.IsDir()
		ctx["mod_time"] = info.ModTime()
	}

	trackedErr := at.WrapAny(err, ctx)
	var te *TrackedError
	if errors.As(trackedErr, &te) {
		te.Category = CategoryFileSystem
	}

	return trackedErr
}

// WrapDownloadError handles file download errors specific to manga downloading
func (at *AutoTracker) WrapDownloadError(err error, url, destPath string, bytesDownloaded int64) error {
	if err == nil {
		return nil
	}

	ctx := map[string]interface{}{
		"download_url":     url,
		"destination_path": destPath,
		"bytes_downloaded": bytesDownloaded,
		"operation":        "download",
	}

	// Classify download-specific error types
	errStr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errStr, "no space"):
		ctx["error_type"] = "insufficient_space"
	case strings.Contains(errStr, "permission"):
		ctx["error_type"] = "permission_denied"
	case strings.Contains(errStr, "connection"):
		ctx["error_type"] = "connection_interrupted"
	case strings.Contains(errStr, "timeout"):
		ctx["error_type"] = "download_timeout"
	default:
		ctx["error_type"] = "download_failed"
	}

	trackedErr := at.WrapAny(err, ctx)
	var te *TrackedError
	if errors.As(trackedErr, &te) {
		if strings.Contains(errStr, "connection") || strings.Contains(errStr, "timeout") {
			te.Category = CategoryNetwork
		} else {
			te.Category = CategoryFileSystem
		}
	}

	return trackedErr
}

// WrapContextError handles context-related errors
func (at *AutoTracker) WrapContextError(err error, ctx context.Context) error {
	if err == nil {
		return nil
	}

	contx := map[string]interface{}{}

	if ctx != nil {
		if deadline, ok := ctx.Deadline(); ok {
			contx["deadline"] = deadline
			contx["time_remaining"] = time.Until(deadline)
		}

		if ctx.Err() != nil {
			contx["context_error"] = ctx.Err().Error()
		}
	}

	category := CategoryUnknown
	if errors.Is(err, context.Canceled) {
		category = CategoryTimeout
		contx["error_type"] = "canceled"
	} else if errors.Is(err, context.DeadlineExceeded) {
		category = CategoryTimeout
		contx["error_type"] = "deadline_exceeded"
	}

	trackedErr := at.WrapAny(err, contx)
	var te *TrackedError
	if errors.As(trackedErr, &te) && category != CategoryUnknown {
		te.Category = category
	}

	return trackedErr
}

// RecoverPanic recovers from panics and converts them to tracked errors
func (at *AutoTracker) RecoverPanic() error {
	if !at.RecoverPanics {
		return nil
	}

	if r := recover(); r != nil {
		// Create error from panic
		var err error
		switch v := r.(type) {
		case error:
			err = v
		case string:
			err = fmt.Errorf("panic: %s", v)
		default:
			err = fmt.Errorf("panic: %v", v)
		}

		// Capture full stack trace
		stackBuf := make([]byte, 4096)
		stackSize := runtime.Stack(stackBuf, false)
		stackTrace := string(stackBuf[:stackSize])

		ctx := map[string]interface{}{
			"panic_type":  fmt.Sprintf("%T", r),
			"panic_value": fmt.Sprintf("%v", r),
			"stack_trace": stackTrace,
		}

		trackedErr := at.WrapAny(err, ctx)
		var te *TrackedError
		if errors.As(trackedErr, &te) {
			te.Category = CategoryPanic
		}

		return trackedErr
	}

	return nil
}

// WrapGoroutineError wraps errors that occur in goroutines
func (at *AutoTracker) WrapGoroutineError(err error, goroutineID int) error {
	if err == nil {
		return nil
	}

	ctx := map[string]interface{}{
		"goroutine_id":   goroutineID,
		"num_goroutines": runtime.NumGoroutine(),
	}

	return at.WrapAny(err, ctx)
}

// Comprehensive error classification
func (at *AutoTracker) classifySpecificError(err error) (ErrorCategory, map[string]interface{}) {
	ctx := make(map[string]interface{})

	// Check custom classifiers first
	for _, classifier := range at.Classifiers {
		if category, ctx := classifier(err); category != CategoryUnknown {
			return category, ctx
		}
	}

	// Network errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		ctx["is_net_error"] = true
		ctx["is_timeout"] = netErr.Timeout()
		ctx["is_temporary"] = netErr.Temporary()

		if netErr.Timeout() {
			return CategoryTimeout, ctx
		}
		return CategoryNetwork, ctx
	}

	// URL errors
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		ctx["operation"] = urlErr.Op
		ctx["url"] = urlErr.URL
		ctx["is_url_error"] = true

		return CategoryNetwork, ctx
	}

	// HTTP errors
	if strings.Contains(err.Error(), "http:") {
		return CategoryNetwork, ctx
	}

	// JSON errors
	var syntaxError *json.SyntaxError
	if errors.As(err, &syntaxError) {
		return CategoryParsing, ctx
	}
	var unmarshalTypeError *json.UnmarshalTypeError
	if errors.As(err, &unmarshalTypeError) {
		return CategoryParsing, ctx
	}

	// Context errors
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return CategoryTimeout, ctx
	}

	// File system errors
	var pathError *os.PathError
	if errors.As(err, &pathError) {
		return CategoryFileSystem, ctx
	}
	var linkError *os.LinkError
	if errors.As(err, &linkError) {
		return CategoryFileSystem, ctx
	}

	// System call errors
	var syscallErr *os.SyscallError
	if errors.As(err, &syscallErr) {
		ctx["syscall"] = syscallErr.Syscall
		ctx["errno"] = syscallErr.Err
		return CategoryFileSystem, ctx
	}

	// Syscall errors
	var errno syscall.Errno
	if errors.As(err, &errno) {
		ctx["errno"] = int(errno)
		ctx["errno_name"] = errno.Error()

		switch {
		case errors.Is(errno, syscall.ECONNREFUSED):
			return CategoryNetwork, ctx
		case errors.Is(errno, syscall.ETIMEDOUT):
			return CategoryTimeout, ctx
		case errors.Is(errno, syscall.ENOENT):
			return CategoryNotFound, ctx
		case errors.Is(errno, syscall.EACCES), errors.Is(errno, syscall.EPERM):
			return CategoryAuth, ctx
		default:
			return CategoryUnknown, ctx
		}
	}

	// Parse specific error messages
	errStr := strings.ToLower(err.Error())

	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "too many requests") {
		return CategoryRateLimit, ctx
	}

	if strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "forbidden") {
		return CategoryAuth, ctx
	}

	if strings.Contains(errStr, "not found") || strings.Contains(errStr, "404") {
		return CategoryNotFound, ctx
	}

	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline") {
		return CategoryTimeout, ctx
	}

	return CategoryUnknown, ctx
}

func (at *AutoTracker) classifyFileSystemError(err error) string {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		switch {
		case errors.Is(pathErr.Err, syscall.ENOENT):
			return "file_not_found"
		case errors.Is(pathErr.Err, syscall.EACCES):
			return "permission_denied"
		case errors.Is(pathErr.Err, syscall.EEXIST):
			return "file_exists"
		case errors.Is(pathErr.Err, syscall.EISDIR):
			return "is_directory"
		case errors.Is(pathErr.Err, syscall.ENOTDIR):
			return "not_directory"
		case errors.Is(pathErr.Err, syscall.ENOSPC):
			return "no_space"
		default:
			return "path_error"
		}
	}

	errStr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errStr, "permission denied"):
		return "permission_denied"
	case strings.Contains(errStr, "no such file"):
		return "file_not_found"
	case strings.Contains(errStr, "file exists"):
		return "file_exists"
	case strings.Contains(errStr, "is a directory"):
		return "is_directory"
	case strings.Contains(errStr, "not a directory"):
		return "not_directory"
	default:
		return "unknown_fs_error"
	}
}

// Middleware functions for automatic error wrapping

// HTTPMiddleware automatically wraps HTTP handler errors
func (at *AutoTracker) HTTPMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := at.RecoverPanic(); err != nil {
				http.Error(w, "Internal server error", 500)
				// Log the tracked error
				fmt.Printf("HTTP handler panic: %s\n", FormatChain(err))
			}
		}()

		next(w, r)
	}
}

// GoroutineWrapper wraps goroutine functions with error tracking
func (at *AutoTracker) GoroutineWrapper(fn func() error) func() {
	return func() {
		defer func() {
			if err := at.RecoverPanic(); err != nil {
				// Log goroutine panic
				fmt.Printf("Goroutine panic: %s\n", FormatChain(err))
			}
		}()

		if err := fn(); err != nil {
			// Extract goroutine ID
			goroutineID := getGoroutineID()
			trackedErr := at.WrapGoroutineError(err, goroutineID)

			// Log goroutine error
			fmt.Printf("Goroutine error: %s\n", FormatChain(trackedErr))
		}
	}
}

// Utility functions

func getGoroutineID() int {
	// Extract goroutine ID from stack trace
	buf := make([]byte, 64)
	buf = buf[:runtime.Stack(buf, false)]

	// Parse "goroutine 123 [running]:"
	idStr := string(buf)
	if idx := strings.Index(idStr, "goroutine "); idx != -1 {
		idStr = idStr[idx+10:]
		if idx := strings.Index(idStr, " "); idx != -1 {
			idStr = idStr[:idx]
			if id, err := strconv.Atoi(idStr); err == nil {
				return id
			}
		}
	}

	return 0
}

// DefaultTracker Global auto tracker instance
var DefaultTracker = NewAutoTracker()

// Auto automatically wraps any error with comprehensive tracking
func Auto(err error, context ...map[string]interface{}) error {
	return DefaultTracker.WrapAny(err, context...)
}

// AutoHTTP wraps HTTP errors
func AutoHTTP(err error, req *http.Request, resp *http.Response) error {
	return DefaultTracker.WrapHTTPError(err, req, resp)
}

// AutoJSON wraps JSON parsing errors
func AutoJSON(err error, data []byte, target interface{}) error {
	return DefaultTracker.WrapJSONError(err, data, target)
}

// AutoFS wraps file system errors
func AutoFS(err error, path string, operation string) error {
	return DefaultTracker.WrapFileSystemError(err, path, operation)
}

// AutoContext wraps context errors
func AutoContext(err error, ctx context.Context) error {
	return DefaultTracker.WrapContextError(err, ctx)
}

// AutoDownload wraps download errors
func AutoDownload(err error, url, destPath string, bytesDownloaded int64) error {
	return DefaultTracker.WrapDownloadError(err, url, destPath, bytesDownloaded)
}

// Recover recovers from panics
func Recover() error {
	return DefaultTracker.RecoverPanic()
}
