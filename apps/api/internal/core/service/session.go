// Package service provides business logic and orchestration for the core module.
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/repository"
)

// SessionService specifies operations for managing user login sessions.
type SessionService interface {
	CreateSession(ctx context.Context, userID int64, ipAddress string, userAgent string) (domain.Session, error)
	ValidateSession(ctx context.Context, token string) (domain.Session, domain.User, error)
	RevokeSession(ctx context.Context, token string) error
	RevokeAllUserSessions(ctx context.Context, userID int64) error
}

type sessionService struct {
	sessions repository.SessionRepository
	users    repository.UserRepository
	ttl      time.Duration
}

// NewSessionService constructs a new SessionService instance.
func NewSessionService(sessions repository.SessionRepository, users repository.UserRepository, ttl time.Duration) SessionService {
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	return &sessionService{
		sessions: sessions,
		users:    users,
		ttl:      ttl,
	}
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func (s *sessionService) CreateSession(ctx context.Context, userID int64, ipAddress string, userAgent string) (domain.Session, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return domain.Session{}, err
	}
	rawToken := hex.EncodeToString(bytes)
	tokenHash := hashToken(rawToken)

	newSession := domain.Session{
		UserID:    userID,
		Token:     tokenHash,
		ExpiresAt: time.Now().Add(s.ttl),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	created, err := s.sessions.Create(ctx, newSession)
	if err != nil {
		return domain.Session{}, err
	}
	created.Token = rawToken
	return created, nil
}

func (s *sessionService) ValidateSession(ctx context.Context, token string) (domain.Session, domain.User, error) {
	if token == "" {
		return domain.Session{}, domain.User{}, domain.ErrSessionNotFound
	}

	tokenHash := hashToken(token)
	sess, err := s.sessions.FindByToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, base.ErrNotFound) {
			return domain.Session{}, domain.User{}, domain.ErrSessionNotFound
		}
		return domain.Session{}, domain.User{}, err
	}

	if sess.ExpiresAt.Before(time.Now()) {
		_ = s.sessions.DeleteByToken(ctx, tokenHash)
		return domain.Session{}, domain.User{}, domain.ErrSessionExpired
	}

	usr, err := s.users.GetByID(ctx, sess.UserID)
	if err != nil {
		return domain.Session{}, domain.User{}, err
	}

	return sess, usr, nil
}

func (s *sessionService) RevokeSession(ctx context.Context, token string) error {
	return s.sessions.DeleteByToken(ctx, hashToken(token))
}

func (s *sessionService) RevokeAllUserSessions(ctx context.Context, userID int64) error {
	return s.sessions.DeleteByUserID(ctx, userID)
}
