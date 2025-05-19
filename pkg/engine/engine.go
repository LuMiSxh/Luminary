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
	"Luminary/pkg/engine/downloader"
	"Luminary/pkg/engine/logger"
	"Luminary/pkg/engine/network"
	"Luminary/pkg/engine/parser"
	"Luminary/pkg/engine/search"
	"Luminary/pkg/provider"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

// Engine is the central component providing services to providers
type Engine struct {
	HTTP        *network.HTTPService
	Download    *downloader.DownloadService
	Parser      *parser.Service
	RateLimiter *network.RateLimiterService
	DOM         *parser.DOMService
	Metadata    *parser.MetadataService
	Logger      *logger.Service
	API         *network.APIService
	Extractor   *parser.ExtractorService
	Pagination  *search.PaginationService
	Search      *search.Service
	WebScraper  *network.WebScraperService

	// Provider registry
	providers     map[string]provider.Provider
	providerMutex sync.RWMutex
}

// New creates a new Engine with default configuration
func New() *Engine {
	// Create basic services first
	parserService := &parser.Service{
		RegexPatterns: map[string]*regexp.Regexp{
			"chapterNumber": regexp.MustCompile(`(?i)(?:chapter|ch\.?|vol\.?|episode|ep\.?)[\s:]*(\d+(?:\.\d+)?)`),
			"volumeNumber":  regexp.MustCompile(`(?i)(?:volume|vol\.?)[\s:]*(\d+)`),
			"title":         regexp.MustCompile(`(?i)<title>(.*?)</title>`),
			"mangaTitle":    regexp.MustCompile(`(?i)(?:manga|comic|series|title)[\s:]*([^,\r\n]+)`),
			"authorName":    regexp.MustCompile(`(?i)(?:author|artist|creator|mangaka)[\s:]*([^,\r\n]+)`),
		},
	}

	// Determine default log file
	logFile := ""
	if homeDir, err := os.UserHomeDir(); err == nil {
		logDir := filepath.Join(homeDir, ".luminary", "logs")
		if err := os.MkdirAll(logDir, 0755); err == nil {
			logFile = filepath.Join(logDir, "luminary.log")
		}
	}

	// Create logger service first so we can use it in other services
	loggerService := logger.NewService(logFile)

	// Create HTTP service with logger
	httpService := &network.HTTPService{
		DefaultClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			},
		},
		RequestOptions: network.RequestOptions{
			Headers:         make(http.Header),
			UserProvider:    "Luminary/1.0",
			Method:          "GET",
			FollowRedirects: true,
		},
		DefaultRetries:     3,
		DefaultTimeout:     30 * time.Second,
		ThrottleTimeAPI:    2 * time.Second,
		ThrottleTimeImages: 500 * time.Millisecond,
		Logger:             loggerService,
	}

	// Set common headers
	httpService.RequestOptions.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	httpService.RequestOptions.Headers.Set("Accept-Language", "en-US,en;q=0.5")

	// Create download service
	downloadService := &downloader.DownloadService{
		Throttle:     500 * time.Millisecond,
		OutputFormat: "png",
		Client:       httpService.DefaultClient,
		Logger:       loggerService,
	}

	// Create rate limiter service with logger
	rateLimiterService := network.NewRateLimiterService(2*time.Second, loggerService)

	// Create DOM service
	domService := &parser.DOMService{}

	// Create engine with all services
	engine := &Engine{
		HTTP:        httpService,
		Download:    downloadService,
		Parser:      parserService,
		RateLimiter: rateLimiterService,
		DOM:         domService,
		Logger:      loggerService,
		providers:   make(map[string]provider.Provider),
	}

	// Create metadata service (depends on parser)
	engine.Metadata = &parser.MetadataService{
		Parser: parserService,
	}

	// Create API service with logger
	engine.API = network.NewAPIService(httpService, rateLimiterService, loggerService)

	// Create the extractor service with logger
	engine.Extractor = parser.NewExtractorService(loggerService)

	// Create the pagination service with dependencies
	engine.Pagination = search.NewPaginationService(engine.API, engine.Extractor, loggerService)

	// Create the search service with dependencies and logger
	engine.Search = search.NewSearchService(
		loggerService,
		engine.API,
		engine.Extractor,
		engine.Pagination,
		rateLimiterService,
	)

	// Create the WebScraper service
	engine.WebScraper = network.NewWebScraperService(httpService, domService, rateLimiterService, loggerService)

	loggerService.Info("Engine initialized successfully")
	return engine
}

// RegisterProvider adds a provider to the registry
func (e *Engine) RegisterProvider(provider provider.Provider) error {
	e.providerMutex.Lock()
	defer e.providerMutex.Unlock()

	if _, exists := e.providers[provider.ID()]; exists {
		return fmt.Errorf("provider with ID '%s' already registered", provider.ID())
	}

	e.providers[provider.ID()] = provider
	return nil
}

// GetProvider retrieves a registered provider by ID
func (e *Engine) GetProvider(id string) (provider.Provider, bool) {
	e.providerMutex.RLock()
	defer e.providerMutex.RUnlock()

	foundProvider, exists := e.providers[id]
	return foundProvider, exists
}

// AllProvider returns all registered providers
func (e *Engine) AllProvider() []provider.Provider {
	e.providerMutex.RLock()
	defer e.providerMutex.RUnlock()

	providers := make([]provider.Provider, 0, len(e.providers))
	for _, a := range e.providers {
		providers = append(providers, a)
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
