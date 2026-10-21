package cms

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/cms/controller"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/platform/authz"
)

func initRoutes(mod CmsModule, r fiber.Router, az *authz.Authorizer) {
	media := r.Group("/workspaces/:wid/cms/media")
	mediaCtl := controller.NewMediaController(mod.MediaService)

	media.Post("/search", az.Require(domain.ObjMedia, "read"), mediaCtl.Search)
	media.Post("/", az.Require(domain.ObjMedia, "create"), mediaCtl.Upload)
	media.Get("/:id", az.Require(domain.ObjMedia, "read"), mediaCtl.GetByID)
	media.Put("/:id", az.Require(domain.ObjMedia, "update"), mediaCtl.Update)
	media.Delete("/:id", az.Require(domain.ObjMedia, "delete"), mediaCtl.Delete)
}
