# Luminary API Mode Guide

This guide covers how to use Luminary's `--api` mode, which provides machine-readable JSON output instead of human-readable text. This is useful for integration with other tools, scripts, or applications.

![Separator](.github/assets/luminary-separator.png)

## Overview

API mode is enabled by adding the `--api` flag to any Luminary command:

```bash
luminary --api [command] [options]
```

All API responses follow a standardized JSON structure:

```json
{
  "status": "success|error|downloading",
  "data": { /* command-specific data */ },
  "error": "error message if status is error"
}
```

## Global Flags

All commands support these global flags in API mode:

- `--api`: Enable API mode (machine-readable JSON output)
- `--concurrency <number>`: Set maximum concurrent operations (default: 5)

![Separator](.github/assets/luminary-separator.png)

## Commands

### Version Command

Get version information in JSON format.

**Usage:**
```bash
luminary --api version
```

**Output:**
```json
{
  "status": "success",
  "data": {
    "version": "0.0.0-dev",
    "go_version": "go1.24.2",
    "os": "darwin",
    "arch": "arm64",
    "log_file": "/Users/username/.luminary/logs/luminary.log"
  }
}
```

**Fields:**
- `version`: Luminary version string
- `go_version`: Go compiler version used to build
- `os`: Operating system (linux, windows, darwin for macOS)
- `arch`: Architecture (amd64, arm64, etc.)
- `log_file`: Path to log file or "disabled" if logging is disabled

---

### Providers Command

List all available manga source providers.

**Usage:**
```bash
luminary --api providers
```

**Output:**
```json
{
  "status": "success",
  "data": {
    "providers": [
      {
        "id": "mgd",
        "name": "MangaDex",
        "description": "World's largest manga community and scanlation site"
      },
      {
        "id": "kmg",
        "name": "KissManga",
        "description": "Read manga online for free at KissManga with daily updates"
      }
    ]
  }
}
```

**Fields:**
- `providers`: Array of provider objects
    - `id`: Short provider identifier (used in commands)
    - `name`: Human-readable provider name
    - `description`: Provider description

---

### Search Command

Search for manga across providers.

**Usage:**
```bash
luminary --api search "manga title" [options]
```

**Options:**
- `--provider <id>`: Search specific provider only
- `--limit <number>`: Maximum results per page (default: 10)
- `--pages <number>`: Number of pages to fetch (default: 1)
- `--sort <method>`: Sort method (relevance, name, newest, updated)
- `--fields <fields>`: Comma-separated fields to search (title, author, genre)
- `--filter <filters>`: Field-specific filters (e.g., genre=fantasy,author=oda)
- `--alt-titles`: Include alternative titles in search (default: true)
- `--all-langs`: Search across all language versions (default: true)

**Output:**
```json
{
  "status": "success",
  "data": {
    "query": "one piece",
    "results": [
      {
        "id": "mgd:manga-id-123",
        "title": "One Piece",
        "provider": "mgd",
        "provider_name": "MangaDex",
        "alt_titles": ["ワンピース", "Wan Pīsu"],
        "authors": ["Oda Eiichiro"],
        "tags": ["Action", "Adventure", "Comedy", "Drama", "Shounen"]
      }
    ],
    "count": 1
  }
}
```

**Fields:**
- `query`: The search query used
- `results`: Array of manga search results
    - `id`: Combined provider:manga-id format
    - `title`: Main manga title
    - `provider`: Provider ID
    - `provider_name`: Provider display name
    - `alt_titles`: Alternative titles (optional)
    - `authors`: List of authors (optional)
    - `tags`: List of genres/tags (optional)
- `count`: Total number of results returned

---

### List Command

List all available manga from providers.

**Usage:**
```bash
luminary --api list [options]
```

**Options:**
- `--provider <id>`: List from specific provider only
- `--limit <number>`: Results per page (default: 50)
- `--pages <number>`: Number of pages to fetch (default: 1, 0 for all)

