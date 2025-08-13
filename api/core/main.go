package core

import (
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"github.com/kilip/omed/api/routes"
)

func Bootstrap() *fiber.App {
	godotenv.Load()

	app := fiber.New()

	routes.Configure(app)
	
	return app
}
