// Package service provides business logic and orchestration for the core module.
package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/core/repository"
	"github.com/kilip/omed/api/internal/platform/oauthx"
)

// AccountRepository defines persistence operations for OAuth accounts.
type AccountRepository interface {
	FindByProviderAndAccountID(ctx context.Context, providerID, accountID string) (domain.Account, error)
	FindByUserID(ctx context.Context, userID int64) ([]domain.Account, error)
	Upsert(ctx context.Context, a domain.Account) (domain.Account, error)
	DeleteByUserID(ctx context.Context, userID int64) error
}

// OAuthService handles third-party OAuth2 flows, user linking, and session initialization.
type OAuthService interface {
	GetAuthURL(req model.SignInSocialRequest) (model.SignInSocialResponse, error)
	HandleCallback(ctx context.Context, provider string, code string, state string, expectedNonce string, ipAddress string, userAgent string) (domain.Session, domain.User, string, error)
}

type oauthService struct {
	accounts       AccountRepository
	users          repository.UserRepository
	sessions       SessionService
	authSecret     string
	apiURL         string
	trustedOrigins []string
	drivers        *oauthx.Registry
	logger         *slog.Logger
}

// OAuthServiceParams defines initialization options for NewOAuthService.
type OAuthServiceParams struct {
	Accounts       AccountRepository
	Users          repository.UserRepository
	Sessions       SessionService
	AuthSecret     string
	ApiURL         string
	TrustedOrigins []string
	Drivers        *oauthx.Registry
	Logger         *slog.Logger
}

// NewOAuthService constructs an OAuthService configured with providers and verification keys.
func NewOAuthService(p OAuthServiceParams) OAuthService {
	logger := p.Logger
	if logger == nil {
		logger = slog.Default()
	}

	apiURL := strings.TrimRight(p.ApiURL, "/")
	if apiURL == "" {
		apiURL = "http://localhost:8001"
	}

	drivers := p.Drivers
	if drivers == nil {
		drivers = oauthx.NewRegistry()
	}

	return &oauthService{
		accounts:       p.Accounts,
		users:          p.Users,
		sessions:       p.Sessions,
		authSecret:     p.AuthSecret,
		apiURL:         apiURL,
		trustedOrigins: p.TrustedOrigins,
		drivers:        drivers,
		logger:         logger,
	}
}

func (s *oauthService) driver(provider string) (oauthx.Driver, error) {
	d, err := s.drivers.Get(provider)
	if err != nil {
		if errors.Is(err, oauthx.ErrUnsupportedProvider) {
			return nil, domain.ErrInvalidProvider
		}
		return nil, err
	}
	return d, nil
}

func (s *oauthService) redirectURL(provider string) string {
	apiURL := strings.TrimRight(s.apiURL, "/")
	if apiURL == "" {
		apiURL = "http://localhost:8001"
	}
	return fmt.Sprintf("%s/auth/callback/%s", apiURL, provider)
}

func profileToUserInfo(p *oauthx.Profile) model.OAuthUserInfo {
	if p == nil {
		return model.OAuthUserInfo{}
	}
	return model.OAuthUserInfo{
		ID:            p.ID,
		Email:         p.Email,
		Name:          p.Name,
		Avatar:        p.Avatar,
		EmailVerified: p.EmailVerified,
		Locale:        p.Locale,
	}
}

func (s *oauthService) GetAuthURL(req model.SignInSocialRequest) (model.SignInSocialResponse, error) {
	d, err := s.driver(req.Provider)
	if err != nil {
		return model.SignInSocialResponse{}, err
	}

	// Validate callbackURL against trusted origins if provided
	if req.CallbackURL != "" {
		if !s.isAllowedURL(req.CallbackURL) {
			return model.SignInSocialResponse{}, fmt.Errorf("callbackURL is not in trusted origins: %s", req.CallbackURL)
		}
	}

	state, nonce, err := s.generateState(req)
	if err != nil {
		return model.SignInSocialResponse{}, err
	}

	authURL := d.GetAuthURL(state, s.redirectURL(req.Provider))

	return model.SignInSocialResponse{
		URL:      authURL,
		Redirect: true,
		Nonce:    nonce,
	}, nil
}

func (s *oauthService) HandleCallback(
	ctx context.Context,
	provider string,
	code string,
	state string,
	expectedNonce string,
	ipAddress string,
	userAgent string,
) (domain.Session, domain.User, string, error) {
	claims, err := s.verifyState(state, provider)
	if err != nil {
		return domain.Session{}, domain.User{}, "", domain.ErrInvalidState
	}

	if expectedNonce == "" || claims.Nonce != expectedNonce {
		return domain.Session{}, domain.User{}, "", domain.ErrInvalidState
	}

	if claims.CallbackURL != "" && !s.isAllowedURL(claims.CallbackURL) {
		return domain.Session{}, domain.User{}, "", domain.ErrInvalidState
	}

	d, err := s.driver(claims.Provider)
	if err != nil {
		return domain.Session{}, domain.User{}, "", err
	}

	token, err := d.ExchangeCode(ctx, code, s.redirectURL(claims.Provider))
	if err != nil {
		return domain.Session{}, domain.User{}, "", fmt.Errorf("%w: %v", domain.ErrOAuthExchangeFailed, err)
	}

	profile, err := d.FetchUserProfile(ctx, token)
	if err != nil {
		return domain.Session{}, domain.User{}, "", fmt.Errorf("%w: %v", domain.ErrFetchUserInfoFailed, err)
	}

	userInfo := profileToUserInfo(profile)
	if claims.Locale != "" {
		userInfo.Locale = claims.Locale
	}

	// Resolve or create user
	usr, err := s.resolveUser(ctx, provider, userInfo)
	if err != nil {
		return domain.Session{}, domain.User{}, "", err
	}

	// Upsert account with fresh token credentials
	var expiry *time.Time
	if !token.Expiry.IsZero() {
		exp := token.Expiry
		expiry = &exp
	}

	idToken, _ := token.Extra("id_token").(string)
	scope, _ := token.Extra("scope").(string)
	if scope == "" {
		scope = strings.Join(d.Scopes(), " ")
	}

	_, err = s.accounts.Upsert(ctx, domain.Account{
		UserID:               usr.ID,
		ProviderID:           provider,
		AccountID:            userInfo.ID,
		AccessToken:          token.AccessToken,
		RefreshToken:         token.RefreshToken,
		IDToken:              idToken,
		AccessTokenExpiresAt: expiry,
		Scope:                scope,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to upsert user oauth account",
			slog.String("provider", provider),
			slog.Int64("user_id", usr.ID),
			slog.Any("error", err),
		)
		return domain.Session{}, domain.User{}, "", fmt.Errorf("failed to save oauth account: %w", err)
	}

	// Create user session in database
	sess, err := s.sessions.CreateSession(ctx, usr.ID, ipAddress, userAgent)
	if err != nil {
		return domain.Session{}, domain.User{}, "", err
	}

	redirectURL := claims.CallbackURL
	if redirectURL == "" {
		redirectURL = "/"
	}

	return sess, usr, redirectURL, nil
}

