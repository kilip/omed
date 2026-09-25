package config

import (
	"context"
	"log"
	"math"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type JWKSManager struct {
	jwksURL string
	kf      atomic.Pointer[keyfunc.Keyfunc]
	ready   atomic.Bool
}

func NewJWKSManager(jwksURL string) *JWKSManager {
	return &JWKSManager{jwksURL: jwksURL}
}

// Start fetches JWKS in background with exponential backoff + jitter.
// Never gives up — auth is a hard dependency, not optional.
func (m *JWKSManager) Start(ctx context.Context) {
	go m.run(ctx)
}

func (m *JWKSManager) run(ctx context.Context) {
	const (
		baseDelay = 1 * time.Second
		maxDelay  = 30 * time.Second
	)
	for attempt := 0; ; attempt++ {
		// keyfunc/v3 default options already include: RefreshInterval (1h),
		// RefreshUnknownKID (true), RefreshRateLimit — covers point #3.
		k, err := keyfunc.NewDefaultCtx(ctx, []string{m.jwksURL})
		if err == nil {
			m.kf.Store(&k)
			m.ready.Store(true)
			log.Printf("jwks: ready (source=%s)", m.jwksURL)
			return
		}

		delay := math.Min(float64(maxDelay), float64(baseDelay)*math.Pow(2, float64(attempt)))
		wait := time.Duration(delay/2) + time.Duration(rand.Int63n(int64(delay)/2+1))

		log.Printf("jwks: fetch failed (attempt %d), retry in %s: %v", attempt+1, wait, err)

		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
	}
}

func (m *JWKSManager) Ready() bool { return m.ready.Load() }

func (m *JWKSManager) Keyfunc() jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		k := m.kf.Load()
		if k == nil {
			return nil, jwt.ErrTokenUnverifiable
		}
		return (*k).Keyfunc(token)
	}
}
