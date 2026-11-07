package config_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/agent/domain"
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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type, sentry-trace, baggage")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	s.Equal("http://trusted.com", resp.Header.Get("Access-Control-Allow-Origin"))
	s.Contains(resp.Header.Get("Access-Control-Allow-Methods"), "POST")
	allowHeaders := strings.ToLower(resp.Header.Get("Access-Control-Allow-Headers"))
	s.Contains(allowHeaders, "sentry-trace")
	s.Contains(allowHeaders, "baggage")
}

func (s *FiberSuite) Test_GetFiber_RecoverMiddleware() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Get("/panic", func(c fiber.Ctx) error {
		panic("unexpected crash")
	})

	req := httptest.NewRequest(fiber.MethodGet, "/panic", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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

func (s *FiberSuite) Test_ErrorHandler_5xxMappedError_LogsCause() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Get("/bad-gateway", func(c fiber.Ctx) error {
		return fmt.Errorf("%w: upstream 500 error occurred", domain.ErrAnalysisFailed)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/bad-gateway", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	s.Equal(fiber.StatusBadGateway, resp.StatusCode)
	var errResp httpx.ErrorResponse
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&errResp))
	s.Equal("FINANCIAL_ANALYSIS_FAILED", errResp.Error.Code)

	logged := s.logBuf.String()
	s.Contains(logged, "upstream 500 error occurred")
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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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

func (s *FiberSuite) Test_SensitiveRoute_OTPRequest_RateLimited() {
	cfg := config.Config{
		RateLimitEnabled:             true,
		Env:                          "production",
		RateLimitGeneralMax:          100,
		RateLimitGeneralExpiration:   1 * time.Minute,
		RateLimitSensitiveMax:        2,
		RateLimitSensitiveExpiration: 1 * time.Minute,
	}
	app := config.GetFiber(cfg, s.logger)
	app.Post("/auth/otp/request", func(c fiber.Ctx) error {
		return c.SendString("otp request ok")
	})

	// 1st request -> 200
	req1 := httptest.NewRequest(fiber.MethodPost, "/auth/otp/request", nil)
	resp1, err := app.Test(req1)
	s.Require().NoError(err)
	_ = resp1.Body.Close()
	s.Equal(fiber.StatusOK, resp1.StatusCode)

	// 2nd request -> 200
	req2 := httptest.NewRequest(fiber.MethodPost, "/auth/otp/request", nil)
	resp2, err := app.Test(req2)
	s.Require().NoError(err)
	_ = resp2.Body.Close()
	s.Equal(fiber.StatusOK, resp2.StatusCode)

	// 3rd request -> 429
	req3 := httptest.NewRequest(fiber.MethodPost, "/auth/otp/request", nil)
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

func parseLastLogEntry(buf string) (map[string]any, error) {
	lines := strings.Split(strings.TrimSpace(buf), "\n")
	var entry map[string]any
	err := json.Unmarshal([]byte(lines[len(lines)-1]), &entry)
	return entry, err
}

func (s *FiberSuite) Test_ErrorDetails_StructuredLogging_On4xx_5xx() {
	s.cfg.LogHTTPBody = false
	app := config.GetFiber(s.cfg, s.logger)

	// 1. 404 mapped error
	app.Get("/test-404", func(c fiber.Ctx) error {
		return base.ErrNotFound
	})
	s.logBuf.Reset()
	req404 := httptest.NewRequest(fiber.MethodGet, "/test-404", nil)
	resp404, err := app.Test(req404)
	s.Require().NoError(err)
	_ = resp404.Body.Close()
	s.Equal(fiber.StatusNotFound, resp404.StatusCode)

	logged404 := s.logBuf.String()
	s.Contains(logged404, `"error":{"code":"NOT_FOUND","message":"not found"}`)
	s.NotContains(logged404, `"request_body"`)
	entry404, err := parseLastLogEntry(logged404)
	s.Require().NoError(err)
	s.Equal("NOT_FOUND: not found", entry404["msg"])

	// 2. 422 validation failure
	app.Post("/test-422", func(c fiber.Ctx) error {
		var p testPayload
		if err := c.Bind().Body(&p); err != nil {
			return err
		}
		return c.SendString("ok")
	})
	s.logBuf.Reset()
	req422 := httptest.NewRequest(fiber.MethodPost, "/test-422", strings.NewReader(`{"name":"","age":10}`))
	req422.Header.Set("Content-Type", "application/json")
	resp422, err := app.Test(req422)
	s.Require().NoError(err)
	_ = resp422.Body.Close()
	s.Equal(fiber.StatusUnprocessableEntity, resp422.StatusCode)

	logged422 := s.logBuf.String()
	s.Contains(logged422, `"code":"VALIDATION_FAILED"`)
	s.Contains(logged422, `"message":"validation failed"`)
	s.Contains(logged422, `"fields":`)
	s.NotContains(logged422, `"request_body"`)
	entry422, err := parseLastLogEntry(logged422)
	s.Require().NoError(err)
	s.Equal("VALIDATION_FAILED: validation failed", entry422["msg"])

	// 3. 500 unknown error
	app.Get("/test-500", func(c fiber.Ctx) error {
		return errors.New("critical failure")
	})
	s.logBuf.Reset()
	req500 := httptest.NewRequest(fiber.MethodGet, "/test-500", nil)
	resp500, err := app.Test(req500)
	s.Require().NoError(err)
	_ = resp500.Body.Close()
	s.Equal(fiber.StatusInternalServerError, resp500.StatusCode)

	logged500 := s.logBuf.String()
	s.Contains(logged500, `"code":"UNKNOWN"`)
	s.Contains(logged500, `"message":"unknown error"`)
	s.Contains(logged500, `"cause":"critical failure"`)
	s.NotContains(logged500, `"request_body"`)
	entry500, err := parseLastLogEntry(logged500)
	s.Require().NoError(err)
	s.Equal("UNKNOWN: unknown error", entry500["msg"])
}

func (s *FiberSuite) Test_ErrorDetails_StructuredLogging_RateLimit_429() {
	cfg := config.Config{
		RateLimitEnabled:             true,
		Env:                          "production",
		RateLimitGeneralMax:          1,
		RateLimitGeneralExpiration:   1 * time.Minute,
		RateLimitSensitiveMax:        10,
		RateLimitSensitiveExpiration: 1 * time.Minute,
	}
	app := config.GetFiber(cfg, s.logger)
	app.Get("/rate-test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req1 := httptest.NewRequest(fiber.MethodGet, "/rate-test", nil)
	resp1, err := app.Test(req1)
	s.Require().NoError(err)
	_ = resp1.Body.Close()
	s.Equal(fiber.StatusOK, resp1.StatusCode)

	s.logBuf.Reset()
	req2 := httptest.NewRequest(fiber.MethodGet, "/rate-test", nil)
	resp2, err := app.Test(req2)
	s.Require().NoError(err)
	defer func() { _ = resp2.Body.Close() }()
	s.Equal(fiber.StatusTooManyRequests, resp2.StatusCode)

	logged := s.logBuf.String()
	s.Contains(logged, `"code":"TOO_MANY_REQUESTS"`)
	s.Contains(logged, `"message":"Too many requests, please try again later"`)
	entry, err := parseLastLogEntry(logged)
	s.Require().NoError(err)
	s.Equal("TOO_MANY_REQUESTS: Too many requests, please try again later", entry["msg"])
}

func (s *FiberSuite) Test_RequestBodyLogging_SanitizesSensitiveFields() {
	s.cfg.LogHTTPBody = true
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/account", func(c fiber.Ctx) error {
		return base.ErrInvalidID
	})

	payload := map[string]any{
		"username":         "alice",
		"password":         "plain-secret-password",
		"pass":             "another-pass",
		"current_password": "old-password",
		"new_password":     "brand-new-password",
		"secret":           "my-secret",
		"secret_key":       "my-secret-key",
		"client_secret":    "oauth-client-secret",
		"token":            "session-token-123",
		"access_token":     "jwt-access-token",
		"refresh_token":    "jwt-refresh-token",
		"id_token":         "oidc-id-token",
		"authorization":    "Bearer secret-bearer",
		"api_key":          "api-key-999",
		"code":             "verification-code-123456",
		"otp":              "654321",
		"credential":       "cred-val",
		"credentials":      "creds-val",
		"private_key":      "-----BEGIN PRIVATE KEY-----",
		"nested": map[string]any{
			"password": "nested-password",
			"details":  "public-detail",
		},
		"items": []any{
			map[string]any{
				"token": "item-token",
				"label": "public-item",
			},
		},
	}
	bodyBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	req := httptest.NewRequest(fiber.MethodPost, "/account", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	s.logBuf.Reset()
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	s.Equal(fiber.StatusBadRequest, resp.StatusCode)

	logged := s.logBuf.String()
	s.Contains(logged, `"request_body"`)

	var logEntry struct {
		RequestBody map[string]any `json:"request_body"`
	}
	s.Require().NoError(json.Unmarshal([]byte(logged), &logEntry))
	s.Require().NotNil(logEntry.RequestBody)

	// Sensitive fields must be redacted
	s.Equal("[REDACTED]", logEntry.RequestBody["password"])
	s.Equal("[REDACTED]", logEntry.RequestBody["pass"])
	s.Equal("[REDACTED]", logEntry.RequestBody["current_password"])
	s.Equal("[REDACTED]", logEntry.RequestBody["new_password"])
	s.Equal("[REDACTED]", logEntry.RequestBody["secret"])
	s.Equal("[REDACTED]", logEntry.RequestBody["secret_key"])
	s.Equal("[REDACTED]", logEntry.RequestBody["client_secret"])
	s.Equal("[REDACTED]", logEntry.RequestBody["token"])
	s.Equal("[REDACTED]", logEntry.RequestBody["access_token"])
	s.Equal("[REDACTED]", logEntry.RequestBody["refresh_token"])
	s.Equal("[REDACTED]", logEntry.RequestBody["id_token"])
	s.Equal("[REDACTED]", logEntry.RequestBody["authorization"])
	s.Equal("[REDACTED]", logEntry.RequestBody["api_key"])
	s.Equal("[REDACTED]", logEntry.RequestBody["code"])
	s.Equal("[REDACTED]", logEntry.RequestBody["otp"])
	s.Equal("[REDACTED]", logEntry.RequestBody["credential"])
	s.Equal("[REDACTED]", logEntry.RequestBody["credentials"])
	s.Equal("[REDACTED]", logEntry.RequestBody["private_key"])

	// Non-sensitive fields preserved
	s.Equal("alice", logEntry.RequestBody["username"])

	// Nested object
	nested, ok := logEntry.RequestBody["nested"].(map[string]any)
	s.Require().True(ok)
	s.Equal("[REDACTED]", nested["password"])
	s.Equal("public-detail", nested["details"])

	// Array of objects
	items, ok := logEntry.RequestBody["items"].([]any)
	s.Require().True(ok)
	s.Require().Len(items, 1)
	item0, ok := items[0].(map[string]any)
	s.Require().True(ok)
	s.Equal("[REDACTED]", item0["token"])
	s.Equal("public-item", item0["label"])
}

func (s *FiberSuite) Test_RequestBodyLogging_NotLoggedOn2xx() {
	s.cfg.LogHTTPBody = true
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/success", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true})
	})

	body := `{"user":"alice","data":"something"}`
	req := httptest.NewRequest(fiber.MethodPost, "/success", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	s.logBuf.Reset()
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	s.Equal(fiber.StatusOK, resp.StatusCode)

	logged := s.logBuf.String()
	s.NotContains(logged, "request_body")
	entry, err := parseLastLogEntry(logged)
	s.Require().NoError(err)
	s.Equal("Incoming request", entry["msg"])
}

func (s *FiberSuite) Test_RequestBodyLogging_NotLoggedWhenDisabled() {
	s.cfg.LogHTTPBody = false
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/failed", func(c fiber.Ctx) error {
		return base.ErrInvalidID
	})

	body := `{"user":"alice","data":"something"}`
	req := httptest.NewRequest(fiber.MethodPost, "/failed", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	s.logBuf.Reset()
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	s.Equal(fiber.StatusBadRequest, resp.StatusCode)

	logged := s.logBuf.String()
	s.NotContains(logged, "request_body")
}

