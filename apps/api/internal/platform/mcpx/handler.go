package mcpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// VerifiedToken represents the verified identity and expiry of an access token.
type VerifiedToken struct {
	UserID    int64
	ExpiresAt time.Time
}

// TokenVerifier validates a raw token string and returns the verified token identity.
type TokenVerifier func(ctx context.Context, token string) (VerifiedToken, error)

// HandlerConfig configures the MCP Fiber HTTP handler.
type HandlerConfig struct {
	Verifier         TokenVerifier
	TokenSettingsURL string
	Logger           *slog.Logger
}

type verifiedTokenContextKey struct{}

// Handler returns a Fiber handler serving the MCP transport over HTTP with Bearer token authentication.
func Handler(r *Registrar, cfg HandlerConfig) fiber.Handler {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	streamableHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return r.Server()
	}, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
	})

	netHandler := auth.RequireBearerToken(func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
		vt, ok := req.Context().Value(verifiedTokenContextKey{}).(VerifiedToken)
		if !ok {
			return nil, auth.ErrInvalidToken
		}
		return &auth.TokenInfo{
			UserID:     strconv.FormatInt(vt.UserID, 10),
			Expiration: vt.ExpiresAt,
		}, nil
	}, &auth.RequireBearerTokenOptions{
		AllowMissingExpiration: true,
	})(streamableHandler)

	adapted := adaptor.HTTPHandler(netHandler)

	return func(c fiber.Ctx) error {
		authHeader := c.Get(fiber.HeaderAuthorization)
		if authHeader == "" {
			return sendGenericUnauthorized(c)
		}

		fields := strings.Fields(authHeader)
		if len(fields) != 2 || !strings.EqualFold(fields[0], "bearer") {
			return sendGenericUnauthorized(c)
		}

		token := fields[1]
		if token == "" {
			return sendGenericUnauthorized(c)
		}

		vt, err := cfg.Verifier(c.Context(), token)
		if err != nil {
			if errors.Is(err, ErrTokenExpired) {
				msg := "Token expired. Create a new access token: " + cfg.TokenSettingsURL
				c.Set(fiber.HeaderWWWAuthenticate, fmt.Sprintf(`Bearer error="invalid_token", error_description=%q`, msg))
				return c.Status(fiber.StatusUnauthorized).JSON(httpx.ErrorResponse{
					Error: httpx.ErrorBody{
						Code:    ErrTokenExpired.Error(),
						Message: msg,
					},
					Meta: httpx.NewMeta(c, ""),
				})
			}

			if errors.Is(err, ErrTokenInvalid) {
				return sendGenericUnauthorized(c)
			}

			logger.ErrorContext(c.Context(), "token verification failed", "error", err)
			return sendGenericUnauthorized(c)
		}

		if vt.UserID <= 0 {
			return sendGenericUnauthorized(c)
		}

		c.Locals(verifiedTokenContextKey{}, vt)
		return adapted(c)
	}
}

func sendGenericUnauthorized(c fiber.Ctx) error {
	c.Set(fiber.HeaderWWWAuthenticate, `Bearer error="invalid_token"`)
	return c.Status(fiber.StatusUnauthorized).JSON(httpx.ErrorResponse{
		Error: httpx.ErrorBody{
			Code:    base.ErrUnauthorized.Error(),
			Message: "invalid token",
		},
		Meta: httpx.NewMeta(c, ""),
	})
}
