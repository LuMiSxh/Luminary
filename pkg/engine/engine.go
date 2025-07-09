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

package engine

import (
	"Luminary/pkg/core"
	"Luminary/pkg/engine/download"
	"Luminary/pkg/engine/logger"
	"Luminary/pkg/engine/network"
	"Luminary/pkg/engine/parser"
	"Luminary/pkg/errors"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Provider interface that providers must implement
type Provider interface {
	ID() string
	Name() string
	Description() string
	SiteURL() string

	Initialize(context.Context) error

	Search(context.Context, string, core.SearchOptions) ([]core.Manga, error)
	GetManga(context.Context, string) (*core.MangaInfo, error)
	GetChapter(context.Context, string) (*core.Chapter, error)
	TryGetMangaForChapter(context.Context, string) (*core.Manga, error)
	DownloadChapter(context.Context, string, string) error
}

// Engine is the central component providing services to providers
type Engine struct {
	// Core services (reduced from 11 to 4)
	Network  *network.Client
	Parser   *parser.Service
	Download *download.Service
	Logger   logger.Logger

	// Provider registry
	providers     map[string]Provider
	providerMutex sync.RWMutex

	// Error formatting options
	debugMode   bool
	verboseMode bool
}

// New creates a new Engine with default configuration
func New() *Engine {
	// Determine default log file
	logFile := ""
	if homeDir, err := os.UserHomeDir(); err == nil {
		logDir := filepath.Join(homeDir, ".luminary", "logs")
		if err := os.MkdirAll(logDir, 0755); err == nil {
			logFile = filepath.Join(logDir, "luminary.log")
		}
	}

	// Create logger first
	log := logger.NewService(logFile)

	// Create simplified services
	networkClient := network.NewClient(log)
	parserService := parser.NewService(log)
	downloadService := download.NewService(networkClient, log)

	engine := &Engine{
		Network:   networkClient,
		Parser:    parserService,
		Download:  downloadService,
		Logger:    log,
		providers: make(map[string]Provider),
	}

	log.Info("Engine initialized successfully")
	return engine
}

// RegisterProvider adds a provider to the registry
func (e *Engine) RegisterProvider(provider Provider) error {
	if provider == nil {
		return errors.Track(fmt.Errorf("provider is nil")).Error()
	}

	e.providerMutex.Lock()
	defer e.providerMutex.Unlock()

	id := provider.ID()
	if id == "" {
		return errors.Track(fmt.Errorf("provider has empty ID")).Error()
	}

	if _, exists := e.providers[id]; exists {
		return errors.Track(fmt.Errorf("provider with ID '%s' already registered", id)).Error()
	}

	e.providers[id] = provider
	e.Logger.Info("Registered provider: %s (%s)", provider.Name(), id)
	return nil
}

// GetProvider retrieves a registered provider by ID
func (e *Engine) GetProvider(id string) (Provider, error) {
	e.providerMutex.RLock()
	defer e.providerMutex.RUnlock()

	provider, exists := e.providers[id]
	if !exists {
		return nil, errors.Track(fmt.Errorf("provider '%s' not found", id)).
			WithContext("available_providers", e.getProviderIDs()).Error()
	}

	return provider, nil
}

// GetProviderOrNil retrieves a provider or returns nil if not found
func (e *Engine) GetProviderOrNil(id string) Provider {
	e.providerMutex.RLock()
	defer e.providerMutex.RUnlock()
	return e.providers[id]
}

// AllProviders returns all registered providers
func (e *Engine) AllProviders() []Provider {
	e.providerMutex.RLock()
	defer e.providerMutex.RUnlock()

	providers := make([]Provider, 0, len(e.providers))
	for _, p := range e.providers {
		providers = append(providers, p)
	}
	return providers
}

// ProviderExists checks if a provider exists
func (e *Engine) ProviderExists(id string) bool {
	e.providerMutex.RLock()
	defer e.providerMutex.RUnlock()
	_, exists := e.providers[id]
	return exists
}

// ProviderCount returns the number of registered providers
func (e *Engine) ProviderCount() int {
	e.providerMutex.RLock()
	defer e.providerMutex.RUnlock()
	return len(e.providers)
}

// InitializeProviders initializes all registered providers
func (e *Engine) InitializeProviders(ctx context.Context) error {
	providers := e.AllProviders()

	for _, provider := range providers {
		if err := provider.Initialize(ctx); err != nil {
			e.Logger.Error("Failed to initialize provider %s: %v", provider.ID(), err)
			// Continue with other providers
		}
	}

	return nil
}

// Shutdown gracefully shuts down the engine
func (e *Engine) Shutdown() error {
	e.Logger.Info("Shutting down engine...")

	// Close logger
	if closer, ok := e.Logger.(interface{ Close() error }); ok {
		return closer.Close()
	}

	return nil
}

// getProviderIDs returns a list of all provider IDs
func (e *Engine) getProviderIDs() []string {
	ids := make([]string, 0, len(e.providers))
	for id := range e.providers {
		ids = append(ids, id)
	}
	return ids
}

// SetDebugMode enables or disables debug mode for error formatting
func (e *Engine) SetDebugMode(enabled bool) {
	e.debugMode = enabled
	if enabled {
		e.Logger.SetLevel(logger.LevelDebug)
		// Enable console output for debug mode
		if loggerService, ok := e.Logger.(*logger.Service); ok {
			loggerService.SetConsoleOutput(true)
		}
		e.Logger.Debug("Debug mode enabled")
	} else {
		// Reset to info level when debug is disabled
		e.Logger.SetLevel(logger.LevelInfo)
		// Only disable console output if verbose mode is also off
		if !e.verboseMode {
			if loggerService, ok := e.Logger.(*logger.Service); ok {
				loggerService.SetConsoleOutput(false)
			}
		}
	}
}

// SetVerboseMode enables or disables verbose mode for error formatting
func (e *Engine) SetVerboseMode(enabled bool) {
	e.verboseMode = enabled
	if enabled {
		e.Logger.SetLevel(logger.LevelDebug)
		// Enable console output for verbose mode
		if loggerService, ok := e.Logger.(*logger.Service); ok {
			loggerService.SetConsoleOutput(true)
		}
		e.Logger.Info("Verbose mode enabled")
	} else {
		// Reset to info level when verbose is disabled
		e.Logger.SetLevel(logger.LevelInfo)
		// Only disable console output if debug mode is also off
		if !e.debugMode {
			if loggerService, ok := e.Logger.(*logger.Service); ok {
				loggerService.SetConsoleOutput(false)
			}
		}
	}
}

// FormatError formats an error based on the current verbosity settings
func (e *Engine) FormatError(err error) string {
	if err == nil {
		return ""
	}

	if e.verboseMode {
		// When verbose, show full tracked error with details
		return errors.FormatCLIDebug(err)
	} else if e.debugMode {
		// When debug, show more error details but not full trace
		return errors.FormatCLI(err)
	} else {
		// Default simple format
		return errors.FormatCLISimple(err)
	}
}
