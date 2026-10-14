// Package repository provides database access implementations for user, account, and session entities using Ent.
package repository

import (
	"context"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/session"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
)

// SessionRepository manages database persistence for active user sessions.
type SessionRepository struct {
	cli *ent.Client
}

// NewSessionRepository creates a new SessionRepository.
func NewSessionRepository(cli *ent.Client) SessionRepository {
	return SessionRepository{cli: cli}
}

// Create inserts a new session record into the database.
func (r SessionRepository) Create(ctx context.Context, s domain.Session) (domain.Session, error) {
	builder := r.cli.Session.Create().
		SetUserID(s.UserID).
		SetToken(s.Token).
		SetExpiresAt(s.ExpiresAt)

	if s.IPAddress != "" {
		builder.SetIPAddress(s.IPAddress)
	}
	if s.UserAgent != "" {
		builder.SetUserAgent(s.UserAgent)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return domain.Session{}, err
	}

	return toSessionDomain(created), nil
}

// FindByToken retrieves a session by its unique token.
func (r SessionRepository) FindByToken(ctx context.Context, token string) (domain.Session, error) {
	var s domain.Session
	record, err := r.cli.Session.Query().
		Where(session.TokenEQ(token)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return s, base.ErrNotFound
		}
		return s, err
	}
	return toSessionDomain(record), nil
}

// FindByUserID retrieves all active sessions for a user ID.
func (r SessionRepository) FindByUserID(ctx context.Context, userID int64) ([]domain.Session, error) {
	records, err := r.cli.Session.Query().
		Where(session.UserIDEQ(userID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]domain.Session, len(records))
	for i, rec := range records {
		res[i] = toSessionDomain(rec)
	}
	return res, nil
}

// DeleteByToken removes a session by its token.
func (r SessionRepository) DeleteByToken(ctx context.Context, token string) error {
	_, err := r.cli.Session.Delete().
		Where(session.TokenEQ(token)).
		Exec(ctx)
	return err
}

// DeleteByUserID removes all sessions belonging to a user ID.
func (r SessionRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	_, err := r.cli.Session.Delete().
		Where(session.UserIDEQ(userID)).
		Exec(ctx)
	return err
}

// DeleteExpired purges expired sessions older than current time.
func (r SessionRepository) DeleteExpired(ctx context.Context) (int, error) {
	return r.cli.Session.Delete().
		Where(session.ExpiresAtLT(time.Now())).
		Exec(ctx)
}

func toSessionDomain(s *ent.Session) domain.Session {
	return domain.Session{
		ID:        s.ID,
		Token:     s.Token,
		UserID:    s.UserID,
		ExpiresAt: s.ExpiresAt,
		IPAddress: s.IPAddress,
		UserAgent: s.UserAgent,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
