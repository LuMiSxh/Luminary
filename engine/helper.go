package engine

import (
	"context"
	"fmt"
	"sync"
)

// AgentHelper encapsulates functions needed by agent operations
type AgentHelper interface {
	ID() string
	Name() string
	Initialize(context.Context) error
	ExtractDomain(string) string
	GetEngine() *Engine
}

// SearchResult is a generic interface for search results
type SearchResult interface{}

// MangaResult is a generic interface for manga results
type MangaResult interface{}

// ChapterResult is a generic interface for chapter results
type ChapterResult interface{}

// PerformSearch encapsulates the common pattern for searching
func PerformSearch[T SearchResult](
	ctx context.Context,
	agent AgentHelper,
	query string,
	options SearchOptions,
	searchFunc func(context.Context, string, SearchOptions) (T, error),
) (T, error) {
	var empty T
	// Initialize if needed
	if err := agent.Initialize(ctx); err != nil {
		return empty, err
	}

	e := agent.GetEngine()

	// Apply rate limiting
	domain := agent.ExtractDomain(agent.GetEngine().HTTP.RequestOptions.Referer)
	e.RateLimiter.Wait(domain)

	// Log search request
	e.Logger.Info("[%s] Searching for: %s (limit: %d)", agent.ID(), query, options.Limit)

	// Try cache first
	cacheKey := fmt.Sprintf("search:%s:%s:%d", agent.ID(), query, options.Limit)
	var cachedResults T
	if e.Cache.Get(cacheKey, &cachedResults) {
		e.Logger.Debug("[%s] Using cached search results for: %s", agent.ID(), query)
		return cachedResults, nil
	}

	// Call the agent-specific implementation passed as searchFunc
	results, err := searchFunc(ctx, query, options)
	if err != nil {
		e.Logger.Error("[%s] Search error: %v", agent.ID(), err)
		return empty, err
	}

	// Cache the results
	if err := e.Cache.Set(cacheKey, results); err != nil {
		return results, err // Return results anyway, just don't cache
	}

	e.Logger.Info("[%s] Found results for: %s", agent.ID(), query)
	return results, nil
}

// PerformGetManga encapsulates the common pattern for retrieving manga details
func PerformGetManga[T MangaResult](
	ctx context.Context,
	agent AgentHelper,
	id string,
	mangaCache map[string]T,
	cacheMutex *sync.RWMutex,
	apiURL string,
	getMangaFunc func(context.Context, string) (T, error),
) (T, error) {
	var empty T
	// Initialize if needed
	if err := agent.Initialize(ctx); err != nil {
		return empty, err
	}

	e := agent.GetEngine()

	// Check memory cache first
	if cacheMutex != nil {
		cacheMutex.RLock()
		manga, found := mangaCache[id]
		cacheMutex.RUnlock()

		if found {
			e.Logger.Debug("[%s] Using cached manga info for: %s", agent.ID(), id)
			return manga, nil
		}
	}

	// Apply rate limiting
	domain := agent.ExtractDomain(apiURL)
	e.RateLimiter.Wait(domain)

	// Try disk cache
	cacheKey := fmt.Sprintf("manga:%s:%s", agent.ID(), id)
	var cachedManga T
	if e.Cache.Get(cacheKey, &cachedManga) {
		e.Logger.Debug("[%s] Using disk-cached manga info for: %s", agent.ID(), id)
		// Store in memory cache too
		if cacheMutex != nil {
			cacheMutex.Lock()
			mangaCache[id] = cachedManga
			cacheMutex.Unlock()
		}
		return cachedManga, nil
	}

	// Log manga retrieval
	e.Logger.Info("[%s] Fetching manga details for: %s", agent.ID(), id)

	// Call the agent-specific implementation
	manga, err := getMangaFunc(ctx, id)
	if err != nil {
		e.Logger.Error("[%s] Failed to get manga %s: %v", agent.ID(), id, err)
		return empty, err
	}

	// Cache the result (both memory and disk)
	if cacheMutex != nil {
		cacheMutex.Lock()
		mangaCache[id] = manga
		cacheMutex.Unlock()
	}

	if err := e.Cache.Set(cacheKey, manga); err != nil {
		return manga, err // Return manga anyway, just don't cache
	}

	return manga, nil
}

// PerformGetChapter encapsulates the common pattern for retrieving chapter details
func PerformGetChapter[T ChapterResult](
	ctx context.Context,
	agent AgentHelper,
	chapterID string,
	apiURL string,
	getChapterFunc func(context.Context, string) (T, error),
) (T, error) {
	var empty T
	// Initialize if needed
	if err := agent.Initialize(ctx); err != nil {
		return empty, err
	}

	e := agent.GetEngine()

	// Apply rate limiting
	domain := agent.ExtractDomain(apiURL)
	e.RateLimiter.Wait(domain)

	// Try cache first
	cacheKey := fmt.Sprintf("chapter:%s:%s", agent.ID(), chapterID)
	var cachedChapter T
	if e.Cache.Get(cacheKey, &cachedChapter) {
		e.Logger.Debug("[%s] Using cached chapter info for: %s", agent.ID(), chapterID)
		return cachedChapter, nil
	}

	// Log chapter retrieval
	e.Logger.Info("[%s] Fetching chapter details for: %s", agent.ID(), chapterID)

	// Call agent-specific implementation
	chapter, err := getChapterFunc(ctx, chapterID)
	if err != nil {
		e.Logger.Error("[%s] Failed to get chapter %s: %v", agent.ID(), chapterID, err)
		return empty, err
	}

	// Cache the result
	if err := e.Cache.Set(cacheKey, chapter); err != nil {
		return chapter, err // Return chapter anyway, just don't cache
	}

	return chapter, nil
}
