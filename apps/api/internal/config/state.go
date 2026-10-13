package config

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/platform/authz"
)

// State holds the shared dependencies and runtime components accessible across modules.
type State struct {
	// Config represents the application configuration.
	Config Config
	// FiberApp is the root Fiber HTTP application instance.
	FiberApp *fiber.App
	// Logger is the application-wide structured logger.
	Logger *slog.Logger
	// EntCli is the database ORM client.
	EntCli *ent.Client
	// Public is the unauthenticated route group router.
	Public fiber.Router
	// Secure is the authenticated route group router.
	Secure fiber.Router
	// Authz is the workspace authorization engine.
	Authz *authz.Authorizer
}
