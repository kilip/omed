// Package user provides user management endpoints, domain logic, and data persistence.
package user

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/user/repository"
	"github.com/kilip/omed/api/internal/user/service"
)

type Config struct {
	FiberApp *fiber.App
	Logger   *slog.Logger
	EntCli   *ent.Client
}

// UserModule represents the user management module implementing base.Module.
type UserModule struct {
	// UserService handles user business logic.
	UserService service.UserService
}

// New constructs a new UserModule instance.
func New(cfg Config) UserModule {
	users := repository.NewUserRepository(cfg.EntCli)
	g := cfg.FiberApp.Group("/users")
	mod := UserModule{
		UserService: service.NewUserService(users),
	}
	initRoutes(mod, g)
	return mod
}

// PreBoot executes pre-initialization tasks for the user module.
func (u UserModule) PreBoot() error {
	return nil
}

// Boot initializes repositories, services, and registers routes for the user module.
func (u UserModule) Boot() error {
	return nil
}
