package config

import (
	"context"
	"log/slog"

	"github.com/MicahParks/keyfunc/v3"
	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/http/middleware"
	"github.com/kilip/omed/finance/internal/repository"
	"github.com/kilip/omed/finance/internal/service"
)

// State the state of application
type State struct {
	Config    Config
	Api       *fiber.App
	Log       *slog.Logger
	EntClient *ent.Client
}

func configureMiddleware(state State) {
	jwks, err := keyfunc.NewDefaultCtx(context.Background(), []string{state.Config.JWKSUrl})
	if err != nil {
		state.Log.Warn("Warning: Initial JWKS fetch failed, retrying in background", "error", err.Error())
	}
	state.Api.Use(jwtware.New(jwtware.Config{
		KeyFunc:   jwks.KeyfuncCtx(context.Background()),
		Extractor: extractors.FromAuthHeader("Bearer"),
		ErrorHandler: func(c fiber.Ctx, err error) error {
			state.Log.WarnContext(c, "JWKS Error", "error", err.Error())
			return err
		},
	}))
	state.Api.Use(middleware.UserInjector)

	users := repository.NewUserRepository(state.EntClient, state.Log)
	works := repository.NewWorkspaceRepository(state.EntClient, state.Log)
	authSvc := service.NewAuthService(users, works, state.Log)
	state.Api.Use(middleware.AuthSnapshotMiddleware(authSvc))
}

func Bootstrap(state State) {
	configureMiddleware(state)
}
