// Package repository provides database access implementations for user and account entities using Ent.
package repository

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/useraccount"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
)

// AccountRepository manages persistence for OAuth accounts.
type AccountRepository struct {
	cli *ent.Client
	key []byte
}

// NewAccountRepository creates a new AccountRepository with token encryption using the given secret.
func NewAccountRepository(cli *ent.Client, secret string) AccountRepository {
	k := sha256.Sum256([]byte(secret))
	return AccountRepository{
		cli: cli,
		key: k[:],
	}
}

func (r AccountRepository) cipher() (cipher.AEAD, error) {
	key := r.key
	if len(key) != 32 {
		k := sha256.Sum256(key)
		key = k[:]
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	return cipher.NewGCM(block)
}

func (r AccountRepository) encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	gcm, err := r.cipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("read random nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (r AccountRepository) decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode token: %w", err)
	}
	gcm, err := r.cipher()
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce := data[:gcm.NonceSize()]
	payload := data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt token: %w", err)
	}
	return string(plaintext), nil
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
	return r.toAccountDomain(a)
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
		da, err := r.toAccountDomain(a)
		if err != nil {
			return nil, err
		}
		res[i] = da
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
		enc, err := r.encrypt(a.AccessToken)
		if err != nil {
			return domain.Account{}, fmt.Errorf("encrypting access token: %w", err)
		}
		builder.SetAccessToken(enc)
	}
	if a.RefreshToken != "" {
		enc, err := r.encrypt(a.RefreshToken)
		if err != nil {
			return domain.Account{}, fmt.Errorf("encrypting refresh token: %w", err)
		}
		builder.SetRefreshToken(enc)
	}
	if a.IDToken != "" {
		enc, err := r.encrypt(a.IDToken)
		if err != nil {
			return domain.Account{}, fmt.Errorf("encrypting id token: %w", err)
		}
		builder.SetIDToken(enc)
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

func (r AccountRepository) toAccountDomain(a *ent.UserAccount) (domain.Account, error) {
	accessToken, err := r.decrypt(a.AccessToken)
	if err != nil {
		return domain.Account{}, fmt.Errorf("decrypting access token: %w", err)
	}
	refreshToken, err := r.decrypt(a.RefreshToken)
	if err != nil {
		return domain.Account{}, fmt.Errorf("decrypting refresh token: %w", err)
	}
	idToken, err := r.decrypt(a.IDToken)
	if err != nil {
		return domain.Account{}, fmt.Errorf("decrypting id token: %w", err)
	}

	acc := domain.Account{
		ID:           a.ID,
		UserID:       a.UserID,
		ProviderID:   a.ProviderID,
		AccountID:    a.AccountID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IDToken:      idToken,
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
	return acc, nil
}
