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
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/fatih/color"
)

//go:embed suggestions.json
var suggestionFS embed.FS

// SuggestionsMap holds suggestions for different error categories
type SuggestionsMap map[string][]string

// CLIFormatter provides user-friendly error formatting for command-line interface
type CLIFormatter struct {
	// ShowDebugInfo controls whether to show detailed technical information
	ShowDebugInfo bool

	// ShowFunctionChain controls whether to show the function call chain
	ShowFunctionChain bool

	// ShowTimestamps controls whether to show when errors occurred
	ShowTimestamps bool

	// ColorEnabled controls whether to use ANSI color codes
	ColorEnabled bool

	// Suggestions holds the loaded suggestion lists
	Suggestions SuggestionsMap

	// Error category styles
	ErrorStyle      *color.Color
	NetworkStyle    *color.Color
	ProviderStyle   *color.Color
	ParsingStyle    *color.Color
	NotFoundStyle   *color.Color
	RateLimitStyle  *color.Color
	AuthStyle       *color.Color
	FileSystemStyle *color.Color
	DownloadStyle   *color.Color
	TimeoutStyle    *color.Color
	PanicStyle      *color.Color

	// Text styles (matching CLI formatter)
	HeaderStyle      *color.Color
	TitleStyle       *color.Color
	SubtitleStyle    *color.Color
	DetailLabelStyle *color.Color
	DetailValueStyle *color.Color
	SectionStyle     *color.Color
	HighlightStyle   *color.Color
	SecondaryStyle   *color.Color
	InfoStyle        *color.Color
	WarningStyle     *color.Color
	SuccessStyle     *color.Color
}

// NewCLIFormatter creates a new CLI error formatter with default settings
func NewCLIFormatter() *CLIFormatter {
	f := &CLIFormatter{
		ShowDebugInfo:     false,
		ShowFunctionChain: false,
		ShowTimestamps:    false,
		ColorEnabled:      true,
		Suggestions:       make(SuggestionsMap),
	}

	// Initialize styles
	f.initStyles()

	// Load suggestions
	f.loadSuggestions()

	return f
}

// initStyles sets up all the color styles to match the CLI formatter
func (f *CLIFormatter) initStyles() {
	// Don't use color if it's disabled
	if !f.ColorEnabled {
		color.NoColor = true
	}

	// Configure error category styles
	f.ErrorStyle = color.New(color.FgRed)
	f.NetworkStyle = color.New(color.FgYellow)
	f.ProviderStyle = color.New(color.FgBlue)
	f.ParsingStyle = color.New(color.FgMagenta)
	f.NotFoundStyle = color.New(color.FgCyan)
	f.RateLimitStyle = color.New(color.FgYellow)
	f.AuthStyle = color.New(color.FgRed)
	f.FileSystemStyle = color.New(color.FgGreen)
	f.DownloadStyle = color.New(color.FgCyan)
	f.TimeoutStyle = color.New(color.FgYellow)
	f.PanicStyle = color.New(color.FgHiRed)

	// Configure text styles to match CLI formatter
	f.HeaderStyle = color.New(color.Bold, color.FgCyan)
	f.TitleStyle = color.New(color.Bold, color.FgWhite)
	f.SubtitleStyle = color.New(color.FgHiWhite)
	f.DetailLabelStyle = color.New(color.FgHiBlue)
	f.DetailValueStyle = color.New(color.FgWhite)
	f.SectionStyle = color.New(color.Underline, color.FgHiCyan)
	f.HighlightStyle = color.New(color.FgMagenta)
	f.SecondaryStyle = color.New(color.FgHiBlack)
	f.InfoStyle = color.New(color.FgBlue)
	f.WarningStyle = color.New(color.FgYellow)
	f.SuccessStyle = color.New(color.FgGreen)
}

