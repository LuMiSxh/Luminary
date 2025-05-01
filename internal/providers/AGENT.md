# Luminary Provider Implementation Guide

> Warning: This document is no longer valid. Please wait for the new version.

This guide explains how to implement a new provider (connector) for Luminary, a manga downloader CLI application. Providers in Luminary are responsible for connecting to specific manga sources or websites and implementing the logic for searching, retrieving, and downloading manga content.

## Understanding the Provider Interface

All providers must implement the `Provider` interface defined in `engine/types.go`:

```go
type Provider interface {
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

## Provider Frameworks in Luminary

Based on the codebase, Luminary provides three main frameworks to simplify provider implementation:

1. **API Framework** (`engine/frameworks/api/api_provider.go`): For sites with RESTful JSON APIs
2. **HTML Framework** (`engine/frameworks/web/html_provider.go`): For generic HTML scraping-based sites
3. **Madara Framework** (`engine/frameworks/web/madara_provider.go`): A specialized framework for sites using the Madara WordPress theme

These frameworks handle common operations and error handling, allowing you to focus on the site-specific details.

## Implementation Approaches

### Option 1: API Framework

Use this approach for sites with a well-defined API that returns structured JSON (like MangaDex).

#### Step 1: Create a New Package

Create a new package in the `providers` directory:

```
/providers/myservice/provider.go
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

#### Step 3: Create and Configure Your Provider

```go
// NewProvider creates a new API-based provider
func NewProvider(e *engine.Engine) engine.Provider {
    // Create API provider configuration
    config := api.APIProviderConfig{
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
            "User-Provider": "Luminary/1.0",
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

    // Create and return the API provider
    return api.NewAPIProvider(e, config)
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
func processChapters(ctx context.Context, provider *api.APIProvider, response interface{}, mangaID string) ([]engine.ChapterInfo, bool, error) {
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

#### Step 1: Create Provider Package and Basic Setup

```go
package myscraper

import (
    "Luminary/engine"
    "Luminary/engine/frameworks/web"
    "context"
    "strings"
)

// MyScraper implements a scraping-based provider
type MyScraper struct {
    htmlProvider  *web.HTMLProvider
    engine     *engine.Engine
    webScraper *engine.WebScraperService
}

// NewProvider creates a new scraper-based provider
func NewProvider(e *engine.Engine) engine.Provider {
    // Create HTML provider config
    htmlConfig := web.HTMLProviderConfig{
        ID:          "msc",
        Name:        "My Scraper",
        SiteURL:     "https://myscraper.com",
        Description: "My manga scraper site",
        Headers: map[string]string{
            "User-Provider": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
        },
    }

    // Create HTML provider
    htmlProvider := web.NewHTMLProvider(e, htmlConfig)

    return &MyScraper{
        htmlProvider:  htmlProvider,
        engine:     e,
        webScraper: e.WebScraper,
    }
}
```

#### Step 2: Implement Basic Methods

```go
// Identity methods - delegate to the HTML provider
func (m *MyScraper) ID() string { return m.htmlProvider.ID() }
func (m *MyScraper) Name() string { return m.htmlProvider.Name() }
func (m *MyScraper) Description() string { return m.htmlProvider.Description() }
func (m *MyScraper) SiteURL() string { return m.htmlProvider.SiteURL() }
func (m *MyScraper) Initialize(ctx context.Context) error { return m.htmlProvider.Initialize(ctx) }
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
    page, err := m.htmlProvider.FetchPage(ctx, searchURL)
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
    page, err := m.htmlProvider.FetchPage(ctx, mangaURL)
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

// NewProvider creates a new Madara-based provider
func NewProvider(e *engine.Engine) engine.Provider {
    // Configure Madara provider
    config := web.MadaraConfig{
        ID:              "mmd",
        Name:            "My Madara Site",
        SiteURL:         "https://mymadara.com",
        Description:     "Read manga online at My Madara Site",
        MangaSelector:   "div.post-title h3 a, div.post-title h5 a",
        ChapterSelector: "li.wp-manga-chapter > a, .chapter-link",
        PageSelector:    "div.page-break source, div.page-break img, .reading-content img",
        Headers: map[string]string{
            "User-Provider": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
            "Referer": "https://mymadara.com/",
        },
        // Optional: Configure AJAX behavior for the site
        UseLegacyAjax: false,
        CustomLoadAction: "madara_load_more",
    }

    // Create and return the Madara provider
    return web.NewMadaraProvider(e, config)
}
```

## Registering Your Provider

Once you've implemented your provider, register it with the engine in `main.go`:

```go
// In main.go
func registerProviders(e *engine.Engine) {
    // Register existing providers
    err := e.RegisterProvider(mangadex.NewProvider(e))
    if err != nil {
        e.Logger.Error("Failed to register MangaDex provider: %v", err)
    }
    
    // Register your new provider
    err = e.RegisterProvider(myservice.NewProvider(e))
    if err != nil {
        e.Logger.Error("Failed to register MyService provider: %v", err)
    }
}
```

## Best Practices

1. **Error Handling**: Use the framework's built-in error handling where possible. The common helper functions in `engine/frameworks/common/provider_helpers.go` wrap errors with provider context.

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

6. **CSS Selectors**: For HTML-based providers, use multiple selector options for robustness:
   ```go
   titleSelectors := []string{".manga-title", "h1.title", ".entry-title"}
   for _, selector := range titleSelectors {
       // Try each selector
   }
   ```

7. **Debugging**: Use the `--debug` flag in commands to enable detailed error information.

## Engine Services Reference

The Luminary engine provides several services that your provider can use:

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

### API-based Provider: MangaDex

The MangaDex provider (`providers/mangadex/provider.go`) provides a comprehensive example of an API-based provider that:
- Defines API endpoints and response types
- Creates extractors for mapping JSON to domain models
- Handles pagination for chapter listing
- Implements custom processing for chapter pages

### HTML-based Provider: Generic Scraper

A generic HTML-based provider would use the HTML framework and implement custom scraping logic for each site's specific HTML structure.

### Madara-based Provider: KissManga

The KissManga provider (`providers/kissmanga/provider.go`) shows how to quickly implement an provider for a Madara-based site by customizing the configuration.