func (s *FiberSuite) Test_RequestBodyLogging_MultipartNotLogged() {
	s.cfg.LogHTTPBody = true
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/upload", func(c fiber.Ctx) error {
		return base.ErrInvalidID
	})

	body := "--boundary\r\nContent-Disposition: form-data; name=\"file\"\r\n\r\ncontent\r\n--boundary--"
	req := httptest.NewRequest(fiber.MethodPost, "/upload", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")

	s.logBuf.Reset()
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	s.Equal(fiber.StatusBadRequest, resp.StatusCode)

	logged := s.logBuf.String()
	s.NotContains(logged, "request_body")
}

func (s *FiberSuite) Test_RequestBodyLogging_MalformedJSON() {
	s.cfg.LogHTTPBody = true
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/validate", func(c fiber.Ctx) error {
		var p testPayload
		if err := c.Bind().Body(&p); err != nil {
			return fiber.ErrBadRequest
		}
		return c.SendString("ok")
	})
	app.Post("/auth/login", func(c fiber.Ctx) error {
		var p testPayload
		if err := c.Bind().Body(&p); err != nil {
			return fiber.ErrBadRequest
		}
		return c.SendString("ok")
	})

	// Non-sensitive route -> logs raw truncated body
	malformed := `{"broken: json`
	req1 := httptest.NewRequest(fiber.MethodPost, "/validate", strings.NewReader(malformed))
	req1.Header.Set("Content-Type", "application/json")

	s.logBuf.Reset()
	resp1, err := app.Test(req1)
	s.Require().NoError(err)
	_ = resp1.Body.Close()
	s.Equal(fiber.StatusBadRequest, resp1.StatusCode)

	logged1 := s.logBuf.String()
	entry1, err := parseLastLogEntry(logged1)
	s.Require().NoError(err)
	reqBody, ok := entry1["request_body"].(map[string]any)
	s.Require().True(ok)
	s.Equal(true, reqBody["_malformed"])
	s.Equal(`{"broken: json`, reqBody["raw"])

	// Sensitive route -> does NOT log raw body
	sensitiveMalformed := `{"password":"secret-broken`
	req2 := httptest.NewRequest(fiber.MethodPost, "/auth/login", strings.NewReader(sensitiveMalformed))
	req2.Header.Set("Content-Type", "application/json")

	s.logBuf.Reset()
	resp2, err := app.Test(req2)
	s.Require().NoError(err)
	_ = resp2.Body.Close()
	s.Equal(fiber.StatusBadRequest, resp2.StatusCode)

	logged2 := s.logBuf.String()
	s.NotContains(logged2, "request_body")
	s.NotContains(logged2, "secret-broken")
}

