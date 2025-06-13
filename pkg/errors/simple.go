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
	"net/url"
	"strings"
	"time"
)

// SIMPLE API - These are the only functions you need to use in most cases
// They automatically detect function names and classify errors

// T (Track) - The main function you'll use everywhere
// Wrap any error with T(err) and it automatically handles everything
func T(err error) error {
	if err == nil {
		return nil
	}
	return Track(err)
}

// TC (Track with Context) - Add custom context data
func TC(err error, context map[string]interface{}) error {
	if err == nil {
		return nil
	}
	return Track(err, context)
}

// TM (Track with Message) - Add a user-friendly message
func TM(err error, message string) error {
	if err == nil {
		return nil
	}
	return TrackWithMessage(err, message)
}

// TN (Track Network) - For network/HTTP errors
func TN(err error) error {
	if err == nil {
		return nil
	}
	return TrackNetwork(err)
}

// TP (Track Provider) - For provider-specific errors
func TP(err error, providerID string) error {
	if err == nil {
		return nil
	}
	return TrackProvider(err, providerID)
}

// Join - Combine multiple errors into a single tracked error
// Usage: err := errors.Join(err1, err2, err3)
func Join(errs ...error) error {
	// Filter out nil errors
	var nonNilErrs []error
	for _, err := range errs {
		if err != nil {
			nonNilErrs = append(nonNilErrs, err)
		}
	}

	// Handle edge cases
	switch len(nonNilErrs) {
	case 0:
		return nil
	case 1:
		return T(nonNilErrs[0])
	}

	// Create combined error message
	messages := make([]string, len(nonNilErrs))
	for i, err := range nonNilErrs {
		messages[i] = err.Error()
	}
	combinedMessage := strings.Join(messages, "; ")
	combinedErr := fmt.Errorf("multiple errors: %s", combinedMessage)

	// Track the combined error
	trackedErr := Track(combinedErr)
	var te *TrackedError
	errors.As(trackedErr, &te)

	// Store original errors in context
	te.Context["original_errors"] = nonNilErrs

	// Merge call chains and contexts from any TrackedErrors
	var combinedCallChain []FunctionCall
	combinedContext := make(map[string]interface{})

	for i, err := range nonNilErrs {
		var existingTE *TrackedError
		if errors.As(err, &existingTE) {
			// Add call chain (avoiding duplicates)
			combinedCallChain = mergeCallChains(combinedCallChain, existingTE.CallChain)

			// Merge context
			for k, v := range existingTE.GetContext() {
				// Add error index prefix to avoid key collisions
				combinedContext[fmt.Sprintf("err%d_%s", i, k)] = v
			}

			// Add category if missing
			if te.Category == CategoryUnknown && existingTE.Category != CategoryUnknown {
				te.Category = existingTE.Category
			}
		}
	}

	// Add the merged call chains if any were found
	if len(combinedCallChain) > 0 {
		// Keep the first entry in te.CallChain (from Track call above)
		// and append the merged chains
		if len(te.CallChain) > 0 {
			te.CallChain = append(te.CallChain, combinedCallChain...)
		} else {
			te.CallChain = combinedCallChain
		}
	}

	// Add merged context
	for k, v := range combinedContext {
		te.Context[k] = v
	}

	return te
}

// mergeCallChains combines call chains while avoiding duplicates
func mergeCallChains(chain1, chain2 []FunctionCall) []FunctionCall {
	if len(chain2) == 0 {
		return chain1
	}

	result := make([]FunctionCall, len(chain1))
	copy(result, chain1)

	// Add entries from chain2 that don't duplicate function+file+line
	for _, call := range chain2 {
		isDuplicate := false
		for _, existing := range result {
			if call.Function == existing.Function &&
				call.File == existing.File &&
				call.Line == existing.Line {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			result = append(result, call)
		}
	}

	return result
}

// AUTOMATIC ERROR DETECTION
// These functions automatically detect and wrap common error types

// Must - Convert any function call to a tracked error
// Usage: result := Must(someFunction())
func Must[V any](value V, err error) V {
	if err != nil {
		panic(T(err))
	}
	return value
}

// Try - Safely execute a function and return tracked error
// Usage: err := Try(func() error { return someFunction() })
func Try(fn func() error) error {
	defer func() {
		if r := recover(); r != nil {
			// Convert panic to tracked error
			var err error
			switch v := r.(type) {
			case error:
				err = v
			default:
				err = fmt.Errorf("panic: %v", v)
			}
			panic(T(err))
		}
	}()

	if err := fn(); err != nil {
		return T(err)
	}
	return nil
}

// WithTimeout - Create context with timeout and automatic error tracking
func WithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(parent, timeout)

	// Return a wrapped cancel function that tracks context errors
	wrappedCancel := func() {
		cancel()
		if ctx.Err() != nil {
			// Context error occurred
			_ = AutoContext(ctx.Err(), ctx)
		}
	}

	return ctx, wrappedCancel, nil
}

// ANALYSIS FUNCTIONS
// These help you understand what went wrong

// IsNetwork - Check if error is network-related
func IsNetwork(err error) bool {
	var te *TrackedError
	if errors.As(err, &te) {
		return te.IsCategory(CategoryNetwork)
	}
	return isNetworkError(strings.ToLower(err.Error()))
}

// IsProvider - Check if error is provider-related
func IsProvider(err error) bool {
	var te *TrackedError
	if errors.As(err, &te) {
		return te.IsCategory(CategoryProvider)
	}
	return false
}

// IsParsing - Check if error is parsing-related
func IsParsing(err error) bool {
	var te *TrackedError
	if errors.As(err, &te) {
		return te.IsCategory(CategoryParsing)
	}
	return isParsingError(strings.ToLower(err.Error()))
}

