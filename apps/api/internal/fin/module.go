// Package fin provides finance management endpoints, domain logic, and data persistence.
package fin

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/fin/mcp"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/fin/seed"
	"github.com/kilip/omed/api/internal/fin/service"
	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/kilip/omed/api/internal/platform/mcpx"
)

// Config encapsulates configuration and dependencies for FinModule.
type Config struct {
	FiberApp               *fiber.App
	Logger                 *slog.Logger
	EntCli                 *ent.Client
	FileService            service.FileServiceContract
	AttachmentMaxSizeMB    int64
	AttachmentAllowedTypes []string
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
	SettingService      service.SettingService
	CashFlowService     service.CashFlowService
	InsightService      service.InsightService
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

	settingRepo := repository.NewSettingRepository(cfg.EntCli)
	settingSvc := service.NewSettingService(
		settingRepo,
		accountRepo,
		service.WithSettingCurrencyRepository(currencyRepo),
	)

	periodRepo := repository.NewPeriodRepository(cfg.EntCli)
	periodSvc := service.NewPeriodService(
		periodRepo,
		service.WithPeriodDependencies(accountRepo, entryRepo, rateRepo, currencyRepo, settingRepo),
	)

	cashFlowRepo := repository.NewCashFlowRepository(cfg.EntCli)
	cashFlowSvc := service.NewCashFlowService(
		cashFlowRepo,
		periodRepo,
		accountRepo,
		rateRepo,
		currencyRepo,
		cfg.FileService,
	)

	txRepo := repository.NewTransactionRepository(cfg.EntCli)
	maxSizeBytes := cfg.AttachmentMaxSizeMB * 1024 * 1024
	txSvc := service.NewTransactionService(
		txRepo,
		periodRepo,
		accountRepo,
		rateRepo,
		currencyRepo,
		service.WithTransactionAttachments(cfg.FileService, service.TransactionAttachmentConfig{
			MaxSizeBytes: maxSizeBytes,
			AllowedTypes: cfg.AttachmentAllowedTypes,
		}),
		service.WithTransactionSettingRepository(settingRepo),
	)

	entrySvc := service.NewEntryService(entryRepo, txRepo, periodRepo, accountRepo, rateRepo, currencyRepo, settingRepo)

	reportSvc := service.NewReportService(entryRepo, accountRepo, currencyRepo)
	insightSvc := service.NewInsightService(reportSvc, accountSvc, cashFlowSvc)

	return &FinModule{
		cfg:                 cfg,
		AccountService:      accountSvc,
		PeriodService:       periodSvc,
		TransactionService:  txSvc,
		EntryService:        entrySvc,
		CurrencyService:     currencySvc,
		ExchangeRateService: rateSvc,
		ReportService:       reportSvc,
		SettingService:      settingSvc,
		CashFlowService:     cashFlowSvc,
		InsightService:      insightSvc,
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

// RegisterMCP registers the finance insight tools with the MCP registrar.
func (m *FinModule) RegisterMCP(r *mcpx.Registrar) error {
	mcp.RegisterTools(r, m.InsightService)
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
