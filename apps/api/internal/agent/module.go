// Package agent provides AI model provider management and agent execution features.
package agent

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	_ "github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/repository"
	"github.com/kilip/omed/api/internal/agent/service"
	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/kilip/omed/api/internal/platform/secretx"
)

// Config encapsulates configuration and dependencies for AgentModule.
type Config struct {
	FiberApp               *fiber.App
	Logger                 *slog.Logger
	EntCli                 *ent.Client
	Cipher                 *secretx.Cipher
	AllowPrivateURLs       bool
	FinanceContext         service.FinanceContext
	AttachmentMaxSizeMB    int
	AttachmentAllowedTypes []string
}

// AgentModule represents the agent module implementing config.Module.
type AgentModule struct {
	cfg              Config
	Authorizer       *authz.Authorizer
	ProviderService  service.ProviderService
	SettingService   service.SettingService
	FinancialService service.FinancialService
}

// New constructs a new AgentModule instance.
func New(cfg Config) *AgentModule {
	repo := repository.NewProviderRepository(cfg.EntCli)
	providerSvc := service.NewProviderService(repo, cfg.Cipher, cfg.AllowPrivateURLs)
	settingRepo := repository.NewSettingRepository(cfg.EntCli)
	settingSvc := service.NewSettingService(settingRepo, providerSvc)
	financialSvc := service.NewFinancialService(settingRepo, providerSvc, cfg.FinanceContext)

	return &AgentModule{
		cfg:              cfg,
		ProviderService:  providerSvc,
		SettingService:   settingSvc,
		FinancialService: financialSvc,
	}
}

// SetAuthorizer sets the Casbin authorizer instance.
func (m *AgentModule) SetAuthorizer(az *authz.Authorizer) {
	m.Authorizer = az
}

// PreBoot executes early initialization tasks before Boot.
func (m *AgentModule) PreBoot() error {
	return nil
}

// Boot registers routes for the agent module.
func (m *AgentModule) Boot() error {
	initRoutes(*m, m.cfg.FiberApp, m.Authorizer)
	return nil
}