// loadSuggestions loads error suggestions from the embedded JSON file
func (f *CLIFormatter) loadSuggestions() {
	// Read from embedded file
	jsonData, err := suggestionFS.ReadFile("suggestions.json")
	if err != nil {
		// If can't read, initialize with empty map
		f.Suggestions = make(SuggestionsMap)
		return
	}

	// Parse JSON data
	err = json.Unmarshal(jsonData, &f.Suggestions)
	if err != nil {
		f.Suggestions = make(SuggestionsMap)
	}
}

// NewDebugCLIFormatter creates a CLI formatter with debug information enabled
func NewDebugCLIFormatter() *CLIFormatter {
	f := &CLIFormatter{
		ShowDebugInfo:     true,
		ShowFunctionChain: true,
		ShowTimestamps:    true,
		ColorEnabled:      true,
		Suggestions:       make(SuggestionsMap),
	}

	f.initStyles()
	f.loadSuggestions()

	return f
}

// Format formats an error for CLI display
func (f *CLIFormatter) Format(err error) string {
	if err == nil {
		return ""
	}

	// Check if it's a tracked error
	var trackedErr *TrackedError
	ok := errors.As(err, &trackedErr)
	if !ok {
		// For non-tracked errors, just show the basic message
		return f.formatSimpleError(err)
	}

	var parts []string

	// Main error message with category-specific formatting
	mainMessage := f.formatMainMessage(trackedErr)
	parts = append(parts, mainMessage)

	// Category-specific guidance
	if guidance := f.getCategoryGuidance(trackedErr); guidance != "" {
		parts = append(parts, "")
		parts = append(parts, guidance)
	}

	// Context information
	if contextInfo := f.formatContextInfo(trackedErr); contextInfo != "" {
		parts = append(parts, "")
		parts = append(parts, contextInfo)
	}

	// Function chain (if enabled)
	if f.ShowFunctionChain && len(trackedErr.CallChain) > 0 {
		parts = append(parts, "")
		parts = append(parts, f.formatFunctionChain(trackedErr))
	}

	// Debug information (if enabled)
	if f.ShowDebugInfo {
		if debugInfo := f.formatDebugInfo(trackedErr); debugInfo != "" {
			parts = append(parts, "")
			parts = append(parts, debugInfo)
		}
	}

	return strings.Join(parts, "\n")
}

// FormatSimple provides a one-line error format for simple display
func (f *CLIFormatter) FormatSimple(err error) string {
	if err == nil {
		return ""
	}

	var trackedErr *TrackedError
	ok := errors.As(err, &trackedErr)
	if !ok {
		return err.Error()
	}

	// Category prefix + message
	prefix := f.getCategoryPrefix(trackedErr.Category)
	message := trackedErr.Error()

	// Get the appropriate style for this category
	style := f.getCategoryStyle(trackedErr.Category)
	return fmt.Sprintf("%s %s", prefix, style.Sprint(message))
}

// formatMainMessage creates the main error message with appropriate styling
func (f *CLIFormatter) formatMainMessage(trackedErr *TrackedError) string {
	category := trackedErr.Category
	message := trackedErr.Error()

	// Use user message if available, otherwise use error message
	if trackedErr.UserMessage != "" {
		message = trackedErr.UserMessage
	}

	prefix := f.getCategoryPrefix(category)

	// Get the appropriate style for this category
	style := f.getCategoryStyle(category)
	return fmt.Sprintf("%s %s", f.HeaderStyle.Sprint(prefix), style.Sprint(message))
}

// formatSimpleError formats non-tracked errors
func (f *CLIFormatter) formatSimpleError(err error) string {
	prefix := "[ERROR]"
	return fmt.Sprintf("%s %s", f.HeaderStyle.Sprint(prefix), f.ErrorStyle.Sprint(err.Error()))
}

