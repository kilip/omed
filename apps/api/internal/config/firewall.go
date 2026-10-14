package config

import (
	"github.com/kilip/omed/api/internal/core/middleware"
	"github.com/kilip/omed/api/internal/core/service"
)

func initFirewall(state State, sessions service.SessionService) {
	app := state.FiberApp
	cfg := state.Config

	app.Use(middleware.NewAuthMiddleware(middleware.AuthConfig{
		SessionService: sessions,
		Logger:         state.Logger,
		CookieName:     cfg.SessionCookieName,
	}))
}
