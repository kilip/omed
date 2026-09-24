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

func initProtected(cfg BootstrapConfig) {
	configureAuth(cfg)
	accountRepository := repository.NewAccountRepository(cfg.DB)
	accountService := service.NewAccountService(cfg.DB, cfg.Logger, accountRepository)

	http.NewAccountController(cfg.Fiber, accountService, cfg.Enforcer, cfg.Logger)
}

func Bootstrap(cfg BootstrapConfig) {
	initProtected(cfg)
}
