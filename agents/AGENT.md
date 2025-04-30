# Luminary Agent Implementation Guide

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

## Agent Types in Luminary

Based on the codebase, there are three main ways to implement an agent:

1. **API-based agent** (like MangaDex): Uses structured API endpoints with JSON responses
2. **HTML-based agent**: Uses web scraping for sites without a formal API
3. **Madara-based agent** (like KissManga): A specialized HTML agent for sites using the Madara WordPress theme

## Implementation Approach

### Option 1: API-based Agent (Example: MangaDex)

Use this approach for sites with a well-defined API that returns structured JSON.

#### Step 1: Create a New Package

Create a new package in the `agents` directory:

```
/agents/myservice/agent.go
```

#### Step 2: Define Your Agent Struct

```go
package myservice

import (
    "Luminary/engine"
    "Luminary/errors"
    "context"
    "fmt"
    "time"
)

// MyService implements the engine.Agent interface
type MyService struct {
    engine        *engine.Engine
    id            string
    name          string
    description   string
    siteURL       string
    apiConfig     engine.APIConfig
    extractorSets map[string]engine.ExtractorSet
}

// NewAgent creates a new MyService agent
func NewAgent(e *engine.Engine) engine.Agent {
    agent := &MyService{
        engine:        e,
        id:            "mys",       // Short ID (2-3 letters recommended)
        name:          "My Service", // Display name
        description:   "Description of my manga service",
        siteURL:       "https://myservice.com",
        extractorSets: make(map[string]engine.ExtractorSet),
    }

    // Configure API endpoints
    agent.configureAPIEndpoints()

    // Configure extractors
    agent.configureExtractors()

    return agent
}
```

#### Step 3: Implement Interface Methods

```go
// Basic identity methods
func (m *MyService) ID() string { return m.id }
func (m *MyService) Name() string { return m.name }
func (m *MyService) Description() string { return m.description }
func (m *MyService) SiteURL() string { return m.siteURL }

// Initialize the agent
func (m *MyService) Initialize(ctx context.Context) error {
    return engine.ExecuteInitialize(ctx, m.engine, m.id, m.name, m.onInitialize)
}

// onInitialize performs service-specific initialization
func (m *MyService) onInitialize(ctx context.Context) error {
    // Perform any initialization (fetch tokens, validate connection, etc.)
    return nil
}
```

#### Step 4: Configure API Endpoints

```go
// configureAPIEndpoints sets up the API configuration
func (m *MyService) configureAPIEndpoints() {
    m.apiConfig = engine.APIConfig{
        BaseURL:      "https://api.myservice.com",
        RateLimitKey: "api.myservice.com",
        RetryCount:   3,
        ThrottleTime: 2 * time.Second,
        DefaultHeaders: map[string]string{
            "User-Agent": "Luminary/1.0",
            "Referer":    "https://myservice.com",
        },
        Endpoints: map[string]engine.APIEndpoint{
            // Manga details endpoint
            "manga": {
                Path:          "/manga/{id}",
                Method:        "GET",
                ResponseType:  &MangaResponse{}, // Define your response type
                PathFormatter: engine.DefaultPathFormatter,
            },
            // Chapter details endpoint
            "chapter": {
                Path:          "/chapter/{id}",
                Method:        "GET",
                ResponseType:  &ChapterResponse{}, // Define your response type
                PathFormatter: engine.DefaultPathFormatter,
            },
            // Search endpoint
            "search": {
                Path:         "/search",
                Method:       "GET",
                ResponseType: &SearchResponse{}, // Define your response type
                QueryFormatter: func(params interface{}) url.Values {
                    queryParams := url.Values{}
                    // Format search parameters
                    if opts, ok := params.(*engine.SearchOptions); ok {
                        if opts.Query != "" {
                            queryParams.Set("query", opts.Query)
                        }
                        // Add other parameters (limit, sort, etc.)
                    }
                    return queryParams
                },
            },
        },
    }
}
```

#### Step 5: Define Response Types

```go
// Response types
type MangaResponse struct {
    // Match API response fields
    ID          string `json:"id"`
    Title       string `json:"title"`
    Description string `json:"description"`
    // Other fields...
}

type ChapterResponse struct {
    // Match API response fields
    ID     string `json:"id"`
    Title  string `json:"title"`
    Number string `json:"number"`
    // Other fields...
}

type SearchResponse struct {
    // Match API response fields
    Results []struct {
        ID    string `json:"id"`
        Title string `json:"title"`
        // Other fields...
    } `json:"results"`
    Total int `json:"total"`
}
```

