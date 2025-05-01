# Luminary Provider Implementation Guide

This guide explains how to implement a new provider (connector) for Luminary, a manga downloader CLI application. Providers in Luminary are responsible for connecting to specific manga sources and implementing the logic for searching, retrieving, and downloading manga content.

![Separator](../../.github/assets/luminary-separator.png)

## Table of Contents

1. [Understanding the Provider System](#understanding-the-provider-system)
2. [Provider Interface](#provider-interface)
3. [Provider Frameworks](#provider-frameworks)
4. [Implementing API-based Providers](#implementing-api-based-providers)
5. [Implementing Web-based Providers](#implementing-web-based-providers)
6. [Implementing Madara-based Providers](#implementing-madara-based-providers)
7. [Registering Your Provider](#registering-your-provider)
8. [Best Practices](#best-practices)
9. [Engine Services Reference](#engine-services-reference)
10. [Examples](#examples)

## Understanding the Provider System

Luminary uses a plugin-like architecture where each manga source is implemented as a provider. The engine provides common utilities and services that providers can use, including:

- HTTP requests and rate limiting
- HTML parsing and DOM traversal
- File downloading
- Error handling
- Logging

Providers implement source-specific logic to integrate with these services, while the engine handles the heavy lifting.

## Provider Interface

All providers must implement the `Provider` interface defined in `pkg/provider/interface.go`:

```go
type Provider interface {
    ID() string                 // Short identifier for the provider
    Name() string               // Display name
    Description() string        // Description of the source
    SiteURL() string            // Base website URL

    Initialize(ctx context.Context) error  // Setup/initialization

    Search(ctx context.Context, query string, options core.SearchOptions) ([]core.Manga, error)
    GetManga(ctx context.Context, id string) (*core.MangaInfo, error)
    GetChapter(ctx context.Context, chapterID string) (*core.Chapter, error)
    TryGetMangaForChapter(ctx context.Context, chapterID string) (*core.Manga, error)
    DownloadChapter(ctx context.Context, chapterID, destDir string) error
}
```

## Provider Frameworks

Luminary includes three main frameworks to simplify provider implementation:

1. **API Framework** (`pkg/provider/api/base_provider.go`): For sites with structured JSON APIs
2. **Web Framework** (`pkg/provider/web/base_provider.go`): For generic HTML scraping-based sites
3. **Madara Framework** (`pkg/provider/madara/madara_agent.go`): Specialized for sites using the Madara WordPress theme

These frameworks handle common operations and error handling, allowing you to focus on the site-specific details.

![Separator](../../.github/assets/luminary-separator.png)

## Implementing API-based Providers

Use this approach for sites with well-defined APIs that return structured JSON (like MangaDex).

### Step 1: Create a New Package

Create a new file in the `internal/providers` directory:

```
internal/providers/myservice.go
```

### Step 2: Define Response Types

Define types that match the API response structure:

```go
package providers

import (
    "Luminary/pkg/engine"
    "Luminary/pkg/engine/core"
    "Luminary/pkg/engine/network"
    "Luminary/pkg/engine/parser"
    "Luminary/pkg/engine/search"
    "Luminary/pkg/provider"
    "Luminary/pkg/provider/api"
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
        ID         string    `json:"id"`
        Title      string    `json:"title"`
        Number     string    `json:"number"`
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

### Step 3: Create and Configure Your Provider

```go
// NewMyServiceProvider creates a new API-based provider
func NewMyServiceProvider(e *engine.Engine) provider.Provider {
    // Create API provider configuration
    config := api.Config{
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
        PaginationConfig: &search.PaginationConfig{
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
            ProcessChapters:   processChapters,
        },
    }

    // Configure extractors
    config.ExtractorSets = configureExtractors()

    // Create and return the API provider
    return api.NewProvider(e, config)
}
```

### Step 4: Implement Helper Functions

```go
// formatSearchQuery formats search parameters for the API
func formatSearchQuery(params interface{}) url.Values {
    queryParams := url.Values{}

    // Handle search options
    if opts, ok := params.(*core.SearchOptions); ok {
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
    chapter := &core.Chapter{
        Info: core.ChapterInfo{
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

    // Set publication date
    chapter.Info.Date = chapterResp.Data.PublishAt

    // Add more chapter processing logic here...

    return chapter, nil
}

// processChapters processes chapter list responses
func processChapters(ctx context.Context, provider *api.Provider, response interface{}, mangaID string) ([]core.ChapterInfo, bool, error) {
    // Cast to expected response type
    chaptersResp, ok := response.(*ChapterListResponse)
    if !ok {
        return nil, false, fmt.Errorf("unexpected response type: %T", response)
    }

    // Process the chapter list response
    var chapters []core.ChapterInfo
    for _, item := range chaptersResp.Data {
        // Extract chapter information
        chapterInfo := core.ChapterInfo{
            ID:    item.ID,
            Title: item.Attributes.Title,
            Date:  item.Attributes.PublishAt,
        }

        // Convert chapter number
        if item.Attributes.Chapter != "" {
            if num, err := strconv.ParseFloat(item.Attributes.Chapter, 64); err == nil {
                chapterInfo.Number = num
            }
        }

        chapters = append(chapters, chapterInfo)
    }

    // Determine if there are more pages
    hasMore := len(chaptersResp.Data) >= chaptersResp.Limit && 
               chaptersResp.Offset+len(chaptersResp.Data) < chaptersResp.Total

    return chapters, hasMore, nil
}
```

### Step 5: Configure Extractors

Extractors map API responses to domain models:

```go
// configureExtractors sets up mapping between API responses and domain models
func configureExtractors() map[string]parser.ExtractorSet {
    extractorSets := make(map[string]parser.ExtractorSet)

    // Manga extractor
    extractorSets["manga"] = parser.ExtractorSet{
        Name:  "MyServiceManga",
        Model: &core.MangaInfo{},
        Extractors: []parser.Extractor{
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
    extractorSets["search"] = parser.ExtractorSet{
        Name:  "MyServiceSearch",
        Model: &core.Manga{},
        Extractors: []parser.Extractor{
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

    // Chapter extractor (if needed)
    extractorSets["chapter"] = parser.ExtractorSet{
        Name:  "MyServiceChapter",
        Model: &core.Chapter{},
        Extractors: []parser.Extractor{
            // Define chapter extractors
        },
    }

    return extractorSets
}
```

## Implementing Web-based Providers

For sites without a structured API, use the Web framework for HTML scraping:

### Step 1: Create Provider File and Basic Setup

```go
package providers

import (
    "Luminary/pkg/engine"
    "Luminary/pkg/engine/core"
    "Luminary/pkg/engine/network"
    "Luminary/pkg/engine/parser"
    "Luminary/pkg/provider"
    "Luminary/pkg/provider/web"
    "context"
    "fmt"
    "net/url"
    "strings"
    "time"
)

// MyScraperProvider implements a scraping-based provider
type MyScraperProvider struct {
    *web.Provider    // Embed the base web provider
    engine        *engine.Engine
}

// NewMyScraperProvider creates a new scraper-based provider
func NewMyScraperProvider(e *engine.Engine) provider.Provider {
    // Create HTML provider config
    config := web.Config{
        ID:          "msc",
        Name:        "My Scraper",
        SiteURL:     "https://myscraper.com",
        Description: "My manga scraper site",
        Headers: map[string]string{
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
        },
    }

    // Create base HTML provider
    htmlProvider := web.NewProvider(e, config)

    // Create and return our custom provider
    return &MyScraperProvider{
        Provider: htmlProvider,
        engine:   e,
    }
}
```

### Step 2: Implement Search Method

```go
// Search implements manga search via web scraping
func (p *MyScraperProvider) Search(ctx context.Context, query string, options core.SearchOptions) ([]core.Manga, error) {
    p.engine.Logger.Info("[%s] Searching for: %s", p.ID(), query)
    
    // Create search URL
    searchURL := fmt.Sprintf("%s/search?q=%s", p.SiteURL(), url.QueryEscape(query))
    
    // Apply options
    if options.Limit > 0 {
        searchURL = fmt.Sprintf("%s&limit=%d", searchURL, options.Limit)
    }
    
    // Fetch search page
    req := p.CreateRequest(searchURL)
    page, err := p.FetchPage(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch search page: %w", err)
    }
    
    // Extract manga items using CSS selectors
    var results []core.Manga
    
    mangaElements, err := page.Find(".manga-item")
    if err != nil || len(mangaElements) == 0 {
        // Try alternative selectors
        mangaElements, err = page.Find(".book-item, .item-list")
        if err != nil {
            return nil, fmt.Errorf("failed to find manga elements: %w", err)
        }
    }
    
    for _, elem := range mangaElements {
        // Extract title and link
        titleElem, err := elem.FindOne(".manga-title a, .title a")
        if err != nil || titleElem == nil {
            continue
        }
        
        title := titleElem.Text()
        href := titleElem.Attr("href")
        if title == "" || href == "" {
            continue
        }
        
        // Extract ID from URL
        id := network.ExtractPathFromURL(href)
        
        // Create manga object
        manga := core.Manga{
            ID:    id,
            Title: title,
        }
        
        // Extract other details if available
        coverElem, _ := elem.FindOne(".cover img, .thumb img")
        if coverElem != nil {
            manga.Cover = coverElem.Attr("src")
        }
        
        // Add to results list
        results = append(results, manga)
        
        // Apply limit if specified
        if options.Limit > 0 && len(results) >= options.Limit {
            break
        }
    }
    
    p.engine.Logger.Info("[%s] Found %d manga for query: %s", p.ID(), len(results), query)
    return results, nil
}
```

### Step 3: Implement GetManga Method

```go
// GetManga retrieves manga details
func (p *MyScraperProvider) GetManga(ctx context.Context, id string) (*core.MangaInfo, error) {
    p.engine.Logger.Info("[%s] Getting manga details for: %s", p.ID(), id)
    
    // Create manga URL
    mangaURL := parser.UrlJoin(p.SiteURL(), id)
    
    // Fetch manga page
    req := p.CreateRequest(mangaURL)
    page, err := p.FetchPage(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch manga page: %w", err)
    }
    
    // Extract manga details using CSS selectors
    mangaInfo := &core.MangaInfo{
        Manga: core.Manga{
            ID: id,
        },
    }
    
    // Extract title
    titleElem, err := page.FindOne("h1.manga-title, .entry-title")
    if err == nil && titleElem != nil {
        mangaInfo.Title = strings.TrimSpace(titleElem.Text())
    }
    
    // Extract description
    descElem, err := page.FindOne(".manga-description, .summary-content")
    if err == nil && descElem != nil {
        mangaInfo.Description = strings.TrimSpace(descElem.Text())
    }
    
    // Extract authors
    authorElems, err := page.Find(".manga-authors a, .author-content a")
    if err == nil && len(authorElems) > 0 {
        for _, elem := range authorElems {
            author := strings.TrimSpace(elem.Text())
            if author != "" {
                mangaInfo.Authors = append(mangaInfo.Authors, author)
            }
        }
    }
    
    // Extract genres/tags
    genreElems, err := page.Find(".genres-content a, .genres a")
    if err == nil && len(genreElems) > 0 {
        for _, elem := range genreElems {
            genre := strings.TrimSpace(elem.Text())
            if genre != "" {
                mangaInfo.Tags = append(mangaInfo.Tags, genre)
            }
        }
    }
    
    // Extract status
    statusElem, err := page.FindOne(".manga-status, .status-content")
    if err == nil && statusElem != nil {
        mangaInfo.Status = strings.TrimSpace(statusElem.Text())
    }
    
    // Extract chapters
    chapterElems, err := page.Find(".chapter-list .chapter-item, .chapters li")
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
            chapterID := network.ExtractPathFromURL(href)
            
            // Get chapter title
            title := strings.TrimSpace(linkElem.Text())
            
            // Extract chapter number from title or URL
            chapterNumber := parser.ExtractChapterNumber(title)
            if chapterNumber == 0 {
                chapterNumber = parser.ExtractChapterNumber(chapterID)
            }
            
            // Create chapter info
            chapterInfo := core.ChapterInfo{
                ID:     chapterID,
                Title:  title,
                Number: chapterNumber,
            }
            
            // Try to extract date if available
            dateElem, err := elem.FindOne(".chapter-date, .date")
            if err == nil && dateElem != nil {
                dateText := strings.TrimSpace(dateElem.Text())
                // Try various date formats
                dateFormats := []string{
                    "January 2, 2006",
                    "Jan 2, 2006",
                    "2006-01-02",
                }
                
                for _, format := range dateFormats {
                    if parsed, err := time.Parse(format, dateText); err == nil {
                        chapterInfo.Date = parsed
                        break
                    }
                }
            }
            
            // If date parsing failed, use current time
            if chapterInfo.Date.IsZero() {
                chapterInfo.Date = time.Now()
            }
            
            mangaInfo.Chapters = append(mangaInfo.Chapters, chapterInfo)
        }
    }
    
    return mangaInfo, nil
}
```

### Step 4: Implement GetChapter Method

```go
// GetChapter retrieves chapter details
func (p *MyScraperProvider) GetChapter(ctx context.Context, chapterID string) (*core.Chapter, error) {
    p.engine.Logger.Info("[%s] Getting chapter details for: %s", p.ID(), chapterID)

    // Create a basic Chapter
    chapter := &core.Chapter{
        Info: core.ChapterInfo{
            ID:    chapterID,
            Title: "",
        },
    }

    // Extract chapter number from the ID
    chapter.Info.Number = parser.ExtractChapterNumber(chapterID)

    // Fetch the chapter page
    chapterURL := parser.UrlJoin(p.SiteURL(), chapterID)
    req := p.CreateRequest(chapterURL)
    page, err := p.FetchPage(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch chapter page: %w", err)
    }

    // Get chapter title
    titleElem, err := page.FindOne("h1.chapter-title, .entry-title")
    if err == nil && titleElem != nil {
        chapter.Info.Title = strings.TrimSpace(titleElem.Text())
    }

    // If still no title, use page title
    if chapter.Info.Title == "" {
        chapter.Info.Title = page.GetTitle()
    }

    // Get page images
    var pages []core.Page
    imageElems, err := page.Find(".chapter-content img, .reading-content img")
    if err == nil && len(imageElems) > 0 {
        for i, elem := range imageElems {
            // Try various attributes for image URL
            imageURL := elem.Attr("src")
            if imageURL == "" {
                imageURL = elem.Attr("data-src")
            }
            if imageURL == "" {
                continue
            }

            // Make absolute URL if needed
            if !strings.HasPrefix(imageURL, "http") {
                imageURL = parser.UrlJoin(p.SiteURL(), imageURL)
            }

            // Extract filename from URL
            urlParts := strings.Split(imageURL, "/")
            filename := urlParts[len(urlParts)-1]
            if filename == "" {
                filename = fmt.Sprintf("page_%03d.jpg", i+1)
            }

            // Create page
            page := core.Page{
                Index:    i,
                URL:      imageURL,
                Filename: filename,
            }

            pages = append(pages, page)
        }
    }

    // Set the pages
    chapter.Pages = pages

    // Try to extract the manga ID from the chapter URL
    parts := strings.Split(chapterID, "/")
    if len(parts) > 0 {
        chapter.MangaID = parts[0]
    }

    return chapter, nil
}
```

### Step 5: Implement Remaining Methods

```go
// TryGetMangaForChapter attempts to get manga info for a chapter
func (p *MyScraperProvider) TryGetMangaForChapter(ctx context.Context, chapterID string) (*core.Manga, error) {
    // Get the chapter to extract manga ID
    chapter, err := p.GetChapter(ctx, chapterID)
    if err != nil {
        return nil, err
    }

    // If manga ID is available in chapter
    if chapter.MangaID != "" {
        // Get manga details
        mangaInfo, err := p.GetManga(ctx, chapter.MangaID)
        if err != nil {
            return nil, err
        }
        return &mangaInfo.Manga, nil
    }

    return nil, fmt.Errorf("couldn't determine manga for chapter %s", chapterID)
}

// DownloadChapter downloads a chapter
// This uses the common implementation from the provider framework
```

## Implementing Madara-based Providers

For sites based on the Madara WordPress theme (common for many manga sites), use the specialized Madara framework:

```go
package providers

import (
    "Luminary/pkg/engine"
    "Luminary/pkg/provider"
    "Luminary/pkg/provider/madara"
)

// NewMadaraProvider creates a new Madara-based provider
func NewMadaraProvider(e *engine.Engine) provider.Provider {
    // Configure Madara provider
    config := madara.Config{
        ID:              "mmd",
        Name:            "My Madara Site",
        SiteURL:         "https://mymadarasite.com",
        Description:     "Read manga online at My Madara Site",
        MangaSelector:   "div.post-title h3 a, div.post-title h5 a",
        ChapterSelector: "li.wp-manga-chapter > a, .chapter-link",
        PageSelector:    "div.page-break source, div.page-break img, .reading-content img",
        Headers: map[string]string{
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
            "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
            "Referer": "https://mymadarasite.com/",
        },
        // Optional: Configure AJAX behavior for the site
        UseLegacyAjax: false,
        CustomLoadAction: "madara_load_more",
    }

    // Create and return the Madara provider
    return madara.NewProvider(e, config)
}
```

## Registering Your Provider

Once you've implemented your provider, register it with the engine in `cmd/luminary/main.go`:

```go
// In cmd/luminary/main.go
func registerProviders(e *engine.Engine) {
    // Register existing providers
    err := e.RegisterProvider(providers.NewMangadexProvider(e))
    if err != nil {
        e.Logger.Error("Failed to register MangaDex provider: %v", err)
    }
    
    // Register your new provider
    err = e.RegisterProvider(providers.NewMyServiceProvider(e))
    if err != nil {
        e.Logger.Error("Failed to register MyService provider: %v", err)
    }
}
```

![Separator](../../.github/assets/luminary-separator.png)

## Best Practices

### Error Handling

Use structured errors for proper context:

```go
if err != nil {
    return nil, &errors.ProviderError{
        ProviderID:   p.ID(),
        ResourceType: "manga",
        ResourceID:   id,
        Message:      "Failed to fetch manga page",
        Err:          err,
    }
}
```

### Logging

Use the engine's logger for consistent logging:

```go
p.engine.Logger.Info("[%s] Fetching manga: %s", p.ID(), id)
p.engine.Logger.Error("[%s] Failed to fetch: %v", p.ID(), err)
```

### Rate Limiting

All frameworks apply rate limiting automatically. For custom implementations, use:

```go
domain := network.ExtractDomain(url)
p.engine.RateLimiter.Wait(domain)
```

### Robust Selectors

For HTML-based providers, use multiple selector options for resilience:

```go
titleSelectors := []string{".manga-title", "h1.title", ".entry-title"}
for _, selector := range titleSelectors {
    titleElem, err := page.FindOne(selector)
    if err == nil && titleElem != nil {
        title = titleElem.Text()
        if title != "" {
            break
        }
    }
}
```

### Context Handling

Always respect the provided context for cancellation and timeouts:

```go
select {
case <-ctx.Done():
    return nil, fmt.Errorf("operation cancelled: %w", ctx.Err())
default:
    // Continue operation
}
```

## Engine Services Reference

The Luminary engine provides several services that your provider can use:

| Service     | Purpose             | Key Methods                                    |
|-------------|---------------------|------------------------------------------------|
| HTTP        | HTTP requests       | `FetchWithRetries`, `FetchJSON`, `FetchString` |
| DOM         | HTML parsing        | `QuerySelector`, `QuerySelectorAll`, `GetText` |
| WebScraper  | Web scraping        | `FetchPage`, `Find`, `FindOne`                 |
| API         | API requests        | `FetchFromAPI`                                 |
| Download    | File downloads      | `DownloadChapter`                              |
| RateLimiter | Rate limiting       | `Wait`, `SetLimit`                             |
| Logger      | Logging             | `Info`, `Error`, `Debug`, `Warn`               |
| Extractor   | Data mapping        | `Extract`, `ExtractList`                       |
| Pagination  | Paginated requests  | `FetchAllPages`                                |
| Search      | Search handling     | `ExecuteSearch`, `SearchAcrossProviders`       |
| Parser      | Regex and utilities | `ExtractChapterNumber`, `CompilePattern`       |

![Separator](../../.github/assets/luminary-separator.png)

## Examples

### API-based Provider: MangaDex

The MangaDex provider (`internal/providers/mangadex.go`) implements an API-based provider that:
- Defines API endpoints and response types
- Creates extractors for mapping JSON to domain models
- Handles pagination for chapter listing
- Implements custom processing for chapter pages

### Madara-based Provider: KissManga

The KissManga provider (`internal/providers/kissmanga.go`) shows how to quickly implement a provider for a Madara-based site by customizing the configuration:

```go
// NewMadaraProvider creates a new KissManga provider using the Madara framework
func NewMadaraProvider(e *engine.Engine) provider.Provider {
    // Configure KissManga-specific settings using the Madara framework
    config := madara.Config{
        ID:              "kmg",
        Name:            "KissManga",
        SiteURL:         "https://kissmanga.in",
        Description:     "Read manga online for free at KissManga with daily updates",
        MangaSelector:   "div.post-title h3 a, div.post-title h5 a",
        ChapterSelector: "li.wp-manga-chapter > a, .chapter-link, div.listing-chapters_wrap a, .wp-manga-chapter a",
        PageSelector:    "div.page-break source, div.page-break img, .reading-content img",
        Headers: map[string]string{
            "User-Agent":     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.102 Safari/537.36",
            "Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
            "Accept-Language": "en-US,en;q=0.9",
            "Connection":      "keep-alive",
            "Referer":         "https://kissmanga.in/",
        },
    }

    // Create the Madara provider with our custom configuration
    return madara.NewProvider(e, config)
}
```

By following this guide, you should be able to implement new providers for Luminary that integrate seamlessly with the engine and provide a consistent experience for users.
