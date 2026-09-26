package testutil

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/config"
)

var cfg config.Config
var state config.State
var fiberApp *fiber.App
var logger *slog.Logger
var jwksMock *JWKSMock

func init() {
	jwksMock = NewJWKSMock()
	cfg = config.GetConfig()
	logger = config.GetLogger(cfg)
	fiberApp = config.GetFiber(cfg, logger)
	state = config.State{
		Config: cfg,
		Fiber:  fiberApp,
	}

	config.Bootstrap(state)
}
