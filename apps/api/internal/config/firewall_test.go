package config_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/config"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/stretchr/testify/suite"
)

type FirewallSuite struct {
	suite.Suite
	logger *slog.Logger
}

type mockSessionService struct {
	validateFn func(ctx context.Context, token string) (domain.Session, domain.User, error)
}

func (m *mockSessionService) CreateSession(ctx context.Context, userID int64, ipAddress string, userAgent string) (domain.Session, error) {
	return domain.Session{}, nil
}

func (m *mockSessionService) ValidateSession(ctx context.Context, token string) (domain.Session, domain.User, error) {
	if m.validateFn != nil {
		return m.validateFn(ctx, token)
	}
	return domain.Session{}, domain.User{}, domain.ErrSessionNotFound
}

func (m *mockSessionService) RevokeSession(ctx context.Context, token string) error {
	return nil
}

func (m *mockSessionService) RevokeAllUserSessions(ctx context.Context, userID int64) error {
	return nil
}

func (s *FirewallSuite) SetupSuite() {
	s.logger = slog.Default()
}

func (s *FirewallSuite) Test_InitFirewall() {
	cfg := config.Config{
		SessionCookieName: "custom_session_cookie",
	}
	app := config.GetFiber(cfg, s.logger)

	state := config.State{
		FiberApp: app,
		Logger:   s.logger,
		Config:   cfg,
	}

	sessions := &mockSessionService{
		validateFn: func(ctx context.Context, token string) (domain.Session, domain.User, error) {
			if token == "valid-session-token" {
				return domain.Session{
					ID:        1,
					UserID:    99,
					Token:     token,
					ExpiresAt: time.Now().Add(time.Hour),
				}, domain.User{
					ID:   99,
					Name: "Test User",
				}, nil
			}
			return domain.Session{}, domain.User{}, domain.ErrSessionNotFound
		},
	}

	config.InitFirewall(state, sessions)

	var capturedUser base.UserContext
	app.All("/internal/health", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})
	app.All("/auth/login", func(c fiber.Ctx) error {
		return c.SendString("auth ok")
	})
	app.All("/api/protected", func(c fiber.Ctx) error {
		if u, ok := c.Locals(base.USER_CONTEXT_KEY).(base.UserContext); ok {
			capturedUser = u
		}
		return c.SendString("protected ok")
	})

	// 1. Bypass /internal/*
	reqInternal := httptest.NewRequest(fiber.MethodGet, "/internal/health", nil)
	respInternal, err := app.Test(reqInternal)
	s.Require().NoError(err)
	defer respInternal.Body.Close()
	s.Equal(fiber.StatusOK, respInternal.StatusCode)

	// 2. Bypass OPTIONS
	reqOpt := httptest.NewRequest(fiber.MethodOptions, "/api/protected", nil)
	respOpt, err := app.Test(reqOpt)
	s.Require().NoError(err)
	defer respOpt.Body.Close()
	s.Equal(fiber.StatusOK, respOpt.StatusCode)

	// 3. Bypass /auth/*
	reqAuthPath := httptest.NewRequest(fiber.MethodGet, "/auth/login", nil)
	respAuthPath, err := app.Test(reqAuthPath)
	s.Require().NoError(err)
	defer respAuthPath.Body.Close()
	s.Equal(fiber.StatusOK, respAuthPath.StatusCode)

	// 4. Unauthorized request without token
	reqNoToken := httptest.NewRequest(fiber.MethodGet, "/api/protected", nil)
	respNoToken, err := app.Test(reqNoToken)
	s.Require().NoError(err)
	defer respNoToken.Body.Close()
	s.Equal(fiber.StatusUnauthorized, respNoToken.StatusCode)

	// 5. Authorized request with Bearer token
	reqAuth := httptest.NewRequest(fiber.MethodGet, "/api/protected", nil)
	reqAuth.Header.Set("Authorization", "Bearer valid-session-token")

	respAuth, err := app.Test(reqAuth)
	s.Require().NoError(err)
	defer respAuth.Body.Close()
	s.Equal(fiber.StatusOK, respAuth.StatusCode)
	s.Equal(int64(99), capturedUser.ID)

	// 6. Authorized request with configurable cookie
	reqCookie := httptest.NewRequest(fiber.MethodGet, "/api/protected", nil)
	reqCookie.AddCookie(&http.Cookie{
		Name:  "custom_session_cookie",
		Value: "valid-session-token",
	})
	respCookie, err := app.Test(reqCookie)
	s.Require().NoError(err)
	defer respCookie.Body.Close()
	s.Equal(fiber.StatusOK, respCookie.StatusCode)
}

func (s *FirewallSuite) Test_InitFirewall_TestRoutes() {
	sessions := &mockSessionService{}

	s.Run("test routes require auth in production", func() {
		cfg := config.Config{Env: "production"}
		app := config.GetFiber(cfg, s.logger)
		state := config.State{FiberApp: app, Logger: s.logger, Config: cfg}
		config.InitFirewall(state, sessions)

		app.All("/test/hello", func(c fiber.Ctx) error {
			return c.SendString("hello")
		})

		req := httptest.NewRequest(fiber.MethodGet, "/test/hello", nil)
		resp, err := app.Test(req)
		s.Require().NoError(err)
		defer resp.Body.Close()
		s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
	})

	s.Run("test routes bypassed in development", func() {
		cfg := config.Config{Env: "development"}
		app := config.GetFiber(cfg, s.logger)
		state := config.State{FiberApp: app, Logger: s.logger, Config: cfg}
		config.InitFirewall(state, sessions)

		app.All("/test/hello", func(c fiber.Ctx) error {
			return c.SendString("hello")
		})

		req := httptest.NewRequest(fiber.MethodGet, "/test/hello", nil)
		resp, err := app.Test(req)
		s.Require().NoError(err)
		defer resp.Body.Close()
		s.Equal(fiber.StatusOK, resp.StatusCode)
	})
}

func TestFirewallSuite(t *testing.T) {
	suite.Run(t, new(FirewallSuite))
}
