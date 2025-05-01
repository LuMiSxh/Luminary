![Luminary Banner](.github/assets/luminary-banner.png)

# Luminary

> A streamlined CLI tool for searching and downloading manga across multiple sources.

![Separator](.github/assets/luminary-separator.png)

## Installation

### Download Release

You can download pre-built binaries from the [releases page](https://github.com/lumisxh/Luminary/releases) on GitHub.
Choose the appropriate binary for your operating system and architecture.

### Build from Source

```bash
# Clone the repository
git clone https://github.com/lumisxh/Luminary.git

# Build the application
cd Luminary
go build ./cmd/luminary

# Optional: Add to your PATH
```

![Separator](.github/assets/luminary-separator.png)

## Features

### Multi-Source Support

Luminary connects to multiple manga sources through a unified interface, allowing you to search and download from
different providers with the same commands.

### Powerful Search Capabilities

Find manga by title, author, genre, or any combination with advanced filtering options.

```bash
# Basic search
luminary search "one piece"

# Search with specific fields
luminary search "dragon" --fields title,author

# Apply filters
luminary search "adventure" --filter genre=fantasy,author="eiichiro oda"

# Control results
luminary search "manga title" --limit 20 --sort popularity
```

### Efficient Downloading

Download chapters directly to your device with configurable options for concurrency and file format.

```bash
# Download a chapter
luminary download provider:chapter-id

# Multiple chapters
luminary download provider:chapter-id-1 provider:chapter-id-2

# Configure download options
luminary download provider:chapter-id --output ./my-manga --format jpeg --concurrent 10
```

### CLI & API Modes

All commands support both interactive human-readable output and machine-readable JSON for integration with other tools.

```bash
# Interactive mode
luminary search "manga title"

# API mode
luminary --api search "manga title"
```

![Separator](.github/assets/luminary-separator.png)

## Usage Examples

### List Available Manga Sources

```bash
luminary providers
```

### List Manga from a Source

```bash
# List all manga
luminary list

# List manga from a specific provider (e.g., "mgd" for MangaDex)
luminary list --provider mgd
```

### Get Detailed Information

```bash
# Get manga details including all chapters
luminary info provider:manga-id
```

### Download Manga

```bash
# Download a specific chapter
luminary download provider:chapter-id
```

![Separator](.github/assets/luminary-separator.png)

## Technical Features

### Structured Error Handling

Luminary provides human-readable error messages with detailed debug information available when needed.

### Fast & Concurrent

Powered by Go's concurrency features, Luminary downloads manga efficiently with configurable concurrency limits.

### Extensible Provider System

A plugin-like architecture makes it easy to add support for new manga sources through the provider interface.

### Rate Limiting

Built-in rate limiting protects manga sources from excessive requests and prevents IP bans.

![Separator](.github/assets/luminary-separator.png)

## Development

This project is still in heavy development.
If you want to contribute, feel free to fork the repository and create a pull
request.
You can also open an issue if you find a bug or have a feature request.

### Adding a New Provider

Luminary supports adding new manga sources through its provider interface. For more information, see
the [Provider Implementation Guide](internal/providers/provider.md).

```go
// Example: Simplified provider implementation
func NewProvider(e *engine.Engine) provider.Provider {
    return &MyProvider{
        engine: e,
        config: MyProviderConfig{
            ID:          "mys",
            Name:        "My Source",
            Description: "My manga source description",
            SiteURL:     "https://mysource.com",
        },
    }
}
```
