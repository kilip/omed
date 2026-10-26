package config_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/config"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/stretchr/testify/suite"
)

type FiberSuite struct {
	suite.Suite
	logBuf *bytes.Buffer
	logger *slog.Logger
	cfg    config.Config
}

func (s *FiberSuite) SetupTest() {
	s.logBuf = new(bytes.Buffer)
	s.logger = slog.New(slog.NewJSONHandler(s.logBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	s.cfg = config.Config{
		TrustedOrigins: []string{"http://trusted.com", "http://localhost:3000"},
	}
}

type testPayload struct {
	Name string `json:"name" validate:"required"`
	Age  int    `json:"age" validate:"gte=18"`
}

func (s *FiberSuite) Test_GetFiber_StructValidator_Valid() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/validate", func(c fiber.Ctx) error {
		var p testPayload
		if err := c.Bind().Body(&p); err != nil {
			return err
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
	})

	body := `{"name":"John Doe","age":25}`
	req := httptest.NewRequest(fiber.MethodPost, "/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *FiberSuite) Test_GetFiber_StructValidator_Invalid() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/validate", func(c fiber.Ctx) error {
		var p testPayload
		if err := c.Bind().Body(&p); err != nil {
			return err
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
	})

	body := `{"name":"","age":15}`
	req := httptest.NewRequest(fiber.MethodPost, "/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusUnprocessableEntity, resp.StatusCode)

	var errResp httpx.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	s.Require().NoError(err)
	s.Equal("VALIDATION_FAILED", errResp.Error.Code)
	s.Require().NotEmpty(errResp.Error.Fields)
}

func (s *FiberSuite) Test_GetFiber_CORS_AllowedOrigin() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Get("/hello", func(c fiber.Ctx) error {
		return c.SendString("world")
	})

	req := httptest.NewRequest(fiber.MethodGet, "/hello", nil)
	req.Header.Set("Origin", "http://trusted.com")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.Equal("http://trusted.com", resp.Header.Get("Access-Control-Allow-Origin"))
	s.Equal("true", resp.Header.Get("Access-Control-Allow-Credentials"))
}

func (s *FiberSuite) Test_GetFiber_CORS_DisallowedOrigin() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Get("/hello", func(c fiber.Ctx) error {
		return c.SendString("world")
	})

	req := httptest.NewRequest(fiber.MethodGet, "/hello", nil)
	req.Header.Set("Origin", "http://untrusted.com")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.Empty(resp.Header.Get("Access-Control-Allow-Origin"))
}

func (s *FiberSuite) Test_GetFiber_CORS_Preflight() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/data", func(c fiber.Ctx) error {
		return c.SendString("created")
	})

	req := httptest.NewRequest(fiber.MethodOptions, "/data", nil)
	req.Header.Set("Origin", "http://trusted.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal("http://trusted.com", resp.Header.Get("Access-Control-Allow-Origin"))
	s.Contains(resp.Header.Get("Access-Control-Allow-Methods"), "POST")
}

func (s *FiberSuite) Test_GetFiber_RecoverMiddleware() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Get("/panic", func(c fiber.Ctx) error {
		panic("unexpected crash")
	})

	req := httptest.NewRequest(fiber.MethodGet, "/panic", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusInternalServerError, resp.StatusCode)
}

func (s *FiberSuite) Test_ErrorHandler_MappedError() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Get("/not-found", func(c fiber.Ctx) error {
		return base.ErrNotFound
	})
	app.Get("/invalid-id", func(c fiber.Ctx) error {
		return base.ErrInvalidID
	})

	// Test NotFound
	reqNF := httptest.NewRequest(fiber.MethodGet, "/not-found", nil)
	respNF, err := app.Test(reqNF)
	s.Require().NoError(err)
	defer func() { _ = respNF.Body.Close() }()

	s.Equal(fiber.StatusNotFound, respNF.StatusCode)
	var errRespNF httpx.ErrorResponse
	s.Require().NoError(json.NewDecoder(respNF.Body).Decode(&errRespNF))
	s.Equal("NOT_FOUND", errRespNF.Error.Code)
	s.Equal("not found", errRespNF.Error.Message)

	// Test InvalidID
	reqID := httptest.NewRequest(fiber.MethodGet, "/invalid-id", nil)
	respID, err := app.Test(reqID)
	s.Require().NoError(err)
	defer func() { _ = respID.Body.Close() }()

	s.Equal(fiber.StatusBadRequest, respID.StatusCode)
	var errRespID httpx.ErrorResponse
	s.Require().NoError(json.NewDecoder(respID.Body).Decode(&errRespID))
	s.Equal("INVALID_ID", errRespID.Error.Code)
	s.Equal("invalid id", errRespID.Error.Message)
}

