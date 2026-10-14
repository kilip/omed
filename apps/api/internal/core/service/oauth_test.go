package service_test

import (
	"context"
	"encoding/json"
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
	s.ctx = testutil.NewAdminContext()

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
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)
}

func (s *OAuthServiceSuite) Test_GetAuthURL_Success() {
	res, err := s.oauthSvc.GetAuthURL(model.SignInSocialRequest{
		Provider:    domain.ProviderGoogle,
		CallbackURL: "http://localhost:3000/dashboard",
	})
	s.Require().NoError(err)
	s.True(res.Redirect)
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
		"127.0.0.1",
		"TestAgent",
	)
	s.Require().ErrorIs(err, domain.ErrInvalidState)
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
