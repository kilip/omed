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

	posts := r.Group("/workspaces/:wid/cms/posts")
	postCtl := controller.NewPostController(mod.PostService)

	posts.Post("/search", az.Require(domain.ObjPost, "read"), postCtl.Search)
	posts.Post("/", az.Require(domain.ObjPost, "create"), postCtl.Create)
	posts.Get("/:id", az.Require(domain.ObjPost, "read"), postCtl.GetByID)
	posts.Put("/:id", az.Require(domain.ObjPost, "update"), postCtl.Update)
	posts.Delete("/:id", az.Require(domain.ObjPost, "delete"), postCtl.Delete)
	posts.Post("/:id/publish", az.Require(domain.ObjPost, "update"), postCtl.Publish)
	posts.Post("/:id/schedule", az.Require(domain.ObjPost, "update"), postCtl.Schedule)
	posts.Post("/:id/unpublish", az.Require(domain.ObjPost, "update"), postCtl.Unpublish)

	pages := r.Group("/workspaces/:wid/cms/pages")
	pageCtl := controller.NewPageController(mod.PageService)

	pages.Post("/search", az.Require(domain.ObjPage, "read"), pageCtl.Search)
	pages.Post("/", az.Require(domain.ObjPage, "create"), pageCtl.Create)
	pages.Get("/:id", az.Require(domain.ObjPage, "read"), pageCtl.GetByID)
	pages.Put("/:id", az.Require(domain.ObjPage, "update"), pageCtl.Update)
	pages.Delete("/:id", az.Require(domain.ObjPage, "delete"), pageCtl.Delete)
	pages.Post("/:id/publish", az.Require(domain.ObjPage, "update"), pageCtl.Publish)
	pages.Post("/:id/schedule", az.Require(domain.ObjPage, "update"), pageCtl.Schedule)
	pages.Post("/:id/unpublish", az.Require(domain.ObjPage, "update"), pageCtl.Unpublish)
}
