// Package core provides user management endpoints, domain logic, and data persistence.
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
	// UserService handles user business logic.
	UserService service.UserService
}

// UserModule is an alias for CoreModule.
type UserModule = CoreModule

// New constructs a new CoreModule instance.
func New(cfg Config) CoreModule {
	users := repository.NewUserRepository(cfg.EntCli)
	g := cfg.FiberApp.Group("/users")
	mod := CoreModule{
		UserService: service.NewUserService(users),
	}
	initRoutes(mod, g)
	return mod
}

// PreBoot executes pre-initialization tasks for the core module.
func (u CoreModule) PreBoot() error {
	return nil
}

// Boot initializes repositories, services, and registers routes for the core module.
func (u CoreModule) Boot() error {
	return nil
}
