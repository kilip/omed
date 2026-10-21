package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/middleware"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func setupTestApp(sessions *mockSessionService, cookieName string) (*fiber.App, *base.UserContext) {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			status, _ := httpx.MapError(err)
			return c.Status(status).SendString(err.Error())
		},
	})
	var capturedUser base.UserContext

	app.Use(middleware.NewAuthMiddleware(middleware.AuthConfig{
		SessionService: sessions,
		CookieName:     cookieName,
	}))

	app.All("/internal/health", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})
	app.All("/auth/callback", func(c fiber.Ctx) error {
		return c.SendString("auth ok")
	})
	app.Get("/invitations/xyz", func(c fiber.Ctx) error {
		return c.SendString("invitation ok")
	})
	app.Post("/invitations/xyz/accept", func(c fiber.Ctx) error {
		return c.SendString("accept ok")
	})
	app.Get("/public/x", func(c fiber.Ctx) error {
		return c.SendString("public get ok")
	})
	app.Head("/public/x", func(c fiber.Ctx) error {
		return c.SendString("public head ok")
	})
	app.Post("/public/x", func(c fiber.Ctx) error {
		return c.SendString("public post ok")
	})
	app.All("/api/protected", func(c fiber.Ctx) error {
		if u, ok := c.Locals(base.USER_CONTEXT_KEY).(base.UserContext); ok {
			capturedUser = u
		}
		return c.SendString("protected ok")
	})

	return app, &capturedUser
}

func TestAuthMiddleware_Bypass(t *testing.T) {
	mockSvc := &mockSessionService{}
	app, _ := setupTestApp(mockSvc, "token")

	// 1. Bypass OPTIONS
	reqOpt := httptest.NewRequest(fiber.MethodOptions, "/api/protected", nil)
	respOpt, err := app.Test(reqOpt)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respOpt.StatusCode)

	// 2. Bypass /internal/*
	reqInternal := httptest.NewRequest(fiber.MethodGet, "/internal/health", nil)
	respInternal, err := app.Test(reqInternal)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respInternal.StatusCode)

	// 3. Bypass /auth/*
	reqAuth := httptest.NewRequest(fiber.MethodGet, "/auth/callback", nil)
	respAuth, err := app.Test(reqAuth)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respAuth.StatusCode)

	// 4. Bypass GET /invitations/:token
	reqInv := httptest.NewRequest(fiber.MethodGet, "/invitations/xyz", nil)
	respInv, err := app.Test(reqInv)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respInv.StatusCode)

	// 5. Do NOT bypass POST /invitations/:token/accept
	reqInvAccept := httptest.NewRequest(fiber.MethodPost, "/invitations/xyz/accept", nil)
	respInvAccept, err := app.Test(reqInvAccept)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, respInvAccept.StatusCode)

	// 6. Bypass GET /public/*
	reqPubGet := httptest.NewRequest(fiber.MethodGet, "/public/x", nil)
	respPubGet, err := app.Test(reqPubGet)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respPubGet.StatusCode)

	// 7. Bypass HEAD /public/*
	reqPubHead := httptest.NewRequest(fiber.MethodHead, "/public/x", nil)
	respPubHead, err := app.Test(reqPubHead)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, respPubHead.StatusCode)

	// 8. Do NOT bypass POST /public/*
	reqPubPost := httptest.NewRequest(fiber.MethodPost, "/public/x", nil)
	respPubPost, err := app.Test(reqPubPost)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, respPubPost.StatusCode)
}

func TestAuthMiddleware_Unauthorized_MissingToken(t *testing.T) {
	mockSvc := &mockSessionService{}
	app, _ := setupTestApp(mockSvc, "token")

	req := httptest.NewRequest(fiber.MethodGet, "/api/protected", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_Unauthorized_InvalidSession(t *testing.T) {
	mockSvc := &mockSessionService{
		validateFn: func(ctx context.Context, token string) (domain.Session, domain.User, error) {
			return domain.Session{}, domain.User{}, domain.ErrSessionNotFound
		},
	}
	app, _ := setupTestApp(mockSvc, "token")

	req := httptest.NewRequest(fiber.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestAuthMiddleware_Authorized_BearerToken(t *testing.T) {
	mockSvc := &mockSessionService{
		validateFn: func(ctx context.Context, token string) (domain.Session, domain.User, error) {
			if token == "valid-token" {
				return domain.Session{
					ID:        1,
					UserID:    42,
					Token:     token,
					ExpiresAt: time.Now().Add(time.Hour),
				}, domain.User{
					ID:    42,
					Name:  "Test User",
					Email: "test@example.com",
				}, nil
			}
			return domain.Session{}, domain.User{}, domain.ErrSessionNotFound
		},
	}
	app, captured := setupTestApp(mockSvc, "token")

	req := httptest.NewRequest(fiber.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, int64(42), captured.ID)
	assert.Equal(t, "Test User", captured.Name)
	assert.True(t, captured.EmailVerified)
}

func TestAuthMiddleware_Authorized_CustomCookie(t *testing.T) {
	customCookie := "custom_session_id"
	mockSvc := &mockSessionService{
		validateFn: func(ctx context.Context, token string) (domain.Session, domain.User, error) {
			if token == "cookie-secret" {
				return domain.Session{
					ID:     2,
					UserID: 99,
				}, domain.User{
					ID:   99,
					Name: "Cookie Monster",
				}, nil
			}
			return domain.Session{}, domain.User{}, domain.ErrSessionNotFound
		},
	}
	app, captured := setupTestApp(mockSvc, customCookie)

	req := httptest.NewRequest(fiber.MethodGet, "/api/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  customCookie,
		Value: "cookie-secret",
	})
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
	assert.Equal(t, int64(99), captured.ID)
	assert.Equal(t, "Cookie Monster", captured.Name)
}
