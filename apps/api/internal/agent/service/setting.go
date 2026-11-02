//go:generate go run go.uber.org/mock/mockgen -source=setting.go -destination=mock/mock_setting.go -package=mock

package service

import (
	"context"
	"errors"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/llmx"
)

// SettingRepository defines persistence operations needed by SettingService.
type SettingRepository interface {
	Get(ctx context.Context) (*ent.AgentSetting, error)
	Upsert(ctx context.Context, enabled bool, providerID *int64) (*ent.AgentSetting, error)
}

// AdapterResolver resolves an LLM adapter by provider ID (or default if nil).
type AdapterResolver interface {
	ResolveAdapter(ctx context.Context, id *int64) (llmx.Adapter, error)
}

// SettingService orchestrates workspace agent setting management.
type SettingService interface {
	Get(ctx context.Context) (domain.Setting, error)
	Update(ctx context.Context, req model.UpdateSettingRequest) (domain.Setting, error)
}

type settingService struct {
	repo     SettingRepository
	resolver AdapterResolver
}

// NewSettingService creates a new SettingService instance.
func NewSettingService(repo SettingRepository, resolver AdapterResolver) SettingService {
	return &settingService{
		repo:     repo,
		resolver: resolver,
	}
}

func toSettingDomain(s *ent.AgentSetting) domain.Setting {
	if s == nil {
		return domain.Setting{}
	}
	return domain.Setting{
		WorkspaceID:         s.WorkspaceID,
		FinancialEnabled:    s.FinancialEnabled,
		FinancialProviderID: s.FinancialProviderID,
	}
}

// Get retrieves workspace agent setting or defaults (disabled) when no row exists.
func (s *settingService) Get(ctx context.Context) (domain.Setting, error) {
	row, err := s.repo.Get(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.Setting{
				WorkspaceID:         base.WorkspaceIDFromContext(ctx),
				FinancialEnabled:    false,
				FinancialProviderID: nil,
			}, nil
		}
		return domain.Setting{}, err
	}
	return toSettingDomain(row), nil
}

// Update updates workspace agent settings with provider validation when enabled.
func (s *settingService) Update(ctx context.Context, req model.UpdateSettingRequest) (domain.Setting, error) {
	enabled := req.FinancialEnabled != nil && *req.FinancialEnabled
	if enabled {
		adapter, err := s.resolver.ResolveAdapter(ctx, req.FinancialProviderID)
		if err != nil {
			if errors.Is(err, domain.ErrProviderNotFound) || errors.Is(err, domain.ErrNoDefaultProvider) {
				return domain.Setting{}, domain.ErrFinancialAgentNoProvider
			}
			return domain.Setting{}, err
		}

		caps := adapter.Capabilities()
		if !caps.Vision || !caps.StructuredOutput {
			return domain.Setting{}, domain.ErrFinancialAgentNoProvider
		}
	}

	row, err := s.repo.Upsert(ctx, enabled, req.FinancialProviderID)
	if err != nil {
		return domain.Setting{}, err
	}
	return toSettingDomain(row), nil
}
