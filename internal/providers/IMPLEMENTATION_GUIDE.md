# Luminary Provider Implementation Guide

This guide explains how to implement a new provider (connector) for Luminary, a manga downloader CLI application. Providers in Luminary are responsible for connecting to specific manga sources and implementing the logic for searching, retrieving, and downloading manga content.

![Separator](../../.github/assets/luminary-separator.png)

## Table of Contents

1. [Understanding the Provider System](#understanding-the-provider-system)
2. [Provider Interface](#provider-interface)
3. [Unified Provider Framework](#unified-provider-framework)
4. [Provider Types](#provider-types)
5. [Auto-Registration System](#auto-registration-system)
6. [Implementation Examples](#implementation-examples)
7. [Best Practices](#best-practices)
8. [Engine Services Reference](#engine-services-reference)
9. [Testing Your Provider](#testing-your-provider)

## Understanding the Provider System

Luminary uses a plugin-like architecture where each manga source is implemented as a provider. The engine provides common utilities and services that providers can use, including:

- Network requests with retries and rate limiting
- HTML and JSON parsing
- File downloading
- Error handling and logging

Providers implement source-specific logic to integrate with these services, while the engine handles the heavy lifting.

## Provider Interface

All providers must implement the `Provider` interface defined in `pkg/engine/provider.go`:

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

While you could implement this interface from scratch, Luminary provides a unified base provider that handles most of the common functionality, allowing you to focus on the source-specific details.

## Unified Provider Framework

Luminary's provider framework is built around the `base.Provider` struct which implements the Provider interface with sensible defaults. The framework allows you to either use the defaults or override specific methods based on your needs.

### Creating a Provider with the Base Framework

The simplest way to create a new provider is to use the `base.New()` builder function:

```go
provider := base.New(engine, base.Config{
    // Basic identity
    ID:          "xyz",              // Short identifier (2-3 chars)
    Name:        "XYZ Manga",        // Display name
    Description: "XYZ manga source", // Description
    SiteURL:     "https://xyzmanga.com",
    
    // Provider type (determines default behavior)
    Type:        base.TypeWeb,  // TypeAPI, TypeWeb, or TypeMadara
    
    // Type-specific configuration
    Web: &base.WebConfig{
        // Web scraping configuration
    },
    
    // Common settings
    Headers: map[string]string{
        "User-Agent": "Luminary/1.0",
    },
    RateLimit: 2 * time.Second,
}).Build()
```

### Configuration Options

The `base.Config` struct contains all the configuration options for a provider:

```go
type Config struct {
    // Identity
    ID          string
    Name        string
    Description string
    SiteURL     string

    // Provider type determines default behavior
    Type Type  // TypeAPI, TypeWeb, or TypeMadara

    // Configuration based on type
    API    *APIConfig    // For TypeAPI
    Web    *WebConfig    // For TypeWeb
    Madara *MadaraConfig // For TypeMadara

    // Common settings
    Headers   map[string]string
    RateLimit time.Duration
    Timeout   time.Duration
}
```

### Customizing Provider Behavior

For more complex providers, you can override specific methods using the builder pattern:

```go
provider := base.New(engine, config).
    WithSearch(customSearchFunction).
    WithGetManga(customGetMangaFunction).
    WithGetChapter(customGetChapterFunction).
    Build()
```

## Provider Types

Luminary supports three main provider types, each with its own default behavior:

### 1. API-based Providers (`TypeAPI`)

For sites with structured JSON APIs (like MangaDex). Configure using `APIConfig`:

```go
APIConfig struct {
    BaseURL        string
    Endpoints      map[string]string
    ResponseMapping map[string]ResponseMap
}
```

### 2. Web-based Providers (`TypeWeb`)

For sites that require HTML scraping. Configure using `WebConfig`:

```go
WebConfig struct {
    SearchPath     string
    MangaPath      string
    Selectors      map[string]string
    FilterSupport  bool
    SupportedFilters map[string]bool
}
```

### 3. Madara-based Providers (`TypeMadara`)

For sites using the Madara WordPress theme (common for manga sites). Configure using `MadaraConfig`:

```go
MadaraConfig struct {
    Selectors       map[string]string
    AjaxSearch      bool
    CustomLoadAction string
}
```

## Auto-Registration System

Luminary uses an auto-registration system to discover and register providers:

1. Create an `init()` function in your provider file
2. Call `registry.Register(YourProviderConstructor)` inside `init()`
3. The provider will be automatically registered when Luminary starts

```go
func init() {
    registry.Register(NewMyProvider)
}

func NewMyProvider(e *engine.Engine) engine.Provider {
    // Create and return your provider
}
```

## Implementation Examples

### Simple Madara-based Provider (KissManga)

```go
package providers

import (
    "Luminary/pkg/engine"
    "Luminary/pkg/provider/base"
    "Luminary/pkg/provider/registry"
    "time"
)

func init() {
    registry.Register(NewKissMangaProvider)
}

func NewKissMangaProvider(e *engine.Engine) engine.Provider {
    return base.New(e, base.Config{
        ID:          "kmg",
        Name:        "KissManga",
        Description: "Read manga online for free at KissManga with daily updates",
        SiteURL:     "https://kissmanga.in",
        Type:        base.TypeMadara,

        Madara: &base.MadaraConfig{
            Selectors: map[string]string{
                "search":      "div.post-title h3 a, div.post-title h5 a",
                "title":       "h1.post-title, .post-title-font",
                "description": ".description-summary, .summary__content",
                "chapters":    "li.wp-manga-chapter > a, .chapter-link",
                "pages":       "div.page-break img, .reading-content img",
                "author":      ".author-content a, .manga-authors a",
                "status":      ".post-status .summary-content",
                "genres":      ".genres-content a",
            },
            AjaxSearch:       true,
            CustomLoadAction: "madara_load_more",
        },

        Headers: map[string]string{
            "User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
            "Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
            "Accept-Language": "en-US,en;q=0.9",
            "Referer":         "https://kissmanga.in/",
        },

        RateLimit: 2 * time.Second,
    }).Build()
}
```

### Custom API Provider (MangaDex)

For sources with unique APIs, you can override default behaviors using the builder pattern:

```go
package providers

import (
    "Luminary/pkg/core"
    "Luminary/pkg/engine"
    "Luminary/pkg/provider/base"
    "Luminary/pkg/provider/registry"
    "context"
    "time"
)

func init() {
    registry.Register(NewMangaDexProvider)
}

func NewMangaDexProvider(e *engine.Engine) engine.Provider {
    b := base.New(e, base.Config{
        ID:      "mgd",
        Name:    "MangaDex",
        SiteURL: "https://mangadex.org",
        Type:    base.TypeAPI,
        API: &base.APIConfig{
            BaseURL: "https://api.mangadex.org",
        },
        RateLimit: 1 * time.Second,
    })

    // Build the base provider to get an instance `p` to pass to custom functions
    p := b.Build().(*base.Provider)

    // Override default operations with custom logic
    return b.WithSearch(customMangaDexSearch(p)).
             WithGetManga(customMangaDexGetManga(p)).
             WithGetChapter(customMangaDexGetChapter(p)).
             Build()
}

// customMangaDexSearch provides the implementation for the Search operation.
func customMangaDexSearch(p *base.Provider) func(context.Context, string, core.SearchOptions) ([]core.Manga, error) {
    return func(ctx context.Context, query string, options core.SearchOptions) ([]core.Manga, error) {
        // ... custom implementation using p.Engine.Network ...
    }
}
// ... other custom functions ...
```

![Separator](../../.github/assets/luminary-separator.png)

## Best Practices

### Error Handling

Use Luminary's structured error system for proper context and tracking:

```go
if err != nil {
    return nil, errors.Track(err).
        WithContext("provider", p.ID()).
        AsProvider(p.ID()).
        Error()
}
```

### Logging

Use the engine's logger for consistent logging:

```go
p.Engine.Logger.Info("Fetching manga: %s", id)
p.Engine.Logger.Error("Failed to fetch: %v", err)
```

### Rate Limiting

The base provider applies rate limiting automatically based on your configuration.

### Robust Selectors

For HTML-based providers, use multiple selector options for resilience:

```go
// In WebConfig
Selectors: map[string]string{
    "title": "h1.manga-title, h1.title, .entry-title",  // Try multiple selectors
    // ... other selectors
}
```

### Context Handling

Always respect the provided context for cancellation and timeouts:

```go
select {
case <-ctx.Done():
    return nil, errors.Track(ctx.Err()).WithContext("operation", "search").Error()
default:
    // Continue operation
}
```

## Engine Services Reference

The Luminary engine provides four primary services that your provider can use:

| Service  | Purpose                             | Key Methods                                             |
|----------|-------------------------------------|--------------------------------------------------------|
| Network  | HTTP requests with retries          | `Request`, `Get`, `Post`, `JSON`                        |
| Parser   | HTML and JSON parsing               | `ParseHTML`, `ParseJSON`, `ExtractChapterNumber`        |
| Download | Concurrent file downloading         | `DownloadChapter`, `DownloadFile`, `BatchDownload`      |
| Logger   | Application-wide structured logging | `Info`, `Debug`, `Error`, `Warn`                        |

## Testing Your Provider

1. Implement your provider in a new file under `internal/providers/`
2. Register it with `registry.Register(NewYourProvider)` in an `init()` function
3. Run Luminary and test your provider with basic commands:

```bash
# List providers (your provider should appear)
luminary providers

# Search using your provider
luminary search "manga title" --provider your-provider-id

# Get manga details
luminary info your-provider-id:manga-id

# Download a chapter
luminary download your-provider-id:chapter-id
```

By following this guide, you should be able to implement new providers for Luminary that integrate seamlessly with the engine and provide a consistent experience for users.
