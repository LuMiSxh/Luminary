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
  "fields": [
    "title",
    "author"
  ],
  // Optional: Fields to search in
  "filters": {
    "genre": "action"
  },
  // Optional: Field-specific filters
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

#### `ListService.List`

Retrieves a list of manga.

**Request Parameters (`args_object`):**

```json
{
  "provider": "optional_provider_id",
  // Optional: If omitted, lists from all.
  "limit": 50,
  // Optional: Max results (default: 50)
  "pages": 1,
  // Optional: Number of pages (default: 1, 0 for all if supported by provider)
  "concurrency": 5
  // Optional: Max concurrent operations (default: 5)
}
```

**Example Request (All Providers):**

```json
{
  "method": "ListService.List",
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
  "mangas": [
    {
      "id": "mgd:200e4c75-ce67-4dbf-8f91-a502f49c20e9",
      "title": "Ironking",
      "provider": "mgd",
      "provider_name": "MangaDex",
      "authors": [
        "Author A"
      ],
      "tags": [
        "Tag1"
      ]
    },
    {
      "id": "kmg:kissmanga/saving-the-world-through-a-game",
      "title": "Saving the World Through a Game",
      "provider": "kmg",
      "provider_name": "KissManga",
      "authors": [
        "Author B"
      ],
      "tags": [
        "Tag2"
      ]
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

- `mangas`: An array of `SearchResultItem` objects (same structure as in `SearchService.Search` results).
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

#### `DownloadService.Download`

Initiates the download of a specific manga chapter.

**Request Parameters (`args_object`):**

```json
{
  "chapter_id": "provider_id:chapter_specific_id",
  // e.g., "mgd:chapter-456"
  "output_dir": "./downloads",
  // Optional: Default is "./downloads"
  "volume": 1,
  // Optional: Override volume number
  "concurrency": 5
  // Optional: Max concurrent page downloads (default: 5)
}
```

**Example Request:**

```json
{
  "method": "DownloadService.Download",
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
  "chapter_id": "chapter-456",
  // The chapter ID part (without provider prefix)
  "provider": "mgd",
  "provider_name": "MangaDex",
  "output_dir": "./my_manga",
  "success": true,
  "message": "Successfully downloaded chapter mgd:chapter-456 to ./my_manga"
}
```

**Fields:**

- `chapter_id`: The original chapter ID part (without the provider prefix).
- `provider`: Provider ID.
- `provider_name`: Provider display name.
- `output_dir`: Directory where the chapter was saved.
- `success`: Boolean indicating if the download was successful.
- `message`: A status message (can include error details if `success` is `false`).

**Note:** If the download fails, `success` will be `false`, and the `message` field will contain the error. The RPC call
itself will still be a "successful" JSON-RPC response unless there's a fundamental issue with the request format or
server. The business logic error is conveyed within the `result` payload.

![Separator](.github/assets/luminary-separator.png)

## Language Support

### Supported Language Codes

Luminary supports the following language codes and names for filtering:

**Common Language Codes (ISO 639-1):**

- `en` - English
- `ja` - Japanese
- `es` - Spanish
- `fr` - French
- `de` - German
- `pt` - Portuguese
- `ru` - Russian
- `ko` - Korean
- `zh` - Chinese
- `it` - Italian
- `ar` - Arabic
- `tr` - Turkish
- `th` - Thai
- `vi` - Vietnamese
- `id` - Indonesian
- `pl` - Polish
- `nl` - Dutch
- `sv` - Swedish
- `da` - Danish
- `no` - Norwegian
- `fi` - Finnish
- `hu` - Hungarian
- `cs` - Czech
- `sk` - Slovak
- `bg` - Bulgarian
- `hr` - Croatian
- `sr` - Serbian
- `sl` - Slovenian
- `et` - Estonian
- `lv` - Latvian
- `lt` - Lithuanian
- `ro` - Romanian
- `el` - Greek
- `he` - Hebrew
- `fa` - Persian
- `hi` - Hindi
- `bn` - Bengali
- `ta` - Tamil
- `te` - Telugu
- `ml` - Malayalam
- `kn` - Kannada
- `gu` - Gujarati
- `pa` - Punjabi
- `ur` - Urdu
- `uk` - Ukrainian

**Extended Language Codes:**

- `zh-cn` - Chinese (Simplified)
- `zh-tw` - Chinese (Traditional)
- `zh-hk` - Chinese (Hong Kong)
- `pt-br` - Portuguese (Brazil)
- `es-la` - Spanish (Latin America)

**Special Values:**

- `unknown` or `none` - Chapters with no specified language

### Language Filtering Examples

**Filter by single language:**

```json
{
  "method": "InfoService.Get",
  "params": [
    {
      "manga_id": "mgd:123",
      "language_filter": "en"
    }
  ],
  "id": 1
}
```

**Filter by multiple languages:**

```json
{
  "method": "InfoService.Get",
  "params": [
    {
      "manga_id": "mgd:123",
      "language_filter": "en,ja,es"
    }
  ],
  "id": 1
}
```

**Filter by language names:**

```json
{
  "method": "InfoService.Get",
  "params": [
    {
      "manga_id": "mgd:123",
      "language_filter": "english,japanese"
    }
  ],
  "id": 1
}
```

**Include unknown language chapters:**

```json
{
  "method": "InfoService.Get",
  "params": [
    {
      "manga_id": "mgd:123",
      "language_filter": "en,unknown"
    }
  ],
  "id": 1
}
```

**Show available languages:**

```json
{
  "method": "InfoService.Get",
  "params": [
    {
      "manga_id": "mgd:123",
      "show_languages": true
    }
  ],
  "id": 1
}
```

![Separator](.github/assets/luminary-separator.png)

## Error Handling

Luminary's JSON-RPC interface provides comprehensive error information with automatic function call tracking and error
classification.

### Standard JSON-RPC Errors

If a JSON-RPC request is malformed, or if an unrecoverable server-side error occurs before a specific service method can
process the business logic, a standard JSON-RPC error response is returned:

```json
{
  "result": null,
  "error": {
    "code": -32600,
    "message": "Invalid Request"
  },
  "id": "some_id"
}
{
  "result": null,
  "error": {
    "code": -32601,
    "message": "Method not found"
  },
  "id": "some_id"
}
{
  "result": null,
  "error": {
    "code": -32602,
    "message": "Invalid params"
  },
  "id": "some_id"
}
{
  "result": null,
  "error": {
    "code": -32700,
    "message": "Parse error"
  },
  "id": null
} 
```

### Enhanced Business Logic Errors

For errors specific to service method execution, Luminary returns enhanced error objects with detailed context and
automatic function call tracking:

```json
{
  "result": null,
  "error": {
    "code": -2004,
    "message": "Search failed for query 'one piece'",
    "data": {
      "query": "one piece",
      "service": "SearchService",
      "method": "Search"
    },
    "function_chain": "SearchService.Search() → ExecuteSearch() → MangadxProvider.Search() → HTTPService.DoRequest()",
    "call_details": [
      {
        "function": "SearchService.Search",
        "short_name": "SearchService.Search",
        "operation": "search",
        "timestamp": "2025-06-13T15:04:05.123Z",
        "file": "services.go",
        "line": 45
      },
      {
        "function": "MangadxProvider.Search",
        "short_name": "MangadxProvider.Search",
        "operation": "search",
        "timestamp": "2025-06-13T15:04:05.124Z",
        "file": "mangadx.go",
        "line": 89,
        "context": {
          "provider_id": "mangadx"
        }
      },
      {
        "function": "HTTPService.DoRequest",
        "short_name": "HTTPService.DoRequest",
        "operation": "http_request",
        "timestamp": "2025-06-13T15:04:05.125Z",
        "file": "client.go",
        "line": 156,
        "context": {
          "url": "https://api.mangadx.org/manga",
          "http_method": "GET",
          "status_code": 0
        }
      }
    ],
    "error_category": "network",
    "original_error": "dial tcp: no such host",
    "root_cause": "no such host",
    "timestamp": "2025-06-13T15:04:05.123Z",
    "service": "SearchService",
    "method": "Search"
  },
  "id": 7
}
```

### Enhanced Error Fields

**Core Fields:**

- `code`: Integer error code (see Error Codes section below)
- `message`: Human-readable error message
- `data`: Request context and additional data

**Function Tracking Fields:**

- `function_chain`: Human-readable function call path (e.g., "A() → B() → C()")
- `call_details`: Array of detailed function call information
    - `function`: Full function name with package path
    - `short_name`: Simplified function name (e.g., "ServiceName.Method")
    - `operation`: Operation being performed (search, download, http_request, etc.)
    - `timestamp`: When this function processed the error (ISO 8601)
    - `file`: Source file name where error was handled
    - `line`: Line number in source file
    - `context`: Function-specific context data

**Error Classification Fields:**

- `error_category`: Error type classification (see Categories section below)
- `original_error`: The original error message that started the chain
- `root_cause`: The deepest underlying error cause
- `timestamp`: When the error occurred
- `service`: RPC service name
- `method`: RPC method name

### Error Codes

Luminary uses structured error codes organized by category:

#### Input/Validation Errors (1000-1099)

- `-1001` - Invalid input data
- `-1002` - Invalid data format
- `-1003` - Validation failed

#### Resource/Provider Errors (1100-1199)

- `-1101` - Provider not found
- `-1102` - Resource (manga/chapter) not found
- `-1103` - Provider error

#### Operation Errors (1200-1299)

- `-1201` - Search failed
- `-1202` - Fetch failed
- `-1203` - Download failed
- `-1204` - List failed

#### Network Errors (2000-2099)

- `-2001` - Network unavailable
- `-2002` - Network timeout
- `-2003` - Connection failed
- `-2004` - DNS failure
- `-2005` - HTTP error

#### Timeout Errors (2100-2199)

- `-2101` - Operation timeout
- `-2102` - Deadline exceeded
- `-2103` - Context canceled

#### Authentication/Authorization (2200-2299)

- `-2201` - Authentication failed
- `-2202` - Rate limited
- `-2203` - Forbidden
- `-2204` - Unauthorized

#### Parsing/Data Errors (3000-3099)

- `-3001` - Parsing failed
- `-3002` - Invalid format
- `-3003` - Data corrupted
- `-3004` - JSON error
- `-3005` - XML error

#### File System Errors (3100-3199)

- `-3101` - File not found
- `-3102` - Permission denied
- `-3103` - File system error
- `-3104` - Insufficient disk space

#### Download Errors (3200-3299)

- `-3201` - Download interrupted
- `-3202` - Download timeout
- `-3203` - Download corrupted
- `-3204` - Download failed

#### System Errors (9000-9099)

- `-9001` - Panic occurred
- `-9002` - Internal error
- `-9099` - Unknown error

### Error Categories

Errors are automatically classified into categories for easier handling:

- `network` - Network connectivity issues, DNS failures, connection problems
- `provider` - Provider-specific issues, site unavailable
- `parsing` - JSON/XML parsing errors, invalid response format
- `validation` - Input validation failures
- `timeout` - Operation timeouts, deadlines exceeded
- `authentication` - Auth failures, rate limiting
- `rate_limit` - Too many requests
- `not_found` - Resource not found
- `filesystem` - File system errors, permissions
- `download` - Download-specific errors
- `panic` - System panics
- `unknown` - Unclassified errors

### Error Examples

#### Network Connectivity Error

When there's no internet connection during a search:

```json
{
  "result": null,
  "error": {
    "code": -2004,
    "message": "Search failed for query 'one piece'",
    "function_chain": "SearchService.Search() → MangadxProvider.Search() → HTTPService.DoRequest()",
    "error_category": "network",
    "original_error": "dial tcp: no such host",
    "call_details": [
      {
        "function": "HTTPService.DoRequest",
        "operation": "http_request",
        "context": {
          "url": "https://api.mangadx.org/manga",
          "http_method": "GET"
        }
      }
    ]
  },
  "id": 1
}
```

#### Provider Not Found Error

```json
{
  "result": null,
  "error": {
    "code": -1101,
    "message": "Provider 'invalid_provider' not found",
    "function_chain": "SearchService.Search() → GetProvider()",
    "error_category": "not_found",
    "data": {
      "provider_id": "invalid_provider"
    }
  },
  "id": 2
}
```

#### Download Permission Error

```json
{
  "result": null,
  "error": {
    "code": -3102,
    "message": "Failed to download chapter 'chapter-123'",
    "function_chain": "DownloadService.Download() → DownloadFile() → os.Create()",
    "error_category": "filesystem",
    "original_error": "permission denied",
    "data": {
      "chapter_id": "chapter-123",
      "destination": "/protected/downloads/"
    }
  },
  "id": 3
}
```

#### JSON Parsing Error

```json
{
  "result": null,
  "error": {
    "code": -3004,
    "message": "Search failed for query 'manga'",
    "function_chain": "SearchService.Search() → MangadxProvider.Search() → json.Unmarshal()",
    "error_category": "parsing",
    "original_error": "invalid character '<' looking for beginning of value",
    "call_details": [
      {
        "function": "json.Unmarshal",
        "operation": "parsing",
        "context": {
          "data_type": "json",
          "data_size": 1024
        }
      }
    ]
  },
  "id": 4
}
```

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
