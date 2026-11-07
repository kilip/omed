package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

const (
	sentryHubLocalsKey     = "omed:sentry_hub"
	sentryPanicCapturedKey = "omed:sentry_panic_captured"
	sentryFlushTimeout     = 2 * time.Second
)

var sensitiveHeaders = map[string]struct{}{
	"authorization":       {},
	"cookie":              {},
	"set-cookie":          {},
	"x-api-key":           {},
	"proxy-authorization": {},
}

func isSensitiveHeader(key string) bool {
	lower := strings.ToLower(strings.TrimSpace(key))
	if _, ok := sensitiveHeaders[lower]; ok {
		return true
	}
	return isSensitiveField(lower)
}

func scrubRawData(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	var parsed any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
		sanitized := sanitizeValue(parsed)
		if b, err := json.Marshal(sanitized); err == nil {
			return string(b)
		}
	}
	if containsSensitiveKeyword(trimmed) {
		return "[REDACTED]"
	}
	if len(trimmed) > maxMalformedRawBytes {
		trimmed = trimmed[:maxMalformedRawBytes]
	}
	return strings.ToValidUTF8(trimmed, "")
}

func scrubQueryString(raw string) string {
	if raw == "" {
		return ""
	}
	vals, err := url.ParseQuery(raw)
	if err != nil {
		if containsSensitiveKeyword(raw) {
			return "[REDACTED]"
		}
		return raw
	}
	changed := false
	for k := range vals {
		if isSensitiveField(k) {
			vals.Set(k, "[REDACTED]")
			changed = true
		}
	}
	if changed {
		return vals.Encode()
	}
	return raw
}

// ScrubSentryEvent strips sensitive headers, cookies, user PII, and request payload fields before sending to Sentry.
func ScrubSentryEvent(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	if event == nil {
		return nil
	}

	if event.Request != nil {
		event.Request.Cookies = ""
		if len(event.Request.Headers) > 0 {
			for k := range event.Request.Headers {
				if isSensitiveHeader(k) {
					delete(event.Request.Headers, k)
				}
			}
		}
		if event.Request.Data != "" {
			event.Request.Data = scrubRawData(event.Request.Data)
		}
		if event.Request.QueryString != "" {
			event.Request.QueryString = scrubQueryString(event.Request.QueryString)
		}
		if len(event.Request.Env) > 0 {
			for k := range event.Request.Env {
				if isSensitiveField(k) {
					delete(event.Request.Env, k)
				}
			}
		}
	}

	// Ensure user object only carries opaque ID and never email, username, name, or IP.
	event.User.Email = ""
	event.User.Username = ""
	event.User.Name = ""
	event.User.IPAddress = ""

	if len(event.Contexts) > 0 {
		for k, ctxMap := range event.Contexts {
			if sanitized, ok := sanitizeValue(map[string]any(ctxMap)).(map[string]any); ok {
				event.Contexts[k] = sentry.Context(sanitized)
			}
		}
	}

	return event
}

// InitSentry initializes the Sentry Go SDK when cfg.SentryDSN is configured.
// When cfg.SentryDSN is empty, it acts as a safe no-op and returns a no-op flush function.
func InitSentry(cfg Config) (func(), error) {
	return initSentryWithTransport(cfg, nil)
}

func initSentryWithTransport(cfg Config, transport sentry.Transport) (func(), error) {
	if strings.TrimSpace(cfg.SentryDSN) == "" {
		return func() {}, nil
	}

	if err := cfg.Validate(); err != nil {
		return func() {}, err
	}

	sampleRate := cfg.SentryTracesSampleRate
	opts := sentry.ClientOptions{
		Dsn:              cfg.SentryDSN,
		Environment:      cfg.GetSentryEnvironment(),
		Release:          cfg.SentryRelease,
		AttachStacktrace: true,
		DataCollection: &sentry.DataCollection{
			UserInfo:   sentry.Set(false),
			Cookies:    &sentry.KeyValueCollectionBehavior{Mode: sentry.CollectionOff},
			HTTPBodies: []sentry.BodyType{sentry.BodyIncomingRequest},
		},
		EnableTracing:         sampleRate > 0,
		TracesSampleRate:      sampleRate,
		BeforeSend:            ScrubSentryEvent,
		BeforeSendTransaction: ScrubSentryEvent,
		Transport:             transport,
	}

	if err := sentry.Init(opts); err != nil {
		return func() {}, err
	}

	return func() {
		sentry.Flush(sentryFlushTimeout)
	}, nil
}

// HubFromFiber retrieves the request-scoped *sentry.Hub from fiber.Ctx locals or context,
// falling back to sentry.CurrentHub().
func HubFromFiber(c fiber.Ctx) *sentry.Hub {
	if c == nil {
		return sentry.CurrentHub()
	}
	if hub, ok := c.Locals(sentryHubLocalsKey).(*sentry.Hub); ok && hub != nil {
		return hub
	}
	if hub := sentry.GetHubFromContext(c.Context()); hub != nil {
		return hub
	}
	return sentry.CurrentHub()
}

