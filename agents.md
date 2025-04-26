# Adding a New Agent to Luminary

This guide walks through the process of adding a new agent to Luminary, a manga downloader/reader application. An agent
in Luminary represents a connector to a specific manga source or website.

## Understanding the Agent Interface

Before starting, familiarize yourself with the `Agent` interface defined in `engine/types.go`:

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

Your agent must implement all these methods.

## Step 1: Create a New Package

Create a new package for your agent in a directory that matches the service name inside the `agents` folder.

Example:

```
/
├── engine/
└── agents/
    └── myservice/
        └── agent.go
```

## Step 2: Define Your Agent Struct

Define a struct for your agent that will implement the `Agent` interface.

```go
package myservice

import (
    "Luminary/engine"
    "context"
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
        id:            "myservice",  // Short identifier, typically 2-3 letters
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

## Step 3: Implement Required Methods

Implement the basic identification methods required by the interface:

```go
// ID returns the agent's identifier
func (m *MyService) ID() string {
    return m.id
}

// Name returns the agent's display name
func (m *MyService) Name() string {
    return m.name
}

// Description returns the agent's description
func (m *MyService) Description() string {
    return m.description
}

// SiteURL returns the agent's website URL
func (m *MyService) SiteURL() string {
    return m.siteURL
}

// Initialize initializes the agent
func (m *MyService) Initialize(ctx context.Context) error {
    return engine.ExecuteInitialize(ctx, m.engine, m.id, m.name, m.onInitialize)
}

// onInitialize performs service-specific initialization
func (m *MyService) onInitialize(ctx context.Context) error {
    // Implement any initialization logic here
    // For example, fetching token, validating connection, etc.
    return nil
}
```

## Step 4: Configure API Endpoints

Define the API endpoints for your service. This includes the base URL, endpoints, and how to format requests.

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
                ResponseType:  &MangaResponse{}, // Define this response type
                PathFormatter: engine.DefaultPathFormatter,
            },

            // Chapter details endpoint
            "chapter": {
                Path:          "/chapter/{id}",
                Method:        "GET",
                ResponseType:  &ChapterResponse{}, // Define this response type
                PathFormatter: engine.DefaultPathFormatter,
            },

            // Search endpoint
            "search": {
                Path:         "/search",
                Method:       "GET",
                ResponseType: &SearchResponse{}, // Define this response type
                QueryFormatter: func (params interface{}) url.Values {
                    queryParams := url.Values{}

                    // Handle search options
                    if opts, ok := params.(*engine.SearchOptions); ok {
                        if opts.Query != "" {
                            queryParams.Set("q", opts.Query)
                        }

                        // Set limit
                        if opts.Limit > 0 {
                            queryParams.Set("limit", strconv.Itoa(opts.Limit))
                        } else {
                            queryParams.Set("limit", "20") // Default
                        }

                        // Add other parameters as needed
                    }

                    return queryParams
                },
            },

            // Add other endpoints as needed
        },
    }
}
```

Next, define the response types for your API:

```go
// Response types
type MangaResponse struct {
    // Define fields that match your API's JSON response
    // Example:
    ID          string `json:"id"`
    Title       string `json:"title"`
    Description string `json:"description"`
    // ... other fields
}

type ChapterResponse struct {
    // Define fields that match your API's JSON response
    // Example:
    ID        string `json:"id"`
    Title     string `json:"title"`
    Number    string `json:"number"`
    // ... other fields
}

type SearchResponse struct {
    // Define fields that match your API's JSON response
    // Example:
    Results []struct {
        ID    string `json:"id"`
        Title string `json:"title"`
        // ... other fields
    } `json:"results"`
    Total int `json:"total"`
}
```

## Step 5: Configure Data Extractors

Configure extractors to map API responses to Luminary's domain models:

