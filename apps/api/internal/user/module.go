package user

import (
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/user/repository"
	"github.com/kilip/omed/api/internal/user/service"
)

type UserModule struct {
	UserService service.UserService
}

func New() UserModule {
	return UserModule{}
}

func (u UserModule) PreBoot(state core.State) error {
	return nil
}

func (u UserModule) Boot(state core.State) error {
	users := repository.NewUserRepository(state.EntCli)
	u.UserService = service.NewUserService(users)

	app := state.FiberApp
	g := app.Group("/users")
	initRoutes(u, g)

	return nil
}
