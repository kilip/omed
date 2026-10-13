// Package core provides user and workspace management endpoints, domain logic, and data persistence.
package core

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/core/repository"
	"github.com/kilip/omed/api/internal/core/service"
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
}

// UserModule is an alias for CoreModule.
type UserModule = CoreModule

// New constructs a new CoreModule instance.
func New(cfg Config) CoreModule {
	users := repository.NewUserRepository(cfg.EntCli)
	workspaces := repository.NewWorkspaceRepository(cfg.EntCli)
	mod := CoreModule{
		cfg:              cfg,
		UserService:      service.NewUserService(users),
		WorkspaceService: service.NewWorkspaceService(workspaces),
	}
	return mod
}

// PreBoot executes pre-initialization tasks for the core module.
func (u CoreModule) PreBoot() error {
	return nil
}

// Boot initializes repositories, services, and registers routes for the core module.
func (u CoreModule) Boot() error {
	initRoutes(u, u.cfg.FiberApp)
	return nil
}