func (s *FiberSuite) Test_RequestBodyLogging_MalformedJSON_SensitiveSuppression() {
	s.cfg.LogHTTPBody = true
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/workspaces/:wid/agent/providers", func(c fiber.Ctx) error {
		var p testPayload
		if err := c.Bind().Body(&p); err != nil {
			return fiber.ErrBadRequest
		}
		return c.SendString("ok")
	})
	app.Post("/tokens", func(c fiber.Ctx) error {
		var p testPayload
		if err := c.Bind().Body(&p); err != nil {
			return fiber.ErrBadRequest
		}
		return c.SendString("ok")
	})
	app.Post("/validate", func(c fiber.Ctx) error {
		var p testPayload
		if err := c.Bind().Body(&p); err != nil {
			return fiber.ErrBadRequest
		}
		return c.SendString("ok")
	})

	// 1. Malformed JSON on /workspaces/1/agent/providers
	malformed1 := `{"name": "broken`
	req1 := httptest.NewRequest(fiber.MethodPost, "/workspaces/1/agent/providers", strings.NewReader(malformed1))
	req1.Header.Set("Content-Type", "application/json")
	s.logBuf.Reset()
	resp1, err := app.Test(req1)
	s.Require().NoError(err)
	_ = resp1.Body.Close()
	s.Equal(fiber.StatusBadRequest, resp1.StatusCode)
	s.NotContains(s.logBuf.String(), "request_body")

	// 2. Malformed JSON on /tokens
	reqTokens := httptest.NewRequest(fiber.MethodPost, "/tokens", strings.NewReader(malformed1))
	reqTokens.Header.Set("Content-Type", "application/json")
	s.logBuf.Reset()
	respTokens, err := app.Test(reqTokens)
	s.Require().NoError(err)
	_ = respTokens.Body.Close()
	s.Equal(fiber.StatusBadRequest, respTokens.StatusCode)
	s.NotContains(s.logBuf.String(), "request_body")

	// 3. Malformed JSON on non-sensitive route containing "apiKey" (case-insensitive)
	malformed2 := `{"apiKey": "abc-broken`
	req2 := httptest.NewRequest(fiber.MethodPost, "/validate", strings.NewReader(malformed2))
	req2.Header.Set("Content-Type", "application/json")
	s.logBuf.Reset()
	resp2, err := app.Test(req2)
	s.Require().NoError(err)
	_ = resp2.Body.Close()
	s.Equal(fiber.StatusBadRequest, resp2.StatusCode)
	s.NotContains(s.logBuf.String(), "request_body")
	s.NotContains(s.logBuf.String(), "abc-broken")
}

