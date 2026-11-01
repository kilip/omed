// Package repository provides database access implementations for user, account, session, and token entities using Ent.
package repository

import (
	"context"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/personalaccesstoken"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
)

// AccessTokenRepository manages database persistence for personal access tokens.
type AccessTokenRepository struct {
	cli *ent.Client
}

// NewAccessTokenRepository creates a new AccessTokenRepository.
func NewAccessTokenRepository(cli *ent.Client) AccessTokenRepository {
	return AccessTokenRepository{cli: cli}
}

// Create inserts a new personal access token record into the database.
func (r AccessTokenRepository) Create(ctx context.Context, t domain.AccessToken) (domain.AccessToken, error) {
	builder := r.cli.PersonalAccessToken.Create().
		SetUserID(t.UserID).
		SetName(t.Name).
		SetTokenHash(t.TokenHash).
		SetPrefix(t.Prefix).
		SetExpiresAt(t.ExpiresAt)

	if t.RevokedAt != nil {
		builder.SetRevokedAt(*t.RevokedAt)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return domain.AccessToken{}, err
	}

	return toAccessTokenDomain(created), nil
}

// FindByHash retrieves a token by its unique hash.
func (r AccessTokenRepository) FindByHash(ctx context.Context, hash string) (domain.AccessToken, error) {
	var t domain.AccessToken
	record, err := r.cli.PersonalAccessToken.Query().
		Where(personalaccesstoken.TokenHashEQ(hash)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return t, base.ErrNotFound
		}
		return t, err
	}
	return toAccessTokenDomain(record), nil
}

// ListByUser retrieves all personal access tokens for a user, newest first.
func (r AccessTokenRepository) ListByUser(ctx context.Context, userID int64) ([]domain.AccessToken, error) {
	records, err := r.cli.PersonalAccessToken.Query().
		Where(personalaccesstoken.UserIDEQ(userID)).
		Order(ent.Desc(personalaccesstoken.FieldCreatedAt), ent.Desc(personalaccesstoken.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]domain.AccessToken, len(records))
	for i, rec := range records {
		res[i] = toAccessTokenDomain(rec)
	}
	return res, nil
}

// CountActiveByUser counts active (non-expired, non-revoked) tokens for a user.
func (r AccessTokenRepository) CountActiveByUser(ctx context.Context, userID int64, now time.Time) (int, error) {
	return r.cli.PersonalAccessToken.Query().
		Where(
			personalaccesstoken.UserIDEQ(userID),
			personalaccesstoken.RevokedAtIsNil(),
			personalaccesstoken.ExpiresAtGT(now),
		).
		Count(ctx)
}

// Revoke marks a personal access token as revoked at the given time.
// It is idempotent and returns base.ErrNotFound if the token does not exist or belong to the user.
func (r AccessTokenRepository) Revoke(ctx context.Context, userID, id int64, at time.Time) error {
	record, err := r.cli.PersonalAccessToken.Query().
		Where(
			personalaccesstoken.IDEQ(id),
			personalaccesstoken.UserIDEQ(userID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return base.ErrNotFound
		}
		return err
	}

	if record.RevokedAt != nil {
		return nil
	}

	return r.cli.PersonalAccessToken.UpdateOne(record).
		SetRevokedAt(at).
		Exec(ctx)
}

func toAccessTokenDomain(t *ent.PersonalAccessToken) domain.AccessToken {
	return domain.AccessToken{
		ID:        t.ID,
		UserID:    t.UserID,
		Name:      t.Name,
		TokenHash: t.TokenHash,
		Prefix:    t.Prefix,
		ExpiresAt: t.ExpiresAt,
		RevokedAt: t.RevokedAt,
		CreatedAt: t.CreatedAt,
	}
}
