package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kilip/omed/api/internal/app"
	"github.com/kilip/omed/api/internal/core"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := core.GetConfig()
	logger := app.GetLogger()
	fiber := app.GetFiber(cfg, logger)

	state := core.State{
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