// getCategoryGuidance provides user-friendly guidance based on error category and suggestions from JSON
func (f *CLIFormatter) getCategoryGuidance(trackedErr *TrackedError) string {
	category := strings.ToLower(string(trackedErr.Category))

	// Default header for guidance
	header := f.SectionStyle.Sprint("Troubleshooting suggestions:")

	// Get suggestions based on specific error patterns first
	var suggestions []string

	if trackedErr.Original != nil {
		errStr := strings.ToLower(trackedErr.Original.Error())

		// Check for specific network error patterns
		if category == "network" {
			switch {
			case strings.Contains(errStr, "no such host"):
				suggestions = f.getSuggestionsForKey("network_no_such_host")
			case strings.Contains(errStr, "connection refused"):
				suggestions = f.getSuggestionsForKey("network_connection_refused")
			case strings.Contains(errStr, "timeout"):
				suggestions = f.getSuggestionsForKey("network_timeout")
			case strings.Contains(errStr, "tls") || strings.Contains(errStr, "certificate"):
				suggestions = f.getSuggestionsForKey("network_tls")
			}
		}

		// Check for specific filesystem error patterns
		if category == "filesystem" {
			switch {
			case strings.Contains(errStr, "permission"):
				suggestions = f.getSuggestionsForKey("file_system_permission")
			case strings.Contains(errStr, "no space"):
				suggestions = f.getSuggestionsForKey("file_system_no_space")
			case strings.Contains(errStr, "no such file"):
				suggestions = f.getSuggestionsForKey("file_system_no_such_file")
			}
		}
	}

	// If no specific suggestions were found, use the general category suggestions
	if len(suggestions) == 0 {
		suggestions = f.getSuggestionsForKey(category)
	}

	// If still no suggestions, return empty string
	if len(suggestions) == 0 {
		return ""
	}

	// Format suggestions
	var formattedSuggestions []string
	for _, suggestion := range suggestions {
		formattedSuggestions = append(formattedSuggestions, fmt.Sprintf("  • %s", f.DetailValueStyle.Sprint(suggestion)))
	}

	// Add URL information for network errors if available
	if category == "network" {
		if url := f.extractURL(trackedErr); url != "" {
			formattedSuggestions = append(formattedSuggestions, "")
			formattedSuggestions = append(formattedSuggestions, fmt.Sprintf("Failed URL: %s", f.DetailValueStyle.Sprint(url)))
		}
	}

	// Add provider ID for provider errors if available
	if category == "provider" {
		if providerID := f.extractProviderID(trackedErr); providerID != "" {
			providerMessage := fmt.Sprintf("Provider '%s' is experiencing issues.", f.HighlightStyle.Sprint(providerID))
			return fmt.Sprintf("%s\n\n%s\n\n%s", providerMessage, header, strings.Join(formattedSuggestions, "\n"))
		}
	}

	return fmt.Sprintf("%s\n%s", header, strings.Join(formattedSuggestions, "\n"))
}

// getSuggestionsForKey returns suggestions for a given key or an empty list if none exist
func (f *CLIFormatter) getSuggestionsForKey(key string) []string {
	if suggestions, ok := f.Suggestions[key]; ok {
		return suggestions
	}
	return []string{}
}

// formatContextInfo extracts and formats relevant context information
func (f *CLIFormatter) formatContextInfo(trackedErr *TrackedError) string {
	var info []string

	// Add resource information
	if resourceInfo := f.extractResourceInfo(trackedErr); resourceInfo != "" {
		info = append(info, resourceInfo)
	}

	// Add HTTP information for network errors
	if trackedErr.Category == CategoryNetwork {
		if httpInfo := f.extractHTTPInfo(trackedErr); httpInfo != "" {
			info = append(info, httpInfo)
		}
	}

	if len(info) == 0 {
		return ""
	}

	return f.SectionStyle.Sprint("Additional information:") + "\n" + strings.Join(info, "\n")
}

