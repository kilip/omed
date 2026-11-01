package config

import (
	"context"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/service"
	"github.com/kilip/omed/api/internal/platform/mcpx"
)

func newMCPHandler(state State, reg *mcpx.Registrar, tokens service.AccessTokenService) fiber.Handler {
	verifier := func(ctx context.Context, token string) (mcpx.VerifiedToken, error) {
		tok, err := tokens.Verify(ctx, token)
		if err != nil {
			if errors.Is(err, domain.ErrTokenExpired) {
				return mcpx.VerifiedToken{}, mcpx.ErrTokenExpired
			}
			return mcpx.VerifiedToken{}, mcpx.ErrTokenInvalid
		}
		return mcpx.VerifiedToken{
			UserID:    tok.UserID,
			ExpiresAt: tok.ExpiresAt,
		}, nil
	}

	tokenURL := strings.TrimRight(state.Config.WebUrl, "/") + "/core/access-tokens"

	return mcpx.Handler(reg, mcpx.HandlerConfig{
		Verifier:         verifier,
		TokenSettingsURL: tokenURL,
		Logger:           state.Logger,
	})
}