func buildHTTPRequestFromFiber(c fiber.Ctx) *http.Request {
	rawURL := c.OriginalURL()
	if rawURL == "" {
		rawURL = c.Path()
	}
	var bodyReader *bytes.Reader
	body := c.Body()
	if len(body) > 0 && len(body) <= maxLoggedBodyBytes && !isSensitivePayloadRoute(c) {
		bodyReader = bytes.NewReader(body)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(c.Context(), c.Method(), rawURL, bodyReader)
	if err != nil {
		return nil
	}
	req.Host = c.Hostname()
	for k, vals := range c.GetReqHeaders() {
		if isSensitiveHeader(k) {
			continue
		}
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	return req
}

func enrichScopeFromFiber(scope *sentry.Scope, c fiber.Ctx) {
	if scope == nil || c == nil {
		return
	}

	if reqID := requestid.FromContext(c); reqID != "" {
		scope.SetTag("request_id", reqID)
	}

	wsID := base.WorkspaceIDFromContext(c)
	if wsID <= 0 {
		if rawWid := c.Params("wid"); rawWid != "" {
			if parsed, err := strconv.ParseInt(rawWid, 10, 64); err == nil && parsed > 0 {
				wsID = parsed
			}
		}
	}
	if wsID > 0 {
		scope.SetTag("workspace_id", strconv.FormatInt(wsID, 10))
	}

	if userID := base.UserIDFromContext(c); userID > 0 {
		scope.SetUser(sentry.User{
			ID: strconv.FormatInt(userID, 10),
		})
	}

	if method := c.Method(); method != "" {
		scope.SetTag("http.method", method)
	}
	if route := routePatternFromFiber(c); route != "" {
		scope.SetTag("http.route", route)
	}

	if httpReq := buildHTTPRequestFromFiber(c); httpReq != nil {
		scope.SetRequest(httpReq)
		if body := c.Body(); len(body) > 0 && len(body) <= maxLoggedBodyBytes && !isSensitivePayloadRoute(c) {
			scope.SetRequestBody([]byte(scrubRawData(string(body))))
		}
	}
}

func routePatternFromFiber(c fiber.Ctx) string {
	if c == nil {
		return ""
	}
	if r := c.Route(); r != nil && r.Path != "" && r.Path != "/" {
		return r.Path
	}
	return c.Path()
}

// SentryMiddleware clones a *sentry.Hub onto the Fiber context, continues distributed traces when enabled,
// enriches scope tags (request_id, workspace_id, user.id), and captures unhandled panics with LevelFatal
// before re-panicking so Fiber's recover middleware handles the HTTP 500 response.
func SentryMiddleware(cfg Config) fiber.Handler {
	tracingEnabled := strings.TrimSpace(cfg.SentryDSN) != "" && cfg.SentryTracesSampleRate > 0

	return func(c fiber.Ctx) error {
		hub := sentry.CurrentHub().Clone()
		c.Locals(sentryHubLocalsKey, hub)

		ctx := sentry.SetHubOnContext(c.Context(), hub)

		var tx *sentry.Span
		sentryTrace := c.Get(sentry.SentryTraceHeader)
		baggage := c.Get("baggage")

		if tracingEnabled {
			txName := fmt.Sprintf("%s %s", c.Method(), c.Path())
			opts := []sentry.SpanOption{
				sentry.WithOpName("http.server"),
				sentry.WithTransactionSource(sentry.SourceURL),
				sentry.WithSpanOrigin(sentry.SpanOriginFiber),
				sentry.ContinueTrace(hub, sentryTrace, baggage),
			}
			tx = sentry.StartTransaction(ctx, txName, opts...)
			ctx = tx.Context()
		} else if sentryTrace != "" || baggage != "" {
			if pc, err := sentry.PropagationContextFromHeaders(sentryTrace, baggage); err == nil {
				hub.Scope().SetPropagationContext(pc)
			}
		}

		c.SetContext(ctx)
		enrichScopeFromFiber(hub.Scope(), c)

		defer func() {
			if r := recover(); r != nil {
				c.Locals(sentryPanicCapturedKey, true)
				enrichScopeFromFiber(hub.Scope(), c)
				hub.WithScope(func(scope *sentry.Scope) {
					scope.SetLevel(sentry.LevelFatal)
					scope.SetTag("error.code", base.ErrUnknown.Error())
					enrichScopeFromFiber(scope, c)
					hub.RecoverWithContext(ctx, r)
				})
				if tx != nil {
					tx.Status = sentry.SpanStatusInternalError
					tx.Finish()
				}
				panic(r)
			}
		}()

		err := c.Next()

		enrichScopeFromFiber(hub.Scope(), c)

		if tx != nil {
			route := routePatternFromFiber(c)
			if r := c.Route(); r != nil && r.Path != "" {
				tx.Name = fmt.Sprintf("%s %s", c.Method(), route)
				tx.Source = sentry.SourceRoute
			}
			status := c.Response().StatusCode()
			if err != nil {
				status, _ = httpx.MapError(err)
			}
			tx.Status = sentry.HTTPtoSpanStatus(status)
			if reqID := requestid.FromContext(c); reqID != "" {
				tx.SetTag("request_id", reqID)
			}
			if wsID := base.WorkspaceIDFromContext(c); wsID > 0 {
				tx.SetTag("workspace_id", strconv.FormatInt(wsID, 10))
			}
			tx.Finish()
		}

		return err
	}
}

// CaptureFiberError reports an HTTP >= 500 error from Fiber's ErrorHandler to Sentry
// with request_id, workspace_id, error.code, http.method, http.route tags, and sentry.User{ID}.
func CaptureFiberError(c fiber.Ctx, err error, code string) {
	if err == nil {
		return
	}
	if c != nil {
		if captured, ok := c.Locals(sentryPanicCapturedKey).(bool); ok && captured {
			return
		}
	}
	hub := HubFromFiber(c)
	if hub == nil || hub.Client() == nil || !hub.Client().IsEnabled() {
		return
	}

	if code == "" {
		code = base.ErrUnknown.Error()
	}

	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelError)
		scope.SetTag("error.code", code)
		if c != nil {
			enrichScopeFromFiber(scope, c)
		}
		hub.CaptureException(err)
	})
}
