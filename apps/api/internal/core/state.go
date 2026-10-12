package core

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
)

type State struct {
	Config   Config
	FiberApp *fiber.App
	Logger   *slog.Logger
	EntCli   *ent.Client
	Public   fiber.Router

	// Secured route group
	Secure fiber.Router
}
