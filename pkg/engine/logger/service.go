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

package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Level represents log severity
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger interface for logging operations
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	SetLevel(level Level)
}

// Service implements the Logger interface
type Service struct {
	level         Level
	logFile       string
	file          *os.File
	logger        *log.Logger
	mu            sync.Mutex
	colorize      bool
	pid           int
	consoleOutput bool
}

// NewService creates a new logger service
func NewService(logFile string) *Service {
	s := &Service{
		level:         LevelInfo,
		logFile:       logFile,
		colorize:      true,
		pid:           os.Getpid(),
		consoleOutput: false, // Default to no console output
	}

	// Setup initial output (file only by default)
	s.updateOutputWriters()

	return s
}

// updateOutputWriters configures the output writers based on current settings
func (s *Service) updateOutputWriters() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var output io.Writer

	// Always try to open log file if specified
	if s.logFile != "" && s.file == nil {
		// Ensure directory exists
		dir := filepath.Dir(s.logFile)
		if err := os.MkdirAll(dir, 0755); err == nil {
			// Open log file
			if file, err := os.OpenFile(s.logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
				s.file = file
			}
		}
	}

	// Set output based on configuration
	if s.consoleOutput {
		if s.file != nil {
			output = io.MultiWriter(os.Stdout, s.file)
		} else {
			output = os.Stdout
		}
	} else {
		if s.file != nil {
			output = s.file
		} else {
			// Fallback to stdout if no file is available
			output = os.Stdout
		}
	}

	// Create logger with empty flags since we handle formatting ourselves
	s.logger = log.New(output, "", 0)
}

// SetConsoleOutput enables or disables console output
func (s *Service) SetConsoleOutput(enabled bool) {
	if s.consoleOutput != enabled {
		s.consoleOutput = enabled
		s.updateOutputWriters()
	}
}

// SetLevel sets the minimum log level
func (s *Service) SetLevel(level Level) {
	s.mu.Lock()
	s.level = level
	s.mu.Unlock()
}

// Close closes the log file if open
func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		return err
	}
	return nil
}

// Debug logs a debug message
func (s *Service) Debug(format string, args ...interface{}) {
	s.log(LevelDebug, format, args...)
}

// Info logs an info message
func (s *Service) Info(format string, args ...interface{}) {
	s.log(LevelInfo, format, args...)
}

// Warn logs a warning message
func (s *Service) Warn(format string, args ...interface{}) {
	s.log(LevelWarn, format, args...)
}

// Error logs an error message
func (s *Service) Error(format string, args ...interface{}) {
	s.log(LevelError, format, args...)
}

// log performs the actual logging
func (s *Service) log(level Level, format string, args ...interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check level
	if level < s.level {
		return
	}

	// Get caller information
	_, file, line, ok := runtime.Caller(2)
	fileInfo := "unknown:0"
	if ok {
		// Extract just the filename from the full path
		file = filepath.Base(file)
		fileInfo = fmt.Sprintf("%s:%d", file, line)
	}

	// Format timestamp with milliseconds and comma separator
	now := time.Now()
	timestamp := fmt.Sprintf("%s,%03d",
		now.Format("2006-01-02 15:04:05"),
		now.Nanosecond()/1000000)

	levelStr := s.levelString(level)
	message := fmt.Sprintf(format, args...)

	// Pad file info to consistent width (23 characters based on log pattern)
	paddedFileInfo := fileInfo
	if len(fileInfo) < 23 {
		paddedFileInfo = fileInfo + strings.Repeat(" ", 23-len(fileInfo))
	}

	// Format log entry: timestamp [pid] LEVEL - file:line        - message
	logEntry := fmt.Sprintf("%s [%d] %-5s - %s - %s",
		timestamp, s.pid, levelStr, paddedFileInfo, message)

	// Output with or without color
	if s.colorize && s.file == nil {
		// Color output for console
		color := s.levelColor(level)
		s.logger.Printf("%s%s%s", color, logEntry, resetColor)
	} else {
		// Plain output for file
		s.logger.Print(logEntry)
	}
}

// levelString returns the string representation of a level
func (s *Service) levelString(level Level) string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ANSI color codes
const (
	resetColor  = "\033[0m"
	redColor    = "\033[31m"
	yellowColor = "\033[33m"
	blueColor   = "\033[34m"
	grayColor   = "\033[90m"
)

// levelColor returns the ANSI color code for a level
func (s *Service) levelColor(level Level) string {
	switch level {
	case LevelDebug:
		return grayColor
	case LevelInfo:
		return blueColor
	case LevelWarn:
		return yellowColor
	case LevelError:
		return redColor
	default:
		return resetColor
	}
}

// LogFile returns the path to the log file
func (s *Service) LogFile() string {
	return s.logFile
}
