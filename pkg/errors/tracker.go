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
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ErrorCategory represents the type of error
type ErrorCategory string

const (
	CategoryUnknown    ErrorCategory = "unknown"
	CategoryNetwork    ErrorCategory = "network"
	CategoryParser     ErrorCategory = "parser"
	CategoryProvider   ErrorCategory = "provider"
	CategoryTimeout    ErrorCategory = "timeout"
	CategoryNotFound   ErrorCategory = "not_found"
	CategoryAuth       ErrorCategory = "auth"
	CategoryRateLimit  ErrorCategory = "rate_limit"
	CategoryFileSystem ErrorCategory = "filesystem"
	CategoryDownload   ErrorCategory = "download"
	CategoryPanic      ErrorCategory = "panic"
)

// TrackedError wraps an error with additional context
type TrackedError struct {
	Original    error
	RootCause   error
	UserMessage string
	Category    ErrorCategory
	CallChain   []FunctionCall
	Context     map[string]interface{}
	StackTrace  []StackFrame
	Timestamp   time.Time
}

// FunctionCall represents a function in the call chain
type FunctionCall struct {
	Function  string
	ShortName string
	Package   string
	File      string
	Line      int
	Operation string
	Context   map[string]interface{}
	Timestamp time.Time
}

// StackFrame represents a stack frame
type StackFrame struct {
	Function string
	File     string
	Line     int
}

// Error implements the error interface
func (e *TrackedError) Error() string {
	if e.UserMessage != "" {
		return e.UserMessage
	}
	if e.Original != nil {
		return e.Original.Error()
	}
	return "unknown error"
}

// Unwrap returns the original error
func (e *TrackedError) Unwrap() error {
	return e.Original
}

// Is implements errors.Is support
func (e *TrackedError) Is(target error) bool {
	return errors.Is(e.Original, target) || errors.Is(e.RootCause, target)
}

// As implements errors.As support
func (e *TrackedError) As(target interface{}) bool {
	if t, ok := target.(**TrackedError); ok {
		*t = e
		return true
	}
	return errors.As(e.Original, &target)
}

// GetContext returns the error context
func (e *TrackedError) GetContext() map[string]interface{} {
	return e.Context
}

// GetCategory returns the error category
func (e *TrackedError) GetCategory() string {
	return string(e.Category)
}

// GetFunctionChain returns the function call chain as a string
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

// GetChain returns the call chain details
func (e *TrackedError) GetChain() []FunctionCall {
	return e.CallChain
}

// GetOriginal returns the original error
func (e *TrackedError) GetOriginal() error {
	return e.Original
}

// GetRootCause returns the root cause error
func (e *TrackedError) GetRootCause() error {
	return e.RootCause
}

// trackError creates a new tracked error
func trackError(err error, context ...map[string]interface{}) *TrackedError {
	if err == nil {
		return nil
	}

	// If already tracked, return as is
	var tracked *TrackedError
	if As(err, &tracked) {
		// Add new function to call chain
		pc, file, line, _ := runtime.Caller(2)
		fn := runtime.FuncForPC(pc)
		if fn == nil { // Add nil check for fn
			return tracked
		}

		call := FunctionCall{
			Function:  fn.Name(),
			ShortName: extractShortName(fn.Name()),
			Package:   extractPackageName(fn.Name()),
			File:      extractFileName(file),
			Line:      line,
			Timestamp: time.Now(),
		}

		if len(context) > 0 && context[0] != nil { // Add nil check for context
			call.Context = context[0]
		}

		tracked.CallChain = append(tracked.CallChain, call)
		return tracked
	}

	// Create new tracked error
	te := &TrackedError{
		Original:   err,
		RootCause:  findRootCause(err),
		Category:   classifyError(err),
		Context:    make(map[string]interface{}),
		Timestamp:  time.Now(),
		StackTrace: captureStackTrace(),
	}

	// Add initial function to call chain
	pc, file, line, _ := runtime.Caller(2)
	fn := runtime.FuncForPC(pc)
	if fn == nil { // Add nil check for fn
		return te
	}

	call := FunctionCall{
		Function:  fn.Name(),
		ShortName: extractShortName(fn.Name()),
		Package:   extractPackageName(fn.Name()),
		File:      extractFileName(file),
		Line:      line,
		Timestamp: time.Now(),
	}

	if len(context) > 0 && context[0] != nil { // Add nil check for context
		call.Context = context[0]
		// Also add to main context
		for k, v := range context[0] {
			te.Context[k] = v
		}
	}

	te.CallChain = []FunctionCall{call}

	return te
}

