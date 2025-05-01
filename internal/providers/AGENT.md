# Luminary Agent Implementation Guide

> Warning: This document is no longer valid. Please wait for the new version.

This guide explains how to implement a new agent (connector) for Luminary, a manga downloader CLI application. Agents in Luminary are responsible for connecting to specific manga sources or websites and implementing the logic for searching, retrieving, and downloading manga content.

## Understanding the Agent Interface

All agents must implement the `Agent` interface defined in `engine/types.go`:

```go
type Agent interface {
    ID() string
    Name() string
    Description() string
    SiteURL() string

    Initialize(ctx context.Context) error

    Search(ctx context.Context, query string, options SearchOptions) ([]Manga, error)
    GetManga(ctx context.Context, id string) (*MangaInfo, error)
    GetChapter(ctx context.Context, chapterID string) (*Chapter, error)
    TryGetMangaForChapter(ctx context.Context, chapterID string) (*Manga, error)
    DownloadChapter(ctx context.Context, chapterID, destDir string) error
}
```

## Agent Frameworks in Luminary

Based on the codebase, Luminary provides three main frameworks to simplify agent implementation:

1. **API Framework** (`engine/frameworks/api/api_agent.go`): For sites with RESTful JSON APIs
2. **HTML Framework** (`engine/frameworks/web/html_agent.go`): For generic HTML scraping-based sites
3. **Madara Framework** (`engine/frameworks/web/madara_agent.go`): A specialized framework for sites using the Madara WordPress theme

These frameworks handle common operations and error handling, allowing you to focus on the site-specific details.

## Implementation Approaches

### Option 1: API Framework

Use this approach for sites with a well-defined API that returns structured JSON (like MangaDex).

#### Step 1: Create a New Package

Create a new package in the `agents` directory:

```
/agents/myservice/agent.go
```

#### Step 2: Define Response Types

First, define types that match the API response structure:

```go
package myservice

import (
    "Luminary/engine"
    "Luminary/engine/frameworks/api"
    "context"
    "time"
)

// Define types that match your API responses
type MangaResponse struct {
    Data struct {
        ID          string `json:"id"`
        Title       string `json:"title"`
        Description string `json:"description"`
        // Other fields...
    } `json:"data"`
}

type ChapterResponse struct {
    Data struct {
        ID         string `json:"id"`
        Title      string `json:"title"`
        Number     string `json:"number"`
        PublishAt  time.Time `json:"publishedAt"`
        // Other fields...
    } `json:"data"`
}

type SearchResponse struct {
    Results []struct {
        ID    string `json:"id"`
        Title string `json:"title"`
        // Other fields...
    } `json:"results"`
    Total int `json:"total"`
}
```

#### Step 3: Create and Configure Your Agent

```go
// NewAgent creates a new API-based agent
func NewAgent(e *engine.Engine) engine.Agent {
    // Create API agent configuration
    config := api.APIAgentConfig{
        // Basic identity
        ID:          "mys",              // Short identifier (2-3 chars)
        Name:        "My Service",       // Display name
        Description: "My manga service", // Description
        SiteURL:     "https://myservice.com",

        // API configuration
        BaseURL:      "https://api.myservice.com",
        RateLimitKey: "api.myservice.com",
        RetryCount:   3,
        ThrottleTime: 2 * time.Second,

        DefaultHeaders: map[string]string{
            "User-Agent": "Luminary/1.0",
            "Referer":    "https://myservice.com",
        },

        // Endpoints configuration
        Endpoints: map[string]api.EndpointConfig{
            "manga": {
                Path:         "/manga/{id}",
                Method:       "GET",
                ResponseType: &MangaResponse{},
            },
            "chapter": {
                Path:         "/chapter/{id}",
                Method:       "GET",
                ResponseType: &ChapterResponse{},
            },
            "search": {
                Path:         "/search",
                Method:       "GET",
                ResponseType: &SearchResponse{},
            },
        },

        // Custom query formatters (optional)
        QueryFormatters: map[string]api.QueryFormatter{
            "search": formatSearchQuery,
            // Add other endpoints that need custom query formatting
        },

        // Custom response processors (optional)
        ResponseProcessors: map[string]api.ResponseProcessor{
            "chapter": processChapterResponse,
            // Add other endpoints that need custom response processing
        },

        // Pagination configuration for search results
        PaginationConfig: &engine.PaginationConfig{
            LimitParam:     "limit",
            OffsetParam:    "offset",
            TotalCountPath: []string{"Total"},
            ItemsPath:      []string{"Results"},
            DefaultLimit:   20,
            MaxLimit:       100,
        },

        // Chapter fetching configuration
        ChapterConfig: api.ChapterFetchConfig{
            EndpointName:      "chapters",
            ResponseItemsPath: []string{"Data"},
            TotalCountPath:    []string{"Total"},
            LimitParamName:    "limit",
            OffsetParamName:   "offset",
            DefaultPageSize:   100,
            MaxPageSize:       100,
            ProcessChapters:   processChapters,  // Custom chapter processing function
        },
    }

    // Configure extractors
    config.ExtractorSets = configureExtractors()

    // Create and return the API agent
    return api.NewAPIAgent(e, config)
}
```

