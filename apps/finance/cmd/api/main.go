package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/config"
)

func main() {
	cfg := config.GetConfig()
	logger := config.GetLogger()
	entClient := config.GetEntClient(cfg)
	api := config.GetFiber(logger)

	state := config.State{
		Config:    cfg,
		Api:       api,
		Log:       logger,
		EntClient: entClient,
	}

	config.WaitForJWKS(state)

	config.Bootstrap(state)

	log.Fatal(api.Listen(fmt.Sprintf(":%d", cfg.Port), fiber.ListenConfig{
		EnablePrefork: true,
	}))
}
