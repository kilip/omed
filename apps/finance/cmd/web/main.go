package main

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/config"
)

// @title						Omed Finance API
// @version						1.0
// @description					Finance service for Omed
// @BasePath					/
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
func main() {
	cfg := config.GetConfig()
	db := config.GetDB(cfg)
	logger := config.GetLogger(cfg)
	app := config.GetFiber(cfg, logger)
	enforcer := config.GetEnforcer()

	config.Bootstrap(config.BootstrapConfig{
		Config:   cfg,
		DB:       db,
		Fiber:    app,
		Logger:   logger,
		Enforcer: enforcer,
	})

	listen := fmt.Sprintf("%v:%v", "", cfg.App.Port)
	log.Fatal(app.Listen(listen, fiber.ListenConfig{
		EnablePrefork: false,
	}))
}
