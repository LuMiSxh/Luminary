![Luminary Banner](.github/assets/luminary-banner.png)

# Luminary

> A streamlined CLI tool for searching and downloading manga across multiple sources.

![Separator](.github/assets/luminary-separator.png)

## Installation

### Download Release

You can download pre-built binaries from the [releases page](https://github.com/lumisxh/Luminary/releases) on GitHub. Choose the appropriate binary for your operating system and architecture.

**NOTE: _Release binaries are not yet available_**

### Build from Source

```bash
# Clone the repository
git clone https://github.com/lumisxh/Luminary.git

# Build the application
cd Luminary
go build

# Optional: Add to your PATH
```

![Separator](.github/assets/luminary-separator.png)

## Features

### Multi-Source Support

Luminary connects to multiple manga sources through a unified interface, allowing you to search and download from different providers with the same commands.



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
luminary download agent:chapter-id

# Multiple chapters
luminary download agent:chapter-id-1 agent:chapter-id-2

# Configure download options
luminary download agent:chapter-id --output ./my-manga --format jpeg --concurrent 10
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
luminary agents
```



### List Manga from a Source

```bash
# List all manga
luminary list

# List manga from a specific agent
luminary list --agent mangadex
```

### Get Detailed Information

```bash
# Get manga details including all chapters
luminary info agent:manga-id
```



![Separator](.github/assets/luminary-separator.png)

## Technical Features

### Structured Error Handling

Luminary provides human-readable error messages with detailed debug information available when needed.

### Fast & Concurrent

Powered by Go's concurrency features, Luminary downloads manga efficiently with configurable concurrency limits.

### Extensible Agent System

A plugin-like architecture makes it easy to add support for new manga sources through the agent interface.

### Rate Limiting

Built-in rate limiting protects manga sources from excessive requests and prevents IP bans.

![Separator](.github/assets/luminary-separator.png)

## Development

This project is still in development. If you want to contribute, feel free to fork the repository and create a pull request. You can also open an issue if you find a bug or have a feature request.

### Adding a New Agent

Luminary supports adding new manga sources through its agent interface. For more information, see the [Agent Implementation Guide](agents/AGENT.md).

```go
// Example: Simplified agent implementation
func NewAgent(e *engine.Engine) engine.Agent {
    return &MyAgent{
        engine: e,
        config: MyAgentConfig{
            ID:          "mys",
            Name:        "My Source",
            Description: "My manga source description",
            SiteURL:     "https://mysource.com",
        },
    }
}
```
