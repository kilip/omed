package fin

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/fin/controller"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/platform/authz"
)

func initRoutes(mod FinModule, r fiber.Router, az *authz.Authorizer) {
	accounts := r.Group("/workspaces/:wid/fin/accounts")
	accountCtl := controller.NewAccountController(mod.AccountService)

	accounts.Post("/search", az.Require(domain.ObjAccount, "read"), accountCtl.Search)
	accounts.Post("/seed", az.Require(domain.ObjAccount, "create"), accountCtl.Seed)
	accounts.Post("/", az.Require(domain.ObjAccount, "create"), accountCtl.Create)
	accounts.Get("/:id", az.Require(domain.ObjAccount, "read"), accountCtl.GetByID)
	accounts.Put("/:id", az.Require(domain.ObjAccount, "update"), accountCtl.Update)
	accounts.Delete("/:id", az.Require(domain.ObjAccount, "delete"), accountCtl.Delete)

	periods := r.Group("/workspaces/:wid/fin/periods")
	periodCtl := controller.NewPeriodController(mod.PeriodService)

	periods.Post("/search", az.Require(domain.ObjPeriod, "read"), periodCtl.Search)
	periods.Post("/", az.Require(domain.ObjPeriod, "create"), periodCtl.Create)
	periods.Get("/:id", az.Require(domain.ObjPeriod, "read"), periodCtl.GetByID)
	periods.Put("/:id", az.Require(domain.ObjPeriod, "update"), periodCtl.Update)
	periods.Delete("/:id", az.Require(domain.ObjPeriod, "delete"), periodCtl.Delete)
	periods.Post("/:id/close", az.Require(domain.ObjPeriod, "update"), periodCtl.Close)
	periods.Post("/:id/reopen", az.Require(domain.ObjPeriod, "update"), periodCtl.Reopen)
}
