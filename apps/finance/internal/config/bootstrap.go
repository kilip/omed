package config

import (
	"log/slog"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/delivery/http"
	"github.com/kilip/omed/finance/internal/infra/repository"
	"github.com/kilip/omed/finance/internal/service"
)

type BootstrapConfig struct {
	Config   Config
	Fiber    *fiber.App
	Logger   *slog.Logger
	DB       *ent.Client
	Enforcer *casbin.Enforcer
}

func InjectActiveWorkspace(svc service.WorkspaceService) fiber.Handler {
	return func(c fiber.Ctx) error {
		user := http.GetUser(c)
		err := svc.EnsureWorkspace(c, user)
		if err != nil {
			return err
		}

		return c.Next()
	}
}

func initProtected(cfg BootstrapConfig) {
	configureAuth(cfg)

	userR := repository.NewUserRepository(cfg.DB)
	workspaceR := repository.NewWorkspaceRepository(cfg.DB)
	wsService := service.NewWorkspaceService(userR, workspaceR)
	cfg.Fiber.Use(InjectActiveWorkspace(wsService))

	accountRepository := repository.NewAccountRepository(cfg.DB)
	accountService := service.NewAccountService(cfg.DB, cfg.Logger, accountRepository)

	http.NewHealthController(cfg.Fiber, cfg.Config.DB.URL)
	http.NewAccountController(cfg.Fiber, accountService, cfg.Enforcer, cfg.Logger)
}

func Bootstrap(cfg BootstrapConfig) {
	initProtected(cfg)
}