#### Step 6: Configure Data Extractors

```go
// configureExtractors sets up data mapping from API responses to domain models
func (m *MyService) configureExtractors() {
    // Manga extractor
    m.extractorSets["manga"] = engine.ExtractorSet{
        Name:  "MyServiceManga",
        Model: &engine.MangaInfo{},
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
            // Additional fields (description, authors, etc.)
        },
    }

    // Similar extractors for search results and chapters
    // ...
}
```

#### Step 7: Implement Core Methods

```go
// Search for manga
func (m *MyService) Search(ctx context.Context, query string, options engine.SearchOptions) ([]engine.Manga, error) {
    return engine.ExecuteSearch(
        ctx,
        m.engine,
        m.id,
        query,
        &options,
        m.apiConfig,
        engine.PaginationConfig{
            LimitParam:     "limit",
            OffsetParam:    "offset",
            TotalCountPath: []string{"Total"},
            ItemsPath:      []string{"Results"},
            DefaultLimit:   20,
            MaxLimit:       100,
        },
        m.extractorSets["search"],
    )
}

// Get manga details
func (m *MyService) GetManga(ctx context.Context, id string) (*engine.MangaInfo, error) {
    return engine.ExecuteGetManga(
        ctx,
        m.engine,
        m.id,
        id,
        m.apiConfig,
        m.extractorSets["manga"],
        m.fetchChaptersForManga, // Implement this function to get chapters
    )
}

// fetchChaptersForManga retrieves chapters for a manga
func (m *MyService) fetchChaptersForManga(ctx context.Context, mangaID string) ([]engine.ChapterInfo, error) {
    // Implementation to fetch chapters (possibly with pagination)
    // ...
}

// Get chapter details
func (m *MyService) GetChapter(ctx context.Context, chapterID string) (*engine.Chapter, error) {
    return engine.ExecuteGetChapter(
        ctx,
        m.engine,
        m.id,
        chapterID,
        m.apiConfig,
        m.extractorSets["chapter"],
        m.processChapterResponse, // Implement custom processing if needed
    )
}

// Get manga info for a chapter
func (m *MyService) TryGetMangaForChapter(ctx context.Context, chapterID string) (*engine.Manga, error) {
    // Get the chapter first to extract manga ID
    chapter, err := m.GetChapter(ctx, chapterID)
    if err != nil {
        return nil, err
    }

    if chapter.MangaID != "" {
        // Get manga details
        mangaInfo, err := m.GetManga(ctx, chapter.MangaID)
        if err != nil {
            return nil, err
        }
        return &mangaInfo.Manga, nil
    }

    return nil, fmt.Errorf("couldn't determine manga for chapter %s", chapterID)
}

// Download a chapter
func (m *MyService) DownloadChapter(ctx context.Context, chapterID, destDir string) error {
    return engine.ExecuteDownloadChapter(
        ctx,
        m.engine,
        m.id,
        m.name,
        chapterID,
        destDir,
        m.GetChapter,
        m.TryGetMangaForChapter,
    )
}
```

### Option 2: HTML-based Agent

For sites without a formal API, you can implement a scraper-based agent:

#### Step 1: Create a New Package and Struct

```go
package mysite

import (
    "Luminary/engine"
    "Luminary/errors"
    "context"
    "fmt"
)

// MySite implements an HTML-based agent
type MySite struct {
    htmlAgent  *engine.HTMLAgent
    engine     *engine.Engine
    webScraper *engine.WebScraperService
}

// NewAgent creates a new MySite agent
func NewAgent(e *engine.Engine) engine.Agent {
    // Create HTML agent config
    htmlConfig := engine.HTMLAgentConfig{
        ID:          "mys",
        Name:        "My Site",
        SiteURL:     "https://mysite.com",
        Description: "My manga site description",
        Headers: map[string]string{
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
        },
    }

    // Create HTML agent
    htmlAgent := engine.NewHTMLAgent(e, htmlConfig)

    return &MySite{
        htmlAgent:  htmlAgent,
        engine:     e,
        webScraper: e.WebScraper,
    }
}
```

#### Step 2: Implement Identity Methods

```go
// Identity methods - delegate to the HTML agent
func (m *MySite) ID() string { return m.htmlAgent.ID() }
func (m *MySite) Name() string { return m.htmlAgent.Name() }
func (m *MySite) Description() string { return m.htmlAgent.Description() }
func (m *MySite) SiteURL() string { return m.htmlAgent.SiteURL() }
func (m *MySite) Initialize(ctx context.Context) error { return m.htmlAgent.Initialize(ctx) }
```

