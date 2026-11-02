package service_test

import (
	"testing"
	"time"

	"github.com/kilip/omed/api/internal/agent/service"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitPerUser(t *testing.T) {
	currentTime := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return currentTime }

	limiter := service.NewRateLimiter(clock, 10, time.Minute)

	user1 := int64(100)
	user2 := int64(200)

	// Consume 10 allowed attempts for user 1
	for i := 0; i < 10; i++ {
		assert.True(t, limiter.Allow(user1), "attempt %d should be allowed", i+1)
	}

	// 11th attempt within a minute should be rate limited
	assert.False(t, limiter.Allow(user1), "11th attempt within 1 minute should be blocked")

	// User 2 should be unaffected
	assert.True(t, limiter.Allow(user2), "user 2 should be unaffected")

	// After clock advances by 1 minute, user 1 should be refilled
	currentTime = currentTime.Add(1 * time.Minute)
	assert.True(t, limiter.Allow(user1), "user 1 should be allowed after 1 minute refill")
}

func TestRateLimiter_CustomCapacity(t *testing.T) {
	currentTime := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return currentTime }

	limiter := service.NewRateLimiter(clock, 2, time.Minute)

	userID := int64(100)

	// Capacity 2 allows two calls
	assert.True(t, limiter.Allow(userID), "first call should be allowed")
	assert.True(t, limiter.Allow(userID), "second call should be allowed")

	// Rejects the third
	assert.False(t, limiter.Allow(userID), "third call should be blocked")

	// Allows again after 30 seconds on the injected clock
	currentTime = currentTime.Add(30 * time.Second)
	assert.True(t, limiter.Allow(userID), "should be allowed after 30 seconds refill")
	assert.False(t, limiter.Allow(userID), "subsequent call without refill should be blocked")
}

func TestRateLimiter_ZeroOrNegativeCapacity(t *testing.T) {
	limiterZero := service.NewRateLimiter(nil, 0, time.Minute)
	assert.False(t, limiterZero.Allow(100), "zero capacity should disallow")

	limiterNegative := service.NewRateLimiter(nil, -1, time.Minute)
	assert.False(t, limiterNegative.Allow(100), "negative capacity should disallow")
}

func TestRateLimiter_NilClock(t *testing.T) {
	limiter := service.NewRateLimiter(nil, 5, time.Minute)
	assert.True(t, limiter.Allow(100), "nil clock defaults to time.Now and allows")
}
