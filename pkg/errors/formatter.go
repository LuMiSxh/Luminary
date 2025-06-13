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
}

// NewCLIFormatter creates a new CLI error formatter with default settings
func NewCLIFormatter() *CLIFormatter {
	return &CLIFormatter{
		ShowDebugInfo:     false,
		ShowFunctionChain: false,
		ShowTimestamps:    false,
		ColorEnabled:      true,
	}
}

// NewDebugCLIFormatter creates a CLI formatter with debug information enabled
func NewDebugCLIFormatter() *CLIFormatter {
	return &CLIFormatter{
		ShowDebugInfo:     true,
		ShowFunctionChain: true,
		ShowTimestamps:    true,
		ColorEnabled:      true,
	}
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
	return fmt.Sprintf("%s %s", prefix, trackedErr.Error())
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
	color := f.getCategoryColor(category)

	if f.ColorEnabled {
		return fmt.Sprintf("%s %s%s%s", prefix, color, message, f.colorReset())
	}

	return fmt.Sprintf("%s %s", prefix, message)
}

// formatSimpleError formats non-tracked errors
func (f *CLIFormatter) formatSimpleError(err error) string {
	prefix := "[ERROR]"
	if f.ColorEnabled {
		return fmt.Sprintf("%s %s%s%s", prefix, f.colorRed(), err.Error(), f.colorReset())
	}
	return fmt.Sprintf("%s %s", prefix, err.Error())
}

// getCategoryGuidance provides user-friendly guidance based on error category
func (f *CLIFormatter) getCategoryGuidance(trackedErr *TrackedError) string {
	switch trackedErr.Category {
	case CategoryNetwork:
		return f.getNetworkGuidance(trackedErr)
	case CategoryProvider:
		return f.getProviderGuidance(trackedErr)
	case CategoryNotFound:
		return f.getNotFoundGuidance(trackedErr)
	case CategoryRateLimit:
		return f.getRateLimitGuidance(trackedErr)
	case CategoryAuth:
		return f.getAuthGuidance(trackedErr)
	case CategoryParsing:
		return f.getParsingGuidance(trackedErr)
	case CategoryFileSystem:
		return f.getFileSystemGuidance(trackedErr)
	case CategoryDownload:
		return f.getDownloadGuidance(trackedErr)
	case CategoryTimeout:
		return f.getTimeoutGuidance(trackedErr)
	default:
		return ""
	}
}

// Network-specific guidance
func (f *CLIFormatter) getNetworkGuidance(trackedErr *TrackedError) string {
	guidance := []string{
		"[NETWORK] Network connectivity issue detected.",
		"",
		"Troubleshooting steps:",
		"  • Check your internet connection",
		"  • Verify the service is accessible",
		"  • Try again in a few moments",
	}

	// Add specific guidance based on the original error
	if trackedErr.Original != nil {
		errStr := strings.ToLower(trackedErr.Original.Error())
		switch {
		case strings.Contains(errStr, "no such host"):
			guidance = append(guidance, "  • DNS resolution failed - check your DNS settings")
		case strings.Contains(errStr, "connection refused"):
			guidance = append(guidance, "  • Server is not responding - the service may be down")
		case strings.Contains(errStr, "timeout"):
			guidance = append(guidance, "  • Request timed out - try again with a slower connection")
		case strings.Contains(errStr, "tls") || strings.Contains(errStr, "certificate"):
			guidance = append(guidance, "  • SSL/TLS certificate issue - check system date/time")
		}
	}

	// Add URL information if available
	if url := f.extractURL(trackedErr); url != "" {
		guidance = append(guidance, "", fmt.Sprintf("Failed URL: %s", url))
	}

	return strings.Join(guidance, "\n")
}

// Provider-specific guidance
func (f *CLIFormatter) getProviderGuidance(trackedErr *TrackedError) string {
	providerID := f.extractProviderID(trackedErr)
	if providerID == "" {
		return "[PROVIDER] The manga provider is experiencing issues. Try a different provider or check back later."
	}

	return fmt.Sprintf("[PROVIDER] Provider '%s' is experiencing issues.\n\nSuggestions:\n  • Try a different provider\n  • Check if the provider's website is accessible\n  • Try again later", providerID)
}

// Not found guidance
func (f *CLIFormatter) getNotFoundGuidance(trackedErr *TrackedError) string {
	return "[NOT FOUND] The requested resource was not found.\n\nSuggestions:\n  • Check the ID/URL is correct\n  • Try searching for the content\n  • The content may have been removed or moved"
}

// Rate limit guidance
func (f *CLIFormatter) getRateLimitGuidance(trackedErr *TrackedError) string {
	return "[RATE LIMIT] Rate limit exceeded.\n\nSuggestions:\n  • Wait a few minutes before trying again\n  • Reduce the number of concurrent requests\n  • Consider using a different provider"
}

