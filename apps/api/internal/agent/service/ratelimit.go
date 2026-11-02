package service

import (
	"sync"
	"time"
)

// RateLimiter manages per-user rate limiting for test operations.
type RateLimiter struct {
	mu       sync.Mutex
	now      func() time.Time
	capacity int
	per      time.Duration
	buckets  map[int64]*userBucket
}

// NewRateLimiter creates a new RateLimiter.
func NewRateLimiter(now func() time.Time, capacity int, per time.Duration) *RateLimiter {
	if now == nil {
		now = time.Now
	}
	return &RateLimiter{
		now:      now,
		capacity: capacity,
		per:      per,
	}
}

// Allow reports whether a test operation is permitted for the given user ID.
func (r *RateLimiter) Allow(userID int64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.capacity <= 0 {
		return false
	}

	if r.buckets == nil {
		r.buckets = make(map[int64]*userBucket)
	}

	now := r.now()
	b, ok := r.buckets[userID]
	if !ok {
		r.buckets[userID] = &userBucket{
			tokens:     float64(r.capacity) - 1.0, // consume 1 immediately from capacity
			lastRefill: now,
		}
		return true
	}

	// Refill tokens evenly over per duration
	elapsed := now.Sub(b.lastRefill)
	if elapsed > 0 && r.per > 0 {
		refill := elapsed.Seconds() * (float64(r.capacity) / r.per.Seconds())
		b.tokens += refill
		if b.tokens > float64(r.capacity) {
			b.tokens = float64(r.capacity)
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