#### Step 4: Implement Helper Functions

```go
// formatSearchQuery formats search parameters for the API
func formatSearchQuery(params interface{}) url.Values {
    queryParams := url.Values{}

    // Handle search options
    if opts, ok := params.(*engine.SearchOptions); ok {
        // Apply the title query
        if opts.Query != "" {
            queryParams.Set("query", opts.Query)
        }

        // Set limit
        if opts.Limit > 0 {
            queryParams.Set("limit", strconv.Itoa(opts.Limit))
        }

        // Apply sorting if specified
        if opts.Sort != "" {
            switch strings.ToLower(opts.Sort) {
            case "relevance":
                queryParams.Set("sort", "relevance")
            case "popularity":
                queryParams.Set("sort", "popularity")
            case "name":
                queryParams.Set("sort", "name")
            }
        }

        // Apply filters if provided
        if opts.Filters != nil {
            for field, value := range opts.Filters {
                switch field {
                case "author":
                    queryParams.Add("author", value)
                case "genre":
                    queryParams.Add("genre", value)
                }
            }
        }
    }

    return queryParams
}

// processChapterResponse handles custom processing for chapter responses
func processChapterResponse(response interface{}, chapterID string) (interface{}, error) {
    // Cast to expected response type
    chapterResp, ok := response.(*ChapterResponse)
    if !ok {
        return nil, fmt.Errorf("unexpected response type: %T", response)
    }

    // Create a new chapter object
    chapter := &engine.Chapter{
        Info: engine.ChapterInfo{
            ID:    chapterID,
            Title: chapterResp.Data.Title,
        },
    }

    // Convert chapter number if present
    if chapterResp.Data.Number != "" {
        if num, err := strconv.ParseFloat(chapterResp.Data.Number, 64); err == nil {
            chapter.Info.Number = num
        }
    }

    // Add more chapter processing logic here
    // ...

    return chapter, nil
}

// processChapters processes chapter list responses
func processChapters(ctx context.Context, agent *api.APIAgent, response interface{}, mangaID string) ([]engine.ChapterInfo, bool, error) {
    // Process the chapter list response
    // ...

    // Return:
    // 1. Array of chapter info
    // 2. Boolean indicating if there are more pages
    // 3. Error if any
    return chapters, hasMore, nil
}
```

#### Step 5: Configure Extractors

```go
// configureExtractors sets up mapping between API responses and domain models
func configureExtractors() map[string]engine.ExtractorSet {
    extractorSets := make(map[string]engine.ExtractorSet)

    // Manga extractor
    extractorSets["manga"] = engine.ExtractorSet{
        Name:  "MyServiceManga",
        Model: &engine.MangaInfo{},
        Extractors: []engine.Extractor{
            {
                Name:       "ID",
                SourcePath: []string{"Data", "ID"},
                TargetPath: "ID",
                Required:   true,
            },
            {
                Name:       "Title",
                SourcePath: []string{"Data", "Title"},
                TargetPath: "Title",
                Required:   true,
            },
            {
                Name:       "Description",
                SourcePath: []string{"Data", "Description"},
                TargetPath: "Description",
                Required:   false,
            },
            // Add more extractors for other fields
        },
    }

    // Search results extractor
    extractorSets["search"] = engine.ExtractorSet{
        Name:  "MyServiceSearch",
        Model: &engine.Manga{},
        Extractors: []engine.Extractor{
            {
                Name:       "ID",
                SourcePath: []string{"ID"},
                TargetPath: "ID",
                Required:   true,
            },
            {
                Name:       "Title",
                SourcePath: []string{"Title"},
                TargetPath: "Title",
                Required:   true,
            },
            // Add more extractors for other fields
        },
    }

    // Add chapter extractor if needed
    // ...

    return extractorSets
}
```