```go
// configureExtractors sets up the data extractors
func (m *MyService) configureExtractors() {
    // Manga extractor set
    m.extractorSets["manga"] = engine.ExtractorSet{
        Name:  "MyServiceManga",
        Model: &engine.MangaInfo{},
        Extractors: []engine.Extractor{
            {
                Name:       "ID",
                SourcePath: []string{"id"},
                TargetPath: "ID",
                Required:   true,
            },
            {
                Name:       "Title",
                SourcePath: []string{"title"},
                TargetPath: "Title",
                Required:   true,
            },
            {
                Name:       "Description",
                SourcePath: []string{"description"},
                TargetPath: "Description",
                Required:   false,
            },
            {
                Name:       "Cover",
                SourcePath: []string{"cover_url"},
                TargetPath: "Cover",
                Required:   false,
            },
            {
                Name:       "Authors",
                SourcePath: []string{"authors"},
                TargetPath: "Authors",
                Transform: func (value interface{}) interface{} {
                    // Transform the authors data if needed
                    // For example, converting from a different format
                    return value
                },
                Required: false,
            },
            // Add more extractors as needed
        },
    }

    // Search results extractor
    m.extractorSets["search"] = engine.ExtractorSet{
        Name:  "MyServiceSearchResults",
        Model: &engine.Manga{},
        Extractors: []engine.Extractor{
            {
                Name:       "ID",
                SourcePath: []string{"id"},
                TargetPath: "ID",
                Required:   true,
            },
            {
                Name:       "Title",
                SourcePath: []string{"title"},
                TargetPath: "Title",
                Required:   true,
            },
            // Add more extractors as needed
        },
    }

    // Chapter extractor
    m.extractorSets["chapter"] = engine.ExtractorSet{
        Name:  "MyServiceChapter",
        Model: &engine.Chapter{},
        Extractors: []engine.Extractor{
            {
                Name:       "ID",
                SourcePath: []string{"id"},
                TargetPath: "Info.ID",
                Required:   true,
            },
            {
                Name:       "Title",
                SourcePath: []string{"title"},
                TargetPath: "Info.Title",
                Required:   false,
            },
            {
                Name:       "Number",
                SourcePath: []string{"number"},
                TargetPath: "Info.Number",
                Transform: func (value interface{}) interface{} {
                    // Convert string to float64 if needed
                    if strVal, ok := value.(string); ok {
                        if numVal, err := strconv.ParseFloat(strVal, 64); err == nil {
                            return numVal
                        }
                    }
                    return 0.0
                },
                Required: false,
            },
            // Add more extractors as needed
        },
    }
}
```

## Step 6: Implement Search Functionality

Implement the `Search` method to search for manga on your service:

```go
// Search implements the engine.Agent interface for searching
func (m *MyService) Search(ctx context.Context, query string, options engine.SearchOptions) ([]engine.Manga, error) {
    // Use the engine helper with appropriate configuration
    return engine.ExecuteSearch(
        ctx,
        m.engine,
        m.id,
        query,
        options,
        m.apiConfig,
        engine.PaginationConfig{
            LimitParam:     "limit",
            OffsetParam:    "offset",
            TotalCountPath: []string{"total"},
            ItemsPath:      []string{"results"},
            DefaultLimit:   20,
            MaxLimit:       100,
        },
        m.extractorSets["search"],
    )
}
```

## Step 7: Implement Manga Retrieval

Implement the `GetManga` method to retrieve manga details:

```go
// GetManga implements the engine.Agent interface for retrieving manga details
func (m *MyService) GetManga(ctx context.Context, id string) (*engine.MangaInfo, error) {
    // Use the engine helper with appropriate configuration
    return engine.ExecuteGetManga(
        ctx,
        m.engine,
        m.id,
        id,
        m.apiConfig,
        m.extractorSets["manga"],
        func(ctx context.Context, mangaID string) ([]engine.ChapterInfo, error) {
            return m.fetchChaptersForManga(ctx, mangaID)
        },
    )
}

// fetchChaptersForManga fetches all chapters for a manga
func (m *MyService) fetchChaptersForManga(ctx context.Context, mangaID string) ([]engine.ChapterInfo, error) {
    // Implement fetching chapters for the given manga ID
    // This might involve additional API calls

    // Example implementation:
    response, err := m.engine.API.FetchFromAPI(
        ctx,
        m.apiConfig,
        "chapters", // Make sure this endpoint is defined in configureAPIEndpoints
        nil,
        mangaID,
    )
    if err != nil {
        return nil, fmt.Errorf("failed to fetch chapters: %w", err)
    }

    // Process the response and convert to ChapterInfo objects
    // ...

    return chapterInfoList, nil
}
```

## Step 8: Implement Chapter Retrieval

Implement the `GetChapter` method to retrieve chapter details:

