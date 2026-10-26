package service_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/workspace"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/core/repository"
	"github.com/kilip/omed/api/internal/core/service"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

type OAuthServiceSuite struct {
	suite.Suite
	client      *ent.Client
	userRepo    repository.UserRepository
	accountRepo repository.AccountRepository
	sessionSvc  service.SessionService
	oauthSvc    service.OAuthService
	mockServer  *httptest.Server
	ctx         context.Context
}

func (s *OAuthServiceSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:oauth_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.userRepo = repository.NewUserRepository(s.client)
	s.accountRepo = repository.NewAccountRepository(s.client)
	sessionRepo := repository.NewSessionRepository(s.client)
	s.sessionSvc = service.NewSessionService(sessionRepo, s.userRepo, 24*time.Hour)
	s.ctx = testutil.NewUserContext()

	// Mock OAuth2 Provider Server (Token exchange & UserInfo)
	s.mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token":  "mock-access-token-123",
				"token_type":    "Bearer",
				"refresh_token": "mock-refresh-token-456",
				"expires_in":    3600,
			})
		case "/userinfo/google":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             "google-1001",
				"email":          "google.user@example.com",
				"name":           "Google User",
				"picture":        "https://example.com/avatar.jpg",
				"verified_email": true,
			})
		case "/userinfo/github":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":         2002,
				"login":      "githubuser",
				"name":       "GitHub User",
				"email":      "github.user@example.com",
				"avatar_url": "https://example.com/gh.png",
			})
		case "/userinfo/google-unverified":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             "google-unverified-1",
				"email":          "existing@example.com",
				"name":           "Google Unverified",
				"picture":        "https://example.com/avatar.jpg",
				"verified_email": false,
			})
		case "/userinfo/google-verified":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             "google-verified-1",
				"email":          "existing@example.com",
				"name":           "Google Verified",
				"picture":        "https://example.com/avatar.jpg",
				"verified_email": true,
			})
		case "/userinfo/github-unverified":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":         2003,
				"login":      "githubunverified",
				"name":       "GitHub Unverified",
				"email":      "existing@example.com",
				"avatar_url": "https://example.com/gh.png",
			})
		case "/userinfo/github-unverified/emails":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"email":    "existing@example.com",
					"primary":  true,
					"verified": false,
				},
			})
		case "/userinfo/github-verified":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":         2004,
				"login":      "githubverified",
				"name":       "GitHub Verified",
				"email":      "existing@example.com",
				"avatar_url": "https://example.com/gh.png",
			})
		case "/userinfo/github-verified/emails":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"email":    "existing@example.com",
					"primary":  true,
					"verified": true,
				},
			})
		case "/userinfo/github-fallback":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":         2005,
				"login":      "githubfallback",
				"name":       "GitHub Fallback",
				"email":      "",
				"avatar_url": "https://example.com/gh.png",
			})
		case "/userinfo/github-fallback/emails":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"email":    "unverified@example.com",
					"primary":  true,
					"verified": false,
				},
				{
					"email":    "verified-fallback@example.com",
					"primary":  false,
					"verified": true,
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))

	s.oauthSvc = service.NewOAuthService(service.OAuthServiceParams{
		Accounts:       s.accountRepo,
		Users:          s.userRepo,
		Sessions:       s.sessionSvc,
		AuthSecret:     "test-secret-key-at-least-32-bytes-long",
		ApiURL:         "http://localhost:8001",
		TrustedOrigins: []string{"http://localhost:3000", "https://app.example.com"},
		GoogleID:       "mock-google-id",
		GoogleSecret:   "mock-google-secret",
		GithubID:       "mock-github-id",
		GithubSecret:   "mock-github-secret",
		HTTPClient:     s.mockServer.Client(),
	})

	// Override Google provider endpoints to point to mock server
	if svcWithReg, ok := s.oauthSvc.(interface {
		RegisterProvider(service.OAuthProviderConfig)
	}); ok {
		svcWithReg.RegisterProvider(service.OAuthProviderConfig{
			ID:           domain.ProviderGoogle,
			ClientID:     "mock-google-id",
			ClientSecret: "mock-google-secret",
			Endpoint: oauth2.Endpoint{
				AuthURL:  s.mockServer.URL + "/auth",
				TokenURL: s.mockServer.URL + "/token",
			},
			Scopes:      []string{"openid", "email"},
			UserInfoURL: s.mockServer.URL + "/userinfo/google",
		})
		svcWithReg.RegisterProvider(service.OAuthProviderConfig{
			ID:           domain.ProviderGithub,
			ClientID:     "mock-github-id",
			ClientSecret: "mock-github-secret",
			Endpoint: oauth2.Endpoint{
				AuthURL:  s.mockServer.URL + "/auth",
				TokenURL: s.mockServer.URL + "/token",
			},
			Scopes:      []string{"read:user", "user:email"},
			UserInfoURL: s.mockServer.URL + "/userinfo/github",
			EmailsURL:   s.mockServer.URL + "/userinfo/github/emails",
		})
	}
}

