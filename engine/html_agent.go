package engine

import (
	"Luminary/errors"
	"context"
	"fmt"
	"strings"
)

// HTMLAgentConfig holds the configuration for HTML-based agents
type HTMLAgentConfig struct {
	ID          string            // Short identifier
	Name        string            // Display name
	SiteURL     string            // Base URL
	Description string            // Site description
	Headers     map[string]string // Default HTTP headers
}

// HTMLAgent provides a base implementation for HTML-based (non-API) agents
type HTMLAgent struct {
	config     HTMLAgentConfig
	engine     *Engine
	webScraper *WebScraperService
}

// NewHTMLAgent creates a new HTML-based agent
func NewHTMLAgent(e *Engine, config HTMLAgentConfig) *HTMLAgent {
	// Create web scraper service if not created yet
	if e.WebScraper == nil {
		e.WebScraper = NewWebScraperService(e.HTTP, e.DOM, e.RateLimiter, e.Logger)
	}

	return &HTMLAgent{
		config:     config,
		engine:     e,
		webScraper: e.WebScraper,
	}
}

// ID returns the agent's identifier
func (a *HTMLAgent) ID() string {
	return a.config.ID
}

// Name returns the agent's display name
func (a *HTMLAgent) Name() string {
	return a.config.Name
}

// Description returns the agent's description
func (a *HTMLAgent) Description() string {
	return a.config.Description
}

// SiteURL returns the agent's website URL
func (a *HTMLAgent) SiteURL() string {
	return a.config.SiteURL
}

// Config returns the agent's configuration
func (a *HTMLAgent) Config() HTMLAgentConfig {
	return a.config
}

// Initialize initializes the agent
func (a *HTMLAgent) Initialize(ctx context.Context) error {
	return ExecuteInitialize(ctx, a.engine, a.ID(), a.Name(), func(ctx context.Context) error {
		// No special initialization needed for HTML agents
		return nil
	})
}

// CreateRequest creates a new scraper request with the agent's default headers
func (a *HTMLAgent) CreateRequest(url string) *ScraperRequest {
	req := NewScraperRequest(url)

	// Add default headers
	for key, value := range a.config.Headers {
		req.SetHeader(key, value)
	}

	return req
}

// FetchPage fetches a web page using the agent's configuration
func (a *HTMLAgent) FetchPage(ctx context.Context, url string) (*WebPage, error) {
	req := a.CreateRequest(url)
	return a.webScraper.FetchPage(ctx, req)
}

// Search implements a basic search for HTML-based agents
// Should be overridden by specific agent types
func (a *HTMLAgent) Search(ctx context.Context, query string, options SearchOptions) ([]Manga, error) {
	a.engine.Logger.Info("[%s] Searching for: %s", a.ID(), query)
	return nil, fmt.Errorf("search not implemented for HTML agent %s", a.ID())
}

// GetManga retrieves manga details
// Should be overridden by specific agent types
func (a *HTMLAgent) GetManga(ctx context.Context, id string) (*MangaInfo, error) {
	a.engine.Logger.Info("[%s] Getting manga details for: %s", a.ID(), id)
	return nil, fmt.Errorf("GetManga not implemented for HTML agent %s", a.ID())
}

// GetChapter retrieves chapter details
// Should be overridden by specific agent types
func (a *HTMLAgent) GetChapter(ctx context.Context, chapterID string) (*Chapter, error) {
	a.engine.Logger.Info("[%s] Getting chapter details for: %s", a.ID(), chapterID)
	return nil, fmt.Errorf("GetChapter not implemented for HTML agent %s", a.ID())
}

