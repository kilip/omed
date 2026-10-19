// Package fin provides finance management endpoints, domain logic, and data persistence.
package fin

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/fin/seed"
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
	cfg                 Config
	AccountService      service.AccountService
	PeriodService       service.PeriodService
	TransactionService  service.TransactionService
	EntryService        service.EntryService
	CurrencyService     service.CurrencyService
	ExchangeRateService service.ExchangeRateService
	Authorizer          *authz.Authorizer
	ReportService       service.ReportService
}

// New constructs a new FinModule instance.
func New(cfg Config) *FinModule {
	currencyRepo := repository.NewCurrencyRepository(cfg.EntCli)
	entryRepo := repository.NewEntryRepository(cfg.EntCli)
	currencySvc := service.NewCurrencyService(currencyRepo, entryRepo)

	rateRepo := repository.NewExchangeRateRepository(cfg.EntCli)
	rateSvc := service.NewExchangeRateService(rateRepo, currencyRepo)

	accountRepo := repository.NewAccountRepository(cfg.EntCli)
	accountSvc := service.NewAccountService(accountRepo)

	periodRepo := repository.NewPeriodRepository(cfg.EntCli)
	periodSvc := service.NewPeriodService(
		periodRepo,
		service.WithPeriodDependencies(accountRepo, entryRepo, rateRepo, currencyRepo),
	)

	txRepo := repository.NewTransactionRepository(cfg.EntCli)
	txSvc := service.NewTransactionService(txRepo, periodRepo, accountRepo, rateRepo, currencyRepo)

	entrySvc := service.NewEntryService(entryRepo, txRepo, periodRepo, accountRepo, rateRepo, currencyRepo)

	reportSvc := service.NewReportService(entryRepo, accountRepo, currencyRepo)

	return &FinModule{
		cfg:                 cfg,
		AccountService:      accountSvc,
		PeriodService:       periodSvc,
		TransactionService:  txSvc,
		EntryService:        entrySvc,
		CurrencyService:     currencySvc,
		ExchangeRateService: rateSvc,
		ReportService:       reportSvc,
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

// Boot registers routes for the finance module and seeds initial currencies if needed.
func (m *FinModule) Boot() error {
	if err := seed.SeedCurrencies(context.Background(), m.cfg.EntCli); err != nil {
		return err
	}
	initRoutes(*m, m.cfg.FiberApp, m.Authorizer)
	return nil
}
