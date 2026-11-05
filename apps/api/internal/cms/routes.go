package cms

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/cms/controller"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/platform/authz"
)

func initRoutes(mod CmsModule, r fiber.Router, az *authz.Authorizer) {
	settings := r.Group("/workspaces/:wid/cms/settings")
	settingCtl := controller.NewSettingController(mod.SettingService)

	settings.Get("/", az.Require(domain.ObjSetting, "read"), settingCtl.Get)
	settings.Put("/", az.Require(domain.ObjSetting, "update"), settingCtl.Update)

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
	posts.Put("/:id/translations/:locale", az.Require(domain.ObjPost, "update"), postCtl.SaveTranslation)
	posts.Delete("/:id/translations/:locale", az.Require(domain.ObjPost, "delete"), postCtl.DeleteTranslation)
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
	pages.Put("/:id/translations/:locale", az.Require(domain.ObjPage, "update"), pageCtl.SaveTranslation)
	pages.Delete("/:id/translations/:locale", az.Require(domain.ObjPage, "delete"), pageCtl.DeleteTranslation)
	pages.Delete("/:id", az.Require(domain.ObjPage, "delete"), pageCtl.Delete)
	pages.Post("/:id/publish", az.Require(domain.ObjPage, "update"), pageCtl.Publish)
	pages.Post("/:id/schedule", az.Require(domain.ObjPage, "update"), pageCtl.Schedule)
	pages.Post("/:id/unpublish", az.Require(domain.ObjPage, "update"), pageCtl.Unpublish)

	publicCtl := controller.NewPublicController(mod.PostService, mod.PageService, mod.MediaService, mod.cfg.ApiURL)
	r.Get("/public/workspaces/:wid/cms/posts", publicCtl.ListPosts)
	r.Get("/public/workspaces/:wid/cms/posts/:slug", publicCtl.GetPostBySlug)
	r.Get("/public/workspaces/:wid/cms/:locale/posts", publicCtl.ListPostsWithLocale)
	r.Get("/public/workspaces/:wid/cms/:locale/posts/:slug", publicCtl.GetPostBySlugWithLocale)
	r.Get("/public/workspaces/:wid/cms/pages", publicCtl.ListPages)
	r.Get("/public/workspaces/:wid/cms/pages/:slug", publicCtl.GetPageBySlug)
	r.Get("/public/workspaces/:wid/cms/:locale/pages", publicCtl.ListPagesWithLocale)
	r.Get("/public/workspaces/:wid/cms/:locale/pages/:slug", publicCtl.GetPageBySlugWithLocale)
	r.Get("/public/workspaces/:wid/cms/media/:id", publicCtl.RedirectMedia)
}
