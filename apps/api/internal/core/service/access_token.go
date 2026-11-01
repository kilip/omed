// Package service provides business logic and orchestration for the core module.
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/core/repository"
)

const (
	// AccessTokenPrefix is the required prefix for personal access tokens.
	AccessTokenPrefix = "omed_pat_"
	// MaxActiveAccessTokens is the maximum number of active tokens a user can hold simultaneously.
	MaxActiveAccessTokens = 20
	// MaxAccessTokenNameLen is the maximum allowed length in runes for a token name.
	MaxAccessTokenNameLen = 100
	// DefaultAccessTokenExpiryDays is the default expiration period in days when not specified.
	DefaultAccessTokenExpiryDays = 90
)

// AccessTokenService specifies business operations for personal access tokens.
type AccessTokenService interface {
	Create(ctx context.Context, userID int64, req model.CreateAccessTokenRequest) (domain.AccessToken, string, error)
	List(ctx context.Context, userID int64) ([]domain.AccessToken, error)
	Revoke(ctx context.Context, userID, id int64) error
	Verify(ctx context.Context, token string) (domain.AccessToken, error)
}

type accessTokenService struct {
	tokens repository.AccessTokenRepository
	now    func() time.Time
}

// NewAccessTokenService constructs a new AccessTokenService instance.
func NewAccessTokenService(tokens repository.AccessTokenRepository) AccessTokenService {
	return &accessTokenService{
		tokens: tokens,
		now:    time.Now,
	}
}

func (s *accessTokenService) Create(ctx context.Context, userID int64, req model.CreateAccessTokenRequest) (domain.AccessToken, string, error) {
	name := strings.TrimSpace(req.Name)
	runeCount := utf8.RuneCountInString(name)
	if runeCount == 0 || runeCount > MaxAccessTokenNameLen {
		return domain.AccessToken{}, "", domain.ErrTokenInvalidName
	}

	expiresInDays := req.ExpiresInDays
	if expiresInDays == 0 {
		expiresInDays = DefaultAccessTokenExpiryDays
	}
	if expiresInDays != 30 && expiresInDays != 90 && expiresInDays != 365 {
		return domain.AccessToken{}, "", domain.ErrTokenInvalidExpiry
	}

	now := s.now()
	count, err := s.tokens.CountActiveByUser(ctx, userID, now)
	if err != nil {
		return domain.AccessToken{}, "", err
	}
	if count >= MaxActiveAccessTokens {
		return domain.AccessToken{}, "", domain.ErrTokenLimitReached
	}

	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return domain.AccessToken{}, "", err
	}
	plaintext := AccessTokenPrefix + base64.RawURLEncoding.EncodeToString(rawBytes)
	prefix := plaintext[:12]
	tokenHash := hashToken(plaintext)
	expiresAt := now.AddDate(0, 0, expiresInDays)

	domainToken := domain.AccessToken{
		UserID:    userID,
		Name:      name,
		TokenHash: tokenHash,
		Prefix:    prefix,
		ExpiresAt: expiresAt,
	}

	created, err := s.tokens.Create(ctx, domainToken)
	if err != nil {
		return domain.AccessToken{}, "", err
	}

	return created, plaintext, nil
}

func (s *accessTokenService) List(ctx context.Context, userID int64) ([]domain.AccessToken, error) {
	list, err := s.tokens.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].TokenHash = ""
	}
	if list == nil {
		list = []domain.AccessToken{}
	}
	return list, nil
}

func (s *accessTokenService) Revoke(ctx context.Context, userID, id int64) error {
	return s.tokens.Revoke(ctx, userID, id, s.now())
}

func (s *accessTokenService) Verify(ctx context.Context, token string) (domain.AccessToken, error) {
	if !strings.HasPrefix(token, AccessTokenPrefix) {
		return domain.AccessToken{}, domain.ErrTokenInvalid
	}

	tokenHash := hashToken(token)
	tok, err := s.tokens.FindByHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, base.ErrNotFound) {
			return domain.AccessToken{}, domain.ErrTokenInvalid
		}
		return domain.AccessToken{}, err
	}

	status := tok.Status(s.now())
	switch status {
	case domain.AccessTokenRevoked:
		return domain.AccessToken{}, domain.ErrTokenInvalid
	case domain.AccessTokenExpired:
		return domain.AccessToken{}, domain.ErrTokenExpired
	case domain.AccessTokenActive:
		return tok, nil
	default:
		return domain.AccessToken{}, domain.ErrTokenInvalid
	}
}
