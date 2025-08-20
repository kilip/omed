package api

import "github.com/gofiber/fiber/v2"

type Controller interface {
	LoadRoutes(app *fiber.App)
}
