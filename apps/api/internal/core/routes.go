package core

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/core/controller"
)

func initRoutes(mod CoreModule, r fiber.Router) {
	users := r.Group("/users")
	ctlU := controller.NewUserController(mod.UserService)
	users.Get("/:id", ctlU.GetByID)
	users.Post("/search", ctlU.Search)
	users.Post("/", ctlU.Create)
	users.Put("/:id", ctlU.Update)
	users.Delete("/:id", ctlU.Delete)

	workspaces := r.Group("/workspaces")
	ctlW := controller.NewWorkspaceController(mod.WorkspaceService)
	workspaces.Get("/:id", ctlW.GetByID)
	workspaces.Post("/search", ctlW.Search)
	workspaces.Post("/", ctlW.Create)
	workspaces.Put("/:id", ctlW.Update)
	workspaces.Delete("/:id", ctlW.Delete)
}
