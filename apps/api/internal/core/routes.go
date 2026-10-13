package core

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/core/controller"
)

func initRoutes(mod CoreModule, r fiber.Router) {
	ctlU := controller.NewUserController(mod.UserService)
	r.Get("/:id", ctlU.GetByID)
	r.Post("/search", ctlU.Search)
	r.Post("/", ctlU.Create)
	r.Put("/:id", ctlU.Update)
	r.Delete(":id", ctlU.Delete)
}
