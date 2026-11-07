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
	providers.Post("/models", az.Require(domain.ObjProvider, "create"), ctl.ListModelsConfig)
	providers.Get("/:id", az.Require(domain.ObjProvider, "read"), ctl.Get)
	providers.Patch("/:id", az.Require(domain.ObjProvider, "update"), ctl.Update)
	providers.Delete("/:id", az.Require(domain.ObjProvider, "delete"), ctl.Delete)
	providers.Post("/:id/default", az.Require(domain.ObjProvider, "update"), ctl.SetDefault)
	providers.Post("/:id/test", az.Require(domain.ObjProvider, "update"), ctl.Test)
	providers.Get("/:id/models", az.Require(domain.ObjProvider, "read"), ctl.ListModels)

	settings := r.Group("/workspaces/:wid/agent/settings")
	settingCtl := controller.NewSettingController(mod.SettingService)

	settings.Get("/", az.Require(domain.ObjSetting, "read"), settingCtl.Get)
	settings.Patch("/", az.Require(domain.ObjSetting, "update"), settingCtl.Update)

	financial := r.Group("/workspaces/:wid/agent/financial")
	financialCtl := controller.NewFinancialController(mod.FinancialService, mod.cfg.AttachmentMaxSizeMB, mod.cfg.AttachmentAllowedTypes)

	financial.Get("/status", az.Require(domain.ObjFinancial, "use"), az.Require("fin:transaction", "create"), financialCtl.Status)
	financial.Post("/analyze", az.Require(domain.ObjFinancial, "use"), az.Require("fin:transaction", "create"), financialCtl.Analyze)

	chat := r.Group("/workspaces/:wid/agent/chat", az.Require(domain.ObjFinancial, "use"), az.Require("fin:transaction", "create"))
	chatCtl := controller.NewChatController(mod.ChatService, mod.cfg.AttachmentMaxSizeMB, mod.cfg.AttachmentAllowedTypes)

	chat.Get("/conversations", chatCtl.ListConversations)
	chat.Post("/conversations", chatCtl.CreateConversation)
	chat.Get("/conversations/:id", chatCtl.GetConversation)
	chat.Delete("/conversations/:id", chatCtl.DeleteConversation)
	chat.Patch("/conversations/:id", chatCtl.UpdateConversation)
	chat.Post("/conversations/:id/messages", chatCtl.SendMessage)
	chat.Post("/messages/:id/confirm", chatCtl.ConfirmDraft)
	chat.Post("/messages/:id/discard", chatCtl.DiscardDraft)
	chat.Get("/attachments/:id/download", chatCtl.DownloadAttachment)
}
