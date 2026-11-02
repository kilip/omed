package repository

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/agentsetting"
	"github.com/kilip/omed/api/internal/base"
)

// SettingRepository defines persistence operations for workspace agent settings.
type SettingRepository interface {
	Get(ctx context.Context) (*ent.AgentSetting, error)
	Upsert(ctx context.Context, enabled bool, providerID *int64) (*ent.AgentSetting, error)
}

type settingRepository struct {
	cli *ent.Client
}

// NewSettingRepository creates a new SettingRepository.
func NewSettingRepository(cli *ent.Client) SettingRepository {
	return &settingRepository{cli: cli}
}

// Get retrieves the agent setting for the current workspace. Returns ent.IsNotFound when no row exists.
func (r *settingRepository) Get(ctx context.Context) (*ent.AgentSetting, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}

	return r.cli.AgentSetting.Query().
		Where(agentsetting.WorkspaceIDEQ(wsID)).
		Only(ctx)
}

// Upsert creates or updates the agent setting for the current workspace keyed on workspace_id.
func (r *settingRepository) Upsert(ctx context.Context, enabled bool, providerID *int64) (*ent.AgentSetting, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}

	builder := r.cli.AgentSetting.Create().
		SetWorkspaceID(wsID).
		SetFinancialEnabled(enabled)

	if providerID != nil {
		builder.SetFinancialProviderID(*providerID)
	}

	if userID := base.UserIDFromContext(ctx); userID > 0 {
		builder.SetCreatedBy(userID).SetUpdatedBy(userID)
	} else {
		builder.SetCreatedBy(0).SetUpdatedBy(0)
	}

	upsert := builder.OnConflict(
		sql.ConflictColumns(agentsetting.FieldWorkspaceID),
	).
		SetFinancialEnabled(enabled).
		UpdateUpdatedAt()

	if providerID != nil {
		upsert.SetFinancialProviderID(*providerID)
	} else {
		upsert.ClearFinancialProviderID()
	}

	if userID := base.UserIDFromContext(ctx); userID > 0 {
		upsert.SetUpdatedBy(userID)
	}

	id, err := upsert.ID(ctx)
	if err != nil {
		return nil, err
	}

	return r.cli.AgentSetting.Get(ctx, id)
}
