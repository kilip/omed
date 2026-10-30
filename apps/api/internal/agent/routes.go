package agent

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/agent/controller"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/platform/authz"
)

func initRoutes(mod AgentModule, r fiber.Router, az *authz.Authorizer) {
	providers := r.Group("/workspaces/:wid/agent/providers")
	ctl := controller.NewProviderController(mod.ProviderService)

	providers.Get("/", az.Require(domain.ObjProvider, "read"), ctl.List)
	providers.Post("/", az.Require(domain.ObjProvider, "create"), ctl.Create)
	providers.Post("/test", az.Require(domain.ObjProvider, "create"), ctl.TestConfig)
	providers.Get("/:id", az.Require(domain.ObjProvider, "read"), ctl.Get)
	providers.Patch("/:id", az.Require(domain.ObjProvider, "update"), ctl.Update)
	providers.Delete("/:id", az.Require(domain.ObjProvider, "delete"), ctl.Delete)
	providers.Post("/:id/default", az.Require(domain.ObjProvider, "update"), ctl.SetDefault)
	providers.Post("/:id/test", az.Require(domain.ObjProvider, "update"), ctl.Test)
}
