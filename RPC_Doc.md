# Luminary JSON-RPC Mode Guide

This guide covers how to use Luminary's JSON-RPC interface, provided by the `luminary-rpc` executable. This interface allows for robust, programmatic integration with other tools, scripts, or applications by communicating over standard input (stdin) and standard output (stdout) using the JSON-RPC 2.0 protocol.

![Separator](.github/assets/luminary-separator.png)

## Overview

The JSON-RPC interface is accessed by running the `luminary-rpc` executable:

```bash
./luminary-rpc
```

Once started, `luminary-rpc` listens for JSON-RPC 2.0 requests on its stdin and sends JSON-RPC 2.0 responses to its stdout. Each request and response must be a single line of JSON.

### JSON-RPC 2.0 Request Format

A typical request to `luminary-rpc` will look like this:

```json
{"method": "ServiceName.MethodName", "params": [args_object], "id": request_id}
```

- `method`: A string containing the name of the service and method to be called (e.g., `"SearchService.Search"`).
- `params`: An array containing a single object with the arguments for the method.
- `id`: A unique identifier for the request, which will be included in the response. Can be a string, number, or null.

### JSON-RPC 2.0 Response Format

A successful response will look like this:

```json
{"result": response_data, "error": null, "id": request_id}
```

- `result`: The data returned by the method call. The structure depends on the method.
- `error`: `null` for successful calls.
- `id`: The `id` from the original request.

An error response will look like this:

```json
{"result": null, "error": {"code": error_code, "message": "error_message"}, "id": request_id}
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
{"method": "VersionService.Get", "params": [{}], "id": 1}
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
{"method": "ProvidersService.List", "params": [{}], "id": 2}
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
  "provider": "optional_provider_id", // Optional: "mgd", "kmg", etc. If omitted, searches all.
  "limit": 10,                       // Optional: Max results per page (default: 10)
  "pages": 1,                        // Optional: Number of pages to fetch (default: 1)
  "sort": "relevance",               // Optional: "relevance", "name", "newest", "updated"
  "fields": ["title", "author"],     // Optional: Fields to search in
  "filters": {"genre": "action"},    // Optional: Field-specific filters
  "include_alt_titles": true,        // Optional: Include alternative titles (default: false)
  "concurrency": 5                   // Optional: Max concurrent operations for this search (default: 5)
}
```

**Example Request:**
```json
{"method": "SearchService.Search", "params": [{"query": "one piece", "provider": "mgd", "limit": 5}], "id": 3}
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
      "alt_titles": ["ワンピース", "Wan Pīsu"],
      "authors": ["Oda Eiichiro"],
      "tags": ["Action", "Adventure", "Comedy", "Drama", "Shounen"]
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
  "provider": "optional_provider_id", // Optional: If omitted, lists from all.
  "limit": 50,                       // Optional: Max results (default: 50)
  "pages": 1,                        // Optional: Number of pages (default: 1, 0 for all if supported by provider)
  "concurrency": 5                   // Optional: Max concurrent operations (default: 5)
}
```

**Example Request (All Providers):**
```json
{"method": "ListService.List", "params": [{"limit": 2}], "id": 4}
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
      "authors": ["Author A"],
      "tags": ["Tag1"]
    },
    {
      "id": "kmg:kissmanga/saving-the-world-through-a-game",
      "title": "Saving the World Through a Game",
      "provider": "kmg",
      "provider_name": "KissManga",
      "authors": ["Author B"],
      "tags": ["Tag2"]
    }
  ],
  "count": 2,
  "provider": "", // Empty if multiple providers
  "provider_name": "Multiple Providers" // Or specific provider name if filtered
}
```
**Fields:**
- `mangas`: An array of `SearchResultItem` objects (same structure as in `SearchService.Search` results).
- `count`: Total number of manga returned.
- `provider`: Provider ID (empty if listing from all, or the specific provider ID if filtered).
- `provider_name`: "Multiple Providers" or the specific provider name if filtered.

---

### InfoService

Retrieves detailed information about a specific manga.

#### `InfoService.Get`

Fetches details for a manga given its combined ID.