// IsTimeout - Check if error is timeout-related
func IsTimeout(err error) bool {
	var te *TrackedError
	if errors.As(err, &te) {
		return te.IsCategory(CategoryTimeout)
	}
	return isTimeoutError(strings.ToLower(err.Error()))
}

// GetCategory - Get error category
func GetCategory(err error) ErrorCategory {
	var te *TrackedError
	if errors.As(err, &te) {
		return te.GetCategory()
	}
	return CategoryUnknown
}

// GetFunctionChain - Get the function call path
func GetFunctionChain(err error) string {
	var te *TrackedError
	if errors.As(err, &te) {
		return te.GetFunctionChain()
	}
	return ""
}

// GetContext - Get error context data
func GetContext(err error) map[string]interface{} {
	var te *TrackedError
	if errors.As(err, &te) {
		return te.GetContext()
	}
	return nil
}

// FORMATTING FUNCTIONS

// FormatChain - Human-readable error chain (enhanced version)
func FormatChain(err error) string {
	var te *TrackedError
	ok := errors.As(err, &te)
	if !ok {
		return err.Error()
	}

	var parts []string

	// User message if available
	if te.UserMessage != "" {
		parts = append(parts, fmt.Sprintf("Error: %s", te.UserMessage))
	} else {
		parts = append(parts, fmt.Sprintf("Error: %s", te.Error()))
	}

	// Function call chain (this is the new improved part)
	if len(te.CallChain) > 0 {
		parts = append(parts, "\nFunction Call Chain:")

		// Calculate total duration if we have timestamps
		var totalTime time.Duration
		if len(te.CallChain) > 1 {
			firstTime := te.CallChain[0].Timestamp
			lastTime := te.CallChain[len(te.CallChain)-1].Timestamp
			totalTime = lastTime.Sub(firstTime)
		}

		for i, call := range te.CallChain {
			timestamp := call.Timestamp.Format("15:04:05.000")
			parts = append(parts, fmt.Sprintf("  %d. [%s] %s() at %s:%d",
				i+1, timestamp, call.ShortName, call.File, call.Line))

			// Show operation if detected
			if call.Operation != "" {
				parts = append(parts, fmt.Sprintf("      Operation: %s", call.Operation))
			}

			// Show context if available
			if len(call.Context) > 0 {
				contextStrs := []string{}
				for k, v := range call.Context {
					contextStrs = append(contextStrs, fmt.Sprintf("%s=%v", k, v))
				}
				parts = append(parts, fmt.Sprintf("      Context: %s", strings.Join(contextStrs, ", ")))
			}
		}

		// Add total time if we have it
		if totalTime > 0 {
			parts = append(parts, fmt.Sprintf("  Total time: %.2fms", float64(totalTime.Microseconds())/1000))
		}
	}

	// Error category
	if te.Category != CategoryUnknown {
		parts = append(parts, fmt.Sprintf("\nCategory: %s", te.Category))
	}

	// Original error
	if te.Original != nil {
		parts = append(parts, fmt.Sprintf("\nOriginal Error: %v", te.Original))
	}

	// Root cause if different
	if te.RootCause != nil && !errors.Is(te.RootCause, te.Original) {
		parts = append(parts, fmt.Sprintf("Root Cause: %v", te.RootCause))
	}

	// Additional context
	if len(te.Context) > 0 {
		parts = append(parts, fmt.Sprintf("\nAdditional Context: %v", te.Context))
	}

	return strings.Join(parts, "\n")
}

// FormatSimple - Simple one-line error format
func FormatSimple(err error) string {
	var te *TrackedError
	ok := errors.As(err, &te)
	if !ok {
		return err.Error()
	}

	chain := te.GetFunctionChain()
	if chain == "" {
		return err.Error()
	}

	return fmt.Sprintf("[%s] %s → %s", te.Category, chain, err.Error())
}

// FormatJSON - JSON format for logging/APIs
func FormatJSON(err error) ([]byte, error) {
	var te *TrackedError
	ok := errors.As(err, &te)
	if !ok {
		// Convert to tracked error first
		errors.As(Track(err), &te)
	}

	return json.MarshalIndent(te, "", "  ")
}

// Debug - Print detailed error information (for development)
func Debug(err error) {
	if err == nil {
		fmt.Println("No error")
		return
	}

	fmt.Println("=== ERROR DEBUG INFO ===")
	fmt.Println(FormatChain(err))

	var te *TrackedError
	if errors.As(err, &te) && len(te.StackTrace) > 0 {
		fmt.Println("\nStack Trace:")
		for i, frame := range te.StackTrace {
			fmt.Printf("  %d. %s at %s:%d\n", i+1, frame.Function, frame.File, frame.Line)
		}
	}

	fmt.Println("========================")
}

// Summary - Get error summary for logging
func Summary(err error) map[string]interface{} {
	if err == nil {
		return map[string]interface{}{"error": false}
	}

	summary := map[string]interface{}{
		"error":   true,
		"message": err.Error(),
		"type":    fmt.Sprintf("%T", err),
	}

	var te *TrackedError
	if errors.As(err, &te) {
		summary["category"] = te.Category
		summary["function_chain"] = te.GetFunctionChain()
		summary["chain_length"] = len(te.CallChain)

		if te.Original != nil {
			summary["original"] = te.Original.Error()
		}

		if len(te.Context) > 0 {
			summary["context"] = te.Context
		}
	}

	return summary
}

func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		// Return a fake URL to avoid nil pointer
		return &url.URL{Scheme: "http", Host: "unknown", Path: rawURL}
	}
	return u
}
