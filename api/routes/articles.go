package routes

import "github.com/gofiber/fiber/v2"

func ArticleRoutes(app *fiber.App){
	group := app.Group("/articles");

	group.Post("/", func(c *fiber.Ctx) error {
		return c.SendString("Article Create")
	})
	
	group.Get("/:id", func(c *fiber.Ctx) error {
		return c.SendString("Article Get id: " + c.Params("id"))
	})

	group.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Article List")
	})

	group.Put("/:id", func(c *fiber.Ctx) error {
		return c.SendString("Article update " + c.Params("id"))
	})

	group.Delete("/:id", func(c *fiber.Ctx) error {
		return c.SendString("Article delete " + c.Params("id"))
	})
	
}
