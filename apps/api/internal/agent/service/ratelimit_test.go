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

	limiter := service.NewRateLimiter(clock)

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
