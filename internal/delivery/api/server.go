package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/internal/utils"
)

func NewServer(conf utils.Config) *fiber.App {
	app := fiber.New(fiber.Config{
		Prefork: true,
	})
	return app
}