func (s *OAuthServiceSuite) TearDownSuite() {
	if s.mockServer != nil {
		s.mockServer.Close()
	}
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *OAuthServiceSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.Session.Delete().Exec(ctx)
	_, _ = s.client.UserAccount.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)

	if svcWithReg, ok := s.oauthSvc.(interface {
		RegisterProvider(service.OAuthProviderConfig)
	}); ok {
		svcWithReg.RegisterProvider(service.OAuthProviderConfig{
			ID:           domain.ProviderGoogle,
			ClientID:     "mock-google-id",
			ClientSecret: "mock-google-secret",
			Endpoint: oauth2.Endpoint{
				AuthURL:  s.mockServer.URL + "/auth",
				TokenURL: s.mockServer.URL + "/token",
			},
			Scopes:      []string{"openid", "email"},
			UserInfoURL: s.mockServer.URL + "/userinfo/google",
		})
		svcWithReg.RegisterProvider(service.OAuthProviderConfig{
			ID:           domain.ProviderGithub,
			ClientID:     "mock-github-id",
			ClientSecret: "mock-github-secret",
			Endpoint: oauth2.Endpoint{
				AuthURL:  s.mockServer.URL + "/auth",
				TokenURL: s.mockServer.URL + "/token",
			},
			Scopes:      []string{"read:user", "user:email"},
			UserInfoURL: s.mockServer.URL + "/userinfo/github",
			EmailsURL:   s.mockServer.URL + "/userinfo/github/emails",
		})
	}
}

func (s *OAuthServiceSuite) Test_GetAuthURL_Success() {
	res, err := s.oauthSvc.GetAuthURL(model.SignInSocialRequest{
		Provider:    domain.ProviderGoogle,
		CallbackURL: "http://localhost:3000/dashboard",
	})
	s.Require().NoError(err)
	s.True(res.Redirect)
	s.NotEmpty(res.Nonce)
	s.Contains(res.URL, s.mockServer.URL+"/auth")
	s.Contains(res.URL, "client_id=mock-google-id")
	s.Contains(res.URL, "state=")
}

func (s *OAuthServiceSuite) Test_GetAuthURL_UntrustedCallbackURL() {
	_, err := s.oauthSvc.GetAuthURL(model.SignInSocialRequest{
		Provider:    domain.ProviderGoogle,
		CallbackURL: "https://evil.com/phish",
	})
	s.Require().Error(err)
	s.Contains(err.Error(), "trusted origins")
}

func (s *OAuthServiceSuite) Test_GetAuthURL_InvalidProvider() {
	_, err := s.oauthSvc.GetAuthURL(model.SignInSocialRequest{
		Provider: "unsupported-provider",
	})
	s.Require().ErrorIs(err, domain.ErrInvalidProvider)
}

