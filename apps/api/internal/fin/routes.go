package fin

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/fin/controller"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/platform/authz"
)

func initRoutes(mod FinModule, r fiber.Router, az *authz.Authorizer) {
	accounts := r.Group("/workspaces/:wid/fin/accounts")
	ctl := controller.NewAccountController(mod.AccountService)

	accounts.Post("/search", az.Require(domain.ObjAccount, "read"), ctl.Search)
	accounts.Get("/tree", az.Require(domain.ObjAccount, "read"), ctl.GetTree)
	accounts.Post("/", az.Require(domain.ObjAccount, "create"), ctl.Create)
	accounts.Get("/:id", az.Require(domain.ObjAccount, "read"), ctl.GetByID)
	accounts.Put("/:id", az.Require(domain.ObjAccount, "update"), ctl.Update)
	accounts.Delete("/:id", az.Require(domain.ObjAccount, "delete"), ctl.Delete)
}
