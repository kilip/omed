package config_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/config"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/stretchr/testify/suite"
)

type mockSentryTransport struct {
	mu      sync.Mutex
	events  []*sentry.Event
	flushed bool
	closed  bool
	options sentry.ClientOptions
}

func (t *mockSentryTransport) Configure(options sentry.ClientOptions) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.options = options
}

func (t *mockSentryTransport) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
}

func (t *mockSentryTransport) Flush(_ time.Duration) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.flushed = true
	return true
}

func (t *mockSentryTransport) FlushWithContext(_ context.Context) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.flushed = true
	return true
}

func (t *mockSentryTransport) Close() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.closed = true
}

func (t *mockSentryTransport) Events() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]*sentry.Event, len(t.events))
	copy(out, t.events)
	return out
}

func (t *mockSentryTransport) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = nil
	t.flushed = false
}

type SentrySuite struct {
	suite.Suite
	logger    *slog.Logger
	transport *mockSentryTransport
}

const testSentryDSN = "https://public@example.com/1"

func (s *SentrySuite) SetupTest() {
	s.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	s.transport = &mockSentryTransport{}
	// Reset global hub to a clean state before each test
	sentry.CurrentHub().BindClient(nil)
}

func (s *SentrySuite) TearDownTest() {
	sentry.CurrentHub().BindClient(nil)
}

