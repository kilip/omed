package service

import (
	"sync"
	"time"
)

// RateLimiter manages per-user rate limiting for test operations.
type RateLimiter struct {
	mu      sync.Mutex
	now     func() time.Time
	buckets map[int64]*userBucket
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter(now func() time.Time) *RateLimiter {
	if now == nil {
		now = time.Now
	}
	return &RateLimiter{now: now}
}

// Allow reports whether a test operation is permitted for the given user ID.
func (r *RateLimiter) Allow(userID int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.buckets == nil {
		r.buckets = make(map[int64]*userBucket)
	}

	now := r.now()
	b, ok := r.buckets[userID]
	if !ok {
		r.buckets[userID] = &userBucket{
			tokens:     9.0, // consume 1 immediately from capacity 10
			lastRefill: now,
		}
		return true
	}

	// Refill tokens: 10 per minute = 10 / 60 = 1/6 tokens per second
	elapsed := now.Sub(b.lastRefill)
	if elapsed > 0 {
		refill := elapsed.Seconds() * (10.0 / 60.0)
		b.tokens += refill
		if b.tokens > 10.0 {
			b.tokens = 10.0
		}
		b.lastRefill = now
	}

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}

	return false
}

type userBucket struct {
	tokens     float64
	lastRefill time.Time
}
