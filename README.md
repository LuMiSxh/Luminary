# Luminary

A streamlined CLI tool for searching and downloading manga across multiple sources.

## Overview

Luminary helps you find and download manga from different providers through a simple command-line interface. Whether you're searching by title, author, or genre, Luminary provides a unified way to access your favorite manga.

## Features

- **Search**: Find manga by title, author, genre, or any combination
- **Multiple Sources**: Access manga from various agents/providers
- **Download**: Save chapters directly to your device for offline reading
- **JSON Output**: Machine-readable output for integration with other tools

## Installation

```bash
# Clone the repository
git clone https://github.com/lumisxh/Luminary.git

# Build the application
cd Luminary
go build

# Optional: Add to your PATH
```

### Download Release

You can also download pre-built binaries from the [releases page]()
on GitHub. Choose the appropriate binary for your operating system and architecture.

** NOTE: _This is not yet available_ **

## Usage

### Search for Manga

```bash
# Basic search
luminary search "one piece"

# Search with specific fields
luminary search "dragon" --fields title,author

# Apply filters
luminary search "adventure" --filter genre=fantasy,author="specific author"

# Control results
luminary search "manga title" --limit 20 --sort popularity
```

### List Available Manga

```bash
# List all manga
luminary list

# List manga from a specific agent
luminary list --agent mangadex
```

### Download Chapters

```bash
# Download a chapter
luminary download agent:chapter-id

# Multiple chapters
luminary download agent:chapter-id-1 agent:chapter-id-2

# Configure download options
luminary download agent:chapter-id --output ./my-manga --format jpeg --concurrent 10
```

### API Mode

All commands support machine-readable JSON output:

```bash
luminary --api search "manga title"
```
