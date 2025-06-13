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

package rpc

import (
	"Luminary/pkg/errors"
	"fmt"
	"strings"
	"time"
)

// Error represents RPC-level errors with automatic function call tracking
type Error struct {
	// Standard RPC error fields
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`

	// Enhanced function call tracking
	FunctionChain string                `json:"function_chain,omitempty"`
	CallDetails   []errors.FunctionCall `json:"call_details,omitempty"`
	ErrorCategory errors.ErrorCategory  `json:"error_category,omitempty"`
	OriginalError string                `json:"original_error,omitempty"`
	RootCause     string                `json:"root_cause,omitempty"`
	Timestamp     time.Time             `json:"timestamp"`

	// Request tracking
	Service   string `json:"service,omitempty"`
	Method    string `json:"method,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("RPC Error %d: %s", e.Code, e.Message)
}

// Enhanced RPC Error codes with better categorization
const (
	// Input/Validation errors (1000-1099)
	ErrCodeInvalidInput     = -1001
	ErrCodeInvalidData      = -1002
	ErrCodeValidationFailed = -1003

	// Resource/Provider errors (1100-1199)
	ErrCodeProviderNotFound = -1101
	ErrCodeResourceNotFound = -1102
	ErrCodeProviderError    = -1103

	// Operation errors (1200-1299)
	ErrCodeSearchFailed = -1201
	ErrCodeFetchFailed  = -1202
	ErrCodeListFailed   = -1203

	// Network errors (2000-2099)
	ErrCodeNetworkUnavailable = -2001
	ErrCodeNetworkTimeout     = -2002
	ErrCodeConnectionFailed   = -2003
	ErrCodeDNSFailure         = -2004
	ErrCodeHTTPError          = -2005

	// Timeout errors (2100-2199)
	ErrCodeTimeout          = -2101
	ErrCodeDeadlineExceeded = -2102
	ErrCodeContextCanceled  = -2103

	// Authentication/Authorization (2200-2299)
	ErrCodeAuthFailed   = -2201
	ErrCodeRateLimited  = -2202
	ErrCodeForbidden    = -2203
	ErrCodeUnauthorized = -2204

	// Parsing/Data errors (3000-3099)
	ErrCodeParsingFailed = -3001
	ErrCodeInvalidFormat = -3002
	ErrCodeDataCorrupted = -3003
	ErrCodeJSONError     = -3004
	ErrCodeXMLError      = -3005

	// File system errors (3100-3199)
	ErrCodeFileNotFound      = -3101
	ErrCodePermissionDenied  = -3102
	ErrCodeFileSystemError   = -3103
	ErrCodeInsufficientSpace = -3104

	// Download errors (3200-3299)
	ErrCodeDownloadInterrupted = -3201
	ErrCodeDownloadTimeout     = -3202
	ErrCodeDownloadCorrupted   = -3203
	ErrCodeDownloadFailed      = -3204

	// System errors (9000-9099)
	ErrCodePanic         = -9001
	ErrCodeInternalError = -9002
	ErrCodeUnknownError  = -9099
)

// NewError creates RPC error from tracked error with automatic analysis
func NewError(err error, service, method string, requestData map[string]interface{}) *Error {
	rpcError := &Error{
		Data:      make(map[string]interface{}),
		Timestamp: time.Now(),
		Service:   service,
		Method:    method,
	}

	// Copy request data
	for k, v := range requestData {
		rpcError.Data[k] = v
	}

	// Handle TrackedError specifically
	if trackedErr, ok := err.(*errors.TrackedError); ok {
		rpcError.Message = trackedErr.Error()
		rpcError.FunctionChain = trackedErr.GetFunctionChain()
		rpcError.CallDetails = trackedErr.GetChain()
		rpcError.ErrorCategory = trackedErr.GetCategory()

		if trackedErr.GetOriginal() != nil {
			rpcError.OriginalError = trackedErr.GetOriginal().Error()
		}

		if trackedErr.GetRootCause() != nil {
			rpcError.RootCause = trackedErr.GetRootCause().Error()
		}

		// Merge context data
		for k, v := range trackedErr.GetContext() {
			rpcError.Data[k] = v
		}

		// Determine error code based on category and context
		rpcError.Code = determineErrorCode(trackedErr)
	} else {
		// For non-tracked errors, still provide basic info
		rpcError.Message = err.Error()
		rpcError.OriginalError = err.Error()
		rpcError.RootCause = err.Error()
		rpcError.ErrorCategory = errors.CategoryUnknown
		rpcError.Code = ErrCodeUnknownError
		rpcError.Data["error"] = err.Error()
	}

	return rpcError
}

