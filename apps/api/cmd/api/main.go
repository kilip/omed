// Package main provides the entry point for starting the Omed API HTTP service.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kilip/omed/api/internal/config"
)

// main initializes configuration, logging, fiber app state, and starts the server.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.GetConfig()
	logger := config.GetLogger()
	fiber := config.GetFiber(cfg, logger)
	entcli := config.GetEntClient(cfg)

	state := config.State{
		Logger:   logger,
		Config:   cfg,
		FiberApp: fiber,
		Public:   fiber.Group("/"),
		Secure:   fiber.Group("/"),
		EntCli:   entcli,
	}

	omed := config.New(state)

	if err := omed.Run(ctx); err != nil {
		slog.Error("app stopped", "err", err)
		os.Exit(1)
	}
}
