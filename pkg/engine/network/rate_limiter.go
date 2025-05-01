package network

import (
	"Luminary/pkg/engine/logger"
	"sync"
	"time"
)

// RateLimiterService manages rate limiting for different domains
type RateLimiterService struct {
	limiters     map[string]*Limiter
	defaultLimit time.Duration
	mu           sync.RWMutex
	Logger       *logger.Service
}

// Limiter represents a rate limiter for a specific domain
type Limiter struct {
	interval time.Duration
	lastUsed time.Time
	mu       sync.Mutex
}

// NewRateLimiterService creates a new rate limiter service
func NewRateLimiterService(defaultLimit time.Duration, logger *logger.Service) *RateLimiterService {
	if logger != nil {
		logger.Debug("[RATELIMITER] Creating new service with default limit of %v", defaultLimit)
	}

	return &RateLimiterService{
		limiters:     make(map[string]*Limiter),
		defaultLimit: defaultLimit,
		Logger:       logger,
	}
}

// SetLimit sets the rate limit for a specific domain
func (r *RateLimiterService) SetLimit(domain string, interval time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Logger != nil {
		r.Logger.Debug("[RATELIMITER] Setting limit for domain '%s' to %v", domain, interval)
	}

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
		if r.Logger != nil {
			r.Logger.Debug("[RATELIMITER] Created new limiter for domain '%s' with default interval %v", domain, r.defaultLimit)
		}
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
			waitTime := limiter.interval - elapsed
			if r.Logger != nil {
				r.Logger.Debug("[RATELIMITER] Domain '%s': waiting %v", domain, waitTime)
			}
			time.Sleep(waitTime)
		}
	}

	// Update last used time
	limiter.lastUsed = time.Now()
	if r.Logger != nil {
		r.Logger.Debug("[RATELIMITER] Domain '%s' updated, next request allowed after %v",
			domain, time.Now().Add(limiter.interval).Format("15:04:05.000"))
	}
}