**Output:**
```json
{
  "status": "success",
  "data": {
    "count": 6,
    "mangas": [
      {
        "id": "mgd:200e4c75-ce67-4dbf-8f91-a502f49c20e9",
        "title": "Ironking",
        "provider": "mgd",
        "provider_name": "MangaDex"
      },
      {
        "id": "kmg:kissmanga/saving-the-world-through-a-game",
        "title": "Saving the World Through a Game",
        "provider": "kmg",
        "provider_name": "KissManga"
      }
    ]
  }
}
```

**With Provider Filter:**
```json
{
  "status": "success",
  "data": {
    "count": 2,
    "mangas": [
      {
        "id": "mgd:0b20b128-d7e1-47c0-a01d-8122db8f8cf2",
        "title": "Yayakoshii Mikan-tachi",
        "provider": "mgd",
        "provider_name": "MangaDex"
      }
    ],
    "provider": "mgd",
    "provider_name": "MangaDex"
  }
}
```

**Fields:**
- `mangas`: Array of manga items
    - `id`: Combined provider:manga-id format
    - `title`: Manga title
    - `provider`: Provider ID
    - `provider_name`: Provider display name
- `count`: Total number of manga returned
- `provider`: Provider ID (only when filtering by provider)
- `provider_name`: Provider name (only when filtering by provider)

---

### Info Command

Get detailed information about a specific manga.

**Usage:**
```bash
luminary --api info provider:manga-id
```

**Output:**
```json
{
  "status": "success",
  "data": {
    "manga": {
      "id": "mgd:manga-123",
      "title": "Manga Title",
      "provider": "mgd",
      "provider_name": "MangaDex",
      "description": "Manga description...",
      "authors": ["Author Name"],
      "status": "ongoing",
      "tags": ["Action", "Adventure"],
      "chapters": [
        {
          "id": "mgd:chapter-456",
          "title": "Chapter 1: Beginning",
          "number": 1.0,
          "date": "2024-01-15T10:30:00Z"
        }
      ],
      "chapter_count": 1
    }
  }
}
```

**Fields:**
- `manga`: Detailed manga object
    - `id`: Combined provider:manga-id format
    - `title`: Manga title
    - `provider`: Provider ID
    - `provider_name`: Provider display name
    - `description`: Manga description/summary
    - `authors`: Array of author names
    - `status`: Publication status (ongoing, completed, etc.)
    - `tags`: Array of genres/tags
    - `chapters`: Array of chapter objects
        - `id`: Combined provider:chapter-id format
        - `title`: Chapter title
        - `number`: Chapter number (float for decimals like 1.5)
        - `date`: Publication date in ISO 8601 format
    - `chapter_count`: Total number of chapters

---

### Download Command

Download manga chapters.

**Usage:**
```bash
luminary --api download provider:chapter-id [provider:chapter-id...] [options]
```

**Options:**
- `--output <directory>`: Output directory (default: ./downloads)
- `--vol <number>`: Set or override volume number

**Output (Initial):**
```json
{
  "status": "downloading",
  "data": {
    "chapter_id": "074ec297-9185-4ca0-bec1-46d2bc6963d5",
    "provider": "mgd",
    "provider_name": "MangaDex",
    "output_dir": "./tmp"
  }
}
```

**Output (Success):**
```json
{
  "status": "success",
  "data": {
    "chapter_id": "074ec297-9185-4ca0-bec1-46d2bc6963d5",
    "message": "Successfully downloaded chapter 074ec297-9185-4ca0-bec1-46d2bc6963d5",
    "provider": "mgd",
    "output_dir": "./tmp"
  }
}
```

**Output (Error):**
```json
{
  "status": "error",
  "error": "Provider [mgd] error: chapter with ID 'chapter-id' - Chapter has no pages to download: invalid input"
}
```

**Fields (Downloading):**
- `chapter_id`: Chapter identifier
- `provider`: Provider ID
- `provider_name`: Provider display name
- `output_dir`: Directory where files will be saved
- `volume`: Volume number (optional, only if --vol flag used)

**Fields (Success):**
- `chapter_id`: Downloaded chapter ID
- `message`: Success message
- `provider`: Provider ID
- `output_dir`: Directory where files were saved

**Note:** The download command outputs two separate JSON objects:
1. First, a "downloading" status to indicate the download has started
2. Then, either a "success" or "error" status with the final result

The `volume` field only appears in the downloading status when the `--vol` flag is used.