### Option 2: HTML Framework (Web Scraping)

For sites without a structured API, use the HTML framework for web scraping:

#### Step 1: Create Agent Package and Basic Setup

```go
package myscraper

import (
    "Luminary/engine"
    "Luminary/engine/frameworks/web"
    "context"
    "strings"
)

// MyScraper implements a scraping-based agent
type MyScraper struct {
    htmlAgent  *web.HTMLAgent
    engine     *engine.Engine
    webScraper *engine.WebScraperService
}

// NewAgent creates a new scraper-based agent
func NewAgent(e *engine.Engine) engine.Agent {
    // Create HTML agent config
    htmlConfig := web.HTMLAgentConfig{
        ID:          "msc",
        Name:        "My Scraper",
        SiteURL:     "https://myscraper.com",
        Description: "My manga scraper site",
        Headers: map[string]string{
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
        },
    }

    // Create HTML agent
    htmlAgent := web.NewHTMLAgent(e, htmlConfig)

    return &MyScraper{
        htmlAgent:  htmlAgent,
        engine:     e,
        webScraper: e.WebScraper,
    }
}
```

#### Step 2: Implement Basic Methods

```go
// Identity methods - delegate to the HTML agent
func (m *MyScraper) ID() string { return m.htmlAgent.ID() }
func (m *MyScraper) Name() string { return m.htmlAgent.Name() }
func (m *MyScraper) Description() string { return m.htmlAgent.Description() }
func (m *MyScraper) SiteURL() string { return m.htmlAgent.SiteURL() }
func (m *MyScraper) Initialize(ctx context.Context) error { return m.htmlAgent.Initialize(ctx) }
```

#### Step 3: Implement Web Scraping Logic