func (s *OAuthServiceSuite) Test_HandleCallback_InvalidState() {
	_, _, _, err := s.oauthSvc.HandleCallback(
		s.ctx,
		domain.ProviderGoogle,
		"valid-code",
		"invalid-state",
		"some-nonce",
		"127.0.0.1",
		"TestAgent",
	)
	s.Require().ErrorIs(err, domain.ErrInvalidState)
}

func (s *OAuthServiceSuite) Test_HandleCallback_NonceValidation() {
	authRes, err := s.oauthSvc.GetAuthURL(model.SignInSocialRequest{
		Provider:    domain.ProviderGoogle,
		CallbackURL: "http://localhost:3000/dashboard",
	})
	s.Require().NoError(err)

	var state string
	for _, part := range splitQuery(authRes.URL) {
		if len(part) > 6 && part[:6] == "state=" {
			state = part[6:]
			break
		}
	}
	s.NotEmpty(state)
	s.NotEmpty(authRes.Nonce)

	// 1. Missing expected nonce should fail with ErrInvalidState
	_, _, _, err = s.oauthSvc.HandleCallback(
		s.ctx,
		domain.ProviderGoogle,
		"mock-code-123",
		state,
		"",
		"127.0.0.1",
		"TestBrowser",
	)
	s.Require().ErrorIs(err, domain.ErrInvalidState)

	// 2. Mismatched expected nonce should fail with ErrInvalidState
	_, _, _, err = s.oauthSvc.HandleCallback(
		s.ctx,
		domain.ProviderGoogle,
		"mock-code-123",
		state,
		"mismatched-nonce",
		"127.0.0.1",
		"TestBrowser",
	)
	s.Require().ErrorIs(err, domain.ErrInvalidState)

	// 3. Matching expected nonce should succeed
	sess, user, _, err := s.oauthSvc.HandleCallback(
		s.ctx,
		domain.ProviderGoogle,
		"mock-code-123",
		state,
		authRes.Nonce,
		"127.0.0.1",
		"TestBrowser",
	)
	s.Require().NoError(err)
	s.NotEmpty(sess.Token)
	s.Equal("google.user@example.com", user.Email)
}

func (s *OAuthServiceSuite) Test_HandleCallback_UntrustedCallbackURL() {
	// Attacker crafts or manipulates state with an untrusted external callback URL
	claims := model.OAuthStateClaims{
		Nonce:       "attacker-nonce",
		Provider:    domain.ProviderGoogle,
		CallbackURL: "https://evil.attacker.com/steal-token",
		ExpiresAt:   time.Now().Add(10 * time.Minute).Unix(),
	}
	data, err := json.Marshal(claims)
	s.Require().NoError(err)
	payload := base64.RawURLEncoding.EncodeToString(data)

	// HMAC sign with auth secret
	mac := hmac.New(sha256.New, []byte("test-secret-key-at-least-32-bytes-long"))
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	state := fmt.Sprintf("%s.%s", payload, sig)

	_, _, _, err = s.oauthSvc.HandleCallback(
		s.ctx,
		domain.ProviderGoogle,
		"mock-code-123",
		state,
		"attacker-nonce",
		"127.0.0.1",
		"TestBrowser",
	)
	s.Require().ErrorIs(err, domain.ErrInvalidState)
}

func (s *OAuthServiceSuite) Test_HandleCallback_TrustedExternalCallbackURL() {
	claims := model.OAuthStateClaims{
		Nonce:       "valid-nonce",
		Provider:    domain.ProviderGoogle,
		CallbackURL: "https://app.example.com/welcome",
		ExpiresAt:   time.Now().Add(10 * time.Minute).Unix(),
	}
	data, err := json.Marshal(claims)
	s.Require().NoError(err)
	payload := base64.RawURLEncoding.EncodeToString(data)

	mac := hmac.New(sha256.New, []byte("test-secret-key-at-least-32-bytes-long"))
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	state := fmt.Sprintf("%s.%s", payload, sig)

	_, _, redirectURL, err := s.oauthSvc.HandleCallback(
		s.ctx,
		domain.ProviderGoogle,
		"mock-code-123",
		state,
		"valid-nonce",
		"127.0.0.1",
		"TestBrowser",
	)
	s.Require().NoError(err)
	s.Equal("https://app.example.com/welcome", redirectURL)
}

