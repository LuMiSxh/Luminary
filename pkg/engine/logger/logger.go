package logger

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// Service provides logging capabilities
type Service struct {
	Verbose bool
	LogFile string
	AppName string // Optional application/component name
	mu      sync.Mutex
}

// Log logs a message
func (l *Service) Log(level string, message string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	formattedMsg := fmt.Sprintf(message, args...)

	// Format timestamp without brackets and with comma for milliseconds
	timestamp := time.Now().Format("2006-01-02 15:04:05,000")

	// Get thread ID or use a fixed value (using goroutine ID is complex)
	threadID := fmt.Sprintf("%4d", os.Getpid()%10000)

	// Build the log entry in the format: timestamp [threadID] LEVEL - category - message
	var logEntry string
	if l.AppName != "" {
		logEntry = fmt.Sprintf("%s [%s] %s - %s - %s", timestamp, threadID, level, l.AppName, formattedMsg)
	} else {
		logEntry = fmt.Sprintf("%s [%s] %s - 0 - %s", timestamp, threadID, level, formattedMsg)
	}

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
