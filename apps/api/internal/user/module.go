// Package user provides user management endpoints, domain logic, and data persistence.
package user

import (
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/user/repository"
	"github.com/kilip/omed/api/internal/user/service"
)

// UserModule represents the user management module implementing base.Module.
type UserModule struct {
	// UserService handles user business logic.
	UserService service.UserService
}

// New constructs a new UserModule instance.
func New() UserModule {
	return UserModule{}
}

// PreBoot executes pre-initialization tasks for the user module.
func (u UserModule) PreBoot(state base.State) error {
	return nil
}

// Boot initializes repositories, services, and registers routes for the user module.
func (u UserModule) Boot(state base.State) error {
	users := repository.NewUserRepository(state.EntCli)
	u.UserService = service.NewUserService(users)

	app := state.FiberApp
	g := app.Group("/users")
	initRoutes(u, g)

	return nil
}
