package agents

import (
	"Luminary/engine"
	"Luminary/utils"
	"context"
	"errors"
	"fmt"
	"golang.org/x/net/html"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// BaseAgent provides a complete implementation of the Agent interface
// with hooks for source-specific customization
type BaseAgent struct {
	id          string
	name        string
	description string
	siteURL     string
	apiURL      string

	// Engine provides services
	Engine *engine.Engine

	// Internal state
	initialized  bool
	lastInitTime time.Time
	initMutex    sync.Mutex
	mangaCache   map[string]*MangaInfo
	cacheMutex   sync.RWMutex
}

// NewBaseAgent creates a new base agent with the provided values
func NewBaseAgent(id, name, description string) *BaseAgent {
	return &BaseAgent{
		id:          id,
		name:        name,
		description: description,
		Engine:      engine.New(),
		mangaCache:  make(map[string]*MangaInfo),
		initialized: false,
	}
}

// ID returns the agent's identifier
func (a *BaseAgent) ID() string {
	return a.id
}

// Name returns the agent's display name
func (a *BaseAgent) Name() string {
	return a.name
}

// Description returns the agent's description
func (a *BaseAgent) Description() string {
	return a.description
}

// SiteURL returns the agent's website URL
func (a *BaseAgent) SiteURL() string {
	return a.siteURL
}

// SetSiteURL sets the agent's website URL
func (a *BaseAgent) SetSiteURL(url string) {
	a.siteURL = url
}

// APIURL returns the agent's API URL
func (a *BaseAgent) APIURL() string {
	return a.apiURL
}

// SetAPIURL sets the agent's API URL
func (a *BaseAgent) SetAPIURL(url string) {
	a.apiURL = url
}

// GetEngine returns the agent's engine instance
func (a *BaseAgent) GetEngine() *engine.Engine {
	return a.Engine
}

// Initialize ensures the agent is properly initialized
func (a *BaseAgent) Initialize(ctx context.Context) error {
	a.initMutex.Lock()
	defer a.initMutex.Unlock()

	// Skip if already initialized recently (within 30 minutes)
	if a.initialized && time.Since(a.lastInitTime) < 30*time.Minute {
		return nil
	}

	// Log initialization
	a.Engine.Logger.Info("Initializing agent: %s (%s)", a.name, a.id)

	// Call the source-specific initialization
	err := a.OnInitialize(ctx)
	if err != nil {
		a.Engine.Logger.Error("Failed to initialize agent %s: %v", a.id, err)
		return fmt.Errorf("failed to initialize agent: %w", err)
	}

	a.initialized = true
	a.lastInitTime = time.Now()
	a.Engine.Logger.Info("Agent initialized: %s", a.id)
	return nil
}

// OnInitialize is meant to be overridden by specific agents
func (a *BaseAgent) OnInitialize(ctx context.Context) error {
	// Default implementation does nothing
	return nil
}

// Search provides a default implementation that returns an error
// Specific agents must override this method
func (a *BaseAgent) Search(ctx context.Context, query string, options engine.SearchOptions) ([]Manga, error) {
	return nil, errors.New("search not implemented by this agent")
}

// GetManga provides a default implementation that returns an error
// Specific agents must override this method
func (a *BaseAgent) GetManga(ctx context.Context, id string) (*MangaInfo, error) {
	return nil, errors.New("manga retrieval not implemented by this agent")
}

// GetChapter provides a default implementation that returns an error
// Specific agents must override this method
func (a *BaseAgent) GetChapter(ctx context.Context, chapterID string) (*Chapter, error) {
	return nil, errors.New("chapter retrieval not implemented by this agent")
}

// TryGetMangaForChapter attempts to get manga info for a chapter
// Specific agents must override this method
func (a *BaseAgent) TryGetMangaForChapter(ctx context.Context, chapterID string) (*Manga, error) {
	return nil, errors.New("manga lookup for chapter not implemented by this agent")
}

// DownloadChapter downloads a chapter with common handling
// This provides a default implementation that agents can use
func (a *BaseAgent) DownloadChapter(ctx context.Context, chapterID, destDir string) error {
	// Initialize if needed
	if err := a.Initialize(ctx); err != nil {
		return err
	}

	// Log download request
	a.Engine.Logger.Info("[%s] Downloading chapter: %s to %s", a.id, chapterID, destDir)

	// Get chapter information
	chapter, err := a.GetChapter(ctx, chapterID)
	if err != nil {
		return err
	}

	// Try to get manga info for proper manga title
	var mangaTitle string
	var mangaID string

	manga, err := a.TryGetMangaForChapter(ctx, chapterID)
	if err == nil && manga != nil {
		mangaTitle = manga.Title
		mangaID = manga.ID
	} else {
		// Fall back to using chapter title
		a.Engine.Logger.Debug("[%s] Couldn't find manga for chapter %s, using fallback title", a.id, chapterID)
		mangaTitle = fmt.Sprintf("%s-%s", a.Name(), chapterID)
	}

	// Extract chapter and volume numbers
	chapterNum := &chapter.Info.Number
	if *chapterNum == 0 {
		chapterNum = nil
	}

	// Check for volume override in context
	var volumeNum *int
	if val := ctx.Value("volume_override"); val != nil {
		if volNum, ok := val.(int); ok && volNum > 0 {
			volumeNum = &volNum
		}
	} else if chapter.Info.Title != "" {
		// Try to extract volume from title if not overridden
		_, extractedVol := a.Engine.Metadata.ExtractChapterInfo(chapter.Info.Title)
		volumeNum = extractedVol
	}

	// Prepare metadata
	metadata := engine.ChapterMetadata{
		MangaID:      mangaID,
		MangaTitle:   mangaTitle,
		ChapterID:    chapterID,
		ChapterNum:   chapterNum,
		VolumeNum:    volumeNum,
		ChapterTitle: chapter.Info.Title,
		AgentID:      a.ID(),
	}

	// Convert pages to download requests
	downloadFiles := make([]engine.DownloadRequest, len(chapter.Pages))
	for i, page := range chapter.Pages {
		downloadFiles[i] = engine.DownloadRequest{
			URL:       page.URL,
			Index:     i + 1,
			Filename:  page.Filename,
			PageCount: len(chapter.Pages),
		}
	}

	// Extract concurrency settings from context or use default
	concurrency := a.Engine.Download.MaxConcurrency
	if contextConcurrency, ok := ctx.Value("concurrency").(int); ok && contextConcurrency > 0 {
		concurrency = contextConcurrency
	}

	// Set up download configuration
	config := engine.DownloadJobConfig{
		Metadata:    metadata,
		OutputDir:   destDir,
		Concurrency: concurrency,
		Files:       downloadFiles,
		WaitDuration: func(isRetry bool) {
			if isRetry {
				time.Sleep(a.Engine.HTTP.ThrottleTimeAPI)
			} else {
				time.Sleep(a.Engine.HTTP.ThrottleTimeImages)
			}
		},
	}

	// Log and start download
	a.Engine.Logger.Info("[%s] Downloading %d pages for chapter %s", a.id, len(chapter.Pages), chapterID)

	// Use the engine's download service to download the chapter
	err = a.Engine.Download.DownloadChapter(ctx, config)
	if err != nil {
		a.Engine.Logger.Error("[%s] Download failed: %v", a.id, err)
		return err
	}

	a.Engine.Logger.Info("[%s] Successfully downloaded chapter %s", a.id, chapterID)
	return nil
}

// FetchJSON is a convenience method for agents to fetch JSON data
func (a *BaseAgent) FetchJSON(ctx context.Context, url string, result interface{}) error {
	// Apply rate limiting
	domain := a.ExtractDomain(url)
	a.Engine.RateLimiter.Wait(domain)

	// Create custom headers
	headers := make(http.Header)
	headers.Set("Accept", "application/json")
	if a.siteURL != "" {
		headers.Set("Referer", a.siteURL)
	}

	// Fetch the JSON
	return a.Engine.HTTP.FetchJSON(ctx, url, result, headers)
}

// FetchHTML is a convenience method for agents to fetch HTML content
func (a *BaseAgent) FetchHTML(ctx context.Context, url string) (string, error) {
	// Apply rate limiting
	domain := a.ExtractDomain(url)
	a.Engine.RateLimiter.Wait(domain)

	// Create custom headers
	headers := make(http.Header)
	headers.Set("Accept", "text/html")
	if a.siteURL != "" {
		headers.Set("Referer", a.siteURL)
	}

	// Fetch the HTML
	return a.Engine.HTTP.FetchString(ctx, url, headers)
}

// ParseDOM is a convenience method to parse HTML into a DOM
func (a *BaseAgent) ParseDOM(html string) (*html.Node, error) {
	return a.Engine.DOM.Parse(html)
}

// ExtractDomain extracts the domain from a URL
func (a *BaseAgent) ExtractDomain(urlStr string) string {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		// If parsing fails, return the whole URL as the domain
		return urlStr
	}
	return parsed.Host
}

// FormatMangaID creates a standardized manga ID
func (a *BaseAgent) FormatMangaID(mangaID string) string {
	return utils.FormatMangaID(a.id, mangaID)
}

// ParseMangaID parses a combined manga ID
func (a *BaseAgent) ParseMangaID(combinedID string) (string, error) {
	agentID, mangaID, err := utils.ParseMangaID(combinedID)
	if err != nil {
		return "", err
	}

	// Check if the agent ID matches
	if agentID != a.id {
		return "", fmt.Errorf("manga ID agent mismatch: expected %s, got %s", a.id, agentID)
	}

	return mangaID, nil
}
