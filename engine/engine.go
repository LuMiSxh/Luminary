package engine

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"
)

// Engine is the central component providing services to agents
type Engine struct {
	HTTP        *HTTPService
	Cache       *CacheService
	Download    *DownloadService
	Parser      *ParserService
	RateLimiter *RateLimiterService
	DOM         *DOMService
	Metadata    *MetadataService
	Logger      *LoggerService
	API         *APIService
	Extractor   *ExtractorService
	Pagination  *PaginationService
}

// New creates a new Engine with default configuration
func New() *Engine {
	// Create basic services first
	parser := &ParserService{
		RegexPatterns: map[string]*regexp.Regexp{
			"chapterNumber": regexp.MustCompile(`(?i)(?:chapter|ch\.?|vol\.?|episode|ep\.?)[\s:]*(\d+(?:\.\d+)?)`),
			"volumeNumber":  regexp.MustCompile(`(?i)(?:volume|vol\.?)[\s:]*(\d+)`),
			"title":         regexp.MustCompile(`(?i)<title>(.*?)</title>`),
			"mangaTitle":    regexp.MustCompile(`(?i)(?:manga|comic|series|title)[\s:]*([^,\r\n]+)`),
			"authorName":    regexp.MustCompile(`(?i)(?:author|artist|creator|mangaka)[\s:]*([^,\r\n]+)`),
		},
	}

	httpService := &HTTPService{
		DefaultClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			},
		},
		RequestOptions: RequestOptions{
			Headers:         make(http.Header),
			UserAgent:       "Luminary/1.0",
			Method:          "GET",
			FollowRedirects: true,
		},
		DefaultRetries:     3,
		DefaultTimeout:     30 * time.Second,
		ThrottleTimeAPI:    2 * time.Second,
		ThrottleTimeImages: 500 * time.Millisecond,
	}

	// Set common headers
	httpService.RequestOptions.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	httpService.RequestOptions.Headers.Set("Accept-Language", "en-US,en;q=0.5")

	// Determine default cache directory
	cacheDir := filepath.Join(os.TempDir(), "luminary-cache")
	if homeDir, err := os.UserHomeDir(); err == nil {
		cacheDir = filepath.Join(homeDir, ".luminary", "cache")
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
	loggerService := &LoggerService{
		Verbose: false,
		LogFile: logFile,
	}

	// Create cache service
	cacheService := NewCacheService(
		24*time.Hour, // TTL of 24 hours
		cacheDir,     // Cache directory
		true,         // Use disk cache
	)

	// Create download service
	downloadService := &DownloadService{
		MaxConcurrency: runtime.NumCPU() * 2, // Default to 2x CPU cores
		Throttle:       500 * time.Millisecond,
		OutputFormat:   "jpeg",
		Client:         httpService.DefaultClient,
		Logger:         loggerService,
	}

	// Create rate limiter service
	rateLimiterService := NewRateLimiterService(2 * time.Second)

	// Configure rate limiters for common sites
	rateLimiterService.SetLimit("api.mangadex.org", 2*time.Second)
	rateLimiterService.SetLimit("mangadex.org", 1*time.Second)

	// Create DOM service
	domService := &DOMService{}

	// Create engine with all services
	engine := &Engine{
		HTTP:        httpService,
		Cache:       cacheService,
		Download:    downloadService,
		Parser:      parser,
		RateLimiter: rateLimiterService,
		DOM:         domService,
		Logger:      loggerService,
	}

	// Create metadata service (depends on parser)
	engine.Metadata = &MetadataService{
		Parser: parser,
	}

	// Create the new services
	engine.Extractor = NewExtractorService(loggerService)
	engine.API = NewAPIService(httpService, rateLimiterService, cacheService, loggerService)
	engine.Pagination = NewPaginationService(engine.API, engine.Extractor, loggerService)

	return engine
}

// CompilePattern is a helper method for ParserService
func (p *ParserService) CompilePattern(pattern string) *regexp.Regexp {
	re, found := p.RegexPatterns[pattern]
	if found {
		return re
	}

	// If not found, compile it
	re = regexp.MustCompile(pattern)
	p.RegexPatterns[pattern] = re
	return re
}

// Shutdown performs cleanup operations
func (e *Engine) Shutdown() {
	// Clean expired cache entries before shutting down
	_, _ = e.Cache.CleanExpired()

	// Perform any cleanup needed
	e.Logger.Info("Engine shutting down")
}

// ExtractDomain extracts the domain from a URL
func (e *Engine) ExtractDomain(urlStr string) string {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		// If parsing fails, return the whole URL as the domain
		return urlStr
	}
	return parsed.Host
}
