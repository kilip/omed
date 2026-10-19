// Package fin provides finance management endpoints, domain logic, and data persistence.
package fin

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/fin/service"
	"github.com/kilip/omed/api/internal/platform/authz"
)

// Config encapsulates configuration and dependencies for FinModule.
type Config struct {
	FiberApp *fiber.App
	Logger   *slog.Logger
	EntCli   *ent.Client
}

// FinModule represents the finance module implementing config.Module.
type FinModule struct {
	cfg            Config
	AccountService service.AccountService
	PeriodService  service.PeriodService
	Authorizer     *authz.Authorizer
}

// New constructs a new FinModule instance.
func New(cfg Config) *FinModule {
	accountRepo := repository.NewAccountRepository(cfg.EntCli)
	accountSvc := service.NewAccountService(accountRepo)

	periodRepo := repository.NewPeriodRepository(cfg.EntCli)
	periodSvc := service.NewPeriodService(periodRepo)

	return &FinModule{
		cfg:            cfg,
		AccountService: accountSvc,
		PeriodService:  periodSvc,
	}
}

// SetAuthorizer sets the Casbin authorizer instance.
func (m *FinModule) SetAuthorizer(az *authz.Authorizer) {
	m.Authorizer = az
}

// PreBoot executes early initialization tasks before Boot.
func (m *FinModule) PreBoot() error {
	return nil
}

// Boot registers routes for the finance module.
func (m *FinModule) Boot() error {
	initRoutes(*m, m.cfg.FiberApp, m.Authorizer)
	return nil
}