func (s *FiberSuite) Test_RequestBodyLogging_RootLevelJSONArray() {
	s.cfg.LogHTTPBody = true
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/batch", func(c fiber.Ctx) error {
		return base.ErrInvalidID
	})

	body := `[{"password":"123","name":"item"}]`
	req := httptest.NewRequest(fiber.MethodPost, "/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	s.logBuf.Reset()
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	s.Equal(fiber.StatusBadRequest, resp.StatusCode)

	entry, err := parseLastLogEntry(s.logBuf.String())
	s.Require().NoError(err)
	items, ok := entry["request_body"].([]any)
	s.Require().True(ok)
	s.Require().Len(items, 1)

	item0, ok := items[0].(map[string]any)
	s.Require().True(ok)
	s.Equal("[REDACTED]", item0["password"])
	s.Equal("item", item0["name"])
}

func (s *FiberSuite) Test_RequestBodyLogging_ContentTypeWithCharset() {
	s.cfg.LogHTTPBody = true
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/account", func(c fiber.Ctx) error {
		return base.ErrInvalidID
	})

	body := `{"password":"secret","name":"john"}`
	req := httptest.NewRequest(fiber.MethodPost, "/account", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	s.logBuf.Reset()
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	s.Equal(fiber.StatusBadRequest, resp.StatusCode)

	entry, err := parseLastLogEntry(s.logBuf.String())
	s.Require().NoError(err)
	reqBody, ok := entry["request_body"].(map[string]any)
	s.Require().True(ok)
	s.Equal("[REDACTED]", reqBody["password"])
	s.Equal("john", reqBody["name"])
}