func (s *OAuthServiceSuite) Test_HandleCallback_SuccessNewUser() {
	authRes, err := s.oauthSvc.GetAuthURL(model.SignInSocialRequest{
		Provider:    domain.ProviderGoogle,
		CallbackURL: "http://localhost:3000/welcome",
	})
	s.Require().NoError(err)

	// Extract state from URL
	var state string
	for _, part := range splitQuery(authRes.URL) {
		if len(part) > 6 && part[:6] == "state=" {
			state = part[6:]
			break
		}
	}
	s.NotEmpty(state)

	sess, user, redirectURL, err := s.oauthSvc.HandleCallback(
		s.ctx,
		domain.ProviderGoogle,
		"mock-code-123",
		state,
		authRes.Nonce,
		"127.0.0.1",
		"TestBrowser",
	)
	s.Require().NoError(err)
	s.Equal("http://localhost:3000/welcome", redirectURL)
	s.Equal("google.user@example.com", user.Email)
	s.Equal("Google User", user.Name)
	s.NotEmpty(sess.Token)
	s.Equal(user.ID, sess.UserID)

	// Verify Account was created
	acc, err := s.accountRepo.FindByProviderAndAccountID(s.ctx, domain.ProviderGoogle, "google-1001")
	s.Require().NoError(err)
	s.Equal(user.ID, acc.UserID)
	s.Equal("mock-access-token-123", acc.AccessToken)

	// Verify personal workspace was auto-created
	ws, err := s.client.Workspace.Query().
		Where(workspace.OwnerIDEQ(user.ID), workspace.PersonalEQ(true)).
		Only(s.ctx)
	s.Require().NoError(err)
	s.True(ws.Personal)
	s.Equal("en", user.Locale)
	s.Equal("USD", user.Currency)
	s.Equal("USD", ws.BaseCurrency)
}

func (s *OAuthServiceSuite) Test_NormalizeLocaleAndCurrency() {
	tests := []struct {
		input            string
		expectedLocale   string
		expectedCurrency string
	}{
		{"id", "id", "IDR"},
		{"id-ID", "id", "IDR"},
		{"ID-id", "id", "IDR"},
		{"id_ID", "id", "IDR"},
		{"en", "en", "USD"},
		{"en-US", "en", "USD"},
		{"en-GB", "en", "USD"},
		{"fr-FR", "en", "USD"},
		{"", "en", "USD"},
	}

	for _, tt := range tests {
		loc, curr := service.NormalizeLocaleAndCurrency(tt.input)
		s.Equal(tt.expectedLocale, loc, "for input: "+tt.input)
		s.Equal(tt.expectedCurrency, curr, "for input: "+tt.input)
	}
}

func (s *OAuthServiceSuite) Test_OAuth_LocalePreserved_Indonesian() {
	authRes, err := s.oauthSvc.GetAuthURL(model.SignInSocialRequest{
		Provider:    domain.ProviderGoogle,
		CallbackURL: "http://localhost:3000/dashboard",
		Locale:      "id-ID",
	})
	s.Require().NoError(err)

	var state string
	for _, part := range splitQuery(authRes.URL) {
		if len(part) > 6 && part[:6] == "state=" {
			state = part[6:]
			break
		}
	}
	s.NotEmpty(state)

	_, user, _, err := s.oauthSvc.HandleCallback(
		s.ctx,
		domain.ProviderGoogle,
		"mock-code-123",
		state,
		authRes.Nonce,
		"127.0.0.1",
		"TestBrowser",
	)
	s.Require().NoError(err)
	s.Equal("id", user.Locale)
	s.Equal("IDR", user.Currency)

	ws, err := s.client.Workspace.Query().
		Where(workspace.OwnerIDEQ(user.ID), workspace.PersonalEQ(true)).
		Only(s.ctx)
	s.Require().NoError(err)
	s.Equal("IDR", ws.BaseCurrency)
}