// formatFunctionChain formats the function call chain
func (f *CLIFormatter) formatFunctionChain(trackedErr *TrackedError) string {
	if len(trackedErr.CallChain) == 0 {
		return ""
	}

	// Start with a header for the function chain section
	parts := []string{f.SectionStyle.Sprint("Function Call Chain:")}

	// Add each function in the call chain
	for i, call := range trackedErr.CallChain {
		timestamp := ""
		if f.ShowTimestamps {
			timestamp = fmt.Sprintf("[%s] ", call.Timestamp.Format("15:04:05.000"))
		}

		funcInfo := fmt.Sprintf("  %d. %s%s() at %s:%s",
			i+1, timestamp, f.HighlightStyle.Sprint(call.ShortName),
			f.SecondaryStyle.Sprint(call.File), f.SecondaryStyle.Sprint(call.Line))

		parts = append(parts, funcInfo)

		// If the function has operation info, add it
		if call.Operation != "" {
			parts = append(parts, fmt.Sprintf("      Operation: %s", f.DetailValueStyle.Sprint(call.Operation)))
		}

		// Add context information if available
		if len(call.Context) > 0 {
			contextStr := fmt.Sprintf("      Context: ")
			var contextItems []string
			for k, v := range call.Context {
				contextItems = append(contextItems, fmt.Sprintf("%s=%v",
					f.DetailLabelStyle.Sprint(k), f.DetailValueStyle.Sprint(v)))
			}
			contextStr += strings.Join(contextItems, ", ")
			parts = append(parts, contextStr)
		}
	}

	// Calculate and show total execution time if timestamps are enabled
	if f.ShowTimestamps && len(trackedErr.CallChain) > 0 {
		duration := trackedErr.CallChain[len(trackedErr.CallChain)-1].Timestamp.Sub(trackedErr.CallChain[0].Timestamp)
		parts = append(parts, fmt.Sprintf("  Total time: %s", f.HighlightStyle.Sprintf("%.2fms", math.Abs(float64(duration.Nanoseconds())/1e6))))
	}

	return strings.Join(parts, "\n")
}

// formatDebugInfo formats detailed debug information
func (f *CLIFormatter) formatDebugInfo(trackedErr *TrackedError) string {
	var parts []string

	parts = append(parts, f.SectionStyle.Sprint("Debug Information"))

	// Error hierarchy
	if trackedErr.Original != nil {
		parts = append(parts, fmt.Sprintf("%s %s",
			f.DetailLabelStyle.Sprint("Original Error:"),
			f.DetailValueStyle.Sprint(trackedErr.Original.Error())))

		// Check if this is a joined error (from errors.Join)
		errorsList, hasErrors := trackedErr.Context["errors"]
		if hasErrors {
			parts = append(parts, "")
			parts = append(parts, f.SubtitleStyle.Sprint("Component Errors:"))

			if errSlice, ok := errorsList.([]error); ok {
				for i, componentErr := range errSlice {
					// Extract the original error message
					errMessage := componentErr.Error()

					// Try to get the underlying error if it's a tracked error
					var componentTracked *TrackedError
					if errors.As(componentErr, &componentTracked) {
						parts = append(parts, fmt.Sprintf("  %d. %s %s",
							i+1,
							f.HighlightStyle.Sprint(errMessage),
							f.SecondaryStyle.Sprintf("(category: %s)", componentTracked.Category)))

						// Show component's context if available
						if len(componentTracked.Context) > 0 {
							parts = append(parts, f.DetailLabelStyle.Sprint("     Context:"))
							for k, v := range componentTracked.Context {
								parts = append(parts, fmt.Sprintf("       %s: %s",
									f.DetailLabelStyle.Sprint(k),
									f.DetailValueStyle.Sprint(v)))
							}
						}

						// Show underlying error if different from the message
						if componentTracked.RootCause != nil &&
							componentTracked.RootCause.Error() != errMessage {
							parts = append(parts, fmt.Sprintf("     %s %s",
								f.DetailLabelStyle.Sprint("Root cause:"),
								f.WarningStyle.Sprint(componentTracked.RootCause.Error())))
						}
					} else {
						// Simple error (not tracked)
						parts = append(parts, fmt.Sprintf("  %d. %s",
							i+1,
							f.DetailValueStyle.Sprint(errMessage)))
					}
				}
			} else {
				// Fallback if errors is not properly typed
				parts = append(parts, fmt.Sprintf("  %s",
					f.DetailValueStyle.Sprint(errorsList)))
			}
		}
	}

	if trackedErr.RootCause != nil && !errors.Is(trackedErr.RootCause, trackedErr.Original) {
		parts = append(parts, fmt.Sprintf("%s %s",
			f.DetailLabelStyle.Sprint("Root Cause:"),
			f.DetailValueStyle.Sprint(trackedErr.RootCause.Error())))
	}

	// Additional context
	if len(trackedErr.Context) > 0 {
		parts = append(parts, "")
		parts = append(parts, f.SubtitleStyle.Sprint("Global Context:"))
		for k, v := range trackedErr.Context {
			// Skip errors as we've already displayed it in a better format
			if k == "errors" || k == "error_count" {
				continue
			}

			parts = append(parts, fmt.Sprintf("  %s %s",
				f.DetailLabelStyle.Sprintf("%s:", k),
				f.DetailValueStyle.Sprint(v)))
		}
	}

	// Stack trace if available
	if len(trackedErr.StackTrace) > 0 {
		parts = append(parts, "")
		parts = append(parts, f.SubtitleStyle.Sprint("Stack Trace:"))
		for i, frame := range trackedErr.StackTrace {
			parts = append(parts, fmt.Sprintf("  %d. %s at %s:%s",
				i+1,
				f.HighlightStyle.Sprint(frame.Function),
				f.SecondaryStyle.Sprint(frame.File),
				f.SecondaryStyle.Sprint(frame.Line)))
		}
	}

	return strings.Join(parts, "\n")
}