func (s *oauthService) resolveUser(ctx context.Context, provider string, info model.OAuthUserInfo) (domain.User, error) {
	// 1. Check if account already exists
	acc, err := s.accounts.FindByProviderAndAccountID(ctx, provider, info.ID)
	if err == nil {
		usr, err := s.users.GetByID(ctx, acc.UserID)
		if err == nil {
			return usr, nil
		}
	}

	// 2. Ensure email is verified before linking to existing user or provisioning new user
	if !info.EmailVerified {
		return domain.User{}, domain.ErrEmailNotVerified
	}

	// 3. Check if user with same email exists
	if info.Email != "" {
		usr, err := s.users.FindByEmail(ctx, info.Email)
		if err == nil {
			return usr, nil
		}
	}

	// 4. User does not exist, provision new user (which also auto-creates personal workspace)
	name := info.Name
	if name == "" {
		name = strings.Split(info.Email, "@")[0]
	}
	if name == "" {
		name = fmt.Sprintf("User-%s", info.ID)
	}

	locale, currency := NormalizeLocaleAndCurrency(info.Locale)
	createdUser, err := s.users.Create(ctx, model.CreateUserRequest{
		Name:          name,
		Email:         info.Email,
		Avatar:        info.Avatar,
		Locale:        locale,
		Currency:      currency,
		EmailVerified: info.EmailVerified,
	})
	if err != nil {
		// In case of race condition on email, try to fetch again
		if errors.Is(err, domain.ErrEmailTaken) {
			return s.users.FindByEmail(ctx, info.Email)
		}
		return domain.User{}, err
	}

	return createdUser, nil
}

func (s *oauthService) generateState(req model.SignInSocialRequest) (string, string, error) {
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return "", "", err
	}
	nonce := hex.EncodeToString(nonceBytes)

	claims := model.OAuthStateClaims{
		Nonce:            nonce,
		Provider:         req.Provider,
		CallbackURL:      req.CallbackURL,
		ErrorCallbackURL: req.ErrorCallbackURL,
		Locale:           req.Locale,
		ExpiresAt:        time.Now().Add(10 * time.Minute).Unix(),
	}

	data, err := json.Marshal(claims)
	if err != nil {
		return "", "", err
	}

	payload := base64.RawURLEncoding.EncodeToString(data)
	sig := s.sign(payload)

	return fmt.Sprintf("%s.%s", payload, sig), nonce, nil
}

func (s *oauthService) verifyState(state string, expectedProvider string) (model.OAuthStateClaims, error) {
	parts := strings.Split(state, ".")
	if len(parts) != 2 {
		return model.OAuthStateClaims{}, errors.New("invalid state format")
	}

	payload, sig := parts[0], parts[1]
	expectedSig := s.sign(payload)
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return model.OAuthStateClaims{}, errors.New("state signature mismatch")
	}

	data, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return model.OAuthStateClaims{}, err
	}

	var claims model.OAuthStateClaims
	if err := json.Unmarshal(data, &claims); err != nil {
		return model.OAuthStateClaims{}, err
	}

	if claims.ExpiresAt < time.Now().Unix() {
		return model.OAuthStateClaims{}, errors.New("state expired")
	}

	if claims.Provider != expectedProvider {
		return model.OAuthStateClaims{}, errors.New("provider mismatch in state")
	}

	return claims, nil
}

func (s *oauthService) sign(payload string) string {
	mac := hmac.New(sha256.New, []byte(s.authSecret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *oauthService) isAllowedURL(targetURL string) bool {
	// Relative paths are allowed
	if strings.HasPrefix(targetURL, "/") && !strings.HasPrefix(targetURL, "//") {
		return true
	}

	parsed, err := url.Parse(targetURL)
	if err != nil || parsed.Host == "" {
		return false
	}

	targetOrigin := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	for _, allowed := range s.trustedOrigins {
		if strings.TrimRight(allowed, "/") == strings.TrimRight(targetOrigin, "/") {
			return true
		}
	}

	return false
}

// NormalizeLocaleAndCurrency normalizes raw browser/client locale string into a supported language code
// and determines the corresponding default currency code.
func NormalizeLocaleAndCurrency(raw string) (string, string) {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if strings.HasPrefix(lower, "id") {
		return "id", "IDR"
	}
	return "en", "USD"
}