func (s *FiberSuite) Test_ErrorHandler_UnknownError() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Get("/unknown", func(c fiber.Ctx) error {
		return errors.New("database disconnected unexpectedly")
	})

	req := httptest.NewRequest(fiber.MethodGet, "/unknown", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusInternalServerError, resp.StatusCode)
	var errResp httpx.ErrorResponse
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&errResp))
	s.Equal(base.ErrUnknown.Error(), errResp.Error.Code)
	s.Equal("unknown error", errResp.Error.Message)

	// Verify logged
	logged := s.logBuf.String()
	s.Contains(logged, "unknown error")
	s.Contains(logged, "database disconnected unexpectedly")
}

func (s *FiberSuite) Test_GetFiber_Timeouts() {
	cfg := config.Config{
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 45 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	app := config.GetFiber(cfg, s.logger)
	fCfg := app.Config()
	s.Equal(15*time.Second, fCfg.ReadTimeout)
	s.Equal(45*time.Second, fCfg.WriteTimeout)
	s.Equal(60*time.Second, fCfg.IdleTimeout)

	// Default fallback when <= 0
	appDef := config.GetFiber(config.Config{}, s.logger)
	defCfg := appDef.Config()
	s.Equal(10*time.Second, defCfg.ReadTimeout)
	s.Equal(30*time.Second, defCfg.WriteTimeout)
	s.Equal(120*time.Second, defCfg.IdleTimeout)
}

func (s *FiberSuite) Test_GetFiber_BodyLimit() {
	// AttachmentMaxSizeMB = 10, CMSMediaMaxSizeMB = 25 -> max 25MB + 1MB
	cfg1 := config.Config{
		AttachmentMaxSizeMB: 10,
		CMSMediaMaxSizeMB:   25,
	}
	app1 := config.GetFiber(cfg1, s.logger)
	s.Equal(int(25*1024*1024)+1024*1024, app1.Config().BodyLimit)

	// AttachmentMaxSizeMB = 50, CMSMediaMaxSizeMB = 25 -> max 50MB + 1MB
	cfg2 := config.Config{
		AttachmentMaxSizeMB: 50,
		CMSMediaMaxSizeMB:   25,
	}
	app2 := config.GetFiber(cfg2, s.logger)
	s.Equal(int(50*1024*1024)+1024*1024, app2.Config().BodyLimit)
}

func (s *FiberSuite) Test_GetFiber_SecurityHeaders() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Get("/test-sec", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest(fiber.MethodGet, "/test-sec", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.Equal("nosniff", resp.Header.Get("X-Content-Type-Options"))
	s.Equal("DENY", resp.Header.Get("X-Frame-Options"))
	s.Equal("0", resp.Header.Get("X-XSS-Protection"))
	s.Equal("no-referrer", resp.Header.Get("Referrer-Policy"))
	// Over non-TLS plain HTTP, HSTS is omitted per RFC 6797
	s.Empty(resp.Header.Get("Strict-Transport-Security"))
}

func (s *FiberSuite) Test_GetFiber_SecurityHeaders_HSTS_Production() {
	cfg := config.Config{
		Env:            "production",
		ApiUrl:         "https://api.omed.local",
		TrustedOrigins: []string{"http://localhost:3000"},
	}
	app := config.GetFiber(cfg, s.logger)
	app.Get("/test-hsts", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest(fiber.MethodGet, "https://api.omed.local/test-hsts", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.Contains(resp.Header.Get("Strict-Transport-Security"), "max-age=31536000")
	s.Contains(resp.Header.Get("Strict-Transport-Security"), "includeSubDomains")
	s.Contains(resp.Header.Get("Strict-Transport-Security"), "preload")
}

func (s *FiberSuite) Test_GetFiber_RateLimiting_General() {
	cfg := config.Config{
		RateLimitEnabled:             true,
		Env:                          "production",
		RateLimitGeneralMax:          2,
		RateLimitGeneralExpiration:   1 * time.Minute,
		RateLimitSensitiveMax:        10,
		RateLimitSensitiveExpiration: 1 * time.Minute,
	}
	app := config.GetFiber(cfg, s.logger)
	app.Get("/items", func(c fiber.Ctx) error {
		return c.SendString("items ok")
	})

	// 1st request -> 200
	req1 := httptest.NewRequest(fiber.MethodGet, "/items", nil)
	resp1, err := app.Test(req1)
	s.Require().NoError(err)
	_ = resp1.Body.Close()
	s.Equal(fiber.StatusOK, resp1.StatusCode)

	// 2nd request -> 200
	req2 := httptest.NewRequest(fiber.MethodGet, "/items", nil)
	resp2, err := app.Test(req2)
	s.Require().NoError(err)
	_ = resp2.Body.Close()
	s.Equal(fiber.StatusOK, resp2.StatusCode)

	// 3rd request -> 429
	req3 := httptest.NewRequest(fiber.MethodGet, "/items", nil)
	resp3, err := app.Test(req3)
	s.Require().NoError(err)
	defer func() { _ = resp3.Body.Close() }()
	s.Equal(fiber.StatusTooManyRequests, resp3.StatusCode)

	var errResp httpx.ErrorResponse
	s.Require().NoError(json.NewDecoder(resp3.Body).Decode(&errResp))
	s.Equal("TOO_MANY_REQUESTS", errResp.Error.Code)
}

func (s *FiberSuite) Test_GetFiber_RateLimiting_Sensitive_Auth() {
	cfg := config.Config{
		RateLimitEnabled:             true,
		Env:                          "production",
		RateLimitGeneralMax:          100,
		RateLimitGeneralExpiration:   1 * time.Minute,
		RateLimitSensitiveMax:        2,
		RateLimitSensitiveExpiration: 1 * time.Minute,
	}
	app := config.GetFiber(cfg, s.logger)
	app.Post("/auth/sign-in/social", func(c fiber.Ctx) error {
		return c.SendString("auth ok")
	})

	// 1st request -> 200
	req1 := httptest.NewRequest(fiber.MethodPost, "/auth/sign-in/social", nil)
	resp1, err := app.Test(req1)
	s.Require().NoError(err)
	_ = resp1.Body.Close()
	s.Equal(fiber.StatusOK, resp1.StatusCode)

	// 2nd request -> 200
	req2 := httptest.NewRequest(fiber.MethodPost, "/auth/sign-in/social", nil)
	resp2, err := app.Test(req2)
	s.Require().NoError(err)
	_ = resp2.Body.Close()
	s.Equal(fiber.StatusOK, resp2.StatusCode)

	// 3rd request -> 429
	req3 := httptest.NewRequest(fiber.MethodPost, "/auth/sign-in/social", nil)
	resp3, err := app.Test(req3)
	s.Require().NoError(err)
	defer func() { _ = resp3.Body.Close() }()
	s.Equal(fiber.StatusTooManyRequests, resp3.StatusCode)
}

func (s *FiberSuite) Test_GetFiber_RateLimiting_Sensitive_Invitations() {
	cfg := config.Config{
		RateLimitEnabled:             true,
		Env:                          "production",
		RateLimitGeneralMax:          100,
		RateLimitGeneralExpiration:   1 * time.Minute,
		RateLimitSensitiveMax:        2,
		RateLimitSensitiveExpiration: 1 * time.Minute,
	}
	app := config.GetFiber(cfg, s.logger)
	app.Post("/workspaces/:wid/invitations", func(c fiber.Ctx) error {
		return c.SendString("invitation created")
	})
	app.Post("/workspaces/:wid/invitations/:id/send", func(c fiber.Ctx) error {
		return c.SendString("invitation sent")
	})

	// 1st request -> 200
	req1 := httptest.NewRequest(fiber.MethodPost, "/workspaces/ws-1/invitations", nil)
	resp1, err := app.Test(req1)
	s.Require().NoError(err)
	_ = resp1.Body.Close()
	s.Equal(fiber.StatusOK, resp1.StatusCode)

	// 2nd request -> 200
	req2 := httptest.NewRequest(fiber.MethodPost, "/workspaces/ws-1/invitations/inv-1/send", nil)
	resp2, err := app.Test(req2)
	s.Require().NoError(err)
	_ = resp2.Body.Close()
	s.Equal(fiber.StatusOK, resp2.StatusCode)

	// 3rd request -> 429
	req3 := httptest.NewRequest(fiber.MethodPost, "/workspaces/ws-1/invitations", nil)
	resp3, err := app.Test(req3)
	s.Require().NoError(err)
	defer func() { _ = resp3.Body.Close() }()
	s.Equal(fiber.StatusTooManyRequests, resp3.StatusCode)
}

func (s *FiberSuite) Test_GetFiber_RateLimiting_HealthBypass() {
	cfg := config.Config{
		RateLimitEnabled:             true,
		Env:                          "production",
		RateLimitGeneralMax:          1,
		RateLimitGeneralExpiration:   1 * time.Minute,
		RateLimitSensitiveMax:        1,
		RateLimitSensitiveExpiration: 1 * time.Minute,
	}
	app := config.GetFiber(cfg, s.logger)
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.SendString("health ok")
	})
	app.Get("/readyz", func(c fiber.Ctx) error {
		return c.SendString("ready ok")
	})

	// Send 5 requests each, none should get 429
	for i := 0; i < 5; i++ {
		reqH := httptest.NewRequest(fiber.MethodGet, "/healthz", nil)
		respH, err := app.Test(reqH)
		s.Require().NoError(err)
		_ = respH.Body.Close()
		s.Equal(fiber.StatusOK, respH.StatusCode)

		reqR := httptest.NewRequest(fiber.MethodGet, "/readyz", nil)
		respR, err := app.Test(reqR)
		s.Require().NoError(err)
		_ = respR.Body.Close()
		s.Equal(fiber.StatusOK, respR.StatusCode)
	}
}

func (s *FiberSuite) Test_GetFiber_RateLimiting_TestEnvSkip() {
	cfg := config.Config{
		RateLimitEnabled:           true,
		Env:                        "test",
		RateLimitGeneralMax:        1,
		RateLimitGeneralExpiration: 1 * time.Minute,
	}
	app := config.GetFiber(cfg, s.logger)
	app.Get("/test-skip", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	// In test env, limiter skips by default
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(fiber.MethodGet, "/test-skip", nil)
		resp, err := app.Test(req)
		s.Require().NoError(err)
		_ = resp.Body.Close()
		s.Equal(fiber.StatusOK, resp.StatusCode)
	}

	// With explicit X-Test-Rate-Limit header, it enforces
	req1 := httptest.NewRequest(fiber.MethodGet, "/test-skip", nil)
	req1.Header.Set("X-Test-Rate-Limit", "true")
	resp1, err := app.Test(req1)
	s.Require().NoError(err)
	_ = resp1.Body.Close()
	s.Equal(fiber.StatusOK, resp1.StatusCode)

	req2 := httptest.NewRequest(fiber.MethodGet, "/test-skip", nil)
	req2.Header.Set("X-Test-Rate-Limit", "true")
	resp2, err := app.Test(req2)
	s.Require().NoError(err)
	_ = resp2.Body.Close()
	s.Equal(fiber.StatusTooManyRequests, resp2.StatusCode)
}

func TestFiberSuite(t *testing.T) {
	suite.Run(t, new(FiberSuite))
}