// GetContext extracts the context from a TrackedError or returns nil
func GetContext(err error) map[string]interface{} {
	var tracked *TrackedError
	if errors.As(err, &tracked) {
		return tracked.Context
	}
	return nil
}

// Helper methods for extracting information

func (f *CLIFormatter) extractProviderID(trackedErr *TrackedError) string {
	context := GetContext(trackedErr)
	if context == nil {
		return ""
	}

	providerID, ok := context["provider_id"]
	if ok {
		if strID, isString := providerID.(string); isString {
			return strID
		}
	}

	// Check call chain for provider context as fallback
	for _, call := range trackedErr.CallChain {
		if providerID, ok := call.Context["provider_id"].(string); ok {
			return providerID
		}
	}

	return ""
}

func (f *CLIFormatter) extractURL(trackedErr *TrackedError) string {
	context := GetContext(trackedErr)
	if context == nil {
		return ""
	}

	url, ok := context["url"]
	if ok {
		if strURL, isString := url.(string); isString {
			return strURL
		}
	}

	// Check call chain for URL context as fallback
	for _, call := range trackedErr.CallChain {
		if url, ok := call.Context["url"].(string); ok {
			return url
		}
	}

	return ""
}

func (f *CLIFormatter) extractResourceInfo(trackedErr *TrackedError) string {
	var parts []string
	context := GetContext(trackedErr)
	if context == nil {
		return ""
	}

	// Check for resource type and ID
	if resourceType, ok := context["resource_type"].(string); ok {
		if resourceID, ok := context["resource_id"].(string); ok {
			parts = append(parts, fmt.Sprintf("%s %s (%s)",
				f.DetailLabelStyle.Sprint("Resource:"),
				f.HighlightStyle.Sprint(resourceID),
				f.SecondaryStyle.Sprint(resourceType)))
		}
	}

	// Check for query
	if query, ok := context["query"].(string); ok {
		parts = append(parts, fmt.Sprintf("%s %s",
			f.DetailLabelStyle.Sprint("Query:"),
			f.DetailValueStyle.Sprint(query)))
	}

	return strings.Join(parts, "\n")
}

