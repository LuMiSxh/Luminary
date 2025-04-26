package engine

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LoggerService provides logging functionality for the engine
type LoggerService struct {
	Verbose     bool   // Controls verbosity level
	DebugMode   bool   // Controls debug output
	LogFile     string // Path to the log file
	initialized bool   // Whether the logger is initialized
	fileLogger  *log.Logger
	mutex       sync.Mutex
}

// initLogger initializes the logger system
func (l *LoggerService) initLogger() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	if l.initialized {
		return nil
	}

	// Set up file logging if a log file is specified
	if l.LogFile != "" {
		// Ensure the directory exists
		logDir := filepath.Dir(l.LogFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		// Open the log file (append mode)
		file, err := os.OpenFile(l.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}

		// Create the file logger
		l.fileLogger = log.New(file, "", log.LstdFlags|log.Lmicroseconds)
	}

	l.initialized = true
	return nil
}

// logToFile writes a message to the log file if configured
func (l *LoggerService) logToFile(level, format string, args ...interface{}) {
	if !l.initialized {
		if err := l.initLogger(); err != nil {
			fmt.Printf("Logger initialization error: %v\n", err)
			return
		}
	}

	if l.fileLogger != nil {
		timestamp := time.Now().Format("2006-01-02 15:04:05.000")
		pid := os.Getpid()
		message := fmt.Sprintf(format, args...)
		l.fileLogger.Printf("%s [%d] %s - %s", timestamp, pid, level, message)
	}
}

// Debug logs debug-level messages
func (l *LoggerService) Debug(format string, args ...interface{}) {
	if l.DebugMode {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
	l.logToFile("DEBUG", format, args...)
}

// Info logs informational messages
func (l *LoggerService) Info(format string, args ...interface{}) {
	if l.Verbose {
		fmt.Printf("[INFO] "+format+"\n", args...)
	}
	l.logToFile("INFO", format, args...)
}

// Warn logs warning messages
func (l *LoggerService) Warn(format string, args ...interface{}) {
	fmt.Printf("[WARN] "+format+"\n", args...)
	l.logToFile("WARN", format, args...)
}

// Error logs error messages
func (l *LoggerService) Error(format string, args ...interface{}) {
	fmt.Printf("[ERROR] "+format+"\n", args...)
	l.logToFile("ERROR", format, args...)
}
