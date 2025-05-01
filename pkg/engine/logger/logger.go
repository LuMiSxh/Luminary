package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// Service provides logging capabilities
type Service struct {
	Verbose    bool   // If true, print all log levels to console (not just ERROR)
	LogFile    string // Optional file to write logs to
	CallerSkip int    // Number of call frames to skip when determining caller
	mu         sync.Mutex
}

// NewService creates a new logger service with default configuration
func NewService(logFile string) *Service {
	return &Service{
		Verbose:    false,
		LogFile:    logFile,
		CallerSkip: 2, // Default to skip Log and the specific log method (Info, Error, etc.)
	}
}

// getSourceInfo returns formatted source file and line information
func (l *Service) getSourceInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown:0"
	}

	// Extract just the filename without the full path
	filename := filepath.Base(file)
	return fmt.Sprintf("%s:%d", filename, line)
}

// Log logs a message
func (l *Service) Log(level string, message string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	formattedMsg := fmt.Sprintf(message, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05,000")
	threadID := fmt.Sprintf("%4d", os.Getpid()%10000)

	// Get source file information
	skip := l.CallerSkip
	if skip <= 0 {
		skip = 2 // Default skip value if not set
	}
	sourceInfo := l.getSourceInfo(skip + 1) // +1 to account for this function call

	// Pad the level and source info for consistent alignment
	paddedLevel := fmt.Sprintf("%-5s", level)            // Left-align level to 5 chars
	paddedSourceInfo := fmt.Sprintf("%-25s", sourceInfo) // Left-align source info to 25 chars

	// Build the log entry with consistent padding
	logEntry := fmt.Sprintf("%s [%s] %s - %s - %s",
		timestamp,
		threadID,
		paddedLevel,
		paddedSourceInfo,
		formattedMsg)

	// Always print errors
	if level == "ERROR" {
		_, _ = fmt.Fprintln(os.Stderr, logEntry)
	} else if l.Verbose {
		fmt.Println(logEntry)
	}

	// Write to the log file if configured
	if l.LogFile != "" {
		if f, err := os.OpenFile(l.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			defer func(f *os.File) {
				err := f.Close()
				if err != nil {
					fmt.Printf("Error closing log file: %s\n", err)
				}
			}(f)
			_, _ = fmt.Fprintln(f, logEntry)
		}
	}
}

// Info logs an info message
func (l *Service) Info(message string, args ...interface{}) {
	l.Log("INFO", message, args...)
}

// Error logs an error message
func (l *Service) Error(message string, args ...interface{}) {
	l.Log("ERROR", message, args...)
}

// Debug logs a debug message
func (l *Service) Debug(message string, args ...interface{}) {
	l.Log("DEBUG", message, args...)
}

// Warn logs a warning message
func (l *Service) Warn(message string, args ...interface{}) {
	l.Log("WARN", message, args...)
}