![Separator](.github/assets/luminary-separator.png)

## Error Handling

All commands can return error responses when something goes wrong:

```json
{
  "status": "error",
  "error": "Error message describing what went wrong"
}
```

**Common Error Types:**
- Provider not found
- Manga/chapter not found
- Network/server errors
- Rate limiting errors
- Invalid input format
- File system errors (for download)

**Example Error Responses:**
```json
{
  "status": "error",
  "error": "provider 'invalid' not found"
}
```

```json
{
  "status": "error",
  "error": "manga 'some-manga-id' not found on MangaDex"
}
```

```json
{
  "status": "error", 
  "error": "Provider [mgd] error: chapter with ID 'chapter-id' - Chapter has no pages to download: invalid input"
}
```

![Separator](.github/assets/luminary-separator.png)

## Integration Examples

### Python Integration

```python
import subprocess
import json
from typing import List, Dict, Optional, Any
from dataclasses import dataclass
from datetime import datetime

@dataclass
class Manga:
    id: str
    title: str
    provider: str
    provider_name: str

@dataclass
class Chapter:
    id: str
    title: str
    number: int
    date: datetime

@dataclass
class MangaInfo:
    id: str
    title: str
    provider: str
    provider_name: str
    description: str
    authors: List[str]
    status: str
    tags: List[str]
    chapters: List[Chapter]
    chapter_count: int

class LuminaryClient:
    def __init__(self, luminary_path: str = "luminary", concurrency: int = 5):
        self.luminary_path = luminary_path
        self.concurrency = concurrency
    
    def _run_command(self, args: List[str]) -> Dict[str, Any]:
        """Execute a luminary command and return parsed JSON response."""
        cmd = [self.luminary_path, "--api", "--concurrency", str(self.concurrency)] + args
        result = subprocess.run(cmd, capture_output=True, text=True)
        
        if result.returncode != 0:
            raise Exception(f"Command failed: {result.stderr}")
        
        response = json.loads(result.stdout)
        
        if response["status"] == "error":
            raise Exception(f"Luminary error: {response['error']}")
        
        return response["data"]
    
    def search(self, query: str, provider: Optional[str] = None, limit: int = 10) -> List[Manga]:
        """Search for manga."""
        args = ["search", query, "--limit", str(limit)]
        if provider:
            args.extend(["--provider", provider])
        
        data = self._run_command(args)
        
        return [
            Manga(
                id=result["id"],
                title=result["title"],
                provider=result["provider"],
                provider_name=result["provider_name"]
            )
            for result in data["results"]
        ]
    
    def get_manga_info(self, manga_id: str) -> MangaInfo:
        """Get detailed manga information."""
        data = self._run_command(["info", manga_id])
        manga = data["manga"]
        
        chapters = [
            Chapter(
                id=ch["id"],
                title=ch["title"],
                number=ch["number"],
                date=datetime.fromisoformat(ch["date"].replace('Z', '+00:00'))
            )
            for ch in manga["chapters"]
        ]
        
        return MangaInfo(
            id=manga["id"],
            title=manga["title"],
            provider=manga["provider"],
            provider_name=manga["provider_name"],
            description=manga["description"],
            authors=manga["authors"],
            status=manga["status"],
            tags=manga["tags"],
            chapters=chapters,
            chapter_count=manga["chapter_count"]
        )
    
    def download_chapter(self, chapter_id: str, output_dir: str = "./downloads") -> str:
        """Download a chapter and return success message."""
        args = ["download", chapter_id, "--output", output_dir]
        
        # For download, we need to handle the streaming JSON responses
        cmd = [self.luminary_path, "--api", "--concurrency", str(self.concurrency)] + args
        process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
        
        responses = []
        for line in process.stdout:
            if line.strip():
                try:
                    response = json.loads(line.strip())
                    responses.append(response)
                    
                    if response["status"] == "error":
                        raise Exception(f"Download failed: {response['error']}")
                    elif response["status"] == "success":
                        return response["data"]["message"]
                except json.JSONDecodeError:
                    continue
        
        process.wait()
        if process.returncode != 0:
            raise Exception(f"Download command failed: {process.stderr.read()}")
        
        return "Download completed"
    
    def list_providers(self) -> List[Dict[str, str]]:
        """List all available providers."""
        data = self._run_command(["providers"])
        return data["providers"]

# Usage examples
if __name__ == "__main__":
    client = LuminaryClient()
    
    try:
        # Search for manga
        print("Searching for 'naruto'...")
        results = client.search("naruto", provider="mgd", limit=5)
        for manga in results:
            print(f"- {manga.title} ({manga.provider_name})")
        
        if results:
            # Get detailed info for first result
            print(f"\nGetting info for: {results[0].title}")
            manga_info = client.get_manga_info(results[0].id)
            print(f"Description: {manga_info.description[:100]}...")
            print(f"Status: {manga_info.status}")
            print(f"Chapters: {manga_info.chapter_count}")
            
            # Download first chapter (uncomment to test)
            # if manga_info.chapters:
            #     chapter = manga_info.chapters[0]
            #     print(f"\nDownloading: {chapter.title}")
            #     message = client.download_chapter(chapter.id)
            #     print(message)
        
        # List providers
        print("\nAvailable providers:")
        providers = client.list_providers()
        for provider in providers:
            print(f"- {provider['name']} ({provider['id']}): {provider['description']}")
    
    except Exception as e:
        print(f"Error: {e}")
```