// Auth guidance
func (f *CLIFormatter) getAuthGuidance(trackedErr *TrackedError) string {
	return "[AUTH] Authentication or authorization failed.\n\nSuggestions:\n  • Check if the provider requires registration\n  • Verify your credentials (if applicable)\n  • The provider may have restricted access"
}

// Parsing guidance
func (f *CLIFormatter) getParsingGuidance(trackedErr *TrackedError) string {
	return "[PARSING] Data parsing failed.\n\nThis usually indicates:\n  • The provider changed their API format\n  • The response was corrupted\n  • This may be a temporary issue - try again later"
}

// File system guidance
func (f *CLIFormatter) getFileSystemGuidance(trackedErr *TrackedError) string {
	if trackedErr.Original != nil {
		errStr := strings.ToLower(trackedErr.Original.Error())
		switch {
		case strings.Contains(errStr, "permission"):
			return "[FILESYSTEM] Permission denied.\n\nSuggestions:\n  • Check file/directory permissions\n  • Run with appropriate privileges\n  • Choose a different output directory"
		case strings.Contains(errStr, "no space"):
			return "[FILESYSTEM] Insufficient disk space.\n\nSuggestions:\n  • Free up disk space\n  • Choose a different output directory\n  • Clean up old downloads"
		case strings.Contains(errStr, "no such file"):
			return "[FILESYSTEM] File or directory not found.\n\nSuggestions:\n  • Check the path exists\n  • Create the directory if needed\n  • Verify file permissions"
		}
	}

	return "[FILESYSTEM] File system error occurred.\n\nSuggestions:\n  • Check file/directory permissions\n  • Ensure sufficient disk space\n  • Verify the path is correct"
}

// Download guidance
func (f *CLIFormatter) getDownloadGuidance(trackedErr *TrackedError) string {
	return "[DOWNLOAD] Download failed.\n\nSuggestions:\n  • Check your internet connection\n  • Ensure sufficient disk space\n  • Try downloading to a different location\n  • Retry the download"
}

// Timeout guidance
func (f *CLIFormatter) getTimeoutGuidance(trackedErr *TrackedError) string {
	return "[TIMEOUT] Operation timed out.\n\nSuggestions:\n  • Try again with a slower connection\n  • Reduce the number of concurrent operations\n  • The service may be experiencing high load"
}

// formatContextInfo extracts and formats relevant context information
func (f *CLIFormatter) formatContextInfo(trackedErr *TrackedError) string {
	var info []string

	// Add provider information
	if providerID := f.extractProviderID(trackedErr); providerID != "" {
		info = append(info, fmt.Sprintf("Provider: %s", providerID))
	}

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

	return strings.Join(info, "\n")
}

// formatFunctionChain formats the function call chain
func (f *CLIFormatter) formatFunctionChain(trackedErr *TrackedError) string {
	if len(trackedErr.CallChain) == 0 {
		return ""
	}

	// Start with a header for the function chain section
	parts := []string{"Function Call Chain:"}

	// Add each function in the call chain
	for i, call := range trackedErr.CallChain {
		timestamp := ""
		if f.ShowTimestamps {
			timestamp = fmt.Sprintf("[%s] ", call.Timestamp.Format("15:04:05.000"))
		}

		funcInfo := fmt.Sprintf("  %d. %s%s() at %s:%d",
			i+1, timestamp, call.ShortName, call.File, call.Line)

		parts = append(parts, funcInfo)

		// If the function has operation info, add it
		if call.Operation != "" {
			parts = append(parts, fmt.Sprintf("      Operation: %s", call.Operation))
		}

		// Add context information if available
		if len(call.Context) > 0 {
			contextStr := fmt.Sprintf("      Context: ")
			contextItems := []string{}
			for k, v := range call.Context {
				contextItems = append(contextItems, fmt.Sprintf("%s=%v", k, v))
			}
			contextStr += strings.Join(contextItems, ", ")
			parts = append(parts, contextStr)
		}
	}

	// Calculate and show total execution time if timestamps are enabled
	if f.ShowTimestamps && len(trackedErr.CallChain) > 0 {
		duration := trackedErr.CallChain[len(trackedErr.CallChain)-1].Timestamp.Sub(trackedErr.CallChain[0].Timestamp)
		parts = append(parts, fmt.Sprintf("  Total time: %.2fms", float64(duration.Nanoseconds())/1e6))
	}

	return strings.Join(parts, "\n")
}