```go
// GetChapter implements the engine.Agent interface for retrieving chapter details
func (m *MyService) GetChapter(ctx context.Context, chapterID string) (*engine.Chapter, error) {
    // Use the engine helper with customized processing function
    return engine.ExecuteGetChapter(
        ctx,
        m.engine,
        m.id,
        chapterID,
        m.apiConfig,
        m.extractorSets["chapter"],
        m.processChapterResponse,
    )
}

// processChapterResponse is a custom handler for chapter API responses
func (m *MyService) processChapterResponse(response interface{}, chapterID string) (*engine.Chapter, error) {
    // Cast to expected response type
    chapterResp, ok := response.(*ChapterResponse)
    if !ok {
        return nil, fmt.Errorf("unexpected response type: %T", response)
    }

    // Create a new chapter object
    chapter := &engine.Chapter{
        Info: engine.ChapterInfo{
            ID: chapterID,
        },
    }

    // Extract chapter information
    chapter.Info.Title = chapterResp.Title

    // Convert chapter number if present
    if chapterResp.Number != "" {
        if num, err := strconv.ParseFloat(chapterResp.Number, 64); err == nil {
            chapter.Info.Number = num
        }
    }

    // Set manga ID if available
    chapter.MangaID = chapterResp.MangaID

    // Fetch pages using a separate API call or from the current response
    // ...

    // Add pages to the chapter
    chapter.Pages = []engine.Page{
        // Example:
        {
            Index:    0,
            URL:      "https://myservice.com/images/chapter1/page1.jpg",
            Filename: "page1.jpg",
        },
        // Add more pages...
    }

    return chapter, nil
}
```

## Step 9: Implement Manga Retrieval for Chapters

Implement the `TryGetMangaForChapter` method:

```go
// TryGetMangaForChapter attempts to get manga info for a chapter
func (m *MyService) TryGetMangaForChapter(ctx context.Context, chapterID string) (*engine.Manga, error) {
    // Fetch chapter details first to get manga ID
    chapter, err := m.GetChapter(ctx, chapterID)
    if err != nil {
        return nil, err
    }

    // If manga ID is available in chapter
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
```

## Step 10: Implement Chapter Download

Implement the `DownloadChapter` method:

```go
// DownloadChapter implements the engine.Agent interface for downloading a chapter
func (m *MyService) DownloadChapter(ctx context.Context, chapterID, destDir string) error {
    // Use the engine helper for downloading
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

## Step 11: Register Your Agent

Finally, register your agent with the engine in your application's initialization:

```go
func RegisterAgents(engine *engine.Engine) {
    // Register existing agents
    // ...

    // Register your new agent
    if err := engine.RegisterAgent(agents.myservice.NewAgent(engine)); err != nil {
        engine.Logger.Error("Failed to register MyService agent: %v", err)
    }
}
```

## Testing Your Agent

To test your agent, focus on these areas:

1. **Search functionality**: Verify that searching returns expected results.
2. **Manga retrieval**: Check that manga details are correctly retrieved.
3. **Chapter retrieval**: Ensure chapters can be retrieved with their pages.
4. **Chapter download**: Confirm that chapters can be downloaded correctly.

Example testing code:

```go
func TestMyServiceAgent(t *testing.T) {
    // Create engine
    e := engine.New()

    // Create and register agent
    agent := agents.myservice.NewAgent(e)
    e.RegisterAgent(agent)

    // Initialize agent
    ctx := context.Background()
    err := agent.Initialize(ctx)
    if err != nil {
        t.Fatalf("Failed to initialize agent: %v", err)
    }

    // Test search
    results, err := agent.Search(ctx, "test query", engine.SearchOptions{Limit: 5})
    if err != nil {
        t.Fatalf("Search failed: %v", err)
    }

    if len(results) == 0 {
        t.Fatalf("Expected search results, got none")
    }

    // Test manga retrieval
    manga, err := agent.GetManga(ctx, results[0].ID)
    if err != nil {
        t.Fatalf("Failed to get manga: %v", err)
    }

    if manga.Title == "" {
        t.Fatalf("Expected manga title, got empty string")
    }

    // Add more tests as needed
}
```

## Conclusion

By following this guide, you've created a new agent for Luminary that connects to your manga service. The agent
leverages Luminary's engine components to handle common tasks like API communication, rate limiting, and data
extraction, while you focus on implementing the service-specific logic.

Remember to test your agent thoroughly to ensure it correctly interacts with your service's API and conforms to
Luminary's expected data models.