### Rust Integration

```rust
use serde::{Deserialize, Serialize};
use std::process::Command;
use std::io::{BufRead, BufReader};
use chrono::{DateTime, Utc};

#[derive(Debug, Deserialize)]
struct ApiResponse<T> {
    status: String,
    data: Option<T>,
    error: Option<String>,
}

#[derive(Debug, Deserialize, Serialize)]
struct Manga {
    id: String,
    title: String,
    provider: String,
    provider_name: String,
}

#[derive(Debug, Deserialize)]
struct SearchData {
    query: String,
    count: u32,
    results: Vec<Manga>,
}

#[derive(Debug, Deserialize)]
struct Chapter {
    id: String,
    title: String,
    number: i32,
    date: DateTime<Utc>,
}

#[derive(Debug, Deserialize)]
struct MangaInfo {
    id: String,
    title: String,
    provider: String,
    provider_name: String,
    description: String,
    authors: Vec<String>,
    status: String,
    tags: Vec<String>,
    chapters: Vec<Chapter>,
    chapter_count: u32,
}

#[derive(Debug, Deserialize)]
struct MangaInfoData {
    manga: MangaInfo,
}

#[derive(Debug, Deserialize)]
struct Provider {
    id: String,
    name: String,
    description: String,
}

#[derive(Debug, Deserialize)]
struct ProvidersData {
    providers: Vec<Provider>,
}

#[derive(Debug, Deserialize)]
struct DownloadData {
    chapter_id: String,
    message: Option<String>,
    provider: String,
    output_dir: String,
}

pub struct LuminaryClient {
    luminary_path: String,
    concurrency: u32,
}

impl LuminaryClient {
    pub fn new(luminary_path: Option<String>, concurrency: Option<u32>) -> Self {
        Self {
            luminary_path: luminary_path.unwrap_or_else(|| "luminary".to_string()),
            concurrency: concurrency.unwrap_or(5),
        }
    }
    
    fn run_command(&self, args: &[&str]) -> Result<serde_json::Value, Box<dyn std::error::Error>> {
        let mut cmd = Command::new(&self.luminary_path);
        cmd.args(&["--api", "--concurrency", &self.concurrency.to_string()])
           .args(args);
        
        let output = cmd.output()?;
        
        if !output.status.success() {
            return Err(format!("Command failed: {}", String::from_utf8_lossy(&output.stderr)).into());
        }
        
        let response: ApiResponse<serde_json::Value> = serde_json::from_slice(&output.stdout)?;
        
        match response.status.as_str() {
            "success" => Ok(response.data.unwrap_or(serde_json::Value::Null)),
            "error" => Err(response.error.unwrap_or("Unknown error".to_string()).into()),
            _ => Err("Unexpected response status".into()),
        }
    }
    
    pub fn search(&self, query: &str, provider: Option<&str>, limit: Option<u32>) -> Result<Vec<Manga>, Box<dyn std::error::Error>> {
        let mut args = vec!["search", query];
        
        let limit_str = limit.unwrap_or(10).to_string();
        args.extend(&["--limit", &limit_str]);
        
        if let Some(p) = provider {
            args.extend(&["--provider", p]);
        }
        
        let data = self.run_command(&args)?;
        let search_data: SearchData = serde_json::from_value(data)?;
        
        Ok(search_data.results)
    }
    
    pub fn get_manga_info(&self, manga_id: &str) -> Result<MangaInfo, Box<dyn std::error::Error>> {
        let data = self.run_command(&["info", manga_id])?;
        let manga_data: MangaInfoData = serde_json::from_value(data)?;
        
        Ok(manga_data.manga)
    }
    
    pub fn download_chapter(&self, chapter_id: &str, output_dir: Option<&str>) -> Result<String, Box<dyn std::error::Error>> {
        let mut args = vec!["download", chapter_id];
        
        if let Some(dir) = output_dir {
            args.extend(&["--output", dir]);
        }
        
        let mut cmd = Command::new(&self.luminary_path);
        cmd.args(&["--api", "--concurrency", &self.concurrency.to_string()])
           .args(&args)
           .stdout(std::process::Stdio::piped());
        
        let mut child = cmd.spawn()?;
        
        if let Some(stdout) = child.stdout.take() {
            let reader = BufReader::new(stdout);
            
            for line in reader.lines() {
                let line = line?;
                if let Ok(response) = serde_json::from_str::<ApiResponse<DownloadData>>(&line) {
                    match response.status.as_str() {
                        "error" => return Err(response.error.unwrap_or("Download failed".to_string()).into()),
                        "success" => {
                            if let Some(data) = response.data {
                                return Ok(data.message.unwrap_or("Download completed".to_string()));
                            }
                        },
                        "downloading" => continue, // Wait for final result
                        _ => continue,
                    }
                }
            }
        }
        
        child.wait()?;
        Ok("Download completed".to_string())
    }
    
    pub fn list_providers(&self) -> Result<Vec<Provider>, Box<dyn std::error::Error>> {
        let data = self.run_command(&["providers"])?;
        let providers_data: ProvidersData = serde_json::from_value(data)?;
        
        Ok(providers_data.providers)
    }
}

// Usage example
fn main() -> Result<(), Box<dyn std::error::Error>> {
    let client = LuminaryClient::new(None, Some(5));
    
    // Search for manga
    println!("Searching for 'naruto'...");
    let results = client.search("naruto", Some("mgd"), Some(5))?;
    
    for manga in &results {
        println!("- {} ({})", manga.title, manga.provider_name);
    }
    
    if let Some(first_manga) = results.first() {
        // Get detailed info
        println!("\nGetting info for: {}", first_manga.title);
        let manga_info = client.get_manga_info(&first_manga.id)?;
        
        println!("Description: {}...", &manga_info.description[..100.min(manga_info.description.len())]);
        println!("Status: {}", manga_info.status);
        println!("Chapters: {}", manga_info.chapter_count);
        
        // Download first chapter (uncomment to test)
        // if let Some(chapter) = manga_info.chapters.first() {
        //     println!("\nDownloading: {}", chapter.title);
        //     let message = client.download_chapter(&chapter.id, Some("./downloads"))?;
        //     println!("{}", message);
        // }
    }
    
    // List providers
    println!("\nAvailable providers:");
    let providers = client.list_providers()?;
    for provider in providers {
        println!("- {} ({}): {}", provider.name, provider.id, provider.description);
    }
    
    Ok(())
}
```

**Dependencies for Rust (Cargo.toml):**
```toml
[dependencies]
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
chrono = { version = "0.4", features = ["serde"] }
```

![Separator](.github/assets/luminary-separator.png)

## Notes

- All timestamps are in ISO 8601 format (UTC)
- Chapter numbers can be integers or floats to support decimal chapters (e.g., 1.5)
- Provider IDs are short codes (2-3 characters) for efficiency
- Combined IDs use the format `provider:id` for consistency
- Array fields may be empty `[]` if no data is available
- Optional fields may be omitted from responses if not available
- The download command outputs two separate JSON responses in sequence
