# Luminary JSON-RPC Mode Guide

This guide covers how to use Luminary's JSON-RPC interface, provided by the `luminary-rpc` executable. This interface
allows for robust, programmatic integration with other tools, scripts, or applications by communicating over standard
input (stdin) and standard output (stdout) using the JSON-RPC 2.0 protocol.

![Separator](.github/assets/luminary-separator.png)

## Overview

The JSON-RPC interface is accessed by running the `luminary-rpc` executable:

```bash
./luminary-rpc
```

Once started, `luminary-rpc` listens for JSON-RPC 2.0 requests on its stdin and sends JSON-RPC 2.0 responses to its
stdout. Each request and response must be a single line of JSON.

### JSON-RPC 2.0 Request Format

A typical request to `luminary-rpc` will look like this:

```json
{
  "method": "ServiceName.MethodName",
  "params": [
    args_object
  ],
  "id": request_id
}
```

- `method`: A string containing the name of the service and method to be called (e.g., `"SearchService.Search"`).
- `params`: An array containing a single object with the arguments for the method.
- `id`: A unique identifier for the request, which will be included in the response. Can be a string, number, or null.

### JSON-RPC 2.0 Response Format

A successful response will look like this:

```json
{
  "result": response_data,
  "error": null,
  "id": request_id
}
```

- `result`: The data returned by the method call. The structure depends on the method.
- `error`: `null` for successful calls.
- `id`: The `id` from the original request.

An error response will look like this:

```json
{
  "result": null,
  "error": {
    "code": error_code,
    "message": "error_message"
  },
  "id": request_id
}
```

- `result`: `null` for error calls.
- `error`: An object containing:
    - `code`: An integer error code (standard JSON-RPC error codes or custom).
    - `message`: A string describing the error.
- `id`: The `id` from the original request, or `null` if the request `id` could not be determined.

![Separator](.github/assets/luminary-separator.png)

## Services and Methods

### VersionService

Provides version information about the Luminary application and its environment.

#### `VersionService.Get`

Retrieves version information.

**Request Parameters (`args_object`):**
An empty object `{}`.

**Example Request:**

```json
{
  "method": "VersionService.Get",
  "params": [
    {}
  ],
  "id": 1
}
```

**Response Data (`response_data`):**

```json
{
  "version": "0.0.0-dev",
  "go_version": "go1.24.2",
  "os": "darwin",
  "arch": "arm64",
  "log_file": "/Users/username/.luminary/logs/luminary.log"
}
```

**Fields:**

- `version`: Luminary version string.
- `go_version`: Go compiler version used to build.
- `os`: Operating system (e.g., "linux", "windows", "darwin").
- `arch`: Architecture (e.g., "amd64", "arm64").
- `log_file`: Path to the log file, or "disabled" if logging is not to a file.

---

### ProvidersService

Lists available manga source providers.

#### `ProvidersService.List`

Retrieves a list of all configured manga providers.

**Request Parameters (`args_object`):**
An empty object `{}`.

**Example Request:**

```json
{
  "method": "ProvidersService.List",
  "params": [
    {}
  ],
  "id": 2
}
```

**Response Data (`response_data`):**
An array of provider information objects.

```json
[
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
```

**Fields (per provider object):**

- `id`: Short unique identifier for the provider.
- `name`: Human-readable name of the provider.
- `description`: A brief description of the provider.

---

### SearchService

Searches for manga across one or all providers.

#### `SearchService.Search`

Performs a manga search.

**Request Parameters (`args_object`):**

```json
{
  "query": "search term",
  "provider": "optional_provider_id",
  // Optional: "mgd", "kmg", etc. If omitted, searches all.
  "limit": 10,
  // Optional: Max results per page (default: 10)
  "pages": 1,
  // Optional: Number of pages to fetch (default: 1)
  "sort": "relevance",
  // Optional: "relevance", "name", "newest", "updated"
  "include_alt_titles": true,
  // Optional: Include alternative titles (default: false)
  "concurrency": 5
  // Optional: Max concurrent operations for this search (default: 5)
}
```

**Example Request:**

```json
{
  "method": "SearchService.Search",
  "params": [
    {
      "query": "one piece",
      "provider": "mgd",
      "limit": 5
    }
  ],
  "id": 3
}
```

**Response Data (`response_data`):**

```json
{
  "query": "one piece",
  "results": [
    {
      "id": "mgd:manga-id-123",
      "title": "One Piece",
      "provider": "mgd",
      "provider_name": "MangaDex",
      "alt_titles": [
        "ワンピース",
        "Wan Pīsu"
      ],
      "authors": [
        "Oda Eiichiro"
      ],
      "tags": [
        "Action",
        "Adventure",
        "Comedy",
        "Drama",
        "Shounen"
      ]
    }
  ],
  "count": 1
}
```

**Fields:**

