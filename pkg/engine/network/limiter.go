// Luminary: A streamlined CLI tool for searching and downloading manga.
// Copyright (C) 2025 Luca M. Schmidt (LuMiSxh)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package network

import (
	"context"
	"net/url"
	"sync"
	"time"

	"Luminary/pkg/errors"
)

// RateLimiter provides per-domain rate limiting
type RateLimiter struct {
	domains map[string]*domainLimiter
	mu      sync.RWMutex
}

// domainLimiter tracks rate limiting for a specific domain
type domainLimiter struct {
	lastRequest time.Time
	delay       time.Duration
	mu          sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		domains: make(map[string]*domainLimiter),
	}
}

// Wait enforces rate limiting for the given URL
func (r *RateLimiter) Wait(ctx context.Context, rawURL string, delay time.Duration) error {
	domain, err := extractDomain(rawURL)
	if err != nil {
		return errors.Track(err).WithContext("url", rawURL).Error()
	}

	limiter := r.getLimiter(domain)
	return limiter.wait(ctx, delay)
}

// WaitForDomain enforces rate limiting for a specific domain
func (r *RateLimiter) WaitForDomain(ctx context.Context, domain string, delay time.Duration) error {
	limiter := r.getLimiter(domain)
	return limiter.wait(ctx, delay)
}

// SetDefaultDelay sets a default delay for a domain
func (r *RateLimiter) SetDefaultDelay(domain string, delay time.Duration) {
	limiter := r.getLimiter(domain)
	limiter.mu.Lock()
	limiter.delay = delay
	limiter.mu.Unlock()
}

// Reset clears rate limiting for a domain
func (r *RateLimiter) Reset(domain string) {
	r.mu.Lock()
	delete(r.domains, domain)
	r.mu.Unlock()
}

// ResetAll clears all rate limiting
func (r *RateLimiter) ResetAll() {
	r.mu.Lock()
	r.domains = make(map[string]*domainLimiter)
	r.mu.Unlock()
}

// getLimiter returns or creates a limiter for a domain
func (r *RateLimiter) getLimiter(domain string) *domainLimiter {
	r.mu.RLock()
	limiter, exists := r.domains[domain]
	r.mu.RUnlock()

	if exists {
		return limiter
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists := r.domains[domain]; exists {
		return limiter
	}

	limiter = &domainLimiter{}
	r.domains[domain] = limiter
	return limiter
}

// wait enforces the rate limit
func (l *domainLimiter) wait(ctx context.Context, delay time.Duration) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Use provided delay or default
	if delay == 0 {
		delay = l.delay
	}
	if delay == 0 {
		return nil // No rate limiting
	}

	// Calculate time to wait
	elapsed := time.Since(l.lastRequest)
	if elapsed < delay {
		waitTime := delay - elapsed

		// Wait with context
		timer := time.NewTimer(waitTime)
		defer timer.Stop()

		select {
		case <-timer.C:
			// Wait completed
		case <-ctx.Done():
			return errors.Track(ctx.Err()).
				WithContext("wait_time", waitTime).
				AsNetwork().
				Error()
		}
	}

	l.lastRequest = time.Now()
	return nil
}

// extractDomain extracts the domain from a URL
func extractDomain(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	return u.Host, nil
}

// ExtractDomain is a public helper to extract domain from URL
func ExtractDomain(rawURL string) string {
	domain, _ := extractDomain(rawURL)
	return domain
}
