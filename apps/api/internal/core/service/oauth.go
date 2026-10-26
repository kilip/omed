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
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/core/repository"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// OAuthService handles third-party OAuth2 flows, user linking, and session initialization.
type OAuthService interface {
	GetAuthURL(req model.SignInSocialRequest) (model.SignInSocialResponse, error)
	HandleCallback(ctx context.Context, provider string, code string, state string, expectedNonce string, ipAddress string, userAgent string) (domain.Session, domain.User, string, error)
}

// OAuthProviderConfig holds credentials and endpoint URLs for an OAuth2 provider.
type OAuthProviderConfig struct {
	ID           string
	ClientID     string
	ClientSecret string
	Endpoint     oauth2.Endpoint
	Scopes       []string
	UserInfoURL  string
	EmailsURL    string
}

type oauthService struct {
	accounts       repository.AccountRepository
	users          repository.UserRepository
	sessions       SessionService
	authSecret     string
	apiURL         string
	trustedOrigins []string
	providers      map[string]OAuthProviderConfig
	httpClient     *http.Client
}

// OAuthServiceParams defines initialization options for NewOAuthService.
type OAuthServiceParams struct {
	Accounts       repository.AccountRepository
	Users          repository.UserRepository
	Sessions       SessionService
	AuthSecret     string
	ApiURL         string
	TrustedOrigins []string
	GoogleID       string
	GoogleSecret   string
	GithubID       string
	GithubSecret   string
	HTTPClient     *http.Client
}

// NewOAuthService constructs an OAuthService configured with providers and verification keys.
func NewOAuthService(p OAuthServiceParams) OAuthService {
	client := p.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	apiURL := strings.TrimRight(p.ApiURL, "/")
	if apiURL == "" {
		apiURL = "http://localhost:8001"
	}

	providers := make(map[string]OAuthProviderConfig)

	if p.GoogleID != "" && p.GoogleSecret != "" {
		providers[domain.ProviderGoogle] = OAuthProviderConfig{
			ID:           domain.ProviderGoogle,
			ClientID:     p.GoogleID,
			ClientSecret: p.GoogleSecret,
			Endpoint:     google.Endpoint,
			Scopes: []string{
				"openid",
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			UserInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
		}
	}

	if p.GithubID != "" && p.GithubSecret != "" {
		providers[domain.ProviderGithub] = OAuthProviderConfig{
			ID:           domain.ProviderGithub,
			ClientID:     p.GithubID,
			ClientSecret: p.GithubSecret,
			Endpoint:     github.Endpoint,
			Scopes: []string{
				"read:user",
				"user:email",
			},
			UserInfoURL: "https://api.github.com/user",
			EmailsURL:   "https://api.github.com/user/emails",
		}
	}

	return &oauthService{
		accounts:       p.Accounts,
		users:          p.Users,
		sessions:       p.Sessions,
		authSecret:     p.AuthSecret,
		apiURL:         apiURL,
		trustedOrigins: p.TrustedOrigins,
		providers:      providers,
		httpClient:     client,
	}
}

// RegisterProvider allows adding or overriding a provider config (useful for testing).
func (s *oauthService) RegisterProvider(cfg OAuthProviderConfig) {
	s.providers[cfg.ID] = cfg
}

func (s *oauthService) getOAuthConfig(provider string) (*oauth2.Config, *OAuthProviderConfig, error) {
	p, ok := s.providers[provider]
	if !ok {
		return nil, nil, domain.ErrInvalidProvider
	}

	redirectURI := fmt.Sprintf("%s/auth/callback/%s", s.apiURL, provider)

	conf := &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		Endpoint:     p.Endpoint,
		Scopes:       p.Scopes,
		RedirectURL:  redirectURI,
	}

	return conf, &p, nil
}

