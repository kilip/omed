package mcpx_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/require"

	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/internal/platform/mcpx"
)

type authRoundTripper struct {
	header string
	base   http.RoundTripper
}

func (rt *authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	if rt.header != "" {
		req.Header.Set("Authorization", rt.header)
	}
	base := rt.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

func Test_Handler_ValidToken_ListToolsAndCallSeeUser(t *testing.T) {
	var resolvedUserID int64
	res := &stubResolver{
		resolve: func(ctx context.Context, userID, workspaceID int64) (authz.Membership, error) {
			resolvedUserID = userID
			return authz.Membership{
				UserID:      userID,
				WorkspaceID: workspaceID,
				Roles:       []string{"fin:viewer"},
			}, nil
		},
	}
	auth := setupTestAuthz(res)
	r := mcpx.NewRegistrar(mcpx.Config{
		Authorizer: auth,
		Resolver:   res,
	})

	mcpx.AddTool(r, mcpx.ToolSpec{
		Name:        "fin_ping",
		Description: "Ping finance",
		ReadOnly:    true,
		Permissions: []mcpx.Permission{
			{Object: "fin:account", Action: "read"},
		},
	}, func(ctx context.Context, in testInput) (testOutput, error) {
		return testOutput{Message: "pong"}, nil
	})

	cfg := mcpx.HandlerConfig{
		Verifier: func(ctx context.Context, token string) (mcpx.VerifiedToken, error) {
			if token == "secret-token-7" {
				return mcpx.VerifiedToken{UserID: 7}, nil
			}
			return mcpx.VerifiedToken{}, mcpx.ErrTokenInvalid
		},
		TokenSettingsURL: "https://app.example.com/core/access-tokens",
	}

	app := fiber.New()
	app.All("/mcp", mcpx.Handler(r, cfg))

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go func() {
		_ = app.Listener(ln)
	}()
	defer func() {
		_ = app.Shutdown()
	}()

	clientTransport := &mcp.StreamableClientTransport{
		Endpoint: fmt.Sprintf("http://%s/mcp", ln.Addr().String()),
		HTTPClient: &http.Client{
			Transport: &authRoundTripper{
				header: "Bearer secret-token-7",
			},
		},
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	cs, err := client.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)
	defer cs.Close()

	listRes, err := cs.ListTools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, listRes.Tools, 1)
	require.Equal(t, "fin_ping", listRes.Tools[0].Name)

	callRes, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "fin_ping",
		Arguments: map[string]any{
			"workspace_id": 1,
		},
	})
	require.NoError(t, err)
	require.False(t, callRes.IsError)
	require.Equal(t, int64(7), resolvedUserID)
}

func Test_Handler_AuthorizationVariants_Return401Generic(t *testing.T) {
	tests := []struct {
		name         string
		header       string
		expectStatus int
		expectCall   bool
	}{
		{
			name:         "missing authorization header",
			header:       "",
			expectStatus: fiber.StatusUnauthorized,
			expectCall:   false,
		},
		{
			name:         "basic auth scheme",
			header:       "Basic x",
			expectStatus: fiber.StatusUnauthorized,
			expectCall:   false,
		},
		{
			name:         "bearer keyword only",
			header:       "Bearer",
			expectStatus: fiber.StatusUnauthorized,
			expectCall:   false,
		},
		{
			name:         "bearer with whitespace only",
			header:       "Bearer   ",
			expectStatus: fiber.StatusUnauthorized,
			expectCall:   false,
		},
		{
			name:         "lowercase bearer with valid token",
			header:       "bearer valid-token",
			expectStatus: fiber.StatusOK,
			expectCall:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var verifierCalls int32
			res := &stubResolver{}
			auth := setupTestAuthz(res)
			r := mcpx.NewRegistrar(mcpx.Config{
				Authorizer: auth,
				Resolver:   res,
			})

			cfg := mcpx.HandlerConfig{
				Verifier: func(ctx context.Context, token string) (mcpx.VerifiedToken, error) {
					atomic.AddInt32(&verifierCalls, 1)
					if token == "valid-token" {
						return mcpx.VerifiedToken{UserID: 1}, nil
					}
					return mcpx.VerifiedToken{}, mcpx.ErrTokenInvalid
				},
				TokenSettingsURL: "https://app.example.com/core/access-tokens",
			}

			app := fiber.New()
			app.All("/mcp", mcpx.Handler(r, cfg))

			req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"ping","id":1}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, tt.expectStatus, resp.StatusCode)

			if tt.expectCall {
				require.Equal(t, int32(1), atomic.LoadInt32(&verifierCalls))
			} else {
				require.Equal(t, int32(0), atomic.LoadInt32(&verifierCalls))
				require.Equal(t, `Bearer error="invalid_token"`, resp.Header.Get("WWW-Authenticate"))

				bodyBytes, err := io.ReadAll(resp.Body)
				require.NoError(t, err)

				var errResp httpx.ErrorResponse
				err = json.Unmarshal(bodyBytes, &errResp)
				require.NoError(t, err)
				require.Equal(t, "UNAUTHORIZED", errResp.Error.Code)
				require.Equal(t, "invalid token", errResp.Error.Message)
			}
		})
	}
}

