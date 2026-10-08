package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kilip/omed/finance/internal/config"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.GetConfig()
	logger := config.GetLogger(cfg)

	config.WaitForJWKS(cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fiber := config.GetFiber(cfg, logger)
	entClient := config.GetEntClient(cfg)

	state := config.State{FiberApp: fiber, Config: cfg, Log: logger, EntClient: entClient}

	config.Bootstrap(state)
	if err := config.StartEventConsumer(ctx, state); err != nil {
		log.Fatal(err)
	}

	go func() { <-ctx.Done(); _ = fiber.Shutdown() }()
	log.Fatal(fiber.Listen(fmt.Sprintf(":%d", cfg.Port)))
}