func (s *OAuthServiceSuite) getStateAndNonce(req model.SignInSocialRequest) (string, string) {
	authRes, err := s.oauthSvc.GetAuthURL(req)
	s.Require().NoError(err)

	for _, part := range splitQuery(authRes.URL) {
		if len(part) > 6 && part[:6] == "state=" {
			return part[6:], authRes.Nonce
		}
	}
	s.Fail("state not found in auth URL")
	return "", ""
}

func (s *OAuthServiceSuite) getState(req model.SignInSocialRequest) string {
	state, _ := s.getStateAndNonce(req)
	return state
}

func (s *OAuthServiceSuite) Test_AccountLinking_Google_UnverifiedEmail_Rejected() {
	// 1. Create existing user
	_, err := s.userRepo.Create(s.ctx, model.CreateUserRequest{
		Name:     "Existing User",
		Email:    "existing@example.com",
		Locale:   "en",
		Currency: "USD",
	})
	s.Require().NoError(err)

	// Override Google provider to return unverified profile
	svcWithReg := s.oauthSvc.(interface {
		RegisterProvider(service.OAuthProviderConfig)
	})
	svcWithReg.RegisterProvider(service.OAuthProviderConfig{
		ID:           domain.ProviderGoogle,
		ClientID:     "mock-google-id",
		ClientSecret: "mock-google-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  s.mockServer.URL + "/auth",
			TokenURL: s.mockServer.URL + "/token",
		},
		Scopes:      []string{"openid", "email"},
		UserInfoURL: s.mockServer.URL + "/userinfo/google-unverified",
	})

	state, nonce := s.getStateAndNonce(model.SignInSocialRequest{
		Provider:    domain.ProviderGoogle,
		CallbackURL: "http://localhost:3000/dashboard",
	})

	_, _, _, err = s.oauthSvc.HandleCallback(
		s.ctx,
		domain.ProviderGoogle,
		"mock-code-123",
		state,
		nonce,
		"127.0.0.1",
		"TestBrowser",
	)
	s.Require().Error(err)
	s.Require().ErrorIs(err, domain.ErrEmailNotVerified)

	// Ensure no account was linked
	_, err = s.accountRepo.FindByProviderAndAccountID(s.ctx, domain.ProviderGoogle, "google-unverified-1")
	s.Require().Error(err)
}

func (s *OAuthServiceSuite) Test_AccountLinking_Google_VerifiedEmail_Success() {
	// 1. Create existing user
	existingUser, err := s.userRepo.Create(s.ctx, model.CreateUserRequest{
		Name:     "Existing User",
		Email:    "existing@example.com",
		Locale:   "en",
		Currency: "USD",
	})
	s.Require().NoError(err)

	// Override Google provider to return verified profile
	svcWithReg := s.oauthSvc.(interface {
		RegisterProvider(service.OAuthProviderConfig)
	})
	svcWithReg.RegisterProvider(service.OAuthProviderConfig{
		ID:           domain.ProviderGoogle,
		ClientID:     "mock-google-id",
		ClientSecret: "mock-google-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  s.mockServer.URL + "/auth",
			TokenURL: s.mockServer.URL + "/token",
		},
		Scopes:      []string{"openid", "email"},
		UserInfoURL: s.mockServer.URL + "/userinfo/google-verified",
	})

	state, nonce := s.getStateAndNonce(model.SignInSocialRequest{
		Provider:    domain.ProviderGoogle,
		CallbackURL: "http://localhost:3000/dashboard",
	})

	sess, user, _, err := s.oauthSvc.HandleCallback(
		s.ctx,
		domain.ProviderGoogle,
		"mock-code-123",
		state,
		nonce,
		"127.0.0.1",
		"TestBrowser",
	)
	s.Require().NoError(err)
	s.Equal(existingUser.ID, user.ID)
	s.Equal(existingUser.Email, user.Email)
	s.NotEmpty(sess.Token)

	// Verify Account was created and linked to existing user
	acc, err := s.accountRepo.FindByProviderAndAccountID(s.ctx, domain.ProviderGoogle, "google-verified-1")
	s.Require().NoError(err)
	s.Equal(existingUser.ID, acc.UserID)
}

