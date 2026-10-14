// Package repository provides database access implementations for user and account entities using Ent.
package repository

import (
	"context"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/useraccount"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
)

// AccountRepository manages persistence for OAuth accounts.
type AccountRepository struct {
	cli *ent.Client
}

// NewAccountRepository creates a new AccountRepository.
func NewAccountRepository(cli *ent.Client) AccountRepository {
	return AccountRepository{cli: cli}
}

// FindByProviderAndAccountID retrieves an account by provider ID and external account ID.
func (r AccountRepository) FindByProviderAndAccountID(ctx context.Context, providerID, accountID string) (domain.Account, error) {
	var acc domain.Account
	a, err := r.cli.UserAccount.Query().
		Where(
			useraccount.ProviderIDEQ(providerID),
			useraccount.AccountIDEQ(accountID),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return acc, base.ErrNotFound
		}
		return acc, err
	}
	return toAccountDomain(a), nil
}

// FindByUserID retrieves all accounts linked to a given user ID.
func (r AccountRepository) FindByUserID(ctx context.Context, userID int64) ([]domain.Account, error) {
	accounts, err := r.cli.UserAccount.Query().
		Where(useraccount.UserIDEQ(userID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]domain.Account, len(accounts))
	for i, a := range accounts {
		res[i] = toAccountDomain(a)
	}
	return res, nil
}

// Upsert inserts a new account or updates an existing account with latest token data.
func (r AccountRepository) Upsert(ctx context.Context, a domain.Account) (domain.Account, error) {
	builder := r.cli.UserAccount.Create().
		SetUserID(a.UserID).
		SetProviderID(a.ProviderID).
		SetAccountID(a.AccountID)

	if a.AccessToken != "" {
		builder.SetAccessToken(a.AccessToken)
	}
	if a.RefreshToken != "" {
		builder.SetRefreshToken(a.RefreshToken)
	}
	if a.IDToken != "" {
		builder.SetIDToken(a.IDToken)
	}
	if a.AccessTokenExpiresAt != nil {
		builder.SetAccessTokenExpiresAt(*a.AccessTokenExpiresAt)
	}
	if a.RefreshTokenExpiresAt != nil {
		builder.SetRefreshTokenExpiresAt(*a.RefreshTokenExpiresAt)
	}
	if a.Scope != "" {
		builder.SetScope(a.Scope)
	}
	if a.Password != "" {
		builder.SetPassword(a.Password)
	}

	err := builder.OnConflictColumns(useraccount.FieldProviderID, useraccount.FieldAccountID).
		UpdateNewValues().
		Exec(ctx)
	if err != nil {
		return domain.Account{}, err
	}

	return r.FindByProviderAndAccountID(ctx, a.ProviderID, a.AccountID)
}

// DeleteByUserID deletes all accounts associated with a user ID.
func (r AccountRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	_, err := r.cli.UserAccount.Delete().
		Where(useraccount.UserIDEQ(userID)).
		Exec(ctx)
	return err
}

func toAccountDomain(a *ent.UserAccount) domain.Account {
	acc := domain.Account{
		ID:           a.ID,
		UserID:       a.UserID,
		ProviderID:   a.ProviderID,
		AccountID:    a.AccountID,
		AccessToken:  a.AccessToken,
		RefreshToken: a.RefreshToken,
		IDToken:      a.IDToken,
		Scope:        a.Scope,
		Password:     a.Password,
		CreatedAt:    a.CreatedAt,
		UpdatedAt:    a.UpdatedAt,
	}
	if !a.AccessTokenExpiresAt.IsZero() {
		acc.AccessTokenExpiresAt = &a.AccessTokenExpiresAt
	}
	if !a.RefreshTokenExpiresAt.IsZero() {
		acc.RefreshTokenExpiresAt = &a.RefreshTokenExpiresAt
	}
	return acc
}
