package core

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/controller"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/platform/authz"
)

func initRoutes(mod CoreModule, r fiber.Router, az *authz.Authorizer) {
	// Authentication routes (Better Auth compatible)
	auth := r.Group("/auth")
	ctlA := controller.NewAuthController(mod.OAuthService, mod.SessionService)
	auth.Post("/sign-in/social", ctlA.SignInSocial)
	auth.Get("/callback/:id", ctlA.Callback)
	auth.Post("/callback/:id", ctlA.Callback)
	auth.Get("/get-session", ctlA.GetSession)
	auth.Post("/sign-out", ctlA.SignOut)

	// Global users management is restricted to superadmin.
	users := r.Group("/users", authz.RequireGlobal(base.UserRoleAdmin))
	ctlU := controller.NewUserController(mod.UserService)
	users.Get("/:id", ctlU.GetByID)
	users.Post("/search", ctlU.Search)
	users.Post("/", ctlU.Create)
	users.Put("/:id", ctlU.Update)
	users.Delete("/:id", ctlU.Delete)

	workspaces := r.Group("/workspaces")
	ctlW := controller.NewWorkspaceController(mod.WorkspaceService)
	ctlM := controller.NewMemberController(mod.MemberService, az)
	ctlI := controller.NewInvitationController(mod.InvitationService)

	// Workspaces collection
	workspaces.Post("/search", ctlW.Search)
	workspaces.Post("/", ctlW.Create)

	// Workspaces single resource
	workspaces.Get("/:wid", az.Require(domain.ObjWorkspace, "read"), ctlW.GetByID)
	workspaces.Put("/:wid", az.Require(domain.ObjWorkspace, "update"), ctlW.Update)
	workspaces.Delete("/:wid", az.RequireOwner(), ctlW.Delete)

	// Workspace permissions for the current user
	workspaces.Get("/:wid/permissions", az.Member(), ctlM.Permissions)

	// Workspace members
	workspaces.Post("/:wid/members/search", az.Require(domain.ObjMember, "read"), ctlM.Search)
	workspaces.Post("/:wid/members", az.Require(domain.ObjMember, "create"), ctlM.Invite)
	workspaces.Put("/:wid/members/:id", az.Require(domain.ObjMember, "update"), ctlM.Update)
	workspaces.Delete("/:wid/members/:id", az.Require(domain.ObjMember, "delete"), ctlM.Delete)

	// Workspace invitations
	workspaces.Post("/:wid/invitations/search", az.Require(domain.ObjInvitation, "read"), ctlI.Search)
	workspaces.Post("/:wid/invitations", az.Require(domain.ObjInvitation, "create"), ctlI.Create)
	workspaces.Delete("/:wid/invitations/:id", az.Require(domain.ObjInvitation, "delete"), ctlI.Revoke)

	// Public & join invitation routes
	invitations := r.Group("/invitations")
	invitations.Get("/:token", ctlI.GetPublic)
	invitations.Post("/:token/accept", ctlI.Accept)
}
