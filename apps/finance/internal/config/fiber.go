package config

import "github.com/gofiber/fiber/v3"

func GetFiber() *fiber.App {
	app := fiber.New()

	return app
}
