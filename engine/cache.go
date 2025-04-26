package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheService provides caching capabilities
type CacheService struct {
	// Cache parameters
	TTL          time.Duration
	CacheDir     string
	UseDiskCache bool

	// Memory cache
	memoryCache map[string]*cacheItem
	mutex       sync.RWMutex
}

type cacheItem struct {
	Value      interface{}
	Expiration time.Time
}

// CacheInfo holds information about the cache
type CacheInfo struct {
	Location   string        `json:"location"`
	Size       int64         `json:"size_bytes"`
	EntryCount int           `json:"entry_count"`
	TTL        time.Duration `json:"ttl"`
}

// NewCacheService creates a new cache service
func NewCacheService(ttl time.Duration, cacheDir string, useDiskCache bool) *CacheService {
	// Create cache directory if it doesn't exist
	if useDiskCache && cacheDir != "" {
		_ = os.MkdirAll(cacheDir, 0755)
	}

	return &CacheService{
		TTL:          ttl,
		CacheDir:     cacheDir,
		UseDiskCache: useDiskCache,
		memoryCache:  make(map[string]*cacheItem),
	}
}

// Get retrieves an item from the cache
func (c *CacheService) Get(key string, value interface{}) bool {
	// Check memory cache first
	c.mutex.RLock()
	item, exists := c.memoryCache[key]
	c.mutex.RUnlock()

	if exists && time.Now().Before(item.Expiration) {
		// Convert the value to the target type
		if err := convertValue(item.Value, value); err == nil {
			return true
		}
	}

	// If not found in memory or expired, try disk cache
	if !c.UseDiskCache || c.CacheDir == "" {
		return false
	}

	cacheFile := filepath.Join(c.CacheDir, hashKey(key))

	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return false
	}

	// Read expiration time from the first line
	var storedItem struct {
		Expiration time.Time
		Value      json.RawMessage
	}

	if err := json.Unmarshal(data, &storedItem); err != nil {
		return false
	}

	// Check if expired
	if time.Now().After(storedItem.Expiration) {
		// Clean up expired file
		_ = os.Remove(cacheFile)
		return false
	}

	// Unmarshal the actual value
	if err := json.Unmarshal(storedItem.Value, value); err != nil {
		return false
	}

	// Update memory cache
	c.mutex.Lock()
	c.memoryCache[key] = &cacheItem{
		Value:      value,
		Expiration: storedItem.Expiration,
	}
	c.mutex.Unlock()

	return true
}

// Set stores an item in the cache
func (c *CacheService) Set(key string, value interface{}) error {
	expiration := time.Now().Add(c.TTL)

	// Update memory cache
	c.mutex.Lock()
	c.memoryCache[key] = &cacheItem{
		Value:      value,
		Expiration: expiration,
	}
	c.mutex.Unlock()

	// If disk cache is disabled, we're done
	if !c.UseDiskCache || c.CacheDir == "" {
		return nil
	}

	// Create cache structure
	storedItem := struct {
		Expiration time.Time   `json:"expiration"`
		Value      interface{} `json:"value"`
	}{
		Expiration: expiration,
		Value:      value,
	}

	// Marshal to JSON
	data, err := json.Marshal(storedItem)
	if err != nil {
		return err
	}

	// Write to disk
	cacheFile := filepath.Join(c.CacheDir, hashKey(key))
	return os.WriteFile(cacheFile, data, 0644)
}

// Clear empties the cache entirely
func (c *CacheService) Clear() error {
	// Clear memory cache
	c.mutex.Lock()
	c.memoryCache = make(map[string]*cacheItem)
	c.mutex.Unlock()

	// Clear disk cache if enabled
	if c.UseDiskCache && c.CacheDir != "" {
		// Read the directory
		entries, err := os.ReadDir(c.CacheDir)
		if err != nil {
			return fmt.Errorf("failed to read cache directory: %w", err)
		}

		// Remove each file
		for _, entry := range entries {
			if !entry.IsDir() { // Skip directories
				err := os.Remove(filepath.Join(c.CacheDir, entry.Name()))
				if err != nil {
					return fmt.Errorf("failed to remove cache file %s: %w", entry.Name(), err)
				}
			}
		}
	}

	return nil
}

// CleanExpired removes expired entries from the cache
func (c *CacheService) CleanExpired() (int, error) {
	count := 0

	// Clean memory cache
	c.mutex.Lock()
	now := time.Now()
	for key, item := range c.memoryCache {
		if now.After(item.Expiration) {
			delete(c.memoryCache, key)
			count++
		}
	}
	c.mutex.Unlock()

	// Clean disk cache if enabled
	if c.UseDiskCache && c.CacheDir != "" {
		entries, err := os.ReadDir(c.CacheDir)
		if err != nil {
			return count, fmt.Errorf("failed to read cache directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue // Skip directories
			}

			cacheFile := filepath.Join(c.CacheDir, entry.Name())
			data, err := os.ReadFile(cacheFile)
			if err != nil {
				continue // Skip files we can't read
			}

			// Read expiration time
			var storedItem struct {
				Expiration time.Time
				Value      json.RawMessage
			}

			if err := json.Unmarshal(data, &storedItem); err != nil {
				// Invalid cache file, remove it
				_ = os.Remove(cacheFile)
				count++
				continue
			}

			// Check if expired
			if now.After(storedItem.Expiration) {
				_ = os.Remove(cacheFile)
				count++
			}
		}
	}

	return count, nil
}

// GetInfo returns information about the cache
func (c *CacheService) GetInfo() (*CacheInfo, error) {
	info := &CacheInfo{
		Location:   c.CacheDir,
		TTL:        c.TTL,
		EntryCount: 0,
		Size:       0,
	}

	// Count memory entries
	c.mutex.RLock()
	info.EntryCount = len(c.memoryCache)
	c.mutex.RUnlock()

	// Get disk cache info if enabled
	if c.UseDiskCache && c.CacheDir != "" {
		// Check if directory exists
		if _, err := os.Stat(c.CacheDir); errors.Is(err, os.ErrNotExist) {
			return info, nil // Return with zero size/entries if dir doesn't exist
		}

		entries, err := os.ReadDir(c.CacheDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read cache directory: %w", err)
		}

		// Count files and calculate total size
		for _, entry := range entries {
			if entry.IsDir() {
				continue // Skip directories
			}

			fileInfo, err := entry.Info()
			if err != nil {
				continue // Skip files we can't stat
			}

			info.Size += fileInfo.Size()
			info.EntryCount++ // Count disk entries only
		}
	}

	return info, nil
}

// convertValue converts a value from one type to another
func convertValue(src, dest interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// hashKey creates a hash of a cache key for disk storage
func hashKey(key string) string {
	// Simple implementation - in a real application you'd use a proper hashing function
	return fmt.Sprintf("%x", key)
}