// Determine appropriate error code based on tracked error analysis
func determineErrorCode(trackedErr *errors.TrackedError) int {
	category := trackedErr.GetCategory()
	context := trackedErr.GetContext()
	originalErr := trackedErr.GetOriginal()

	// Category-based code selection
	switch category {
	case errors.CategoryNetwork:
		// Check for specific network error types
		if originalErr != nil {
			errStr := strings.ToLower(originalErr.Error())
			switch {
			case strings.Contains(errStr, "no such host"), strings.Contains(errStr, "dns"):
				return ErrCodeDNSFailure
			case strings.Contains(errStr, "connection refused"):
				return ErrCodeConnectionFailed
			case strings.Contains(errStr, "timeout"), strings.Contains(errStr, "deadline"):
				return ErrCodeNetworkTimeout
			default:
				return ErrCodeNetworkUnavailable
			}
		}
		return ErrCodeNetworkUnavailable

	case errors.CategoryTimeout:
		if strings.Contains(strings.ToLower(originalErr.Error()), "canceled") {
			return ErrCodeContextCanceled
		}
		return ErrCodeTimeout

	case errors.CategoryParsing:
		if strings.Contains(strings.ToLower(originalErr.Error()), "json") {
			return ErrCodeJSONError
		}
		return ErrCodeParsingFailed

	case errors.CategoryAuth:
		return ErrCodeAuthFailed

	case errors.CategoryRateLimit:
		return ErrCodeRateLimited

	case errors.CategoryNotFound:
		// Check context to determine if it's provider or resource
		if _, ok := context["provider_id"]; ok {
			return ErrCodeProviderNotFound
		}
		return ErrCodeResourceNotFound

	case errors.CategoryFileSystem:
		errStr := strings.ToLower(originalErr.Error())
		switch {
		case strings.Contains(errStr, "no such file"), strings.Contains(errStr, "not found"):
			return ErrCodeFileNotFound
		case strings.Contains(errStr, "permission"), strings.Contains(errStr, "access"):
			return ErrCodePermissionDenied
		case strings.Contains(errStr, "no space"), strings.Contains(errStr, "disk full"):
			return ErrCodeInsufficientSpace
		default:
			return ErrCodeFileSystemError
		}

	case errors.CategoryDownload:
		errStr := strings.ToLower(originalErr.Error())
		switch {
		case strings.Contains(errStr, "interrupted"), strings.Contains(errStr, "connection"):
			return ErrCodeDownloadInterrupted
		case strings.Contains(errStr, "timeout"), strings.Contains(errStr, "deadline"):
			return ErrCodeDownloadTimeout
		case strings.Contains(errStr, "corrupted"), strings.Contains(errStr, "checksum"):
			return ErrCodeDownloadCorrupted
		default:
			return ErrCodeDownloadFailed
		}

	case errors.CategoryPanic:
		return ErrCodePanic

	case errors.CategoryProvider:
		return ErrCodeProviderError

	default:
		// Try to infer from function chain or context
		if len(trackedErr.CallChain) > 0 {
			lastCall := trackedErr.CallChain[len(trackedErr.CallChain)-1]
			switch lastCall.Operation {
			case "search":
				return ErrCodeSearchFailed
			case "download":
				return ErrCodeDownloadFailed
			case "get", "fetch":
				return ErrCodeFetchFailed
			case "validation":
				return ErrCodeValidationFailed
			}
		}

		return ErrCodeInternalError
	}
}

// Simplified error creation functions (much easier to use)

// SearchFailed - Create search failure error
func SearchFailed(err error, query string) *Error {
	trackedErr := errors.T(err) // Automatically tracks function chain
	requestData := map[string]interface{}{"query": query}
	return NewError(trackedErr, "SearchService", "Search", requestData)
}

// DownloadFailed - Create download failure error
func DownloadFailed(err error, chapterID string) *Error {
	trackedErr := errors.T(err)
	requestData := map[string]interface{}{"chapter_id": chapterID}
	return NewError(trackedErr, "DownloadService", "Download", requestData)
}

// ProviderNotFound - Create provider not found error
func ProviderNotFound(providerID string) *Error {
	baseErr := fmt.Errorf("provider '%s' not found", providerID)
	trackedErr := errors.T(baseErr)
	requestData := map[string]interface{}{"provider_id": providerID}
	return NewError(trackedErr, "ProviderService", "GetProvider", requestData)
}

// ResourceNotFound - Create resource not found error
func ResourceNotFound(resourceType, resourceID string) *Error {
	baseErr := fmt.Errorf("%s '%s' not found", resourceType, resourceID)
	trackedErr := errors.T(baseErr)
	requestData := map[string]interface{}{
		"resource_type": resourceType,
		"resource_id":   resourceID,
	}
	return NewError(trackedErr, "ResourceService", "GetResource", requestData)
}

