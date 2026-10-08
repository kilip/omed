package main

import (
	"fmt"
	"log"

	"github.com/kilip/omed/finance/internal/config"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.GetConfig()
	logger := config.GetLogger(cfg)

	// wait for jwks before we continue
	config.WaitForJWKS(cfg, logger)

	fiber := config.GetFiber(cfg, logger)
	entClient := config.GetEntClient(cfg)

	state := config.State{
		FiberApp:  fiber,
		Config:    cfg,
		Log:       logger,
		EntClient: entClient,
	}

	config.Bootstrap(state)
	slisten := fmt.Sprintf(":%d", cfg.Port)
	log.Fatal(fiber.Listen(slisten))
}
