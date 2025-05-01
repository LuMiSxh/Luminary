package engine

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"
)

// Engine is the central component providing services to agents
type Engine struct {
	HTTP        *HTTPService
	Download    *DownloadService
	Parser      *ParserService
	RateLimiter *RateLimiterService
	DOM         *DOMService
	Metadata    *MetadataService
	Logger      *LoggerService
	API         *APIService
	Extractor   *ExtractorService
	Pagination  *PaginationService
	Search      *SearchService
	WebScraper  *WebScraperService

	// Agent registry
	agents      map[string]Agent
	agentsMutex sync.RWMutex
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

	// Create HTTP service with logger
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
		Logger:             loggerService,
	}

	// Set common headers
	httpService.RequestOptions.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	httpService.RequestOptions.Headers.Set("Accept-Language", "en-US,en;q=0.5")

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
	rateLimiterService.SetLimit("kissmanga.in", 2*time.Second)

	// Create DOM service
	domService := &DOMService{}

	// Create engine with all services
	engine := &Engine{
		HTTP:        httpService,
		Download:    downloadService,
		Parser:      parser,
		RateLimiter: rateLimiterService,
		DOM:         domService,
		Logger:      loggerService,
		agents:      make(map[string]Agent),
	}

	// Create metadata service (depends on parser)
	engine.Metadata = &MetadataService{
		Parser: parser,
	}

	// Create API service with logger
	engine.API = NewAPIService(httpService, rateLimiterService, loggerService)

	// Create the extractor service with logger
	engine.Extractor = NewExtractorService(loggerService)

	// Create the pagination service with dependencies
	engine.Pagination = NewPaginationService(engine.API, engine.Extractor, loggerService)

	// Create the search service with dependencies and logger
	engine.Search = NewSearchService(
		loggerService,
		engine.API,
		engine.Extractor,
		engine.Pagination,
		rateLimiterService,
	)

	// Create the WebScraper service
	engine.WebScraper = NewWebScraperService(httpService, domService, rateLimiterService, loggerService)

	loggerService.Info("Engine initialized successfully")
	return engine
}

// RegisterAgent adds an agent to the registry
func (e *Engine) RegisterAgent(agent Agent) error {
	e.agentsMutex.Lock()
	defer e.agentsMutex.Unlock()

	if _, exists := e.agents[agent.ID()]; exists {
		return fmt.Errorf("agent with ID '%s' already registered", agent.ID())
	}

	e.agents[agent.ID()] = agent
	return nil
}

// GetAgent retrieves an agent by ID
func (e *Engine) GetAgent(id string) (Agent, bool) {
	e.agentsMutex.RLock()
	defer e.agentsMutex.RUnlock()

	agent, exists := e.agents[id]
	return agent, exists
}

// AllAgents returns all registered agents
func (e *Engine) AllAgents() []Agent {
	e.agentsMutex.RLock()
	defer e.agentsMutex.RUnlock()

	agents := make([]Agent, 0, len(e.agents))
	for _, a := range e.agents {
		agents = append(agents, a)
	}
	return agents
}

// AgentExists checks if an agent exists
func (e *Engine) AgentExists(id string) bool {
	e.agentsMutex.RLock()
	defer e.agentsMutex.RUnlock()

	_, exists := e.agents[id]
	return exists
}