**Request Parameters (`args_object`):**
```json
{
  "manga_id": "provider_id:manga_specific_id" // e.g., "mgd:manga-id-123"
}
```

**Example Request:**
```json
{"method": "InfoService.Get", "params": [{"manga_id": "mgd:manga-id-123"}], "id": 5}
```

**Response Data (`response_data`):**
```json
{
  "id": "mgd:manga-id-123",
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
      "date": "2024-01-15T10:30:00Z" // ISO 8601 format
    }
  ],
  "chapter_count": 1
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
- `chapters`: Array of `ChapterInfo` objects.
    - `id`: Combined provider ID and chapter ID (e.g., "mgd:chapter-456").
    - `title`: Chapter title.
    - `number`: Chapter number (float, e.g., 1.0, 1.5).
    - `date`: Publication date in ISO 8601 format (optional).
- `chapter_count`: Total number of chapters.

---

### DownloadService

Handles downloading of manga chapters.

#### `DownloadService.Download`

Initiates the download of a specific manga chapter.

**Request Parameters (`args_object`):**
```json
{
  "chapter_id": "provider_id:chapter_specific_id", // e.g., "mgd:chapter-456"
  "output_dir": "./downloads",                     // Optional: Default is "./downloads"
  "volume": 1,                                     // Optional: Override volume number
  "concurrency": 5                                 // Optional: Max concurrent page downloads (default: 5)
}
```

**Example Request:**
```json
{"method": "DownloadService.Download", "params": [{"chapter_id": "mgd:chapter-456", "output_dir": "./my_manga"}], "id": 6}
```

**Response Data (`response_data`):**
```json
{
  "chapter_id": "chapter-456", // The chapter ID part (without provider prefix)
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

**Note:** If the download fails, `success` will be `false`, and the `message` field will contain the error. The RPC call itself will still be a "successful" JSON-RPC response unless there's a fundamental issue with the request format or server. The business logic error is conveyed within the `result` payload.

![Separator](.github/assets/luminary-separator.png)

## Error Handling

If a JSON-RPC request is malformed, or if an unrecoverable server-side error occurs before a specific service method can process the business logic, a standard JSON-RPC error response is returned:

```json
{"result": null, "error": {"code": -32600, "message": "Invalid Request"}, "id": "some_id"}
{"result": null, "error": {"code": -32601, "message": "Method not found"}, "id": "some_id"}
{"result": null, "error": {"code": -32602, "message": "Invalid params"}, "id": "some_id"}
{"result": null, "error": {"code": -32700, "message": "Parse error"}, "id": null} 
```

For errors specific to a service method's execution (e.g., "provider not found", "manga not found"), the JSON-RPC call itself is successful (i.e., `error` field in the main JSON-RPC response is `null`), but the `result` payload will indicate the failure. For example, `DownloadService.Download` returns a `success: false` field in its result. Other services might return an error directly as the JSON-RPC error object if the method signature in Go returns an `error`.

**Example of a business logic error returned as a JSON-RPC error:**
If `SearchService.Search` is called with an invalid provider ID:
```json
{"result": null, "error": {"code": 1, "message": "provider 'invalid_provider' not found"}, "id": 7} 
```
*(Note: The specific error code for business logic errors might vary or be a generic one).*

![Separator](.github/assets/luminary-separator.png)

## Integration Examples

### Python Integration

```python
import subprocess
import json
import sys
from typing import List, Dict, Optional, Any, Union

class LuminaryRPCClient:
    def __init__(self, rpc_executable_path: str = "./luminary-rpc"):
        self.rpc_executable_path = rpc_executable_path
        self.process = None
        self._start_process()
        self.request_id_counter = 0

    def _start_process(self):
        try:
            self.process = subprocess.Popen(
                [self.rpc_executable_path],
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE, # Capture stderr for debugging
                text=True,
                bufsize=1 # Line buffered
            )
        except FileNotFoundError:
            print(f"Error: luminary-rpc executable not found at {self.rpc_executable_path}", file=sys.stderr)
            sys.exit(1)
        except Exception as e:
            print(f"Error starting luminary-rpc: {e}", file=sys.stderr)
            sys.exit(1)

    def _send_request(self, method: str, params: Optional[Dict[str, Any]] = None) -> Any:
        if not self.process or self.process.poll() is not None:
            # Process died or was never started, try to restart
            print("RPC process is not running. Attempting to restart...", file=sys.stderr)
            self._start_process()
            if not self.process or self.process.poll() is not None:
                 raise ConnectionError("Failed to start or connect to luminary-rpc process.")


        self.request_id_counter += 1
        request_id = self.request_id_counter
        
        rpc_request = {
            "method": method,
            "params": [params if params is not None else {}],
            "id": request_id
        }
        
        try:
            # print(f"Sending: {json.dumps(rpc_request)}", file=sys.stderr) # Debug
            self.process.stdin.write(json.dumps(rpc_request) + "\n")
            self.process.stdin.flush()

            # Read response
            response_line = self.process.stdout.readline()
            # print(f"Received: {response_line.strip()}", file=sys.stderr) # Debug
            
            if not response_line:
                # Check stderr for clues if process died
                stderr_output = self.process.stderr.read()
                raise ConnectionError(f"No response from luminary-rpc. Process might have died. Stderr: {stderr_output}")

            response = json.loads(response_line)
        except BrokenPipeError:
            stderr_output = self.process.stderr.read()
            raise ConnectionError(f"Broken pipe. luminary-rpc process likely crashed. Stderr: {stderr_output}")
        except Exception as e:
            # Catch any other exception during send/receive
            stderr_output = self.process.stderr.read()
            raise ConnectionError(f"Exception during RPC communication: {e}. Stderr: {stderr_output}")


        if response.get("id") != request_id:
            raise ValueError("Response ID does not match request ID.")
            
        if response.get("error"):
            raise Exception(f"RPC Error: {response['error'].get('message', 'Unknown error')} (Code: {response['error'].get('code', 0)})")
            
        return response.get("result")

    def get_version(self) -> Dict[str, Any]:
        return self._send_request("VersionService.Get")

    def list_providers(self) -> List[Dict[str, Any]]:
        return self._send_request("ProvidersService.List")

    def search_manga(self, query: str, provider: Optional[str] = None, limit: int = 10, pages: int = 1) -> Dict[str, Any]:
        params = {"query": query, "limit": limit, "pages": pages}
        if provider:
            params["provider"] = provider
        return self._send_request("SearchService.Search", params)

    def list_manga(self, provider: Optional[str] = None, limit: int = 50, pages: int = 1) -> Dict[str, Any]:
        params = {"limit": limit, "pages": pages}
        if provider:
            params["provider"] = provider
        return self._send_request("ListService.List", params)

    def get_manga_info(self, manga_id: str) -> Dict[str, Any]:
        return self._send_request("InfoService.Get", {"manga_id": manga_id})

    def download_chapter(self, chapter_id: str, output_dir: Optional[str] = None) -> Dict[str, Any]:
        params = {"chapter_id": chapter_id}
        if output_dir:
            params["output_dir"] = output_dir
        return self._send_request("DownloadService.Download", params)

    def close(self):
        if self.process and self.process.poll() is None:
            self.process.terminate()
            self.process.wait(timeout=5)
        self.process = None

# Usage Example
if __name__ == "__main__":
    client = LuminaryRPCClient()
    try:
        version_info = client.get_version()
        print("Version Info:", json.dumps(version_info, indent=2))

        providers = client.list_providers()
        print("\nProviders:", json.dumps(providers, indent=2))

        if providers:
            first_provider_id = providers[0]["id"]
            print(f"\nSearching for 'Solo Leveling' on '{first_provider_id}':")
            search_results = client.search_manga("Solo Leveling", provider=first_provider_id, limit=3)
            print(json.dumps(search_results, indent=2))

            if search_results and search_results.get("results"):
                first_manga_id = search_results["results"][0]["id"]
                print(f"\nGetting info for manga '{first_manga_id}':")
                manga_info = client.get_manga_info(first_manga_id)
                print(json.dumps(manga_info, indent=2))

                # Example Download (be cautious with actual downloads)
                # if manga_info and manga_info.get("chapters"):
                #     first_chapter_id = manga_info["chapters"][0]["id"]
                #     print(f"\nDownloading chapter '{first_chapter_id}':")
                #     download_status = client.download_chapter(first_chapter_id, output_dir="./rpc_downloads")
                #     print(json.dumps(download_status, indent=2))
            
            print(f"\nListing manga from '{first_provider_id}':")
            list_results = client.list_manga(provider=first_provider_id, limit=2)
            print(json.dumps(list_results, indent=2))


    except Exception as e:
        print(f"\nAn error occurred: {e}", file=sys.stderr)
    finally:
        client.close()
```

### Rust Integration

```rust
use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::io::{BufRead, BufReader, Write};
use std::process::{Command, Stdio, Child, ChildStdin, ChildStdout};
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Mutex; // For request_id_counter if making client thread-safe

#[derive(Serialize, Deserialize, Debug)]
struct JsonRpcRequest<'a> {
    method: &'a str,
    params: Vec<Value>,
    id: usize,
}

#[derive(Serialize, Deserialize, Debug)]
struct JsonRpcResponse {
    result: Option<Value>,
    error: Option<JsonRpcError>,
    id: usize,
}

#[derive(Serialize, Deserialize, Debug)]
struct JsonRpcError {
    code: i32,
    message: String,
}

pub struct LuminaryRpcClient {
    process: Child,
    stdin: ChildStdin,
    stdout: BufReader<ChildStdout>,
    request_id_counter: AtomicUsize,
}

impl LuminaryRpcClient {
    pub fn new(rpc_executable_path: &str) -> Result<Self, Box<dyn std::error::Error>> {
        let mut process = Command::new(rpc_executable_path)
            .stdin(Stdio::piped())
            .stdout(Stdio::piped())
            .stderr(Stdio::piped()) // Capture stderr for debugging
            .spawn()?;

        let stdin = process.stdin.take().ok_or("Failed to open stdin")?;
        let stdout = BufReader::new(process.stdout.take().ok_or("Failed to open stdout")?);

        Ok(LuminaryRpcClient {
            process,
            stdin,
            stdout,
            request_id_counter: AtomicUsize::new(1),
        })
    }

    fn send_request(&mut self, method: &str, params_obj: Value) -> Result<Value, Box<dyn std::error::Error>> {
        let request_id = self.request_id_counter.fetch_add(1, Ordering::SeqCst);
        let request = JsonRpcRequest {
            method,
            params: vec![params_obj],
            id: request_id,
        };

        let request_json = serde_json::to_string(&request)? + "\n";
        // eprintln!("Sending: {}", request_json.trim()); // Debug
        self.stdin.write_all(request_json.as_bytes())?;
        self.stdin.flush()?;

        let mut response_line = String::new();
        self.stdout.read_line(&mut response_line)?;
        // eprintln!("Received: {}", response_line.trim()); // Debug

        if response_line.is_empty() {
            // Check stderr if process died
            let mut stderr_output = String::new();
            if let Some(mut stderr) = self.process.stderr.take() {
                 std::io::Read::read_to_string(&mut stderr, &mut stderr_output)?;
            }
            return Err(format!("No response from luminary-rpc. Process might have died. Stderr: {}", stderr_output).into());
        }
        
        let response: JsonRpcResponse = serde_json::from_str(&response_line)?;

        if response.id != request_id {
            return Err("Response ID mismatch".into());
        }

        if let Some(err) = response.error {
            return Err(format!("RPC Error (Code {}): {}", err.code, err.message).into());
        }

        response.result.ok_or_else(|| "Missing result field in RPC response".into())
    }

    // --- Service Methods ---
    pub fn get_version(&mut self) -> Result<Value, Box<dyn std::error::Error>> {
        self.send_request("VersionService.Get", serde_json::json!({}))
    }

    pub fn list_providers(&mut self) -> Result<Value, Box<dyn std::error::Error>> {
        self.send_request("ProvidersService.List", serde_json::json!({}))
    }
    
    #[derive(Serialize)]
    struct SearchParams<'a> {
        query: &'a str,
        #[serde(skip_serializing_if = "Option::is_none")]
        provider: Option<&'a str>,
        #[serde(skip_serializing_if = "Option::is_none")]
        limit: Option<u32>,
        // Add other params as needed
    }

    pub fn search_manga(&mut self, query: &str, provider: Option<&str>, limit: Option<u32>) -> Result<Value, Box<dyn std::error::Error>> {
        let params = SearchParams { query, provider, limit };
        self.send_request("SearchService.Search", serde_json::to_value(params)?)
    }
    
    #[derive(Serialize)]
    struct ListParams<'a> {
        #[serde(skip_serializing_if = "Option::is_none")]
        provider: Option<&'a str>,
        #[serde(skip_serializing_if = "Option::is_none")]
        limit: Option<u32>,
         #[serde(skip_serializing_if = "Option::is_none")]
        pages: Option<u32>,
    }

    pub fn list_manga(&mut self, provider: Option<&str>, limit: Option<u32>, pages: Option<u32>) -> Result<Value, Box<dyn std::error::Error>> {
        let params = ListParams { provider, limit, pages };
        self.send_request("ListService.List", serde_json::to_value(params)?)
    }
    
    #[derive(Serialize)]
    struct InfoParams<'a> {
        manga_id: &'a str,
    }
    pub fn get_manga_info(&mut self, manga_id: &str) -> Result<Value, Box<dyn std::error::Error>> {
        let params = InfoParams { manga_id };
        self.send_request("InfoService.Get", serde_json::to_value(params)?)
    }

    #[derive(Serialize)]
    struct DownloadParams<'a> {
        chapter_id: &'a str,
        #[serde(skip_serializing_if = "Option::is_none")]
        output_dir: Option<&'a str>,
    }
    pub fn download_chapter(&mut self, chapter_id: &str, output_dir: Option<&str>) -> Result<Value, Box<dyn std::error::Error>> {
        let params = DownloadParams { chapter_id, output_dir };
        self.send_request("DownloadService.Download", serde_json::to_value(params)?)
    }
}

impl Drop for LuminaryRpcClient {
    fn drop(&mut self) {
        // Ensure the child process is killed when the client is dropped
        let _ = self.process.kill();
        let _ = self.process.wait(); // Wait to reap the zombie process
    }
}

// Usage Example
fn main() -> Result<(), Box<dyn std::error::Error>> {
    let mut client = LuminaryRpcClient::new("./luminary-rpc")?;

    let version_info = client.get_version()?;
    println!("Version Info:\n{}", serde_json::to_string_pretty(&version_info)?);

    let providers = client.list_providers()?;
    println!("\nProviders:\n{}", serde_json::to_string_pretty(&providers)?);
    
    if let Some(provider_array) = providers.as_array() {
        if !provider_array.is_empty() {
            if let Some(first_provider_id) = provider_array[0].get("id").and_then(|v| v.as_str()) {
                 println!("\nSearching for 'Bleach' on '{}':", first_provider_id);
                 let search_results = client.search_manga("Bleach", Some(first_provider_id), Some(2))?;
                 println!("{}", serde_json::to_string_pretty(&search_results)?);

                 if let Some(results_obj) = search_results.as_object() {
                     if let Some(manga_list) = results_obj.get("results").and_then(|r| r.as_array()) {
                         if !manga_list.is_empty() {
                             if let Some(manga_id) = manga_list[0].get("id").and_then(|id_val| id_val.as_str()) {
                                 println!("\nGetting info for manga '{}':", manga_id);
                                 let manga_info = client.get_manga_info(manga_id)?;
                                 println!("{}", serde_json::to_string_pretty(&manga_info)?);
                             }
                         }
                     }
                 }
            }
        }
    }
    Ok(())
}
```
**Dependencies for Rust (Cargo.toml):**
```toml
[dependencies]
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
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

This RPC interface provides a more structured and robust way to integrate Luminary's functionalities compared to parsing CLI output.
