package network

import (
	"sync"
	"time"
)

// RateLimiterService manages rate limiting for different domains
type RateLimiterService struct {
	limiters     map[string]*Limiter
	defaultLimit time.Duration
	mu           sync.RWMutex
}

// Limiter represents a rate limiter for a specific domain
type Limiter struct {
	interval time.Duration
	lastUsed time.Time
	mu       sync.Mutex
}

// NewRateLimiterService creates a new rate limiter service
func NewRateLimiterService(defaultLimit time.Duration) *RateLimiterService {
	return &RateLimiterService{
		limiters:     make(map[string]*Limiter),
		defaultLimit: defaultLimit,
	}
}

// SetLimit sets the rate limit for a specific domain
func (r *RateLimiterService) SetLimit(domain string, interval time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.limiters[domain] = &Limiter{interval: interval}
}

// Wait waits until the rate limit allows a request
func (r *RateLimiterService) Wait(domain string) {
	r.mu.RLock()
	limiter, exists := r.limiters[domain]
	if !exists {
		// Create a new limiter if one doesn't exist
		r.mu.RUnlock()
		r.mu.Lock()
		limiter = &Limiter{interval: r.defaultLimit}
		r.limiters[domain] = limiter
		r.mu.Unlock()
	} else {
		r.mu.RUnlock()
	}

	// Acquire limiter lock
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	// Check if we need to wait
	if !limiter.lastUsed.IsZero() {
		elapsed := time.Since(limiter.lastUsed)
		if elapsed < limiter.interval {
			time.Sleep(limiter.interval - elapsed)
		}
	}

	// Update last used time
	limiter.lastUsed = time.Now()
}
