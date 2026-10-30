package repository

import (
	"context"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/agentprovider"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/entx"
	"github.com/kilip/omed/api/internal/platform/llmx"
)

// NewProvider holds data for creating an agent provider.
type NewProvider struct {
	Name         string
	Kind         llmx.Kind
	BaseURL      string
	Model        string
	APIKeyEnc    string
	KeyHint      string
	Capabilities *llmx.Capabilities
	IsDefault    bool
}

// ProviderUpdate holds mutable fields for an agent provider.
type ProviderUpdate struct {
	Name         *string
	BaseURL      *string
	Model        *string
	APIKeyEnc    *string
	KeyHint      *string
	Capabilities *llmx.Capabilities
}

// ProviderRepository defines persistence operations for agent providers.
type ProviderRepository interface {
	Create(ctx context.Context, p NewProvider) (*ent.AgentProvider, error)
	List(ctx context.Context) ([]*ent.AgentProvider, error)
	Get(ctx context.Context, id int64) (*ent.AgentProvider, error)
	GetDefault(ctx context.Context) (*ent.AgentProvider, error)
	Update(ctx context.Context, id int64, u ProviderUpdate) (*ent.AgentProvider, error)
	Delete(ctx context.Context, id int64) error
	SetDefault(ctx context.Context, id int64) (*ent.AgentProvider, error)
}

type providerRepository struct {
	cli *ent.Client
}

// NewProviderRepository creates a new ProviderRepository.
func NewProviderRepository(cli *ent.Client) ProviderRepository {
	return &providerRepository{cli: cli}
}

func (r *providerRepository) Create(ctx context.Context, p NewProvider) (*ent.AgentProvider, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}

	var created *ent.AgentProvider
	err := entx.WithTx(ctx, r.cli, func(tx *ent.Tx) error {
		if p.IsDefault {
			if err := tx.AgentProvider.Update().
				Where(agentprovider.WorkspaceIDEQ(wsID), agentprovider.IsDefaultEQ(true)).
				SetIsDefault(false).
				Exec(ctx); err != nil {
				return err
			}
		}

		builder := tx.AgentProvider.Create().
			SetWorkspaceID(wsID).
			SetName(p.Name).
			SetKind(agentprovider.Kind(p.Kind)).
			SetBaseURL(p.BaseURL).
			SetModel(p.Model).
			SetAPIKeyEnc(p.APIKeyEnc).
			SetKeyHint(p.KeyHint).
			SetIsDefault(p.IsDefault)

		if p.Capabilities != nil {
			builder.SetCapabilities(p.Capabilities)
		}

		var err error
		created, err = builder.Save(ctx)
		return err
	})

	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, domain.ErrProviderNameTaken
		}
		return nil, err
	}

	return created, nil
}

func (r *providerRepository) List(ctx context.Context) ([]*ent.AgentProvider, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}

	return r.cli.AgentProvider.Query().
		Where(agentprovider.WorkspaceIDEQ(wsID)).
		WithCreator().
		WithUpdater().
		Order(ent.Desc(agentprovider.FieldIsDefault), ent.Asc(agentprovider.FieldName)).
		All(ctx)
}

func (r *providerRepository) Get(ctx context.Context, id int64) (*ent.AgentProvider, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}

	p, err := r.cli.AgentProvider.Query().
		Where(agentprovider.IDEQ(id), agentprovider.WorkspaceIDEQ(wsID)).
		WithCreator().
		WithUpdater().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, domain.ErrProviderNotFound
		}
		return nil, err
	}
	return p, nil
}

func (r *providerRepository) GetDefault(ctx context.Context) (*ent.AgentProvider, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}

	p, err := r.cli.AgentProvider.Query().
		Where(agentprovider.WorkspaceIDEQ(wsID), agentprovider.IsDefaultEQ(true)).
		WithCreator().
		WithUpdater().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, domain.ErrNoDefaultProvider
		}
		return nil, err
	}
	return p, nil
}

func (r *providerRepository) Update(ctx context.Context, id int64, u ProviderUpdate) (*ent.AgentProvider, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}

	builder := r.cli.AgentProvider.UpdateOneID(id).
		Where(agentprovider.WorkspaceIDEQ(wsID))

	if u.Name != nil {
		builder.SetName(*u.Name)
	}
	if u.BaseURL != nil {
		builder.SetBaseURL(*u.BaseURL)
	}
	if u.Model != nil {
		builder.SetModel(*u.Model)
	}
	if u.APIKeyEnc != nil {
		builder.SetAPIKeyEnc(*u.APIKeyEnc)
	}
	if u.KeyHint != nil {
		builder.SetKeyHint(*u.KeyHint)
	}
	if u.Capabilities != nil {
		builder.SetCapabilities(u.Capabilities)
	}

	p, err := builder.Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, domain.ErrProviderNotFound
		}
		if ent.IsConstraintError(err) {
			return nil, domain.ErrProviderNameTaken
		}
		return nil, err
	}
	return p, nil
}

func (r *providerRepository) Delete(ctx context.Context, id int64) error {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return base.ErrWorkspaceRequired
	}

	affected, err := r.cli.AgentProvider.Delete().
		Where(agentprovider.IDEQ(id), agentprovider.WorkspaceIDEQ(wsID)).
		Exec(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return domain.ErrProviderNotFound
	}
	return nil
}

func (r *providerRepository) SetDefault(ctx context.Context, id int64) (*ent.AgentProvider, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID == 0 {
		return nil, base.ErrWorkspaceRequired
	}

	var updated *ent.AgentProvider
	err := entx.WithTx(ctx, r.cli, func(tx *ent.Tx) error {
		exists, err := tx.AgentProvider.Query().
			Where(agentprovider.IDEQ(id), agentprovider.WorkspaceIDEQ(wsID)).
			Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return domain.ErrProviderNotFound
		}

		if err := tx.AgentProvider.Update().
			Where(agentprovider.WorkspaceIDEQ(wsID), agentprovider.IsDefaultEQ(true), agentprovider.IDNEQ(id)).
			SetIsDefault(false).
			Exec(ctx); err != nil {
			return err
		}

		updated, err = tx.AgentProvider.UpdateOneID(id).
			Where(agentprovider.WorkspaceIDEQ(wsID)).
			SetIsDefault(true).
			Save(ctx)
		return err
	})

	if err != nil {
		return nil, err
	}
	return updated, nil
}