// TryGetMangaForChapter attempts to get manga info for a chapter
// Could be overridden by specific agent types for optimization
func (a *HTMLAgent) TryGetMangaForChapter(ctx context.Context, chapterID string) (*Manga, error) {
	// Get the chapter to extract manga ID
	chapter, err := a.GetChapter(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	// If manga ID is available in chapter
	if chapter.MangaID != "" {
		// Get manga details
		mangaInfo, err := a.GetManga(ctx, chapter.MangaID)
		if err != nil {
			return nil, err
		}
		return &mangaInfo.Manga, nil
	}

	return nil, fmt.Errorf("couldn't determine manga for chapter %s", chapterID)
}

// DownloadChapter downloads a chapter
// Could be overridden by specific agent types
func (a *HTMLAgent) DownloadChapter(ctx context.Context, chapterID, destDir string) error {
	return ExecuteDownloadChapter(
		ctx,
		a.engine,
		a.ID(),
		a.Name(),
		chapterID,
		destDir,
		a.GetChapter,
		a.TryGetMangaForChapter,
	)
}

// MadaraAgent extends HTMLAgent with Madara-specific functionality
type MadaraAgent struct {
	*HTMLAgent
	madaraScraper *MadaraScraper
}

// NewMadaraAgent creates a new Madara-based agent
func NewMadaraAgent(e *Engine, config HTMLAgentConfig, selectors map[string]string) *MadaraAgent {
	// Create base HTML agent
	htmlAgent := NewHTMLAgent(e, config)

	// Create Madara scraper
	madaraScraper := NewMadaraScraper(e.WebScraper, config.SiteURL, e.Logger)

	// Set selectors if provided
	if selectors != nil {
		madaraScraper.SetSelectors(
			selectors["mangas"],
			selectors["chapters"],
			selectors["pages"],
		)
	}

	return &MadaraAgent{
		HTMLAgent:     htmlAgent,
		madaraScraper: madaraScraper,
	}
}

// Search implements manga search for Madara sites
func (a *MadaraAgent) Search(ctx context.Context, query string, options SearchOptions) ([]Manga, error) {
	a.engine.Logger.Info("[%s] Searching for: %s", a.ID(), query)

	limit := options.Limit
	if limit <= 0 {
		limit = 100
	}

	pages := options.Pages
	if pages <= 0 {
		pages = 1
	}

	var allManga []Manga

	for page := 0; page < pages; page++ {
		manga, err := a.madaraScraper.FetchMangaList(ctx, page, limit)
		if err != nil {
			return nil, &errors.AgentError{
				AgentID: a.ID(),
				Message: fmt.Sprintf("Failed to search for '%s'", query),
				Err:     err,
			}
		}

		// Filter based on query if provided
		if query != "" {
			var filtered []Manga
			lowerQuery := strings.ToLower(query)

			for _, m := range manga {
				if strings.Contains(strings.ToLower(m.Title), lowerQuery) {
					filtered = append(filtered, m)
				}
			}

			manga = filtered
		}

		allManga = append(allManga, manga...)

		// If we got fewer results than requested, we're at the end
		if len(manga) < limit {
			break
		}
	}

	return allManga, nil
}

// GetManga retrieves manga details for Madara sites
func (a *MadaraAgent) GetManga(ctx context.Context, id string) (*MangaInfo, error) {
	a.engine.Logger.Info("[%s] Getting manga details for: %s", a.ID(), id)

	// Create a basic MangaInfo
	mangaInfo := &MangaInfo{
		Manga: Manga{
			ID:    id,
			Title: "", // Will be filled by the page title
		},
	}

	// Fetch the manga page
	page, err := a.FetchPage(ctx, UrlJoin(a.SiteURL(), id))
	if err != nil {
		return nil, &errors.AgentError{
			AgentID:      a.ID(),
			ResourceType: "manga",
			ResourceID:   id,
			Message:      "Failed to fetch manga page",
			Err:          err,
		}
	}

	// Get manga title
	mangaInfo.Title = page.GetTitle()

	// Get manga description
	descElement, err := page.FindOne(".description-summary")
	if err == nil && descElement != nil {
		mangaInfo.Description = descElement.Text()
	}

	// Get manga authors
	var authors []string
	authorElements, err := page.Find(".author-content a")
	if err == nil {
		for _, authorElement := range authorElements {
			authors = append(authors, authorElement.Text())
		}
	}
	mangaInfo.Authors = authors

	// Get manga tags
	var tags []string
	tagElements, err := page.Find(".genres-content a")
	if err == nil {
		for _, tagElement := range tagElements {
			tags = append(tags, tagElement.Text())
		}
	}
	mangaInfo.Tags = tags

	// Get manga status
	statusElement, err := page.FindOne(".status")
	if err == nil && statusElement != nil {
		mangaInfo.Status = statusElement.Text()
	}

	// Get chapters
	chapters, err := a.madaraScraper.FetchMangaChapters(ctx, id)
	if err != nil {
		a.engine.Logger.Warn("[%s] Failed to fetch chapters for manga %s: %v", a.ID(), id, err)
		// Continue with empty chapters list
		mangaInfo.Chapters = []ChapterInfo{}
	} else {
		mangaInfo.Chapters = chapters
	}

	// If we still don't have a title, use the ID
	if mangaInfo.Title == "" {
		mangaInfo.Title = id
	}

	return mangaInfo, nil
}

// GetChapter retrieves chapter details for Madara sites
func (a *MadaraAgent) GetChapter(ctx context.Context, chapterID string) (*Chapter, error) {
	a.engine.Logger.Info("[%s] Getting chapter details for: %s", a.ID(), chapterID)

	// Create a basic Chapter
	chapter := &Chapter{
		Info: ChapterInfo{
			ID:    chapterID,
			Title: "",
		},
	}

	// Extract chapter number from the ID
	chapter.Info.Number = ExtractChapterNumber(chapterID)

	// Get pages
	pages, err := a.madaraScraper.FetchChapterPages(ctx, chapterID)
	if err != nil {
		return nil, &errors.AgentError{
			AgentID:      a.ID(),
			ResourceType: "chapter",
			ResourceID:   chapterID,
			Message:      "Failed to get pages",
			Err:          err,
		}
	}

	// Set the pages
	chapter.Pages = pages

	// Try to extract the manga ID from the chapter URL
	// For Madara sites, the manga ID is usually the first part of the chapter URL
	parts := strings.Split(chapterID, "/")
	if len(parts) > 0 {
		chapter.MangaID = parts[0]
	}

	return chapter, nil
}

// CreateMadaraAgent is a convenience function for creating Madara-based agents
func CreateMadaraAgent(e *Engine, id, name, siteURL, description string) Agent {
	config := HTMLAgentConfig{
		ID:          id,
		Name:        name,
		SiteURL:     siteURL,
		Description: description,
		Headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.131 Safari/537.36",
		},
	}

	// Default Madara selectors
	selectors := map[string]string{
		"mangas":   "div.post-title h3 a, div.post-title h5 a",
		"chapters": "li.wp-manga-chapter > a",
		"pages":    "div.page-break source",
	}

	return NewMadaraAgent(e, config, selectors)
}