- `query`: The search query used.
- `results`: An array of `SearchResultItem` objects.
    - `id`: Combined provider ID and manga ID (e.g., "mgd:manga-id-123").
    - `title`: Main title of the manga.
    - `provider`: ID of the provider.
    - `provider_name`: Display name of the provider.
    - `alt_titles`: Array of alternative titles (optional).
    - `authors`: Array of author names (optional).
    - `tags`: Array of genre/tag strings (optional).
- `count`: Total number of results returned.

---

### ListService

Lists manga from providers, typically without a specific search query.

#### `ListService.Latest`

Retrieves a list of the latest manga.

**Request Parameters (`args_object`):**

```json
{
  "provider": "optional_provider_id",
  // Optional: If omitted, lists from all.
  "limit": 50,
  // Optional: Max results (default: 50)
  "page": 1
  // Optional: Page number (default: 1)
}
```

**Example Request (All Providers):**

```json
{
  "method": "ListService.Latest",
  "params": [
    {
      "limit": 2
    }
  ],
  "id": 4
}
```

**Response Data (`response_data`):**

```json
{
  "results": [
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
  ],
  "count": 2,
  "provider": "",
  // Empty if multiple providers
  "provider_name": "Multiple Providers"
  // Or specific provider name if filtered
}
```

**Fields:**

- `results`: An array of `ListItem` objects with basic manga information.
- `count`: Total number of manga returned.
- `provider`: Provider ID (empty if listing from all, or the specific provider ID if filtered).
- `provider_name`: "Multiple Providers" or the specific provider name if filtered.

---

### InfoService

Retrieves detailed information about a specific manga, with optional language filtering.

#### `InfoService.Get`

Fetches details for a manga given its combined ID, with optional chapter language filtering.

**Request Parameters (`args_object`):**

```json
{
  "manga_id": "provider_id:manga_specific_id",
  // e.g., "mgd:manga-id-123"
  "language_filter": "en,ja",
  // Optional: Comma-separated language codes/names to filter chapters
  "show_languages": true
  // Optional: Include available languages in response (default: false)
}
```

**Example Request (Basic):**

```json
{
  "method": "InfoService.Get",
  "params": [
    {
      "manga_id": "mgd:manga-id-123"
    }
  ],
  "id": 5
}
```

**Example Request (With Language Filter):**

```json
{
  "method": "InfoService.Get",
  "params": [
    {
      "manga_id": "mgd:manga-id-123",
      "language_filter": "en,ja",
      "show_languages": true
    }
  ],
  "id": 5
}
```

**Response Data (`response_data`):**

```json
{
  "id": "mgd:manga-id-123",
  "title": "Manga Title",
  "provider": "mgd",
  "provider_name": "MangaDex",
  "description": "Manga description...",
  "authors": [
    "Author Name"
  ],
  "status": "ongoing",
  "tags": [
    "Action",
    "Adventure"
  ],
  "chapters": [
    {
      "id": "mgd:chapter-456",
      "title": "Chapter 1: Beginning",
      "number": 1.0,
      "date": "2024-01-15T10:30:00Z",
      // ISO 8601 format, nullable (null if unavailable)
      "language": "en"
      // ISO 639-1 language code, nullable (null if unavailable)
    },
    {
      "id": "mgd:chapter-457",
      "title": "第1話: 始まり",
      "number": 1.0,
      "date": "2024-01-15T10:30:00Z",
      "language": "ja"
    }
  ],
  "chapter_count": 2,
  "last_updated": "2024-04-10T12:00:00Z",
  // Nullable date in ISO 8601 format
  "available_languages": [
    "en",
    "ja",
    "es",
    "fr"
  ],
  // Optional: Only included if show_languages=true
  "filtered_chapters": true,
  // Optional: Only included if language filtering was applied
  "original_chapter_count": 15
  // Optional: Only included if chapters were filtered
}
```

**Fields:**

- `id`: Combined provider ID and manga ID.
- `title`: Manga title.
- `provider`: Provider ID.
- `provider_name`: Provider display name.
- `description`: Manga description/summary.
- `authors`: Array of author names.
- `status`: Publication status (e.g., "ongoing", "completed").
- `tags`: Array of genres/tags.
- `chapters`: Array of `ChapterInfo` objects (filtered by language if `language_filter` was specified).
    - `id`: Combined provider ID and chapter ID (e.g., "mgd:chapter-456").
    - `title`: Chapter title.
    - `number`: Chapter number (float, e.g., 1.0, 1.5).
    - `date`: Publication date in ISO 8601 format (null if unavailable).
    - `language`: Language code in ISO 639-1 format (e.g., "en", "ja", "fr") (null if unavailable).
