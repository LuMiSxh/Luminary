package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheService provides caching for various types of data with both memory and disk caching
type CacheService struct {
	// Memory cache storage
	memCache map[string]cacheEntry
	memMutex sync.RWMutex

	// Disk cache settings
	CacheDir     string
	UseDiskCache bool
	TTL          time.Duration
}

// cacheEntry represents a cached item with expiration
type cacheEntry struct {
	data       interface{}
	expiration time.Time
}

// NewCacheService creates a new cache service
func NewCacheService(ttl time.Duration, cacheDir string, useDiskCache bool) *CacheService {
	// Create cache directories if using disk cache
	if useDiskCache && cacheDir != "" {
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			fmt.Printf("WARNING: Failed to create cache directory: %v\n", err)
		}
	}

	return &CacheService{
		memCache:     make(map[string]cacheEntry),
		TTL:          ttl,
		CacheDir:     cacheDir,
		UseDiskCache: useDiskCache,
	}
}

// Get retrieves an item from cache
func (c *CacheService) Get(key string, result interface{}) bool {
	// Try memory cache first
	c.memMutex.RLock()
	entry, found := c.memCache[key]
	c.memMutex.RUnlock()

	// If found in memory and not expired
	if found && time.Now().Before(entry.expiration) {
		// Try to copy the data to the result
		if result != nil {
			bytes, err := json.Marshal(entry.data)
			if err == nil {
				return json.Unmarshal(bytes, result) == nil
			}
		}
		return true
	}

	// If not in memory and disk cache is enabled, try disk
	if c.UseDiskCache && c.CacheDir != "" {
		cacheFile := filepath.Join(c.CacheDir, key+".json")
		if data, err := os.ReadFile(cacheFile); err == nil {
			// Check if file is expired by looking at its modification time
			info, err := os.Stat(cacheFile)
			if err == nil && time.Since(info.ModTime()) < c.TTL {
				// File exists and is not expired
				if err := json.Unmarshal(data, result); err == nil {
					// Store back in memory cache for faster access next time
					var cachedCopy interface{}
					if err := json.Unmarshal(data, &cachedCopy); err == nil {
						err := c.Set(key, cachedCopy)
						if err != nil {
							return false
						}
					}
					return true
				}
			} else {
				// File is expired, remove it
				_ = os.Remove(cacheFile)
			}
		}
	}

	return false
}

// Set stores an item in cache
func (c *CacheService) Set(key string, value interface{}) error {
	// Store in memory with expiration
	expiration := time.Now().Add(c.TTL)

	c.memMutex.Lock()
	c.memCache[key] = cacheEntry{
		data:       value,
		expiration: expiration,
	}
	c.memMutex.Unlock()

	// Store on disk if enabled
	if c.UseDiskCache && c.CacheDir != "" {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal cache data: %w", err)
		}

		cacheFile := filepath.Join(c.CacheDir, key+".json")
		if err := os.WriteFile(cacheFile, data, 0644); err != nil {
			return fmt.Errorf("failed to write cache file: %w", err)
		}
	}

	return nil
}

// Delete removes an item from the cache
func (c *CacheService) Delete(key string) error {
	// Remove from memory
	c.memMutex.Lock()
	delete(c.memCache, key)
	c.memMutex.Unlock()

	// Remove from disk if enabled
	if c.UseDiskCache && c.CacheDir != "" {
		cacheFile := filepath.Join(c.CacheDir, key+".json")
		if err := os.Remove(cacheFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove cache file: %w", err)
		}
	}

	return nil
}

// Clear removes all items from the cache
func (c *CacheService) Clear() error {
	// Clear memory cache
	c.memMutex.Lock()
	c.memCache = make(map[string]cacheEntry)
	c.memMutex.Unlock()

	// Clear disk cache if enabled
	if c.UseDiskCache && c.CacheDir != "" {
		// Read all files in the cache directory
		entries, err := os.ReadDir(c.CacheDir)
		if err != nil {
			return fmt.Errorf("failed to read cache directory: %w", err)
		}

		// Remove all .json files
		for _, entry := range entries {
			if entry.Type().IsRegular() && filepath.Ext(entry.Name()) == ".json" {
				cacheFile := filepath.Join(c.CacheDir, entry.Name())
				if err := os.Remove(cacheFile); err != nil {
					return fmt.Errorf("failed to remove cache file %s: %w", entry.Name(), err)
				}
			}
		}
	}

	return nil
}

// CleanExpired removes expired items from the cache
func (c *CacheService) CleanExpired() error {
	now := time.Now()

	// Clean memory cache
	c.memMutex.Lock()
	for key, entry := range c.memCache {
		if now.After(entry.expiration) {
			delete(c.memCache, key)
		}
	}
	c.memMutex.Unlock()

	// Clean disk cache if enabled
	if c.UseDiskCache && c.CacheDir != "" {
		// Read all files in the cache directory
		entries, err := os.ReadDir(c.CacheDir)
		if err != nil {
			return fmt.Errorf("failed to read cache directory: %w", err)
		}

		// Check each file's modification time
		for _, entry := range entries {
			if entry.Type().IsRegular() && filepath.Ext(entry.Name()) == ".json" {
				cacheFile := filepath.Join(c.CacheDir, entry.Name())
				info, err := os.Stat(cacheFile)
				if err == nil && time.Since(info.ModTime()) > c.TTL {
					// File is expired, remove it
					if err := os.Remove(cacheFile); err != nil {
						return fmt.Errorf("failed to remove expired cache file %s: %w", entry.Name(), err)
					}
				}
			}
		}
	}

	return nil
}