func (f *CLIFormatter) extractHTTPInfo(trackedErr *TrackedError) string {
	var parts []string
	context := GetContext(trackedErr)
	if context == nil {
		return ""
	}

	// Check for HTTP method directly
	if method, ok := context["http_method"].(string); ok {
		info := fmt.Sprintf("%s %s",
			f.DetailLabelStyle.Sprint("HTTP Method:"),
			f.DetailValueStyle.Sprint(method))

		if statusCode, ok := context["status_code"].(int); ok && statusCode > 0 {
			info += fmt.Sprintf(", %s %s",
				f.DetailLabelStyle.Sprint("Status:"),
				f.DetailValueStyle.Sprint(statusCode))
		}

		parts = append(parts, info)
	} else if method, ok := context["method"].(string); ok {
		// Alternative keys that might be used
		info := fmt.Sprintf("%s %s",
			f.DetailLabelStyle.Sprint("HTTP Method:"),
			f.DetailValueStyle.Sprint(method))

		if statusCode, ok := context["status_code"].(int); ok && statusCode > 0 {
			info += fmt.Sprintf(", %s %s",
				f.DetailLabelStyle.Sprint("Status:"),
				f.DetailValueStyle.Sprint(statusCode))
		}

		parts = append(parts, info)
	} else {
		// Then check call chain as fallback
		for _, call := range trackedErr.CallChain {
			if method, ok := call.Context["method"].(string); ok {
				info := fmt.Sprintf("%s %s",
					f.DetailLabelStyle.Sprint("HTTP Method:"),
					f.DetailValueStyle.Sprint(method))

				if statusCode, ok := call.Context["status_code"].(int); ok && statusCode > 0 {
					info += fmt.Sprintf(", %s %s",
						f.DetailLabelStyle.Sprint("Status:"),
						f.DetailValueStyle.Sprint(statusCode))
				}

				parts = append(parts, info)
				break
			}
		}
	}

	return strings.Join(parts, "\n")
}

// Category prefix helpers

func (f *CLIFormatter) getCategoryPrefix(category ErrorCategory) string {
	switch category {
	case CategoryNetwork:
		return "[NETWORK]"
	case CategoryProvider:
		return "[PROVIDER]"
	case CategoryParser:
		return "[PARSING]"
	case CategoryNotFound:
		return "[NOT FOUND]"
	case CategoryRateLimit:
		return "[RATE LIMIT]"
	case CategoryAuth:
		return "[AUTH]"
	case CategoryFileSystem:
		return "[FILESYSTEM]"
	case CategoryDownload:
		return "[DOWNLOAD]"
	case CategoryTimeout:
		return "[TIMEOUT]"
	case CategoryPanic:
		return "[PANIC]"
	default:
		return "[ERROR]"
	}
}

// getCategoryStyle returns the appropriate color style for each category
func (f *CLIFormatter) getCategoryStyle(category ErrorCategory) *color.Color {
	switch category {
	case CategoryNetwork:
		return f.NetworkStyle
	case CategoryProvider:
		return f.ProviderStyle
	case CategoryParser:
		return f.ParsingStyle
	case CategoryNotFound:
		return f.NotFoundStyle
	case CategoryRateLimit:
		return f.RateLimitStyle
	case CategoryAuth:
		return f.AuthStyle
	case CategoryFileSystem:
		return f.FileSystemStyle
	case CategoryDownload:
		return f.DownloadStyle
	case CategoryTimeout:
		return f.TimeoutStyle
	case CategoryPanic:
		return f.PanicStyle
	default:
		return f.ErrorStyle
	}
}

// WithWriter returns a copy of the formatter that writes to the given writer
func (f *CLIFormatter) WithWriter(w io.Writer) *CLIFormatter {
	newFormatter := *f
	return &newFormatter
}

// Global formatters for easy use
var (
	DefaultCLIFormatter = NewCLIFormatter()
	DebugCLIFormatter   = NewDebugCLIFormatter()
)

// Convenience functions

// FormatCLISimple formats an error for simple CLI display
func FormatCLISimple(err error) string {
	return DefaultCLIFormatter.FormatSimple(err)
}

// FormatCLIDebug formats an error with debug information
func FormatCLIDebug(err error) string {
	return DebugCLIFormatter.Format(err)
}