func (s *OAuthServiceSuite) Test_AccountLinking_GitHub_UnverifiedEmail_Rejected() {
	// 1. Create existing user
	_, err := s.userRepo.Create(s.ctx, model.CreateUserRequest{
		Name:     "Existing User",
		Email:    "existing@example.com",
		Locale:   "en",
		Currency: "USD",
	})
	s.Require().NoError(err)

	// Override GitHub provider to return unverified profile
	svcWithReg := s.oauthSvc.(interface {
		RegisterProvider(service.OAuthProviderConfig)
	})
	svcWithReg.RegisterProvider(service.OAuthProviderConfig{
		ID:           domain.ProviderGithub,
		ClientID:     "mock-github-id",
		ClientSecret: "mock-github-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  s.mockServer.URL + "/auth",
			TokenURL: s.mockServer.URL + "/token",
		},
		Scopes:      []string{"read:user", "user:email"},
		UserInfoURL: s.mockServer.URL + "/userinfo/github-unverified",
		EmailsURL:   s.mockServer.URL + "/userinfo/github-unverified/emails",
	})

	state, nonce := s.getStateAndNonce(model.SignInSocialRequest{
		Provider:    domain.ProviderGithub,
		CallbackURL: "http://localhost:3000/dashboard",
	})

	_, _, _, err = s.oauthSvc.HandleCallback(
		s.ctx,
		domain.ProviderGithub,
		"mock-code-123",
		state,
		nonce,
		"127.0.0.1",
		"TestBrowser",
	)
	s.Require().Error(err)
	s.Require().ErrorIs(err, domain.ErrEmailNotVerified)

	// Ensure no account was linked
	_, err = s.accountRepo.FindByProviderAndAccountID(s.ctx, domain.ProviderGithub, "2003")
	s.Require().Error(err)
}

func (s *OAuthServiceSuite) Test_AccountLinking_GitHub_VerifiedEmail_Success() {
	// 1. Create existing user
	existingUser, err := s.userRepo.Create(s.ctx, model.CreateUserRequest{
		Name:     "Existing User",
		Email:    "existing@example.com",
		Locale:   "en",
		Currency: "USD",
	})
	s.Require().NoError(err)

	// Override GitHub provider to return verified profile
	svcWithReg := s.oauthSvc.(interface {
		RegisterProvider(service.OAuthProviderConfig)
	})
	svcWithReg.RegisterProvider(service.OAuthProviderConfig{
		ID:           domain.ProviderGithub,
		ClientID:     "mock-github-id",
		ClientSecret: "mock-github-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  s.mockServer.URL + "/auth",
			TokenURL: s.mockServer.URL + "/token",
		},
		Scopes:      []string{"read:user", "user:email"},
		UserInfoURL: s.mockServer.URL + "/userinfo/github-verified",
		EmailsURL:   s.mockServer.URL + "/userinfo/github-verified/emails",
	})

	state, nonce := s.getStateAndNonce(model.SignInSocialRequest{
		Provider:    domain.ProviderGithub,
		CallbackURL: "http://localhost:3000/dashboard",
	})

	sess, user, _, err := s.oauthSvc.HandleCallback(
		s.ctx,
		domain.ProviderGithub,
		"mock-code-123",
		state,
		nonce,
		"127.0.0.1",
		"TestBrowser",
	)
	s.Require().NoError(err)
	s.Equal(existingUser.ID, user.ID)
	s.Equal(existingUser.Email, user.Email)
	s.NotEmpty(sess.Token)

	// Verify Account was created and linked to existing user
	acc, err := s.accountRepo.FindByProviderAndAccountID(s.ctx, domain.ProviderGithub, "2004")
	s.Require().NoError(err)
	s.Equal(existingUser.ID, acc.UserID)
}