// Helper functions

func extractShortName(fullName string) string {
	idx := strings.LastIndex(fullName, ".")
	if idx == -1 {
		return fullName
	}

	shortName := fullName[idx+1:]

	// Handle method calls like "(*Provider).Search"
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

	// Remove the function name
	packageParts := parts[:len(parts)-1]
	return strings.Join(packageParts, ".")
}

func extractFileName(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}

func findRootCause(err error) error {
	root := err
	for {
		unwrapped := errors.Unwrap(root)
		if unwrapped == nil {
			break
		}
		root = unwrapped
	}
	return root
}

func classifyError(err error) ErrorCategory {
	if err == nil {
		return CategoryUnknown
	}

	errStr := strings.ToLower(err.Error())

	// Network errors
	if strings.Contains(errStr, "dial tcp") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network") {
		return CategoryNetwork
	}

	// Timeout errors
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") {
		return CategoryTimeout
	}

	// Not found errors
	if strings.Contains(errStr, "not found") ||
		strings.Contains(errStr, "404") {
		return CategoryNotFound
	}

	// Auth errors
	if strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "forbidden") ||
		strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "403") {
		return CategoryAuth
	}

	// Rate limit errors
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "too many requests") ||
		strings.Contains(errStr, "429") {
		return CategoryRateLimit
	}

	// File system errors
	if strings.Contains(errStr, "no such file") ||
		strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "file exists") {
		return CategoryFileSystem
	}

	// Parser errors
	if strings.Contains(errStr, "json") ||
		strings.Contains(errStr, "xml") ||
		strings.Contains(errStr, "parse") ||
		strings.Contains(errStr, "unmarshal") {
		return CategoryParser
	}

	return CategoryUnknown
}

func captureStackTrace() []StackFrame {
	var frames []StackFrame

	// Capture up to 10 frames, skipping the first 3 (this function and its callers)
	for i := 3; i < 13; i++ {
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

// Public functions for compatibility

// Is reports whether any error in err's chain matches target
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target
func As(err error, target interface{}) bool {
	if err == nil {
		return false
	}

	// We do NOT write "&target" here because we want to support both pointer and non-pointer targets
	//goland:noinspection GoErrorsAs
	return errors.As(err, target)
}

// Unwrap returns the result of calling the Unwrap method on err
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// Join returns an error that wraps the given errors
func Join(errs ...error) error {
	// Filter out nil errors
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
	}

	// Create combined error
	combinedErr := fmt.Errorf("multiple errors: %d errors occurred", len(nonNil))

	// Track the combined error with proper error handling
	te := trackError(combinedErr)
	if te == nil {
		return combinedErr // Fallback if tracking fails
	}

	te.Context["error_count"] = len(nonNil)
	te.Context["errors"] = nonNil

	// Collect call chains from all source errors
	var allCallChains []FunctionCall

	// First add the current call chain (where Join was called)
	allCallChains = append(allCallChains, te.CallChain...)

	// Then add call chains from all original errors
	for _, err := range nonNil {
		var tracked *TrackedError
		if err != nil && As(err, &tracked) && tracked != nil {
			allCallChains = append(allCallChains, tracked.CallChain...)
		}
	}

	// Replace the call chain with the merged one
	te.CallChain = allCallChains

	// Determine predominant category
	categoryCount := make(map[ErrorCategory]int)
	for _, err := range nonNil {
		var tracked *TrackedError
		if err != nil && As(err, &tracked) && tracked != nil {
			categoryCount[tracked.Category]++
		}
	}

	var maxCategory = CategoryUnknown
	maxCount := 0
	for cat, count := range categoryCount {
		if count > maxCount {
			maxCategory = cat
			maxCount = count
		}
	}

	te.Category = maxCategory

	return te
}
