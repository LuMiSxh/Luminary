package engine

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// LoggerService provides logging capabilities
type LoggerService struct {
	Verbose bool
	LogFile string
	mu      sync.Mutex
}

// Log logs a message
func (l *LoggerService) Log(level string, message string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	formattedMsg := fmt.Sprintf(message, args...)
	logEntry := fmt.Sprintf("[%s] %s: %s", time.Now().Format(time.RFC3339), level, formattedMsg)

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
func (l *LoggerService) Info(message string, args ...interface{}) {
	l.Log("INFO", message, args...)
}

// Error logs an error message
func (l *LoggerService) Error(message string, args ...interface{}) {
	l.Log("ERROR", message, args...)
}

// Debug logs a debug message
func (l *LoggerService) Debug(message string, args ...interface{}) {
	l.Log("DEBUG", message, args...)
}

// Warn logs a warning message
func (l *LoggerService) Warn(message string, args ...interface{}) {
	l.Log("WARN", message, args...)
}
