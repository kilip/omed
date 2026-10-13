// Package main provides the entry point for starting the Omed API HTTP service.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kilip/omed/api/internal/app"
	"github.com/kilip/omed/api/internal/base"
)

// main initializes configuration, logging, fiber app state, and starts the server.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := base.GetConfig()
	logger := app.GetLogger()
	fiber := app.GetFiber(cfg, logger)

	state := base.State{
		Logger:   logger,
		Config:   cfg,
		FiberApp: fiber,
		Public:   fiber.Group("/"),
		Secure:   fiber.Group("/"),
	}

	app.WaitForJWKS(cfg, logger)

	a := app.New(state)

	if err := a.Run(ctx); err != nil {
		slog.Error("app stopped", "err", err)
		os.Exit(1)
	}
}