func (s *FiberSuite) Test_RequestBodyLogging_ExtendedSensitiveFields_AndPreservedKeys() {
	s.cfg.LogHTTPBody = true
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/account", func(c fiber.Ctx) error {
		return base.ErrInvalidID
	})

	payload := map[string]any{
		"confirm_password": "secret-confirm",
		"api_token":        "tok-12345",
		"webhook_secret":   "whsec_xyz",
		"totp":             "123456",
		"pin":              "4321",
		"private_key":      "priv-key-content",
		"passphrase":       "user-passphrase",
		"accesskey":        "ak-999",
		"idempotencyKey":   "idem-key-val",
		"sortKey":          "asc",
		"fileKey":          "files/document.pdf",
		"groupKey":         "grp-1",
	}
	bodyBytes, err := json.Marshal(payload)
	s.Require().NoError(err)

	req := httptest.NewRequest(fiber.MethodPost, "/account", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	s.logBuf.Reset()
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	s.Equal(fiber.StatusBadRequest, resp.StatusCode)

	entry, err := parseLastLogEntry(s.logBuf.String())
	s.Require().NoError(err)
	reqBody, ok := entry["request_body"].(map[string]any)
	s.Require().True(ok)

	// Extended sensitive fields must be redacted
	s.Equal("[REDACTED]", reqBody["confirm_password"])
	s.Equal("[REDACTED]", reqBody["api_token"])
	s.Equal("[REDACTED]", reqBody["webhook_secret"])
	s.Equal("[REDACTED]", reqBody["totp"])
	s.Equal("[REDACTED]", reqBody["pin"])
	s.Equal("[REDACTED]", reqBody["private_key"])
	s.Equal("[REDACTED]", reqBody["passphrase"])
	s.Equal("[REDACTED]", reqBody["accesskey"])

	// Non-sensitive *Key fields must be preserved
	s.Equal("idem-key-val", reqBody["idempotencyKey"])
	s.Equal("asc", reqBody["sortKey"])
	s.Equal("files/document.pdf", reqBody["fileKey"])
	s.Equal("grp-1", reqBody["groupKey"])
}

