package ssh

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiter prevents SSH command flooding per server
type RateLimiter struct {
	mu              sync.Mutex
	serverLimits    map[string]*serverLimit
	defaultInterval time.Duration
	maxConcurrent   int
}

type serverLimit struct {
	lastExecution time.Time
	running       int
}

// NewRateLimiter creates a rate limiter with default settings
func NewRateLimiter(defaultInterval time.Duration, maxConcurrent int) *RateLimiter {
	if defaultInterval == 0 {
		defaultInterval = 10 * time.Second // default: 10s between commands
	}
	if maxConcurrent == 0 {
		maxConcurrent = 2 // default: max 2 concurrent commands per server
	}

	return &RateLimiter{
		serverLimits:    make(map[string]*serverLimit),
		defaultInterval: defaultInterval,
		maxConcurrent:   maxConcurrent,
	}
}

// Allow checks if a command can be executed for a server
func (r *RateLimiter) Allow(serverID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	limit, exists := r.serverLimits[serverID]

	if !exists {
		// First command for this server
		r.serverLimits[serverID] = &serverLimit{
			lastExecution: now,
			running:       1,
		}
		return nil
	}

	// Check concurrent commands
	if limit.running >= r.maxConcurrent {
		return fmt.Errorf("max concurrent commands (%d) reached for server %s", r.maxConcurrent, serverID)
	}

	// Check time interval
	elapsed := now.Sub(limit.lastExecution)
	if elapsed < r.defaultInterval {
		remaining := r.defaultInterval - elapsed
		return fmt.Errorf("rate limit: wait %v before next command", remaining.Round(time.Second))
	}

	// Allow execution
	limit.lastExecution = now
	limit.running++

	return nil
}

// Release marks a command as finished
func (r *RateLimiter) Release(serverID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if limit, exists := r.serverLimits[serverID]; exists {
		limit.running--
		if limit.running < 0 {
			limit.running = 0
		}
	}
}

// AllowWithRelease wraps Allow and returns a release function
func (r *RateLimiter) AllowWithRelease(serverID string) (func(), error) {
	if err := r.Allow(serverID); err != nil {
		return nil, err
	}

	return func() {
		r.Release(serverID)
	}, nil
}

// Reset clears rate limit state for a server
func (r *RateLimiter) Reset(serverID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.serverLimits, serverID)
}

// GetStats returns current rate limit stats for a server
func (r *RateLimiter) GetStats(serverID string) (running int, lastExec time.Time, exists bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	limit, exists := r.serverLimits[serverID]
	if !exists {
		return 0, time.Time{}, false
	}

	return limit.running, limit.lastExecution, true
}