#### Step 3: Implement Core Methods with Web Scraping

```go
// Search for manga
func (m *MySite) Search(ctx context.Context, query string, options engine.SearchOptions) ([]engine.Manga, error) {
    m.engine.Logger.Info("[%s] Searching for: %s", m.ID(), query)
    
    // Create request
    req := engine.NewScraperRequest(m.SiteURL() + "/search")
    req.SetMethod("GET")
    
    // Add query parameters
    if query != "" {
        req.URL += "?q=" + url.QueryEscape(query)
    }
    
    // Fetch page
    page, err := m.webScraper.FetchPage(ctx, req)
    if err != nil {
        return nil, &errors.AgentError{
            AgentID: m.ID(),
            Message: fmt.Sprintf("Failed to search: %v", err),
            Err:     err,
        }
    }
    
    // Extract manga from search results
    var results []engine.Manga
    
    // Example: Find all manga items with a CSS selector
    mangaElements, err := page.Find(".manga-item")
    if err == nil {
        for _, elem := range mangaElements {
            // Extract manga details from elements
            linkElement, err := elem.FindOne("a.title")
            if err != nil {
                continue
            }
            
            href := linkElement.Attr("href")
            title := linkElement.Text()
            
            // Extract ID from URL
            id := engine.ExtractPathFromURL(href)
            
            manga := engine.Manga{
                ID:    id,
                Title: title,
            }
            
            results = append(results, manga)
        }
    }
    
    return results, nil
}

// Similar implementations for GetManga, GetChapter, etc.
// ...
```

### Option 3: Using the Madara Framework

For sites based on the Madara WordPress theme (like KissManga), you can use the Madara agent framework:

```go
package mymadarasite

import (
    "Luminary/agents"
    "Luminary/engine"
)

// NewAgent creates a new agent for a Madara-based site
func NewAgent(e *engine.Engine) engine.Agent {
    // Create a Madara agent with site-specific configuration
    config := madara.DefaultConfig(
        "mmd",
        "My Madara Site",
        "https://mymadarasite.com",
        "Read manga online for free at My Madara Site",
    )
    
    // Customize selectors if needed
    config.MangaSelector = "div.post-title h3 a, div.post-title h5 a"
    config.ChapterSelector = "li.wp-manga-chapter > a, .chapter-link"
    config.PageSelector = "div.page-break source, div.page-break img"
    
    // Customize headers if needed
    config.Headers = map[string]string{
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36",
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
        "Referer": "https://mymadarasite.com/",
    }
    
    // Create and return the Madara agent
    return madara.NewAgent(e, config)
}
```

## Registering Your Agent

Once you've implemented your agent, you need to register it with the engine in `main.go`:

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

1. **Error Handling**: Use the `errors.AgentError` type to provide context for errors. The engine provides helper functions like `ExecuteGetManga` and `ExecuteSearch` that handle common error cases.

2. **Logging**: Use the engine's logger (`e.Logger`) for consistent logging.

3. **Rate Limiting**: Respect the site's rate limits by using the engine's rate limiter service.

4. **Extractors**: When designing extractors, make field names match exactly between `SourcePath` and the struct they're targeting.

5. **Testing**: Test your agent with basic operations (search, get manga, get chapter) before integrating it.

6. **Robustness**: Implement fallbacks and handle edge cases (like missing data) gracefully.

## Common Engine Services

The Luminary engine provides several services that your agent can use:

- **HTTP Service**: For making HTTP requests
- **DOM Service**: For parsing and querying HTML
- **WebScraper Service**: For more advanced web scraping
- **API Service**: For interacting with RESTful APIs
- **Download Service**: For downloading files with concurrency control
- **RateLimiter Service**: For respecting rate limits
- **Logger Service**: For logging
- **Extractor Service**: For mapping data between formats
- **Pagination Service**: For handling paginated API responses
- **Search Service**: For standardized search functionality

## Example Cases

### API-based Agent: MangaDex

The MangaDex agent (`agents/mangadex/agent.go`) provides a good example of an API-based agent that:
- Defines API endpoints
- Creates extractors for mapping JSON to domain models
- Handles pagination for chapter listing
- Implements search with filters

### HTML-based Agent: Madara Framework

The Madara agent framework (`engine/html_agent.go` and `agents/madara.go`) demonstrates how to implement scraping-based agents for sites using the popular Madara WordPress theme.

### Simple Madara Implementation: KissManga

The KissManga agent (`agents/kissmanga/agent.go`) shows how to quickly implement an agent for a Madara-based site by customizing the configuration.
