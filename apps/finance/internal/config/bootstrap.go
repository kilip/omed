package config

import (
	"context"
	"errors"
	"log/slog"

	"github.com/MicahParks/keyfunc/v3"
	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/kilip/omed/finance/internal/http"
	"github.com/kilip/omed/finance/internal/http/middleware"
	slogfiber "github.com/samber/slog-fiber"
)

var ErrValueNotFound = errors.New("value not found")

type State struct {
	Fiber  *fiber.App
	Config Config
	Logger *slog.Logger
}

func initAuthenticatedEndpoint(state State) {
	jwks, err := keyfunc.NewDefaultCtx(context.Background(), []string{state.Config.JWKSUrl})
	if err != nil {
		state.Logger.Warn("Warning: Initial JWKS fetch failed, retrying in background", "error", err.Error())
	}
	state.Fiber.Use(jwtware.New(jwtware.Config{
		KeyFunc:   jwks.KeyfuncCtx(context.Background()),
		Extractor: extractors.FromAuthHeader("Bearer"),
		ErrorHandler: func(c fiber.Ctx, err error) error {
			state.Logger.WarnContext(c, "JWKS Error", "error", err.Error())
			if err.Error() == "value not found" {
				return errors.Join(http.ErrInvalidToken, err)
			}
			return err
		},
	}))

	state.Fiber.Use(middleware.UserInjector)

	state.Fiber.Get("/ping", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"hello": "world",
			"user":  http.GetUser(c),
		})
	})
}

func Bootstrap(state State) {
	state.Fiber.Use(slogfiber.New(state.Logger))
	state.Fiber.Use(requestid.New())
	initAuthenticatedEndpoint(state)
}
