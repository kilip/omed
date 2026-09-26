package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/config"
)

func main() {
	cfg := config.GetConfig()
	logger := config.GetLogger(cfg)
	api := config.GetFiber(cfg, logger)

	state := config.State{
		Fiber:  api,
		Logger: logger,
		Config: cfg,
	}

	config.Bootstrap(state)

	listenConfig := fiber.ListenConfig{EnablePrefork: true}
	if err := api.Listen(fmt.Sprintf(":%d", cfg.Port), listenConfig); err != nil {
		log.Fatalf("Can't start fiber: %s", err)
	}
}