func (s *OAuthServiceSuite) Test_AccountLinking_NewUser_UnverifiedEmail_Rejected() {
	svcWithReg := s.oauthSvc.(interface {
		RegisterProvider(service.OAuthProviderConfig)
	})
	svcWithReg.RegisterProvider(service.OAuthProviderConfig{
		ID:           domain.ProviderGoogle,
		ClientID:     "mock-google-id",
		ClientSecret: "mock-google-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  s.mockServer.URL + "/auth",
			TokenURL: s.mockServer.URL + "/token",
		},
		Scopes:      []string{"openid", "email"},
		UserInfoURL: s.mockServer.URL + "/userinfo/google-unverified",
	})

	state, nonce := s.getStateAndNonce(model.SignInSocialRequest{
		Provider:    domain.ProviderGoogle,
		CallbackURL: "http://localhost:3000/dashboard",
	})

	_, _, _, err := s.oauthSvc.HandleCallback(
		s.ctx,
		domain.ProviderGoogle,
		"mock-code-123",
		state,
		nonce,
		"127.0.0.1",
		"TestBrowser",
	)
	s.Require().Error(err)
	s.Require().ErrorIs(err, domain.ErrEmailNotVerified)

	// Ensure user was NOT created in DB
	_, err = s.userRepo.FindByEmail(s.ctx, "existing@example.com")
	s.Require().Error(err)
}

func (s *OAuthServiceSuite) Test_FetchGithubEmail_PicksVerifiedNonPrimaryWhenPrimaryUnverified() {
	svcWithReg := s.oauthSvc.(interface {
		RegisterProvider(service.OAuthProviderConfig)
	})
	svcWithReg.RegisterProvider(service.OAuthProviderConfig{
		ID:           domain.ProviderGithub,
		ClientID:     "mock-github-id",
		ClientSecret: "mock-github-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  s.mockServer.URL + "/auth",
			TokenURL: s.mockServer.URL + "/token",
		},
		Scopes:      []string{"read:user", "user:email"},
		UserInfoURL: s.mockServer.URL + "/userinfo/github-fallback",
		EmailsURL:   s.mockServer.URL + "/userinfo/github-fallback/emails",
	})

	state, nonce := s.getStateAndNonce(model.SignInSocialRequest{
		Provider:    domain.ProviderGithub,
		CallbackURL: "http://localhost:3000/dashboard",
	})

	sess, user, _, err := s.oauthSvc.HandleCallback(
		s.ctx,
		domain.ProviderGithub,
		"mock-code-123",
		state,
		nonce,
		"127.0.0.1",
		"TestBrowser",
	)
	s.Require().NoError(err)
	s.Equal("verified-fallback@example.com", user.Email)
	s.NotEmpty(sess.Token)
}

func splitQuery(rawURL string) []string {
	var parts []string
	idx := -1
	for i, c := range rawURL {
		if c == '?' {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil
	}
	query := rawURL[idx+1:]
	start := 0
	for i, c := range query {
		if c == '&' {
			parts = append(parts, query[start:i])
			start = i + 1
		}
	}
	parts = append(parts, query[start:])
	return parts
}

func TestOAuthServiceSuite(t *testing.T) {
	suite.Run(t, new(OAuthServiceSuite))
}
