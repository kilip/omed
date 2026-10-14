// Package core provides user and workspace management endpoints, domain logic, and data persistence.
package core

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/core/repository"
	"github.com/kilip/omed/api/internal/core/service"
	"github.com/kilip/omed/api/internal/platform/authz"
)

type Config struct {
	FiberApp *fiber.App
	Logger   *slog.Logger
	EntCli   *ent.Client
}

// CoreModule represents the core module implementing base.Module.
type CoreModule struct {
	cfg Config
	// UserService handles user business logic.
	UserService service.UserService
	// WorkspaceService handles workspace business logic.
	WorkspaceService service.WorkspaceService
	// MemberService handles workspace member business logic.
	MemberService service.MemberService
	// InvitationService handles workspace invitation business logic.
	InvitationService service.InvitationService
	// Authorizer handles workspace-scoped authorization checks.
	Authorizer *authz.Authorizer
}

// UserModule is an alias for CoreModule.
type UserModule = CoreModule

// New constructs a new CoreModule instance.
func New(cfg Config) *CoreModule {
	users := repository.NewUserRepository(cfg.EntCli)
	workspaces := repository.NewWorkspaceRepository(cfg.EntCli)
	members := repository.NewMemberRepository(cfg.EntCli)
	invitations := repository.NewInvitationRepository(cfg.EntCli)

	userSvc := service.NewUserService(users)
	workspaceSvc := service.NewWorkspaceService(workspaces)
	memberSvc := service.NewMemberService(members, workspaceSvc, userSvc)
	invitationSvc := service.NewInvitationService(invitations, workspaceSvc, members, userSvc)

	mod := &CoreModule{
		cfg:               cfg,
		UserService:       userSvc,
		WorkspaceService:  workspaceSvc,
		MemberService:     memberSvc,
		InvitationService: invitationSvc,
	}

	InitEntCli(cfg.EntCli)
	return mod
}

// SetAuthorizer configures the authorizer used by the core module.
func (u *CoreModule) SetAuthorizer(az *authz.Authorizer) {
	u.Authorizer = az
	u.MemberService.SetAuthorizer(az)
	u.InvitationService.SetAuthorizer(az)
}

// PreBoot executes pre-initialization tasks for the core module.
func (u *CoreModule) PreBoot() error {
	return nil
}

// Boot initializes repositories, services, and registers routes for the core module.
func (u *CoreModule) Boot() error {
	initRoutes(*u, u.cfg.FiberApp, u.Authorizer)
	return nil
}
