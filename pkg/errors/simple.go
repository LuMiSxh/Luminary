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
	"strings"
)

// T (Track) - Wrap any error with automatic tracking
func T(err error) error {
	if err == nil {
		return nil
	}
	return Track(err)
}

// TC - Track with custom context data
func TC(err error, context map[string]interface{}) error {
	if err == nil {
		return nil
	}
	return Track(err, context)
}

// TM - Track with user-friendly message
func TM(err error, message string) error {
	if err == nil {
		return nil
	}
	return TrackWithMessage(err, message)
}

// TN - Track network errors
func TN(err error) error {
	if err == nil {
		return nil
	}
	return TrackNetwork(err)
}

// TP - Track provider-specific errors
func TP(err error, providerID string) error {
	if err == nil {
		return nil
	}
	return TrackProvider(err, providerID)
}

// WithCategory - Explicitly set the error category
func WithCategory(err error, category ErrorCategory) error {
	if err == nil {
		return nil
	}

	trackedErr := T(err)
	var te *TrackedError
	if errors.As(trackedErr, &te) {
		te.Category = category
	}

	return trackedErr
}

// ForceCategory - Force change category of an existing error
func ForceCategory(err error, category ErrorCategory) error {
	if err == nil {
		return nil
	}

	var te *TrackedError
	if errors.As(err, &te) {
		newTE := *te
		newTE.Category = category
		return &newTE
	}

	return WithCategory(err, category)
}

// Join - Combine multiple errors into a single tracked error
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
	combinedErr := fmt.Errorf("multiple errors: %s", strings.Join(messages, "; "))

	// Track the combined error
	trackedErr := Track(combinedErr)
	var te *TrackedError
	errors.As(trackedErr, &te)

	// Store original errors in context
	te.Context["original_errors"] = nonNilErrs

	// Merge call chains and contexts
	var combinedCallChain []FunctionCall
	combinedContext := make(map[string]interface{})
	categoryCounter := make(map[ErrorCategory]int)

	for i, err := range nonNilErrs {
		var existingTE *TrackedError
		if errors.As(err, &existingTE) {
			categoryCounter[existingTE.Category]++
			combinedCallChain = mergeCallChains(combinedCallChain, existingTE.CallChain)

			for k, v := range existingTE.GetContext() {
				combinedContext[fmt.Sprintf("err%d_%s", i+1, k)] = v
			}
		}
	}

	// Determine predominant category
	var predominantCategory = CategoryUnknown
	maxCount := 0

	for category, count := range categoryCounter {
		if count > maxCount || (count == maxCount && category != CategoryUnknown) {
			maxCount = count
			predominantCategory = category
		}
	}

	// Set the category if found a predominant one
	if predominantCategory != CategoryUnknown {
		te.Category = predominantCategory
	}

	// Add the merged call chains
	if len(combinedCallChain) > 0 {
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

// Must - Convert any function call to a tracked error
func Must[V any](value V, err error) V {
	if err != nil {
		panic(T(err))
	}
	return value
}

// Try - Safely execute a function and return tracked error
func Try(fn func() error) error {
	defer func() {
		if r := recover(); r != nil {
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

// GetContext - Get error context data
func GetContext(err error) map[string]interface{} {
	var te *TrackedError
	if errors.As(err, &te) {
		return te.GetContext()
	}
	return nil
}
