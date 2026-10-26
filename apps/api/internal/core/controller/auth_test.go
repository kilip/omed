package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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
	handleCallbackFunc func(ctx context.Context, provider string, code string, state string, expectedNonce string, ip string, ua string) (domain.Session, domain.User, string, error)
}

func (m *mockOAuthService) GetAuthURL(req model.SignInSocialRequest) (model.SignInSocialResponse, error) {
	if m.getAuthURLFunc != nil {
		return m.getAuthURLFunc(req)
	}
	return model.SignInSocialResponse{
		URL:      "https://accounts.google.com/o/oauth2/v2/auth?mock=1",
		Redirect: true,
		Nonce:    "mock-nonce-12345",
	}, nil
}

func (m *mockOAuthService) HandleCallback(ctx context.Context, provider string, code string, state string, expectedNonce string, ip string, ua string) (domain.Session, domain.User, string, error) {
	if m.handleCallbackFunc != nil {
		return m.handleCallbackFunc(ctx, provider, code, state, expectedNonce, ip, ua)
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
		ID:            10,
		Name:          "Mock User",
		Email:         "mock@example.com",
		EmailVerified: true,
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
	s.Empty(res.Nonce, "nonce must not be serialized into JSON response")

	// Verify nonce cookie is set
	var foundNonceCookie bool
	for _, c := range resp.Cookies() {
		if c.Name == controller.OAuthNonceCookieName && c.Value == "mock-nonce-12345" {
			foundNonceCookie = true
			s.True(c.HttpOnly)
			s.Equal("/", c.Path)
			s.Equal(http.SameSiteLaxMode, c.SameSite)
		}
	}
	s.True(foundNonceCookie, "expected oauth nonce cookie to be set")
}

func (s *AuthControllerSuite) Test_SignInSocial_CookieSecure_Enabled() {
	app := fiber.New()
	ctl := controller.NewAuthController(s.oauthSvc, s.sessSvc, controller.WithCookieSecure(true))
	app.Post("/auth/sign-in/social", ctl.SignInSocial)

	reqBody := model.SignInSocialRequest{
		Provider: "google",
	}
	b, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(fiber.MethodPost, "/auth/sign-in/social", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	var found bool
	for _, c := range resp.Cookies() {
		if c.Name == controller.OAuthNonceCookieName {
			found = true
			s.True(c.Secure)
			s.True(c.HttpOnly)
			s.Equal("/", c.Path)
			s.Equal(http.SameSiteLaxMode, c.SameSite)
		}
	}
	s.True(found)
}

func (s *AuthControllerSuite) Test_SignInSocial_CookieSecure_Disabled() {
	app := fiber.New()
	ctl := controller.NewAuthController(s.oauthSvc, s.sessSvc, controller.WithCookieSecure(false))
	app.Post("/auth/sign-in/social", ctl.SignInSocial)

	reqBody := model.SignInSocialRequest{
		Provider: "google",
	}
	b, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(fiber.MethodPost, "/auth/sign-in/social", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	var found bool
	for _, c := range resp.Cookies() {
		if c.Name == controller.OAuthNonceCookieName {
			found = true
			s.False(c.Secure)
			s.True(c.HttpOnly)
			s.Equal("/", c.Path)
			s.Equal(http.SameSiteLaxMode, c.SameSite)
		}
	}
	s.True(found)
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
	var capturedNonce string
	s.oauthSvc.handleCallbackFunc = func(ctx context.Context, provider string, code string, state string, expectedNonce string, ip string, ua string) (domain.Session, domain.User, string, error) {
		capturedNonce = expectedNonce
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

	req := httptest.NewRequest(fiber.MethodGet, "/auth/callback/google?code=valid-code&state=valid-state", nil)
	req.AddCookie(&http.Cookie{
		Name:  controller.OAuthNonceCookieName,
		Value: "mock-nonce-12345",
	})

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusFound, resp.StatusCode)
	s.Equal("http://localhost:3000/dashboard", resp.Header.Get("Location"))
	s.Equal("mock-nonce-12345", capturedNonce)

	// Verify cookies: session cookie set, oauth nonce cookie cleared
	cookies := resp.Cookies()
	s.NotEmpty(cookies)
	var foundTokenCookie bool
	var nonceCleared bool
	for _, c := range cookies {
		if c.Name == "omed_session" && c.Value == "mock-session-token" {
			foundTokenCookie = true
			s.True(c.HttpOnly)
		}
		if c.Name == controller.OAuthNonceCookieName && c.MaxAge == -1 {
			nonceCleared = true
			s.True(c.HttpOnly)
			s.Equal("/", c.Path)
			s.Equal(http.SameSiteLaxMode, c.SameSite)
		}
	}
	s.True(foundTokenCookie)
	s.True(nonceCleared, "expected oauth nonce cookie to be cleared")
}

func (s *AuthControllerSuite) Test_Callback_MissingNonceCookie() {
	s.oauthSvc.handleCallbackFunc = func(ctx context.Context, provider string, code string, state string, expectedNonce string, ip string, ua string) (domain.Session, domain.User, string, error) {
		if expectedNonce == "" {
			return domain.Session{}, domain.User{}, "", domain.ErrInvalidState
		}
		return domain.Session{Token: "valid"}, domain.User{}, "/", nil
	}

	req := httptest.NewRequest(fiber.MethodGet, "/auth/callback/google?code=valid-code&state=valid-state", nil)
	// no nonce cookie added

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusBadRequest, resp.StatusCode)

	// Nonce cookie should still be cleared
	var nonceCleared bool
	for _, c := range resp.Cookies() {
		if c.Name == controller.OAuthNonceCookieName && c.MaxAge == -1 {
			nonceCleared = true
		}
	}
	s.True(nonceCleared, "expected oauth nonce cookie to be cleared on missing nonce")
}

func (s *AuthControllerSuite) Test_Callback_WrongNonceCookie() {
	s.oauthSvc.handleCallbackFunc = func(ctx context.Context, provider string, code string, state string, expectedNonce string, ip string, ua string) (domain.Session, domain.User, string, error) {
		if expectedNonce != "correct-nonce" {
			return domain.Session{}, domain.User{}, "", domain.ErrInvalidState
		}
		return domain.Session{Token: "valid"}, domain.User{}, "/", nil
	}

	req := httptest.NewRequest(fiber.MethodGet, "/auth/callback/google?code=valid-code&state=valid-state", nil)
	req.AddCookie(&http.Cookie{
		Name:  controller.OAuthNonceCookieName,
		Value: "wrong-nonce",
	})

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusBadRequest, resp.StatusCode)

	var nonceCleared bool
	for _, c := range resp.Cookies() {
		if c.Name == controller.OAuthNonceCookieName && c.MaxAge == -1 {
			nonceCleared = true
		}
	}
	s.True(nonceCleared, "expected oauth nonce cookie to be cleared on wrong nonce")
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
		Name:  "omed_session",
		Value: "mock-session-token",
	})

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	var res model.SessionResponse
	_ = json.NewDecoder(resp.Body).Decode(&res)
	s.Require().NotNil(res.Session)
	s.Require().NotNil(res.User)
	s.Equal("mock@example.com", res.User.Email)
	s.True(res.User.EmailVerified)
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
	s.Require().NotNil(res.User)
	s.True(res.User.EmailVerified)
}

