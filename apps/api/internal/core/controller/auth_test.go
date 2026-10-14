package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/config"
	"github.com/kilip/omed/api/internal/core/controller"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/stretchr/testify/suite"
)

type mockOAuthService struct {
	getAuthURLFunc     func(req model.SignInSocialRequest) (model.SignInSocialResponse, error)
	handleCallbackFunc func(ctx context.Context, provider string, code string, state string, ip string, ua string) (domain.Session, domain.User, string, error)
}

func (m *mockOAuthService) GetAuthURL(req model.SignInSocialRequest) (model.SignInSocialResponse, error) {
	if m.getAuthURLFunc != nil {
		return m.getAuthURLFunc(req)
	}
	return model.SignInSocialResponse{
		URL:      "https://accounts.google.com/o/oauth2/v2/auth?mock=1",
		Redirect: true,
	}, nil
}

func (m *mockOAuthService) HandleCallback(ctx context.Context, provider string, code string, state string, ip string, ua string) (domain.Session, domain.User, string, error) {
	if m.handleCallbackFunc != nil {
		return m.handleCallbackFunc(ctx, provider, code, state, ip, ua)
	}
	return domain.Session{
		ID:        1,
		Token:     "mock-session-token",
		UserID:    10,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, domain.User{
		ID:    10,
		Name:  "Mock User",
		Email: "mock@example.com",
	}, "http://localhost:3000/dashboard", nil
}

type mockSessionService struct {
	createSessionFunc         func(ctx context.Context, userID int64, ipAddress string, userAgent string) (domain.Session, error)
	validateSessionFunc       func(ctx context.Context, token string) (domain.Session, domain.User, error)
	revokeSessionFunc         func(ctx context.Context, token string) error
	revokeAllUserSessionsFunc func(ctx context.Context, userID int64) error
}

func (m *mockSessionService) CreateSession(ctx context.Context, userID int64, ipAddress string, userAgent string) (domain.Session, error) {
	if m.createSessionFunc != nil {
		return m.createSessionFunc(ctx, userID, ipAddress, userAgent)
	}
	return domain.Session{Token: "test-token", UserID: userID}, nil
}

func (m *mockSessionService) ValidateSession(ctx context.Context, token string) (domain.Session, domain.User, error) {
	if m.validateSessionFunc != nil {
		return m.validateSessionFunc(ctx, token)
	}
	return domain.Session{
		ID:        1,
		Token:     token,
		UserID:    10,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}, domain.User{
		ID:    10,
		Name:  "Mock User",
		Email: "mock@example.com",
	}, nil
}

func (m *mockSessionService) RevokeSession(ctx context.Context, token string) error {
	if m.revokeSessionFunc != nil {
		return m.revokeSessionFunc(ctx, token)
	}
	return nil
}

func (m *mockSessionService) RevokeAllUserSessions(ctx context.Context, userID int64) error {
	if m.revokeAllUserSessionsFunc != nil {
		return m.revokeAllUserSessionsFunc(ctx, userID)
	}
	return nil
}

type AuthControllerSuite struct {
	suite.Suite
	app      *fiber.App
	oauthSvc *mockOAuthService
	sessSvc  *mockSessionService
}

func (s *AuthControllerSuite) SetupTest() {
	s.oauthSvc = &mockOAuthService{}
	s.sessSvc = &mockSessionService{}

	s.app = fiber.New(fiber.Config{
		ErrorHandler: config.ErrorHandler(slog.Default()),
	})

	ctl := controller.NewAuthController(s.oauthSvc, s.sessSvc)
	auth := s.app.Group("/auth")
	auth.Post("/sign-in/social", ctl.SignInSocial)
	auth.Get("/callback/:id", ctl.Callback)
	auth.Post("/callback/:id", ctl.Callback)
	auth.Get("/get-session", ctl.GetSession)
	auth.Post("/sign-out", ctl.SignOut)
}

func (s *AuthControllerSuite) Test_SignInSocial_Success() {
	reqBody := model.SignInSocialRequest{
		Provider:    "google",
		CallbackURL: "http://localhost:3000/dashboard",
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(fiber.MethodPost, "/auth/sign-in/social", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	var res model.SignInSocialResponse
	_ = json.NewDecoder(resp.Body).Decode(&res)
	s.True(res.Redirect)
	s.Contains(res.URL, "https://accounts.google.com")
}

func (s *AuthControllerSuite) Test_SignInSocial_MissingProvider() {
	reqBody := model.SignInSocialRequest{
		Provider: "",
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(fiber.MethodPost, "/auth/sign-in/social", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *AuthControllerSuite) Test_Callback_SuccessRedirect() {
	req := httptest.NewRequest(fiber.MethodGet, "/auth/callback/google?code=valid-code&state=valid-state", nil)

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusFound, resp.StatusCode)
	s.Equal("http://localhost:3000/dashboard", resp.Header.Get("Location"))

	// Verify cookie
	cookies := resp.Cookies()
	s.NotEmpty(cookies)
	var foundTokenCookie bool
	for _, c := range cookies {
		if c.Name == "omed_session" && c.Value == "mock-session-token" {
			foundTokenCookie = true
			s.True(c.HttpOnly)
		}
	}
	s.True(foundTokenCookie)
}

func (s *AuthControllerSuite) Test_Callback_ProviderError() {
	req := httptest.NewRequest(fiber.MethodGet, "/auth/callback/google?error=access_denied", nil)

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *AuthControllerSuite) Test_GetSession_WithCookie() {
	req := httptest.NewRequest(fiber.MethodGet, "/auth/get-session", nil)
	req.AddCookie(&http.Cookie{
		Name:  "token",
		Value: "mock-session-token",
	})

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	var res model.SessionResponse
	_ = json.NewDecoder(resp.Body).Decode(&res)
	s.Require().NotNil(res.Session)
	s.Require().NotNil(res.User)
	s.Equal("mock-session-token", res.Session.Token)
	s.Equal("mock@example.com", res.User.Email)
}

func (s *AuthControllerSuite) Test_GetSession_WithBearer() {
	req := httptest.NewRequest(fiber.MethodGet, "/auth/get-session", nil)
	req.Header.Set("Authorization", "Bearer mock-session-token")

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	var res model.SessionResponse
	_ = json.NewDecoder(resp.Body).Decode(&res)
	s.Require().NotNil(res.Session)
	s.Equal("mock-session-token", res.Session.Token)
}

func (s *AuthControllerSuite) Test_GetSession_MissingToken() {
	req := httptest.NewRequest(fiber.MethodGet, "/auth/get-session", nil)

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *AuthControllerSuite) Test_SignOut_Success() {
	req := httptest.NewRequest(fiber.MethodPost, "/auth/sign-out", nil)
	req.AddCookie(&http.Cookie{
		Name:  "token",
		Value: "mock-session-token",
	})

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	// Check cookie expired
	var cleared bool
	for _, c := range resp.Cookies() {
		if c.Name == "omed_session" && c.MaxAge == -1 {
			cleared = true
		}
	}
	s.True(cleared)
}

func TestAuthControllerSuite(t *testing.T) {
	suite.Run(t, new(AuthControllerSuite))
}
