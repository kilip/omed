package config

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/kilip/omed/finance/ent"
	slogfiber "github.com/samber/slog-fiber"
)

type State struct {
	FiberApp  *fiber.App
	Config    Config
	Log       *slog.Logger
	EntClient *ent.Client
}

func initFiber(state State) {
	app := state.FiberApp
	app.Use(slogfiber.New(state.Log))
	app.Use(requestid.New())

	// init healthcheck
	// Use the default probe on the conventional endpoints
	app.Use(healthcheck.New(healthcheck.Config{
		ResponseFormat: healthcheck.FormatJSON,
	}))

	// cors config
	app.Use(cors.New(cors.Config{
		AllowOrigins: state.Config.TrustedOrigins,
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
	}))
}

func Bootstrap(state State) {
	initFiber(state)
}