func (s *SentrySuite) Test_InitSentry_NoopWhenDSNEmpty() {
	cfg := config.Config{
		Env:       "development",
		SentryDSN: "",
	}

	flush, err := config.InitSentry(cfg)
	s.Require().NoError(err)
	s.Require().NotNil(flush)
	flush()

	s.False(sentry.CurrentHub().Client().IsEnabled())

	app := config.GetFiber(cfg, s.logger)
	app.Use(requestid.New())
	app.Get("/err", func(c fiber.Ctx) error {
		return errors.New("boom without sentry")
	})

	req := httptest.NewRequest(fiber.MethodGet, "/err", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	s.Equal(fiber.StatusInternalServerError, resp.StatusCode)
	s.Empty(s.transport.Events())
}

func (s *SentrySuite) Test_InitSentry_InvalidSampleRate() {
	cfg := config.Config{
		Env:                    "development",
		SentryDSN:              testSentryDSN,
		SentryTracesSampleRate: 2.0,
	}

	_, err := config.InitSentry(cfg)
	s.Require().Error(err)
	s.Contains(err.Error(), "OMED_SENTRY_TRACES_SAMPLE_RATE must be between 0 and 1")
}

func (s *SentrySuite) Test_ErrorHandler_Captures5xx_WithTagsAndUser_Ignores4xx() {
	cfg := config.Config{
		Env:                    "test",
		SentryDSN:              testSentryDSN,
		SentryEnvironment:      "staging",
		SentryRelease:          "omed@1.0.0",
		SentryTracesSampleRate: 0,
	}

	flush, err := config.InitSentryWithTransport(cfg, s.transport)
	s.Require().NoError(err)
	defer flush()

	app := config.GetFiber(cfg, s.logger)
	app.Use(requestid.New())

	// Simulate auth + workspace context middleware
	app.Use(func(c fiber.Ctx) error {
		c.Locals(base.USER_CONTEXT_KEY, base.UserContext{
			ID:   42,
			Name: "Alice Sensitive",
		})
		c.Locals(base.WORKSPACE_CONTEXT_KEY, int64(99))
		return c.Next()
	})

	app.Get("/workspaces/:wid/not-found", func(c fiber.Ctx) error {
		return base.ErrNotFound
	})
	app.Get("/workspaces/:wid/bad-request", func(c fiber.Ctx) error {
		return base.ErrInvalidID
	})
	app.Get("/workspaces/:wid/crash", func(c fiber.Ctx) error {
		return errors.New("unexpected db failure")
	})

	// 1. 404 Not Found -> ignored by Sentry
	req404 := httptest.NewRequest(fiber.MethodGet, "/workspaces/99/not-found", nil)
	req404.Header.Set("X-Request-ID", "req-404")
	resp404, err := app.Test(req404)
	s.Require().NoError(err)
	_ = resp404.Body.Close()
	s.Equal(fiber.StatusNotFound, resp404.StatusCode)
	s.Empty(s.transport.Events())

	// 2. 400 Bad Request -> ignored by Sentry
	req400 := httptest.NewRequest(fiber.MethodGet, "/workspaces/99/bad-request", nil)
	req400.Header.Set("X-Request-ID", "req-400")
	resp400, err := app.Test(req400)
	s.Require().NoError(err)
	_ = resp400.Body.Close()
	s.Equal(fiber.StatusBadRequest, resp400.StatusCode)
	s.Empty(s.transport.Events())

	// 3. 500 Unknown Error -> captured with request_id, workspace_id, user.id, error.code
	req500 := httptest.NewRequest(fiber.MethodGet, "/workspaces/99/crash", nil)
	req500.Header.Set("X-Request-ID", "req-500-abc")
	resp500, err := app.Test(req500)
	s.Require().NoError(err)
	defer func() { _ = resp500.Body.Close() }()
	s.Equal(fiber.StatusInternalServerError, resp500.StatusCode)

	events := s.transport.Events()
	s.Require().Len(events, 1)
	ev := events[0]
	s.Equal(sentry.LevelError, ev.Level)
	s.Equal("staging", ev.Environment)
	s.Equal("omed@1.0.0", ev.Release)
	s.Equal("req-500-abc", ev.Tags["request_id"])
	s.Equal("99", ev.Tags["workspace_id"])
	s.Equal(base.ErrUnknown.Error(), ev.Tags["error.code"])
	s.Equal("GET", ev.Tags["http.method"])
	s.Equal("/workspaces/:wid/crash", ev.Tags["http.route"])
	s.Equal("42", ev.User.ID)
	s.Empty(ev.User.Email)
	s.Empty(ev.User.Name)
	s.Empty(ev.User.Username)
}

func (s *SentrySuite) Test_ErrorHandler_5xxMappedError_IncludesCauseAndCapturesSentry() {
	cfg := config.Config{
		Env:                    "test",
		SentryDSN:              testSentryDSN,
		SentryEnvironment:      "staging",
		SentryRelease:          "omed@1.0.0",
		SentryTracesSampleRate: 0,
	}

	flush, err := config.InitSentryWithTransport(cfg, s.transport)
	s.Require().NoError(err)
	defer flush()

	app := config.GetFiber(cfg, s.logger)
	app.Use(requestid.New())

	app.Post("/workspaces/:wid/agent/chat/messages", func(c fiber.Ctx) error {
		upstreamErr := errors.New("upstream gemini 500 internal error")
		return fmt.Errorf("%w: %w", domain.ErrAnalysisFailed, upstreamErr)
	})

	req := httptest.NewRequest(fiber.MethodPost, "/workspaces/10/agent/chat/messages", strings.NewReader(`{"content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "req-gemini-502")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	s.Equal(fiber.StatusBadGateway, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	var errResp httpx.ErrorResponse
	err = json.Unmarshal(bodyBytes, &errResp)
	s.Require().NoError(err)
	s.Equal("FINANCIAL_ANALYSIS_FAILED", errResp.Error.Code)
	s.Equal("financial analysis failed", errResp.Error.Message)

	events := s.transport.Events()
	s.Require().Len(events, 1)
	ev := events[0]
	s.Equal(sentry.LevelError, ev.Level)
	s.Equal("req-gemini-502", ev.Tags["request_id"])
	s.Equal("FINANCIAL_ANALYSIS_FAILED", ev.Tags["error.code"])
	s.Equal("POST", ev.Tags["http.method"])
	s.Equal("/workspaces/:wid/agent/chat/messages", ev.Tags["http.route"])

	s.Require().NotEmpty(ev.Exception)
	var exceptionValues []string
	for _, exc := range ev.Exception {
		exceptionValues = append(exceptionValues, exc.Value)
	}
	allExceptions := strings.Join(exceptionValues, " | ")
	s.Contains(allExceptions, "upstream gemini 500 internal error")
}

func (s *SentrySuite) Test_PanicRecovery_CapturedBySentry_AndReturns500() {
	cfg := config.Config{
		Env:       "test",
		SentryDSN: testSentryDSN,
	}

	flush, err := config.InitSentryWithTransport(cfg, s.transport)
	s.Require().NoError(err)
	defer flush()

	app := config.GetFiber(cfg, s.logger)
	app.Use(requestid.New())
	app.Use(func(c fiber.Ctx) error {
		c.Locals(base.USER_CONTEXT_KEY, base.UserContext{ID: 7})
		c.Locals(base.WORKSPACE_CONTEXT_KEY, int64(15))
		return c.Next()
	})
	app.Get("/workspaces/:wid/panic", func(c fiber.Ctx) error {
		panic("nil pointer dereference in handler")
	})

	req := httptest.NewRequest(fiber.MethodGet, "/workspaces/15/panic", nil)
	req.Header.Set("X-Request-ID", "req-panic-999")
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	s.Equal(fiber.StatusInternalServerError, resp.StatusCode)
	var errResp httpx.ErrorResponse
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&errResp))
	s.Equal(base.ErrUnknown.Error(), errResp.Error.Code)

	// Wait only capturing panic once (recover.New converts panic to error which hits ErrorHandler;
	// verify panic event was captured with LevelFatal and expected tags)
	events := s.transport.Events()
	s.Require().Len(events, 1)
	ev := events[0]
	s.Equal(sentry.LevelFatal, ev.Level)
	s.Equal("req-panic-999", ev.Tags["request_id"])
	s.Equal("15", ev.Tags["workspace_id"])
	s.Equal(base.ErrUnknown.Error(), ev.Tags["error.code"])
	s.Equal("7", ev.User.ID)
}

func (s *SentrySuite) Test_BeforeSend_ScrubsSensitiveHeadersAndBody() {
	cfg := config.Config{
		Env:       "test",
		SentryDSN: testSentryDSN,
	}

	flush, err := config.InitSentryWithTransport(cfg, s.transport)
	s.Require().NoError(err)
	defer flush()

	app := config.GetFiber(cfg, s.logger)
	app.Use(requestid.New())
	app.Post("/items", func(c fiber.Ctx) error {
		return errors.New("failed to save item")
	})

	body := `{"name":"Widget","password":"super-secret-password","api_token":"tok_live_123","nested":{"clientSecret":"cs_999","keep":"visible"}}`
	req := httptest.NewRequest(fiber.MethodPost, "/items?token=query-secret&page=1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-jwt-token")
	req.Header.Set("Cookie", "omed_session=secret-cookie-val")
	req.Header.Set("X-Custom-Safe", "safe-header-value")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	s.Equal(fiber.StatusInternalServerError, resp.StatusCode)

	events := s.transport.Events()
	s.Require().Len(events, 1)
	ev := events[0]
	s.Require().NotNil(ev.Request)

	// Sensitive headers & cookies must be stripped
	s.Empty(ev.Request.Cookies)
	for k, v := range ev.Request.Headers {
		lower := strings.ToLower(k)
		s.NotEqual("authorization", lower)
		s.NotEqual("cookie", lower)
		s.NotEqual("set-cookie", lower)
		s.NotContains(v, "secret-jwt-token")
		s.NotContains(v, "secret-cookie-val")
	}
	s.Equal("safe-header-value", ev.Request.Headers["X-Custom-Safe"])

	// Query string sensitive params must be scrubbed
	s.NotContains(ev.Request.QueryString, "query-secret")
	s.Contains(ev.Request.QueryString, "page=1")

	// Request body sensitive fields must be scrubbed
	s.Require().NotEmpty(ev.Request.Data)
	s.NotContains(ev.Request.Data, "super-secret-password")
	s.NotContains(ev.Request.Data, "tok_live_123")
	s.NotContains(ev.Request.Data, "cs_999")

	var scrubbedBody map[string]any
	s.Require().NoError(json.Unmarshal([]byte(ev.Request.Data), &scrubbedBody))
	s.Equal("Widget", scrubbedBody["name"])
	s.Equal("[REDACTED]", scrubbedBody["password"])
	s.Equal("[REDACTED]", scrubbedBody["api_token"])
	nested, ok := scrubbedBody["nested"].(map[string]any)
	s.Require().True(ok)
	s.Equal("[REDACTED]", nested["clientSecret"])
	s.Equal("visible", nested["keep"])
}

func (s *SentrySuite) Test_ScrubSentryEvent_DirectUnit() {
	ev := sentry.NewEvent()
	ev.User = sentry.User{
		ID:        "123",
		Email:     "leak@example.com",
		Username:  "leakuser",
		Name:      "Leak Name",
		IPAddress: "1.2.3.4",
	}
	ev.Request = &sentry.Request{
		URL:         "https://api.omed.local/auth/login",
		Method:      "POST",
		Cookies:     "omed_session=secret",
		QueryString: "code=123456&redirect=/home",
		Headers: map[string]string{
			"Authorization": "Bearer secret",
			"Cookie":        "omed_session=secret",
			"Set-Cookie":    "omed_session=secret",
			"Accept":        "application/json",
		},
		Data: `{"password":"my-password","email":"user@example.com"}`,
	}
	ev.Contexts = map[string]sentry.Context{
		"extra": {
			"apiKey": "sk-12345",
			"normal": "ok",
		},
	}

	scrubbed := config.ScrubSentryEvent(ev, nil)
	s.Require().NotNil(scrubbed)
	s.Equal("123", scrubbed.User.ID)
	s.Empty(scrubbed.User.Email)
	s.Empty(scrubbed.User.Username)
	s.Empty(scrubbed.User.Name)
	s.Empty(scrubbed.User.IPAddress)
	s.Empty(scrubbed.Request.Cookies)
	s.NotContains(scrubbed.Request.Headers, "Authorization")
	s.NotContains(scrubbed.Request.Headers, "Cookie")
	s.NotContains(scrubbed.Request.Headers, "Set-Cookie")
	s.Equal("application/json", scrubbed.Request.Headers["Accept"])
	s.NotContains(scrubbed.Request.QueryString, "123456")
	s.NotContains(scrubbed.Request.Data, "my-password")
	s.Equal("[REDACTED]", scrubbed.Contexts["extra"]["apiKey"])
	s.Equal("ok", scrubbed.Contexts["extra"]["normal"])
}

func (s *SentrySuite) Test_TracingMiddleware_ContinuesTraceHeadersAndSendsTransaction() {
	cfg := config.Config{
		Env:                    "test",
		SentryDSN:              testSentryDSN,
		SentryTracesSampleRate: 1.0,
	}

	flush, err := config.InitSentryWithTransport(cfg, s.transport)
	s.Require().NoError(err)
	defer flush()

	app := config.GetFiber(cfg, s.logger)
	app.Use(requestid.New())
	app.Get("/workspaces/:wid/accounts", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"ok": true})
	})

	incomingTraceID := "12345678901234567890123456789012"
	incomingSpanID := "1234567890123456"
	req := httptest.NewRequest(fiber.MethodGet, "/workspaces/55/accounts", nil)
	req.Header.Set("sentry-trace", incomingTraceID+"-"+incomingSpanID+"-1")
	req.Header.Set("baggage", "sentry-environment=test,sentry-trace_id="+incomingTraceID)
	req.Header.Set("X-Request-ID", "req-trace-1")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()
	s.Equal(fiber.StatusOK, resp.StatusCode)

	events := s.transport.Events()
	s.Require().Len(events, 1)
	txEvent := events[0]
	s.Equal("transaction", txEvent.Type)
	s.Equal("GET /workspaces/:wid/accounts", txEvent.Transaction)
	s.Equal("req-trace-1", txEvent.Tags["request_id"])
	s.Equal("55", txEvent.Tags["workspace_id"])
	s.Equal(incomingTraceID, txEvent.Contexts["trace"]["trace_id"].(sentry.TraceID).String())
}

func TestSentrySuite(t *testing.T) {
	suite.Run(t, new(SentrySuite))
}