func (s *FiberSuite) Test_RequestBodyLogging_MalformedJSON_TruncatedAndValidUTF8() {
	s.cfg.LogHTTPBody = true
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/validate", func(c fiber.Ctx) error {
		return fiber.ErrBadRequest
	})

	// 1500 bytes with invalid UTF-8 sequence at the end
	raw := strings.Repeat("a", 1023) + string([]byte{0xff, 0xfe}) + strings.Repeat("b", 500)
	req := httptest.NewRequest(fiber.MethodPost, "/validate", strings.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")

	s.logBuf.Reset()
	resp, err := app.Test(req)
	s.Require().NoError(err)
	_ = resp.Body.Close()
	s.Equal(fiber.StatusBadRequest, resp.StatusCode)

	entry, err := parseLastLogEntry(s.logBuf.String())
	s.Require().NoError(err)
	reqBody, ok := entry["request_body"].(map[string]any)
	s.Require().True(ok)
	s.Equal(true, reqBody["_malformed"])
	rawStr, ok := reqBody["raw"].(string)
	s.Require().True(ok)
	s.LessOrEqual(len(rawStr), 1024)
}

func (s *FiberSuite) Test_RequestBodyLogging_ExceedsMaxSize() {
	s.cfg.LogHTTPBody = true
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/large", func(c fiber.Ctx) error {
		return base.ErrInvalidID
	})

	// Create JSON > 64KB
	largeData := strings.Repeat("x", 65*1024)
	largeBody := `{"data":"` + largeData + `"}`
	req := httptest.NewRequest(fiber.MethodPost, "/large", strings.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")

	s.logBuf.Reset()
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	s.Equal(fiber.StatusBadRequest, resp.StatusCode)

	logged := s.logBuf.String()
	s.NotContains(logged, "request_body")
}

func TestFiberSuite(t *testing.T) {
	suite.Run(t, new(FiberSuite))
}
