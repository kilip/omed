package core

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/core/controller"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/platform/authz"
)

func initRoutes(mod CoreModule, r fiber.Router, az *authz.Authorizer) {
	// Authentication routes (Better Auth compatible)
	auth := r.Group("/auth")
	cookieSecure := strings.HasPrefix(mod.cfg.ApiUrl, "https://") || mod.cfg.Env == "production"
	ctlA := controller.NewAuthController(
		mod.OAuthService,
		mod.SessionService,
		controller.WithCookieName(mod.cfg.SessionCookieName),
		controller.WithCookieSecure(cookieSecure),
	)
	auth.Post("/sign-in/social", ctlA.SignInSocial)
	auth.Get("/callback/:id", ctlA.Callback)
	auth.Post("/callback/:id", ctlA.Callback)
	auth.Get("/get-session", ctlA.GetSession)
	auth.Post("/sign-out", ctlA.SignOut)

	users := r.Group("/users")
	ctlU := controller.NewUserController(mod.UserService)
	ctlT := controller.NewAccessTokenController(mod.AccessTokenService)
	users.Get("/me", ctlU.GetMe)
	users.Put("/me", ctlU.UpdateMe)
	users.Post("/me/access-tokens", ctlT.Create)
	users.Get("/me/access-tokens", ctlT.List)
	users.Delete("/me/access-tokens/:id", ctlT.Revoke)

	workspaces := r.Group("/workspaces")
	ctlW := controller.NewWorkspaceController(mod.WorkspaceService)
	ctlM := controller.NewMemberController(mod.MemberService, az)
	ctlI := controller.NewInvitationController(mod.InvitationService)
	ctlF := controller.NewFileController(mod.FileService)

	// Workspaces collection
	workspaces.Post("/search", ctlW.Search)
	workspaces.Post("/", ctlW.Create)

	// Workspaces single resource
	workspaces.Get("/:wid", az.Require(domain.ObjWorkspace, "read"), ctlW.GetByID)
	workspaces.Put("/:wid", az.Require(domain.ObjWorkspace, "update"), ctlW.Update)
	workspaces.Delete("/:wid", az.RequireOwner(), ctlW.Delete)

	// Workspace permissions for the current user
	workspaces.Get("/:wid/permissions", az.Member(), ctlM.Permissions)

	// Roles that can be assigned in this workspace
	workspaces.Get("/:wid/roles", az.Member(), ctlM.Roles)

	// Workspace members
	workspaces.Post("/:wid/members/search", az.Require(domain.ObjMember, "read"), ctlM.Search)
	workspaces.Post("/:wid/members", az.Require(domain.ObjMember, "create"), ctlM.Invite)
	workspaces.Put("/:wid/members/:id", az.Require(domain.ObjMember, "update"), ctlM.Update)
	workspaces.Delete("/:wid/members/:id", az.Require(domain.ObjMember, "delete"), ctlM.Delete)

	// Workspace invitations
	workspaces.Post("/:wid/invitations/search", az.Require(domain.ObjInvitation, "read"), ctlI.Search)
	workspaces.Post("/:wid/invitations", az.Require(domain.ObjInvitation, "create"), ctlI.Create)
	workspaces.Post("/:wid/invitations/:id/send", az.Require(domain.ObjInvitation, "create"), ctlI.Send)
	workspaces.Delete("/:wid/invitations/:id", az.Require(domain.ObjInvitation, "delete"), ctlI.Revoke)

	// Workspace files
	workspaces.Post("/:wid/files/search", az.Require(domain.ObjFile, "read"), ctlF.Search)
	workspaces.Post("/:wid/files", az.Require(domain.ObjFile, "write"), ctlF.Upload)
	workspaces.Get("/:wid/files/:id/download", az.Require(domain.ObjFile, "read"), ctlF.Download)
	workspaces.Get("/:wid/files/:id/presign", az.Require(domain.ObjFile, "read"), ctlF.Presign)
	workspaces.Get("/:wid/files/:id", az.Require(domain.ObjFile, "read"), ctlF.GetByID)
	workspaces.Put("/:wid/files/:id", az.Require(domain.ObjFile, "write"), ctlF.Update)
	workspaces.Delete("/:wid/files/:id", az.Require(domain.ObjFile, "write"), ctlF.Delete)

	// Public & join invitation routes
	invitations := r.Group("/invitations")
	invitations.Get("/:token", ctlI.GetPublic)
	invitations.Post("/:token/accept", ctlI.Accept)
}
