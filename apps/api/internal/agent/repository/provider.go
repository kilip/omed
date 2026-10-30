package repository

import (
	"context"
	"errors"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/platform/llmx"
)

var errUnimplemented = errors.New("unimplemented")

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
	return nil, errUnimplemented
}

func (r *providerRepository) List(ctx context.Context) ([]*ent.AgentProvider, error) {
	return nil, errUnimplemented
}

func (r *providerRepository) Get(ctx context.Context, id int64) (*ent.AgentProvider, error) {
	return nil, errUnimplemented
}

func (r *providerRepository) GetDefault(ctx context.Context) (*ent.AgentProvider, error) {
	return nil, errUnimplemented
}

func (r *providerRepository) Update(ctx context.Context, id int64, u ProviderUpdate) (*ent.AgentProvider, error) {
	return nil, errUnimplemented
}

func (r *providerRepository) Delete(ctx context.Context, id int64) error {
	return errUnimplemented
}

func (r *providerRepository) SetDefault(ctx context.Context, id int64) (*ent.AgentProvider, error) {
	return nil, errUnimplemented
}