// formatDebugInfo formats detailed debug information
func (f *CLIFormatter) formatDebugInfo(trackedErr *TrackedError) string {
	var parts []string

	parts = append(parts, "=== DEBUG INFORMATION ===")

	// Error hierarchy
	if trackedErr.Original != nil {
		parts = append(parts, fmt.Sprintf("Original Error: %s", trackedErr.Original.Error()))
	}

	if trackedErr.RootCause != nil && !errors.Is(trackedErr.RootCause, trackedErr.Original) {
		parts = append(parts, fmt.Sprintf("Root Cause: %s", trackedErr.RootCause.Error()))
	}

	// Additional context
	if len(trackedErr.Context) > 0 {
		parts = append(parts, "", "Global Context:")
		for k, v := range trackedErr.Context {
			parts = append(parts, fmt.Sprintf("  %s: %v", k, v))
		}
	}

	// Stack trace if available
	if len(trackedErr.StackTrace) > 0 {
		parts = append(parts, "", "Stack Trace:")
		for i, frame := range trackedErr.StackTrace {
			parts = append(parts, fmt.Sprintf("  %d. %s at %s:%d",
				i+1, frame.Function, frame.File, frame.Line))
		}
	}

	parts = append(parts, "========================")

	return strings.Join(parts, "\n")
}

// Helper methods for extracting information

func (f *CLIFormatter) extractProviderID(trackedErr *TrackedError) string {
	// Use GetContext from simple.go
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
	// Use GetContext from simple.go
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
			parts = append(parts, fmt.Sprintf("Resource: %s (%s)", resourceID, resourceType))
		}
	}

	// Check for query
	if query, ok := context["query"].(string); ok {
		parts = append(parts, fmt.Sprintf("Query: %s", query))
	}

	return strings.Join(parts, "\n")
}

func (f *CLIFormatter) extractHTTPInfo(trackedErr *TrackedError) string {
	var parts []string
	context := GetContext(trackedErr)
	if context == nil {
		return ""
	}

	// First check the main context
	if method, ok := context["http_method"].(string); ok {
		info := fmt.Sprintf("HTTP Method: %s", method)

		if statusCode, ok := context["status_code"].(int); ok && statusCode > 0 {
			info += fmt.Sprintf(", Status: %d", statusCode)
		}

		parts = append(parts, info)
	} else {
		// Then check call chain as fallback
		for _, call := range trackedErr.CallChain {
			if method, ok := call.Context["http_method"].(string); ok {
				info := fmt.Sprintf("HTTP Method: %s", method)

				if statusCode, ok := call.Context["status_code"].(int); ok && statusCode > 0 {
					info += fmt.Sprintf(", Status: %d", statusCode)
				}

				parts = append(parts, info)
				break
			}
		}
	}

	return strings.Join(parts, "\n")
}

// Category prefix and color helpers

func (f *CLIFormatter) getCategoryPrefix(category ErrorCategory) string {
	switch category {
	case CategoryNetwork:
		return "[NETWORK]"
	case CategoryProvider:
		return "[PROVIDER]"
	case CategoryParsing:
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

func (f *CLIFormatter) getCategoryColor(category ErrorCategory) string {
	if !f.ColorEnabled {
		return ""
	}

	switch category {
	case CategoryNetwork:
		return "\033[33m" // Yellow
	case CategoryProvider:
		return "\033[34m" // Blue
	case CategoryParsing:
		return "\033[35m" // Magenta
	case CategoryNotFound:
		return "\033[36m" // Cyan
	case CategoryRateLimit:
		return "\033[33m" // Yellow
	case CategoryAuth:
		return "\033[31m" // Red
	case CategoryFileSystem:
		return "\033[32m" // Green
	case CategoryDownload:
		return "\033[36m" // Cyan
	case CategoryTimeout:
		return "\033[33m" // Yellow
	case CategoryPanic:
		return "\033[91m" // Bright Red
	default:
		return "\033[31m" // Red
	}
}

func (f *CLIFormatter) colorRed() string {
	if f.ColorEnabled {
		return "\033[31m"
	}
	return ""
}

func (f *CLIFormatter) colorReset() string {
	if f.ColorEnabled {
		return "\033[0m"
	}
	return ""
}

// Global formatters for easy use
var (
	DefaultCLIFormatter = NewCLIFormatter()
	DebugCLIFormatter   = NewDebugCLIFormatter()
)

// Convenience functions

// FormatCLI formats an error for CLI display using the default formatter
func FormatCLI(err error) string {
	return DefaultCLIFormatter.Format(err)
}

// FormatCLISimple formats an error for simple CLI display
func FormatCLISimple(err error) string {
	return DefaultCLIFormatter.FormatSimple(err)
}

// FormatCLIDebug formats an error with debug information
func FormatCLIDebug(err error) string {
	return DebugCLIFormatter.Format(err)
}