```go
// Search implements manga search via web scraping
func (m *MyScraper) Search(ctx context.Context, query string, options engine.SearchOptions) ([]engine.Manga, error) {
    m.engine.Logger.Info("[%s] Searching for: %s", m.ID(), query)
    
    // Create search URL
    searchURL := m.SiteURL() + "/search"
    if query != "" {
        searchURL += "?q=" + url.QueryEscape(query)
    }
    
    // Fetch search page
    page, err := m.htmlAgent.FetchPage(ctx, searchURL)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch search page: %w", err)
    }
    
    // Extract manga items using CSS selectors
    var results []engine.Manga
    
    mangaElements, err := page.Find(".manga-item")
    if err == nil && len(mangaElements) > 0 {
        for _, elem := range mangaElements {
            // Extract title and link
            titleElem, err := elem.FindOne(".manga-title a")
            if err != nil || titleElem == nil {
                continue
            }
            
            title := titleElem.Text()
            href := titleElem.Attr("href")
            if title == "" || href == "" {
                continue
            }
            
            // Extract ID from URL
            id := engine.ExtractPathFromURL(href)
            
            // Create manga object
            manga := engine.Manga{
                ID:    id,
                Title: title,
            }
            
            // Optionally extract more details like cover, authors, etc.
            // ...
            
            results = append(results, manga)
        }
    }
    
    m.engine.Logger.Info("[%s] Found %d manga for query: %s", m.ID(), len(results), query)
    return results, nil
}

// GetManga retrieves manga details
func (m *MyScraper) GetManga(ctx context.Context, id string) (*engine.MangaInfo, error) {
    m.engine.Logger.Info("[%s] Getting manga details for: %s", m.ID(), id)
    
    // Create manga URL
    mangaURL := engine.UrlJoin(m.SiteURL(), id)
    
    // Fetch manga page
    page, err := m.htmlAgent.FetchPage(ctx, mangaURL)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch manga page: %w", err)
    }
    
    // Extract manga details using CSS selectors
    mangaInfo := &engine.MangaInfo{
        Manga: engine.Manga{
            ID: id,
        },
    }
    
    // Extract title
    titleElem, err := page.FindOne("h1.manga-title")
    if err == nil && titleElem != nil {
        mangaInfo.Title = titleElem.Text()
    }
    
    // Extract description
    descElem, err := page.FindOne(".manga-description")
    if err == nil && descElem != nil {
        mangaInfo.Description = descElem.Text()
    }
    
    // Extract authors
    authorElems, err := page.Find(".manga-authors a")
    if err == nil && len(authorElems) > 0 {
        for _, elem := range authorElems {
            author := elem.Text()
            if author != "" {
                mangaInfo.Authors = append(mangaInfo.Authors, author)
            }
        }
    }
    
    // Extract genres/tags
    genreElems, err := page.Find(".manga-genres a")
    if err == nil && len(genreElems) > 0 {
        for _, elem := range genreElems {
            genre := elem.Text()
            if genre != "" {
                mangaInfo.Tags = append(mangaInfo.Tags, genre)
            }
        }
    }
    
    // Extract status
    statusElem, err := page.FindOne(".manga-status")
    if err == nil && statusElem != nil {
        mangaInfo.Status = statusElem.Text()
    }
    
    // Extract chapters
    chapterElems, err := page.Find(".chapter-list .chapter-item")
    if err == nil && len(chapterElems) > 0 {
        for _, elem := range chapterElems {
            // Get chapter link
            linkElem, err := elem.FindOne("a")
            if err != nil || linkElem == nil {
                continue
            }
            
            href := linkElem.Attr("href")
            if href == "" {
                continue
            }
            
            // Extract chapter ID from URL
            chapterID := engine.ExtractPathFromURL(href)
            
            // Get chapter title
            title := linkElem.Text()
            
            // Extract chapter number from title
            chapterNumber := engine.ExtractChapterNumber(title)
            
            // Create chapter info
            chapterInfo := engine.ChapterInfo{
                ID:     chapterID,
                Title:  title,
                Number: chapterNumber,
                Date:   time.Now(), // Use current time as fallback
            }
            
            // Try to extract date if available
            dateElem, err := elem.FindOne(".chapter-date")
            if err == nil && dateElem != nil {
                dateText := dateElem.Text()
                // Parse date (implementation depends on site format)
                // ...
            }
            
            mangaInfo.Chapters = append(mangaInfo.Chapters, chapterInfo)
        }
    }
    
    return mangaInfo, nil
}

// GetChapter retrieves chapter details
func (m *MyScraper) GetChapter(ctx context.Context, chapterID string) (*engine.Chapter, error) {
    // Implementation similar to GetManga, but for chapters
    // ...
}

// TryGetMangaForChapter attempts to get manga info for a chapter
func (m *MyScraper) TryGetMangaForChapter(ctx context.Context, chapterID string) (*engine.Manga, error) {
    // Implementation to find a manga from a chapter
    // ...
}

// DownloadChapter downloads a chapter
func (m *MyScraper) DownloadChapter(ctx context.Context, chapterID, destDir string) error {
    // Get the chapter first to extract pages
    chapter, err := m.GetChapter(ctx, chapterID)
    if err != nil {
        return err
    }
    
    // Get manga info for title/metadata
    manga, err := m.TryGetMangaForChapter(ctx, chapterID)
    if err != nil {
        // If manga info not available, create basic info
        manga = &engine.Manga{
            ID:    "unknown",
            Title: "Unknown Manga",
        }
    }
    
    // Use the common download implementation
    return engine.ExecuteDownloadChapter(
        ctx,
        m.engine,
        m.ID(),
        m.Name(),
        chapterID,
        destDir,
        m.GetChapter,
        m.TryGetMangaForChapter,
    )
}
```

### Option 3: Madara Framework

For sites based on the Madara WordPress theme (common for many manga sites), use the specialized Madara framework:

```go
package mymadara

import (
    "Luminary/engine"
    "Luminary/engine/frameworks/web"
)

// NewAgent creates a new Madara-based agent
func NewAgent(e *engine.Engine) engine.Agent {
    // Configure Madara agent
    config := web.MadaraConfig{
        ID:              "mmd",
        Name:            "My Madara Site",
        SiteURL:         "https://mymadara.com",
        Description:     "Read manga online at My Madara Site",
        MangaSelector:   "div.post-title h3 a, div.post-title h5 a",
        ChapterSelector: "li.wp-manga-chapter > a, .chapter-link",
        PageSelector:    "div.page-break source, div.page-break img, .reading-content img",
        Headers: map[string]string{
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
            "Referer": "https://mymadara.com/",
        },
        // Optional: Configure AJAX behavior for the site
        UseLegacyAjax: false,
        CustomLoadAction: "madara_load_more",
    }

    // Create and return the Madara agent
    return web.NewMadaraAgent(e, config)
}
```

## Registering Your Agent

Once you've implemented your agent, register it with the engine in `main.go`:

```go
// In main.go
func registerAgents(e *engine.Engine) {
    // Register existing agents
    err := e.RegisterAgent(mangadex.NewAgent(e))
    if err != nil {
        e.Logger.Error("Failed to register MangaDex agent: %v", err)
    }
    
    // Register your new agent
    err = e.RegisterAgent(myservice.NewAgent(e))
    if err != nil {
        e.Logger.Error("Failed to register MyService agent: %v", err)
    }
}
```

## Best Practices

1. **Error Handling**: Use the framework's built-in error handling where possible. The common helper functions in `engine/frameworks/common/agent_helpers.go` wrap errors with agent context.

2. **Logging**: Use the engine's logger (`e.Logger`) for consistent logging:
   ```go
   e.Logger.Info("[%s] Fetching manga: %s", a.ID(), id)
   e.Logger.Error("[%s] Failed to fetch: %v", a.ID(), err)
   ```

3. **Rate Limiting**: All frameworks automatically apply rate limiting. For custom implementations, use:
   ```go
   domain := e.ExtractDomain(url)
   e.RateLimiter.Wait(domain)
   ```

4. **Extractors**: When defining extractors:
    - Make sure field names in `SourcePath` match exactly the case of the struct field names
    - Use `Required: true` for essential fields
    - Provide transformation functions for complex mappings

5. **Timeout Handling**: All operations should use the provided context for cancellation:
   ```go
   ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
   defer cancel()
   ```

6. **CSS Selectors**: For HTML-based agents, use multiple selector options for robustness:
   ```go
   titleSelectors := []string{".manga-title", "h1.title", ".entry-title"}
   for _, selector := range titleSelectors {
       // Try each selector
   }
   ```

7. **Debugging**: Use the `--debug` flag in commands to enable detailed error information.

## Engine Services Reference

The Luminary engine provides several services that your agent can use:

| Service     | Purpose            | Key Methods                                    |
|-------------|--------------------|------------------------------------------------|
| HTTP        | HTTP requests      | `FetchWithRetries`, `FetchJSON`, `FetchString` |
| DOM         | HTML parsing       | `QuerySelector`, `QuerySelectorAll`, `GetText` |
| WebScraper  | Web scraping       | `FetchPage`                                    |
| API         | API requests       | `FetchFromAPI`                                 |
| Download    | File downloads     | `DownloadChapter`                              |
| RateLimiter | Rate limiting      | `Wait`, `SetLimit`                             |
| Logger      | Logging            | `Info`, `Error`, `Debug`, `Warn`               |
| Extractor   | Data mapping       | `Extract`, `ExtractList`                       |
| Pagination  | Paginated requests | `FetchAllPages`                                |
| Search      | Search handling    | `ExecuteSearch`, `SearchAcrossProviders`       |

## Examples

### API-based Agent: MangaDex

The MangaDex agent (`agents/mangadex/agent.go`) provides a comprehensive example of an API-based agent that:
- Defines API endpoints and response types
- Creates extractors for mapping JSON to domain models
- Handles pagination for chapter listing
- Implements custom processing for chapter pages

### HTML-based Agent: Generic Scraper

A generic HTML-based agent would use the HTML framework and implement custom scraping logic for each site's specific HTML structure.

### Madara-based Agent: KissManga

The KissManga agent (`agents/kissmanga/agent.go`) shows how to quickly implement an agent for a Madara-based site by customizing the configuration.
