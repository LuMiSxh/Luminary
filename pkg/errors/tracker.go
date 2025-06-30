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
	"errors"
	"runtime"
	"strings"
	"time"
)

// TrackedError wraps errors with automatic function call chain tracking
type TrackedError struct {
	Original    error                  `json:"original_error"`
	RootCause   error                  `json:"root_cause"`
	CallChain   []FunctionCall         `json:"call_chain"`
	Context     map[string]interface{} `json:"context,omitempty"`
	UserMessage string                 `json:"user_message,omitempty"`
	Category    ErrorCategory          `json:"category"`
	StackTrace  []StackFrame           `json:"stack_trace,omitempty"`
}

// FunctionCall represents a single function in the call chain
type FunctionCall struct {
	Function  string                 `json:"function"`
	ShortName string                 `json:"short_name"`
	Package   string                 `json:"package"`
	File      string                 `json:"file"`
	Line      int                    `json:"line"`
	Timestamp time.Time              `json:"timestamp"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Operation string                 `json:"operation,omitempty"`
}

// StackFrame represents a single frame in the stack trace
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// ErrorCategory helps classify different types of errors
type ErrorCategory string

const (
	CategoryNetwork    ErrorCategory = "network"
	CategoryProvider   ErrorCategory = "provider"
	CategoryParsing    ErrorCategory = "parsing"
	CategoryValidation ErrorCategory = "validation"
	CategoryTimeout    ErrorCategory = "timeout"
	CategoryAuth       ErrorCategory = "authentication"
	CategoryRateLimit  ErrorCategory = "rate_limit"
	CategoryNotFound   ErrorCategory = "not_found"
	CategoryFileSystem ErrorCategory = "filesystem"
	CategoryDownload   ErrorCategory = "download"
	CategoryPanic      ErrorCategory = "panic"
	CategoryUnknown    ErrorCategory = "unknown"
)

func (e *TrackedError) Error() string {
	if e.UserMessage != "" {
		return e.UserMessage
	}
	if e.Original != nil {
		return e.Original.Error()
	}
	return "unknown error"
}

func (e *TrackedError) Unwrap() error {
	return e.Original
}

func (e *TrackedError) Is(target error) bool {
	return (e.Original != nil && Is(e.Original, target)) ||
		(e.RootCause != nil && Is(e.RootCause, target))
}

// GetOriginal returns the original error that started this chain
func (e *TrackedError) GetOriginal() error {
	return e.Original
}

// GetRootCause returns the deepest error in the chain
func (e *TrackedError) GetRootCause() error {
	return e.RootCause
}

// GetChain returns the full function call chain
func (e *TrackedError) GetChain() []FunctionCall {
	return e.CallChain
}

// GetFunctionChain returns the function call path as a string
func (e *TrackedError) GetFunctionChain() string {
	if len(e.CallChain) == 0 {
		return ""
	}

	functions := make([]string, len(e.CallChain))
	for i, call := range e.CallChain {
		functions[i] = call.ShortName
	}

	return strings.Join(functions, " → ")
}

// GetCategory returns the error classification
func (e *TrackedError) GetCategory() ErrorCategory {
	return e.Category
}

// IsCategory checks if error belongs to a specific category
func (e *TrackedError) IsCategory(category ErrorCategory) bool {
	return e.Category == category
}

// GetLastFunction returns the most recent function in the chain
func (e *TrackedError) GetLastFunction() *FunctionCall {
	if len(e.CallChain) == 0 {
		return nil
	}
	return &e.CallChain[len(e.CallChain)-1]
}

// GetFirstFunction returns the first function in the chain
func (e *TrackedError) GetFirstFunction() *FunctionCall {
	if len(e.CallChain) == 0 {
		return nil
	}
	return &e.CallChain[0]
}

// GetContext returns the context data associated with the error
func (e *TrackedError) GetContext() map[string]interface{} {
	if e.Context == nil {
		return make(map[string]interface{})
	}
	return e.Context
}

// Track automatically wraps an error with function call tracking
func Track(err error, context ...map[string]interface{}) error {
	if err == nil {
		return nil
	}

	pc, file, line, ok := getCaller()
	if !ok {
		return createFallbackError(err, context...)
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return createFallbackError(err, context...)
	}

	fullFuncName := fn.Name()
	functionCall := FunctionCall{
		Function:  fullFuncName,
		ShortName: extractShortFunctionName(fullFuncName),
		Package:   extractPackageName(fullFuncName),
		File:      extractFileName(file),
		Line:      line,
		Timestamp: time.Now(),
		Operation: detectOperation(extractShortFunctionName(fullFuncName)),
	}

	// Merge context if provided
	if len(context) > 0 {
		functionCall.Context = context[0]
	}

	// If already a TrackedError, add to chain but preserve the original category
	var trackedErr *TrackedError
	if errors.As(err, &trackedErr) {
		trackedErr.CallChain = append(trackedErr.CallChain, functionCall)
		return trackedErr
	}

	// Create new TrackedError
	return &TrackedError{
		Original:   err,
		RootCause:  findRootCause(err),
		CallChain:  []FunctionCall{functionCall},
		Context:    make(map[string]interface{}),
		Category:   classifyError(err),
		StackTrace: captureStackTrace(),
	}
}

// getCaller walks up the stack to find the first non-tracking function
func getCaller() (uintptr, string, int, bool) {
	internalPatterns := []string{
		"pkg/errors/tracker.go",
		"pkg/errors/simple.go",
		"pkg/errors/coverage.go",
	}

	for i := 1; i < 10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		isInternal := false
		for _, pattern := range internalPatterns {
			if strings.Contains(file, pattern) {
				isInternal = true
				break
			}
		}

		if !isInternal {
			return pc, file, line, true
		}
	}

	return runtime.Caller(1)
}

// TrackWithMessage wraps an error with a user-friendly message
func TrackWithMessage(err error, userMessage string, context ...map[string]interface{}) error {
	trackedErr := Track(err, context...)
	var te *TrackedError
	if errors.As(trackedErr, &te) {
		te.UserMessage = userMessage
	}
	return trackedErr
}

// TrackWithContext wraps an error and adds context data
func TrackWithContext(err error, contextData map[string]interface{}) error {
	trackedErr := Track(err, contextData)
	var te *TrackedError
	if errors.As(trackedErr, &te) {
		for k, v := range contextData {
			te.Context[k] = v
		}
	}
	return trackedErr
}

// TrackNetwork marks an error as network-related
func TrackNetwork(err error, context ...map[string]interface{}) error {
	trackedErr := Track(err, context...)

	var te *TrackedError
	if errors.As(trackedErr, &te) && te.Category == CategoryUnknown {
		te.Category = CategoryNetwork
	}

	return trackedErr
}

// TrackProvider marks an error as provider-related
func TrackProvider(err error, providerID string, context ...map[string]interface{}) error {
	ctx := map[string]interface{}{"provider_id": providerID}
	if len(context) > 0 {
		for k, v := range context[0] {
			ctx[k] = v
		}
	}

	trackedErr := Track(err, ctx)

	var te *TrackedError
	if errors.As(trackedErr, &te) && te.Category == CategoryUnknown {
		te.Category = CategoryProvider
	}

	return trackedErr
}

// Helper functions

func extractShortFunctionName(fullName string) string {
	idx := strings.LastIndex(fullName, ".")
	if idx == -1 {
		return fullName
	}

	shortName := fullName[idx+1:]

	// Handle method calls like "(*SearchService).Search"
	if strings.Contains(fullName, "(*") && strings.Contains(fullName, ")") {
		start := strings.LastIndex(fullName, "(*")
		end := strings.Index(fullName[start:], ").")
		if start != -1 && end != -1 {
			structName := fullName[start+2 : start+end]
			return structName + "." + shortName
		}
	}

	return shortName
}

func extractPackageName(fullName string) string {
	parts := strings.Split(fullName, ".")
	if len(parts) <= 1 {
		return "unknown"
	}

	packageParts := parts[:len(parts)-1]
	packageName := strings.Join(packageParts, ".")

	if idx := strings.LastIndex(packageName, "/"); idx != -1 {
		return packageName[idx+1:]
	}

	return packageName
}

func extractFileName(fullPath string) string {
	if idx := strings.LastIndex(fullPath, "/"); idx != -1 {
		return fullPath[idx+1:]
	}
	return fullPath
}

func detectOperation(functionName string) string {
	lower := strings.ToLower(functionName)

	switch {
	case strings.Contains(lower, "search"):
		return "search"
	case strings.Contains(lower, "download"):
		return "download"
	case strings.Contains(lower, "get"), strings.Contains(lower, "fetch"):
		return "get"
	case strings.Contains(lower, "parse"):
		return "parse"
	case strings.Contains(lower, "validate"):
		return "validate"
	case strings.Contains(lower, "http"), strings.Contains(lower, "request"):
		return "http_request"
	case strings.Contains(lower, "auth"):
		return "authentication"
	default:
		return ""
	}
}

func classifyError(err error) ErrorCategory {
	if err == nil {
		return CategoryUnknown
	}

	errStr := strings.ToLower(err.Error())

	switch {
	case isNetworkError(errStr):
		return CategoryNetwork
	case isParsingError(errStr):
		return CategoryParsing
	case isTimeoutError(errStr):
		return CategoryTimeout
	case isAuthError(errStr):
		return CategoryAuth
	case isNotFoundError(errStr):
		return CategoryNotFound
	case isRateLimitError(errStr):
		return CategoryRateLimit
	case isFileSystemError(errStr):
		return CategoryFileSystem
	default:
		return CategoryUnknown
	}
}

func isNetworkError(errStr string) bool {
	networkPatterns := []string{
		"dial tcp", "connection refused", "no such host", "network is unreachable",
		"connection reset", "timeout", "tls", "certificate", "dns", "no route to host",
	}

	for _, pattern := range networkPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

func isParsingError(errStr string) bool {
	parsingPatterns := []string{
		"json", "xml", "yaml", "parse", "unmarshal", "invalid character",
		"unexpected end", "syntax error",
	}

	for _, pattern := range parsingPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

func isTimeoutError(errStr string) bool {
	return strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded")
}

func isAuthError(errStr string) bool {
	authPatterns := []string{"unauthorized", "forbidden", "authentication", "invalid token"}

	for _, pattern := range authPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

func isNotFoundError(errStr string) bool {
	return strings.Contains(errStr, "not found") || strings.Contains(errStr, "404")
}

func isRateLimitError(errStr string) bool {
	return strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "too many requests")
}

func isFileSystemError(errStr string) bool {
	fsPatterns := []string{"no such file", "permission denied", "file exists", "directory"}

	for _, pattern := range fsPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

func captureStackTrace() []StackFrame {
	var frames []StackFrame

	// Capture up to 20 frames, skipping the first two (this function and getCaller)
	for i := 2; i < 22; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		frames = append(frames, StackFrame{
			Function: fn.Name(),
			File:     extractFileName(file),
			Line:     line,
		})
	}

	return frames
}

func findRootCause(err error) error {
	root := err
	for {
		if unwrapped := Unwrap(root); unwrapped != nil {
			root = unwrapped
		} else {
			break
		}
	}
	return root
}

func createFallbackError(err error, context ...map[string]interface{}) error {
	functionCall := FunctionCall{
		Function:  "unknown",
		ShortName: "unknown",
		Package:   "unknown",
		File:      "unknown",
		Line:      0,
		Timestamp: time.Now(),
	}

	if len(context) > 0 {
		functionCall.Context = context[0]
	}

	return &TrackedError{
		Original:   err,
		RootCause:  findRootCause(err),
		CallChain:  []FunctionCall{functionCall},
		Context:    make(map[string]interface{}),
		Category:   classifyError(err),
		StackTrace: captureStackTrace(),
	}
}
