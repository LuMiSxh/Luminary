# Refactoring Plan: Streamline Engine and Agents Architecture

## Goal

Eliminate `BaseAgent` and all caching mechanisms. Integrate agent registry directly into the `Engine`. Move core agent types and context helpers into the `engine` package. Agents will implement the `engine.Agent` interface directly, relying on enhanced, type-safe engine helper functions for common operations.

## Implementation Plan

### 1. Remove Caching Infrastructure

*   Delete the `engine/cache.go` file entirely.
*   Remove the `Cache` field from the `Engine` struct in `engine/engine.go`.
*   Remove the `CacheService` instantiation (`NewCacheService(...)`) within `engine.New()` in `engine/engine.go`.
*   Remove the `Cache` field from the `APIService` struct in `engine/api.go`.
*   Remove the `CacheService` parameter from `NewAPIService` in `engine/api.go`.
*   Remove all code related to cache checks (`Cache.Get(...)`) and cache writes (`Cache.Set(...)`) from `engine/api.go` (`FetchFromAPI`).
*   Remove the `Shutdown()` method from `engine/engine.go` if its only purpose was cache cleanup.

### 2. Integrate Agent Registry Into Engine

*   Add an `agents` field to the `Engine` struct: `agents map[string]Agent`.
*   Add a mutex to protect concurrent access: `agentsMutex sync.RWMutex`.
*   Add engine methods to manage agents:
    *   `RegisterAgent(agent Agent) error` - registers an agent with error if duplicate ID
    *   `GetAgent(id string) (Agent, bool)` - gets agent by ID with existence check
    *   `AllAgents() []Agent` - returns a slice of all agents
    *   `AgentExists(id string) bool` - checks if agent exists
*   Update the `engine.New()` function to initialize the agents map.

### 3. Centralize Agent Types and Context in Engine

*   **Move Core Types:**
    *   Create `engine/types.go`.
    *   Move `Manga`, `MangaInfo`, `ChapterInfo`, `Chapter`, `Page` struct definitions from `agents/types.go` to `engine/types.go`.
    *   Update all engine services and agent implementations to import these types from `engine`.
*   **Move Agent Interface:**
    *   Move the `Agent` interface definition from `agents/types.go` to `engine/types.go` (or a new `engine/agent.go`).
    *   Modify the `Agent` interface:
        *   Remove `GetEngine()`. Agents will hold a direct reference.
        *   Remove `ExtractDomain()`. Use `engine.ExtractDomain(...)` directly if needed.
        *   Remove `APIURL()`. API config is internal to the agent struct.
        *   Keep `ID()`, `Name()`, `Description()`, `SiteURL()`.
        *   Keep `Initialize(ctx context.Context) error`.
        *   Update method signatures to use types from `engine` package.
*   **Move Context Helpers:**
    *   Create `engine/context.go` (or add to `engine/utils.go` if one exists).
    *   Move `ContextKey`, `WithConcurrency`, `GetConcurrency` from `agents/context.go` to the new engine location.

### 4. Eliminate BaseAgent

*   Delete the `agents/agent.go` file containing `BaseAgent`.

### 5. Develop Engine Helper Functions (e.g., in `engine/agent_helpers.go`)

*   **`ExecuteInitialize(ctx context.Context, engine *Engine, agentID, agentName string, initFunc func(context.Context) error) error`**:
    *   Handles logging and calls the agent's `initFunc`.
    *   No state tracking - each CLI invocation begins with fresh state.
    *   Returns errors directly from `initFunc` with proper context.
*   **`ExecuteSearch[ResultType engine.Manga](ctx, engine, agentID, searchOptions, apiConfig, paginationConfig, extractorSet) ([]ResultType, error)`**:
    *   Handles logging, rate limiting, calls `engine.Pagination.FetchAllPages`.
    *   Returns typed results and detailed errors with context.
*   **`ExecuteGetManga[MangaInfoType, ChapterInfoType](...)`, `ExecuteGetChapter[ResultType](...)`, `ExecuteDownloadChapter(...)`**:
    *   All follow the pattern of handling cross-cutting concerns, calling appropriate engine services, returning typed results.
    *   Each carefully wraps errors with context about the operation/agent for better debugging.

### 6. Refactor Agent Implementations (in `agents/<agent_name>/agent.go`)

*   Each agent struct:
    *   Implements the `engine.Agent` interface directly.
    *   Holds an `*engine.Engine` instance (passed during construction).
    *   Stores agent-specific configurations.
    *   Methods call the corresponding engine helpers, passing appropriate params.
    *   Clear error handling with wrapping for context.
*   In each agent package:
    *   Create a constructor that takes an engine instance: `func NewAgent(e *engine.Engine) engine.Agent`.
    *   Update the `init()` function to create the agent but DO NOT register it yet (will be registered at application startup).
    *   Export `NewAgentName` constructor for use during application initialization.

### 7. Review and Refine Engine Services

*   **Merge `ParserService` into `MetadataService`.**
*   Review error propagation in all services to ensure:
    *   Errors include context about operation/source/parameters.
    *   No silent failures - all errors should be surfaced to callers.
    *   Consider using errors.Wrap() pattern for more context.
*   Review all services for assumptions about initialization state.

### 8. Update Command Implementation

*   Update commands that use agents:
    *   `cmd/agents.go`: Use `engine.AllAgents()` instead of `agents.All()`.
    *   `cmd/download.go`: Use `engine.GetAgent()` instead of `agents.Get()`.
    *   `cmd/info.go`: Update agent retrieval and type references.
    *   `cmd/list.go`: Update agent retrieval and type references.
    *   `cmd/search.go`: Update agent retrieval and type references.
*   Add engine initialization to root command or shared state:
    *   Create the engine once at startup.
    *   Register all agents with the engine during initialization.
*   Remove or update `cmd/management.go`:
    *   Remove cache commands.
    *   Update any remaining engine management commands.

### 9. Central Application Initialization

*   In `main.go` or a dedicated initialization file:
    *   Create the engine instance.
    *   Register all available agents using exported constructors.
    *   Make the engine available to commands.
*   Consider using command context or a singleton for the engine:
    *   `cmd.SetupEngine(engine)` or `cmd.InitializeWithEngine(engine)`.

### 10. Cleanup

*   Delete the old `engine/helper.go` file.
*   Delete `engine/parser.go` (merged into MetadataService).
*   Delete `agents/types.go` (moved to engine).
*   Delete `agents/registry.go` (integrated into engine).
*   Delete `agents/context.go` (moved to engine).
*   Remove any remaining unused code related to `BaseAgent` or caching.