func (s *oauthService) GetAuthURL(req model.SignInSocialRequest) (model.SignInSocialResponse, error) {
	conf, _, err := s.getOAuthConfig(req.Provider)
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

	authURL := conf.AuthCodeURL(state, oauth2.AccessTypeOffline)

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

	conf, pCfg, err := s.getOAuthConfig(provider)
	if err != nil {
		return domain.Session{}, domain.User{}, "", err
	}

	// Attach custom HTTP client to exchange context if configured
	exchangeCtx := context.WithValue(ctx, oauth2.HTTPClient, s.httpClient)
	token, err := conf.Exchange(exchangeCtx, code)
	if err != nil {
		return domain.Session{}, domain.User{}, "", fmt.Errorf("%w: %v", domain.ErrOAuthExchangeFailed, err)
	}

	// Fetch user info from provider
	userInfo, err := s.fetchUserInfo(exchangeCtx, pCfg, token)
	if err != nil {
		return domain.Session{}, domain.User{}, "", fmt.Errorf("%w: %v", domain.ErrFetchUserInfoFailed, err)
	}

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
	if scope == "" && pCfg != nil {
		scope = strings.Join(pCfg.Scopes, " ")
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
		// Log or handle error, proceed if non-fatal
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
		Name:     name,
		Email:    info.Email,
		Avatar:   info.Avatar,
		Locale:   locale,
		Currency: currency,
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

func (s *oauthService) fetchUserInfo(ctx context.Context, p *OAuthProviderConfig, token *oauth2.Token) (model.OAuthUserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.UserInfoURL, nil)
	if err != nil {
		return model.OAuthUserInfo{}, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return model.OAuthUserInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return model.OAuthUserInfo{}, fmt.Errorf("provider returned status %d: %s", resp.StatusCode, string(b))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.OAuthUserInfo{}, err
	}

	switch p.ID {
	case domain.ProviderGoogle:
		var gUser struct {
			ID            string `json:"id"`
			Email         string `json:"email"`
			VerifiedEmail bool   `json:"verified_email"`
			EmailVerified bool   `json:"email_verified"`
			Name          string `json:"name"`
			Picture       string `json:"picture"`
			Locale        string `json:"locale"`
		}
		if err := json.Unmarshal(body, &gUser); err != nil {
			return model.OAuthUserInfo{}, err
		}
		return model.OAuthUserInfo{
			ID:            gUser.ID,
			Email:         gUser.Email,
			Name:          gUser.Name,
			Avatar:        gUser.Picture,
			EmailVerified: gUser.VerifiedEmail || gUser.EmailVerified,
			Locale:        gUser.Locale,
		}, nil

	case domain.ProviderGithub:
		var ghUser struct {
			ID            int64  `json:"id"`
			Login         string `json:"login"`
			Name          string `json:"name"`
			Email         string `json:"email"`
			AvatarURL     string `json:"avatar_url"`
			EmailVerified *bool  `json:"email_verified"`
			Verified      *bool  `json:"verified"`
		}
		if err := json.Unmarshal(body, &ghUser); err != nil {
			return model.OAuthUserInfo{}, err
		}

		userEmail := ghUser.Email
		emailVerified := false

		if ghUser.EmailVerified != nil {
			emailVerified = *ghUser.EmailVerified
		} else if ghUser.Verified != nil {
			emailVerified = *ghUser.Verified
		}

		// GitHub users often have private emails or unverified profile emails.
		// Query /user/emails to get a verified email address.
		if vEmail, ok := s.fetchGithubEmail(ctx, token.AccessToken); ok {
			userEmail = vEmail
			emailVerified = true
		}

		displayName := ghUser.Name
		if displayName == "" {
			displayName = ghUser.Login
		}

		return model.OAuthUserInfo{
			ID:            strconv.FormatInt(ghUser.ID, 10),
			Email:         userEmail,
			Name:          displayName,
			Avatar:        ghUser.AvatarURL,
			EmailVerified: emailVerified,
		}, nil

	default:
		return model.OAuthUserInfo{}, fmt.Errorf("unsupported provider %s", p.ID)
	}
}

func (s *oauthService) fetchGithubEmail(ctx context.Context, accessToken string) (string, bool) {
	emailsURL := "https://api.github.com/user/emails"
	if p, ok := s.providers[domain.ProviderGithub]; ok && p.EmailsURL != "" {
		emailsURL = p.EmailsURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, emailsURL, nil)
	if err != nil {
		return "", false
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", false
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", false
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, true
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email, true
		}
	}

	return "", false
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