func Test_Handler_ExpiredToken_Returns401WithLink(t *testing.T) {
	res := &stubResolver{}
	auth := setupTestAuthz(res)
	r := mcpx.NewRegistrar(mcpx.Config{
		Authorizer: auth,
		Resolver:   res,
	})

	settingsURL := "https://app.example.com/core/access-tokens"
	expectedMsg := "Token expired. Create a new access token: " + settingsURL

	cfg := mcpx.HandlerConfig{
		Verifier: func(ctx context.Context, token string) (mcpx.VerifiedToken, error) {
			return mcpx.VerifiedToken{}, mcpx.ErrTokenExpired
		},
		TokenSettingsURL: settingsURL,
	}

	app := fiber.New()
	app.All("/mcp", mcpx.Handler(r, cfg))

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"ping","id":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer expired-token")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)

	expectedHeader := fmt.Sprintf(`Bearer error="invalid_token", error_description=%q`, expectedMsg)
	require.Equal(t, expectedHeader, resp.Header.Get("WWW-Authenticate"))

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var errResp httpx.ErrorResponse
	err = json.Unmarshal(bodyBytes, &errResp)
	require.NoError(t, err)
	require.Equal(t, "TOKEN_EXPIRED", errResp.Error.Code)
	require.Equal(t, expectedMsg, errResp.Error.Message)
}

func Test_Handler_InvalidAndUnknownVerifierErrors_Return401Generic(t *testing.T) {
	tests := []struct {
		name          string
		verifierErr   error
		expectLogMsg  string
		expectLogging bool
	}{
		{
			name:          "invalid token error",
			verifierErr:   mcpx.ErrTokenInvalid,
			expectLogging: false,
		},
		{
			name:          "unknown verifier error",
			verifierErr:   errors.New("db down"),
			expectLogMsg:  "db down",
			expectLogging: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuf, nil))

			res := &stubResolver{}
			auth := setupTestAuthz(res)
			r := mcpx.NewRegistrar(mcpx.Config{
				Authorizer: auth,
				Resolver:   res,
			})

			cfg := mcpx.HandlerConfig{
				Verifier: func(ctx context.Context, token string) (mcpx.VerifiedToken, error) {
					return mcpx.VerifiedToken{}, tt.verifierErr
				},
				TokenSettingsURL: "https://app.example.com/core/access-tokens",
				Logger:           logger,
			}

			app := fiber.New()
			app.All("/mcp", mcpx.Handler(r, cfg))

			req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"ping","id":1}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer some-token")

			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
			require.Equal(t, `Bearer error="invalid_token"`, resp.Header.Get("WWW-Authenticate"))

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var errResp httpx.ErrorResponse
			err = json.Unmarshal(bodyBytes, &errResp)
			require.NoError(t, err)
			require.Equal(t, "UNAUTHORIZED", errResp.Error.Code)
			require.Equal(t, "invalid token", errResp.Error.Message)

			if tt.expectLogging {
				require.Contains(t, logBuf.String(), tt.expectLogMsg)
			} else {
				require.Empty(t, logBuf.String())
			}
		})
	}
}

func Test_Handler_SessionCookieOnly_Returns401(t *testing.T) {
	var verifierCalls int32
	res := &stubResolver{}
	auth := setupTestAuthz(res)
	r := mcpx.NewRegistrar(mcpx.Config{
		Authorizer: auth,
		Resolver:   res,
	})

	cfg := mcpx.HandlerConfig{
		Verifier: func(ctx context.Context, token string) (mcpx.VerifiedToken, error) {
			atomic.AddInt32(&verifierCalls, 1)
			return mcpx.VerifiedToken{UserID: 1}, nil
		},
		TokenSettingsURL: "https://app.example.com/core/access-tokens",
	}

	app := fiber.New()
	app.All("/mcp", mcpx.Handler(r, cfg))

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"ping","id":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "omed_session",
		Value: "session-secret-x",
	})

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
	require.Equal(t, int32(0), atomic.LoadInt32(&verifierCalls))
	require.Equal(t, `Bearer error="invalid_token"`, resp.Header.Get("WWW-Authenticate"))

	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var errResp httpx.ErrorResponse
	err = json.Unmarshal(bodyBytes, &errResp)
	require.NoError(t, err)
	require.Equal(t, "UNAUTHORIZED", errResp.Error.Code)
	require.Equal(t, "invalid token", errResp.Error.Message)
}

func Test_Handler_NeverLogsToken(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))

	res := &stubResolver{}
	auth := setupTestAuthz(res)
	r := mcpx.NewRegistrar(mcpx.Config{
		Authorizer: auth,
		Resolver:   res,
	})

	secretToken := "secret-token-xyz-12345"

	cfg := mcpx.HandlerConfig{
		Verifier: func(ctx context.Context, token string) (mcpx.VerifiedToken, error) {
			return mcpx.VerifiedToken{}, errors.New("db error")
		},
		TokenSettingsURL: "https://app.example.com/core/access-tokens",
		Logger:           logger,
	}

	app := fiber.New()
	app.All("/mcp", mcpx.Handler(r, cfg))

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"ping","id":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+secretToken)

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	logs := logBuf.String()
	require.NotEmpty(t, logs)
	require.NotContains(t, logs, secretToken)
	require.NotContains(t, logs, "Authorization")
}