func (s *AuthControllerSuite) Test_GetSession_UnverifiedEmail() {
	s.sessSvc.validateSessionFunc = func(ctx context.Context, token string) (domain.Session, domain.User, error) {
		return domain.Session{
			ID:        1,
			Token:     token,
			UserID:    10,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}, domain.User{
			ID:            10,
			Name:          "Unverified User",
			Email:         "unverified@example.com",
			EmailVerified: false,
		}, nil
	}

	req := httptest.NewRequest(fiber.MethodGet, "/auth/get-session", nil)
	req.Header.Set("Authorization", "Bearer mock-session-token")

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	var res model.SessionResponse
	_ = json.NewDecoder(resp.Body).Decode(&res)
	s.Require().NotNil(res.Session)
	s.Require().NotNil(res.User)
	s.False(res.User.EmailVerified)
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

func (s *AuthControllerSuite) Test_Callback_CookieSecure_Enabled() {
	app := fiber.New()
	ctl := controller.NewAuthController(s.oauthSvc, s.sessSvc, controller.WithCookieSecure(true))
	app.Get("/auth/callback/:id", ctl.Callback)

	req := httptest.NewRequest(fiber.MethodGet, "/auth/callback/google?code=valid-code&state=valid-state", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusFound, resp.StatusCode)

	cookies := resp.Cookies()
	s.NotEmpty(cookies)
	var found bool
	for _, c := range cookies {
		if c.Name == "omed_session" {
			found = true
			s.True(c.Secure)
			s.True(c.HttpOnly)
			s.Equal("/", c.Path)
			s.Equal(http.SameSiteLaxMode, c.SameSite)
		}
	}
	s.True(found)
}

func (s *AuthControllerSuite) Test_Callback_CookieSecure_Disabled() {
	app := fiber.New()
	ctl := controller.NewAuthController(s.oauthSvc, s.sessSvc, controller.WithCookieSecure(false))
	app.Get("/auth/callback/:id", ctl.Callback)

	req := httptest.NewRequest(fiber.MethodGet, "/auth/callback/google?code=valid-code&state=valid-state", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusFound, resp.StatusCode)

	cookies := resp.Cookies()
	s.NotEmpty(cookies)
	var found bool
	for _, c := range cookies {
		if c.Name == "omed_session" {
			found = true
			s.False(c.Secure)
			s.True(c.HttpOnly)
			s.Equal("/", c.Path)
			s.Equal(http.SameSiteLaxMode, c.SameSite)
		}
	}
	s.True(found)
}

func (s *AuthControllerSuite) Test_SignOut_CookieAttributes_SecureEnabled() {
	app := fiber.New()
	ctl := controller.NewAuthController(s.oauthSvc, s.sessSvc, controller.WithCookieSecure(true))
	app.Post("/auth/sign-out", ctl.SignOut)

	req := httptest.NewRequest(fiber.MethodPost, "/auth/sign-out", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	var found bool
	for _, c := range resp.Cookies() {
		if c.Name == "omed_session" {
			found = true
			s.Equal(-1, c.MaxAge)
			s.True(c.Secure)
			s.True(c.HttpOnly)
			s.Equal("/", c.Path)
			s.Equal(http.SameSiteLaxMode, c.SameSite)
		}
	}
	s.True(found)
}

func (s *AuthControllerSuite) Test_SignOut_CookieAttributes_SecureDisabled() {
	app := fiber.New()
	ctl := controller.NewAuthController(s.oauthSvc, s.sessSvc, controller.WithCookieSecure(false))
	app.Post("/auth/sign-out", ctl.SignOut)

	req := httptest.NewRequest(fiber.MethodPost, "/auth/sign-out", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	var found bool
	for _, c := range resp.Cookies() {
		if c.Name == "omed_session" {
			found = true
			s.Equal(-1, c.MaxAge)
			s.False(c.Secure)
			s.True(c.HttpOnly)
			s.Equal("/", c.Path)
			s.Equal(http.SameSiteLaxMode, c.SameSite)
		}
	}
	s.True(found)
}

func (s *AuthControllerSuite) Test_CookieSecure_PerEnvironmentAndURL() {
	cases := []struct {
		name           string
		apiURL         string
		env            string
		expectedSecure bool
	}{
		{"production with http", "http://localhost:8001", "production", true},
		{"production with https", "https://api.omed.com", "production", true},
		{"development with https", "https://api.omed.com", "development", true},
		{"development with http", "http://localhost:8001", "development", false},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			app := fiber.New()
			cookieSecure := strings.HasPrefix(tc.apiURL, "https://") || tc.env == "production"
			ctl := controller.NewAuthController(
				s.oauthSvc,
				s.sessSvc,
				controller.WithCookieSecure(cookieSecure),
			)
			app.Get("/auth/callback/:id", ctl.Callback)

			req := httptest.NewRequest(fiber.MethodGet, "/auth/callback/google?code=valid-code&state=valid-state", nil)
			resp, err := app.Test(req)
			s.Require().NoError(err)
			s.Equal(fiber.StatusFound, resp.StatusCode)

			cookies := resp.Cookies()
			s.NotEmpty(cookies)
			var found bool
			for _, c := range cookies {
				if c.Name == "omed_session" {
					found = true
					s.Equal(tc.expectedSecure, c.Secure)
					s.True(c.HttpOnly)
					s.Equal("/", c.Path)
					s.Equal(http.SameSiteLaxMode, c.SameSite)
				}
			}
			s.True(found)
		})
	}
}

func TestAuthControllerSuite(t *testing.T) {
	suite.Run(t, new(AuthControllerSuite))
}