// InvalidInput - Create invalid input error
func InvalidInput(field, value string) *Error {
	baseErr := fmt.Errorf("invalid value '%s' for field '%s'", value, field)
	trackedErr := errors.T(baseErr)
	requestData := map[string]interface{}{
		"invalid_field": field,
		"invalid_value": value,
	}
	return NewError(trackedErr, "ValidationService", "ValidateInput", requestData)
}

// NetworkError - Create network error
func NetworkError(err error) *Error {
	trackedErr := errors.TN(err) // Automatically categorizes as network
	return NewError(trackedErr, "NetworkService", "Request", map[string]interface{}{})
}

// Timeout - Create timeout error
func Timeout(operation string, duration time.Duration) *Error {
	baseErr := fmt.Errorf("operation '%s' timed out after %v", operation, duration)
	trackedErr := errors.T(baseErr)
	requestData := map[string]interface{}{
		"operation": operation,
		"timeout":   duration.String(),
	}
	return NewError(trackedErr, "TimeoutService", operation, requestData)
}

// DownloadInterrupted - Create download interruption error
func DownloadInterrupted(err error, url, destPath string, bytesDownloaded int64) *Error {
	trackedErr := errors.AutoDownload(err, url, destPath, bytesDownloaded)
	requestData := map[string]interface{}{
		"url":              url,
		"destination":      destPath,
		"bytes_downloaded": bytesDownloaded,
	}
	return NewError(trackedErr, "DownloadService", "DownloadFile", requestData)
}

// Helper methods for error analysis

// IsNetworkIssue checks if error is network-related
func (e *Error) IsNetworkIssue() bool {
	return e.ErrorCategory == errors.CategoryNetwork ||
		(e.Code >= ErrCodeNetworkUnavailable && e.Code <= ErrCodeHTTPError)
}

// IsProviderIssue checks if error is provider-related
func (e *Error) IsProviderIssue() bool {
	return e.ErrorCategory == errors.CategoryProvider ||
		(e.Code >= ErrCodeProviderNotFound && e.Code <= ErrCodeProviderError)
}

// IsTimeout checks if error is timeout-related
func (e *Error) IsTimeout() bool {
	return e.ErrorCategory == errors.CategoryTimeout ||
		(e.Code >= ErrCodeTimeout && e.Code <= ErrCodeContextCanceled)
}

// IsParsingIssue checks if error is parsing-related
func (e *Error) IsParsingIssue() bool {
	return e.ErrorCategory == errors.CategoryParsing ||
		(e.Code >= ErrCodeParsingFailed && e.Code <= ErrCodeJSONError)
}

// GetSuggestedAction returns suggested action based on error type
func (e *Error) GetSuggestedAction() string {
	switch {
	case e.IsNetworkIssue():
		return "Check your internet connection and try again"
	case e.IsTimeout():
		return "The operation took too long. Try again or increase timeout"
	case e.IsProviderIssue():
		return "The provider may be temporarily unavailable. Try a different provider"
	case e.IsParsingIssue():
		return "The response format was unexpected. This may be a provider issue"
	default:
		return "An unexpected error occurred. Please try again"
	}
}

// GetHTTPStatusCode returns appropriate HTTP status code
func (e *Error) GetHTTPStatusCode() int {
	switch {
	case e.Code >= ErrCodeInvalidInput && e.Code < ErrCodeProviderNotFound:
		return 400 // Bad Request
	case e.Code >= ErrCodeProviderNotFound && e.Code < ErrCodeSearchFailed:
		return 404 // Not Found
	case e.IsTimeout():
		return 408 // Request Timeout
	case e.Code == ErrCodeRateLimited:
		return 429 // Too Many Requests
	case e.IsNetworkIssue():
		return 503 // Service Unavailable
	case e.Code == ErrCodeForbidden || e.Code == ErrCodeUnauthorized:
		return 403 // Forbidden
	default:
		return 500 // Internal Server Error
	}
}

// ExtractRequestContext helper - now much simpler
func ExtractRequestContext(serviceName, methodName string, args interface{}) map[string]interface{} {
	context := map[string]interface{}{
		"service": serviceName,
		"method":  methodName,
	}

	// Type-specific context extraction (simplified)
	switch v := args.(type) {
	case *SearchRequest:
		if v.Query != "" {
			context["query"] = v.Query
		}
		if v.Provider != "" {
			context["provider"] = v.Provider
		}
	case *DownloadRequest:
		if v.ChapterID != "" {
			context["chapter_id"] = v.ChapterID
		}
	case *InfoRequest:
		if v.MangaID != "" {
			context["manga_id"] = v.MangaID
		}
	case *ListRequest:
		if v.Provider != "" {
			context["provider"] = v.Provider
		}
	}

	return context
}