- `chapter_count`: Total number of chapters returned (after filtering, if applied).
- `last_updated`: When the manga was last updated (null if unavailable).
- `available_languages`: Array of all available language codes for this manga (only included if `show_languages=true`).
- `filtered_chapters`: Boolean indicating whether the chapters list was filtered by language (only included if filtering
  was applied).
- `original_chapter_count`: Original number of chapters before language filtering (only included if filtering was
  applied).

#### Language Filtering

The `language_filter` parameter accepts:

- **Language codes** (ISO 639-1): `"en"`, `"ja"`, `"es"`, `"fr"`, `"de"`, `"pt"`, `"ru"`, `"ko"`, `"zh"`, `"it"`, etc.
- **Extended codes**: `"zh-cn"`, `"zh-tw"`, `"zh-hk"`, `"pt-br"`, `"es-la"`
- **Language names** (case-insensitive): `"english"`, `"japanese"`, `"spanish"`, `"french"`, etc.
- **Special values**: `"unknown"` or `"none"` for chapters with no specified language
- **Multiple languages**: Comma-separated list, e.g., `"en,ja"` or `"english,japanese"`

**Language Filter Examples:**

```json
// Filter by English only
{
  "manga_id": "mgd:123",
  "language_filter": "en"
}

// Filter by English and Japanese
{
  "manga_id": "mgd:123",
  "language_filter": "en,ja"
}

// Filter by language names
{
  "manga_id": "mgd:123",
  "language_filter": "english,japanese"
}

// Show available languages without filtering
{
  "manga_id": "mgd:123",
  "show_languages": true
}

// Filter by English and show all available languages
{
  "manga_id": "mgd:123",
  "language_filter": "en",
  "show_languages": true
}

// Include chapters with unknown language
{
  "manga_id": "mgd:123",
  "language_filter": "en,unknown"
}
```

---

### DownloadService

Handles downloading of manga chapters.

#### `DownloadService.Chapter`

Initiates the download of a specific manga chapter.

**Request Parameters (`args_object`):**

```json
{
  "chapter_id": "provider_id:chapter_specific_id",
  // e.g., "mgd:chapter-456"
  "output_dir": "./downloads"
  // Optional: Default is "./downloads"
}
```

**Example Request:**

```json
{
  "method": "DownloadService.Chapter",
  "params": [
    {
      "chapter_id": "mgd:chapter-456",
      "output_dir": "./my_manga"
    }
  ],
  "id": 6
}
```

**Response Data (`response_data`):**

```json
{
  "success": true,
  "message": "Chapter downloaded successfully",
  "path": "./my_manga",
  "page_count": 24
}
```

**Fields:**

- `success`: Boolean indicating if the download was successful.
- `message`: A status message (can include error details if `success` is `false`).
- `path`: Directory where the chapter was saved.
- `page_count`: Number of pages downloaded (optional).

**Note:** If the download fails, `success` will be `false`, and the `message` field will contain the error. The RPC call
itself will still be a "successful" JSON-RPC response unless there's a fundamental issue with the request format or
server. The business logic error is conveyed within the `result` payload.

![Separator](.github/assets/luminary-separator.png)

## Error Handling

When an error occurs, the `error` field in the JSON-RPC response will be non-null. The `error.message` field will contain a descriptive string.

```json
{
  "result": null,
  "error": {
    "code": -32603,
    "message": "rpc: service/method request ill-formed: Providers.NonExistentMethod"
  },
  "id": 1
}
```

For application-level errors, Luminary's advanced error system provides detailed, formatted messages.

**Example (Provider Not Found):**
```json
{
  "result": null,
  "error": {
    "code": 0,
    "message": "[PROVIDER] provider 'invalid-id' not found"
  },
  "id": 5
}
```
The client can parse this message to understand the error category (e.g., `[PROVIDER]`). If the server were run in a debug mode (not currently supported via RPC), the message would contain a full stack trace and context.

![Separator](.github/assets/luminary-separator.png)

## Notes

- All communication is line-delimited JSON. Each request and each response must be on a single line.
- Ensure the `luminary-rpc` executable is in your PATH or provide the full path to it.
- The `id` field in requests is crucial for matching responses.
- Timestamps (like chapter dates) are generally in ISO 8601 format (UTC).
- Provider and entity IDs are case-sensitive.
- Optional fields in request parameters can be omitted.
- Optional fields in responses may be absent if no data is available.
- Chapter dates and languages will be null when the information is not available.
- Language codes follow the ISO 639-1 standard (two-letter codes like "en", "ja", "fr").
- Language filtering is case-insensitive and supports both codes and full language names.
- When using language filtering, the `chapters` array will only contain chapters matching the specified languages.
- The `available_languages` field is only included when `show_languages=true` is specified in the request.
- Language filtering can significantly reduce the number of chapters returned, which is useful for manga with many
  translations.

This RPC interface provides a more structured and robust way to integrate Luminary's functionalities compared to parsing
CLI output.
