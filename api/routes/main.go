package routes

import "github.com/gofiber/fiber/v2"

func Configure(app *fiber.App){
	ArticleRoutes(app)
}
