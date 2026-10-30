//go:generate go run go.uber.org/mock/mockgen -source=provider.go -destination=mock/mock_provider.go -package=mock

package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/kilip/omed/api/internal/agent/repository"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/llmx"
	"github.com/kilip/omed/api/internal/platform/secretx"
)

var (
	// DefaultTestTimeout is default timeout for connection testing.
	DefaultTestTimeout = 20 * time.Second
)

// ProviderRepository defines persistence operations needed by ProviderService.
type ProviderRepository interface {
	Create(ctx context.Context, p repository.NewProvider) (*ent.AgentProvider, error)
	List(ctx context.Context) ([]*ent.AgentProvider, error)
	Get(ctx context.Context, id int64) (*ent.AgentProvider, error)
	GetDefault(ctx context.Context) (*ent.AgentProvider, error)
	Update(ctx context.Context, id int64, u repository.ProviderUpdate) (*ent.AgentProvider, error)
	Delete(ctx context.Context, id int64) error
	SetDefault(ctx context.Context, id int64) (*ent.AgentProvider, error)
}

// ProviderService orchestrates AI model provider management and resolution.
type ProviderService interface {
	List(ctx context.Context) ([]domain.Provider, error)
	Get(ctx context.Context, id int64) (domain.Provider, error)
	Create(ctx context.Context, req model.CreateProviderRequest) (domain.Provider, error)
	Update(ctx context.Context, id int64, req model.UpdateProviderRequest) (domain.Provider, error)
	Delete(ctx context.Context, id int64) error
	SetDefault(ctx context.Context, id int64) (domain.Provider, error)
	Test(ctx context.Context, id int64) (model.TestResult, error)
	TestConfig(ctx context.Context, req model.TestProviderRequest) (model.TestResult, error)
	ResolveAdapter(ctx context.Context, providerID *int64) (llmx.Adapter, error)
}

// ProviderOption configures ProviderService.
type ProviderOption func(*providerService)

// WithAdapterFactory overrides the adapter factory function.
func WithAdapterFactory(f llmx.Factory) ProviderOption {
	return func(s *providerService) {
		s.adapterFactory = f
	}
}

// WithClock overrides the clock function.
func WithClock(now func() time.Time) ProviderOption {
	return func(s *providerService) {
		s.now = now
	}
}

// WithTestTimeout overrides the test timeout.
func WithTestTimeout(d time.Duration) ProviderOption {
	return func(s *providerService) {
		s.testTimeout = d
	}
}

type providerService struct {
	repo           ProviderRepository
	cipher         *secretx.Cipher
	allowPrivate   bool
	adapterFactory llmx.Factory
	now            func() time.Time
	testTimeout    time.Duration
	limiter        *RateLimiter
}

// NewProviderService creates a new ProviderService implementation.
func NewProviderService(repo repository.ProviderRepository, cipher *secretx.Cipher, allowPrivate bool, opts ...ProviderOption) ProviderService {
	s := &providerService{
		repo:           repo,
		cipher:         cipher,
		allowPrivate:   allowPrivate,
		adapterFactory: llmx.New,
		now:            time.Now,
		testTimeout:    DefaultTestTimeout,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.limiter = NewRateLimiter(s.now)
	return s
}

func toDomain(p *ent.AgentProvider) domain.Provider {
	if p == nil {
		return domain.Provider{}
	}
	return domain.Provider{
		ID:           p.ID,
		WorkspaceID:  p.WorkspaceID,
		Name:         p.Name,
		Kind:         llmx.Kind(p.Kind),
		BaseURL:      p.BaseURL,
		Model:        p.Model,
		KeyHint:      p.KeyHint,
		HasKey:       p.APIKeyEnc != "",
		IsDefault:    p.IsDefault,
		Capabilities: p.Capabilities,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

func (s *providerService) List(ctx context.Context) ([]domain.Provider, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Provider, 0, len(rows))
	for _, r := range rows {
		out = append(out, toDomain(r))
	}
	return out, nil
}

func (s *providerService) Get(ctx context.Context, id int64) (domain.Provider, error) {
	row, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Provider{}, err
	}
	return toDomain(row), nil
}

func (s *providerService) Create(ctx context.Context, req model.CreateProviderRequest) (domain.Provider, error) {
	name := strings.TrimSpace(req.Name)
	if len(name) < 1 || len(name) > 100 {
		return domain.Provider{}, base.NewUnprocessableEntityError("name must be between 1 and 100 characters")
	}

	modelName := strings.TrimSpace(req.Model)
	if len(modelName) < 1 || len(modelName) > 200 {
		return domain.Provider{}, base.NewUnprocessableEntityError("model must be between 1 and 200 characters")
	}

	kind := llmx.Kind(req.Kind)
	if kind != llmx.KindOpenAICompat && kind != llmx.KindAnthropic && kind != llmx.KindGemini {
		return domain.Provider{}, base.NewUnprocessableEntityError("invalid provider kind")
	}

	if s.cipher == nil || !s.cipher.Enabled() {
		return domain.Provider{}, domain.ErrSecretKeyMissing
	}

	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		return domain.Provider{}, domain.ErrProviderKeyRequired
	}

	baseURL := strings.TrimSpace(req.BaseURL)
	if kind == llmx.KindOpenAICompat && baseURL == "" {
		return domain.Provider{}, domain.ErrInvalidBaseURL
	}
	if baseURL != "" {
		if err := llmx.ValidateBaseURL(ctx, baseURL, s.allowPrivate); err != nil {
			return domain.Provider{}, fmt.Errorf("%w: %w", domain.ErrInvalidBaseURL, err)
		}
	}

	wsID := base.WorkspaceIDFromContext(ctx)
	enc, err := s.cipher.Encrypt(apiKey, strconv.FormatInt(wsID, 10))
	if err != nil {
		return domain.Provider{}, err
	}
	hint := secretx.KeyHint(apiKey)

	isDefault := req.IsDefault
	if !isDefault {
		if existing, err := s.repo.List(ctx); err == nil && len(existing) == 0 {
			isDefault = true
		}
	}

	created, err := s.repo.Create(ctx, repository.NewProvider{
		Name:         name,
		Kind:         kind,
		BaseURL:      baseURL,
		Model:        modelName,
		APIKeyEnc:    enc,
		KeyHint:      hint,
		Capabilities: req.Capabilities,
		IsDefault:    isDefault,
	})
	if err != nil {
		return domain.Provider{}, err
	}
	return toDomain(created), nil
}

func (s *providerService) Update(ctx context.Context, id int64, req model.UpdateProviderRequest) (domain.Provider, error) {
	if s.cipher == nil || !s.cipher.Enabled() {
		return domain.Provider{}, domain.ErrSecretKeyMissing
	}

	existing, err := s.repo.Get(ctx, id)
	if err != nil {
		return domain.Provider{}, err
	}

	u := repository.ProviderUpdate{
		Capabilities: req.Capabilities,
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if len(trimmed) < 1 || len(trimmed) > 100 {
			return domain.Provider{}, base.NewUnprocessableEntityError("name must be between 1 and 100 characters")
		}
		u.Name = &trimmed
	}

	if req.Model != nil {
		trimmed := strings.TrimSpace(*req.Model)
		if len(trimmed) < 1 || len(trimmed) > 200 {
			return domain.Provider{}, base.NewUnprocessableEntityError("model must be between 1 and 200 characters")
		}
		u.Model = &trimmed
	}

	if req.BaseURL != nil {
		trimmed := strings.TrimSpace(*req.BaseURL)
		if llmx.Kind(existing.Kind) == llmx.KindOpenAICompat && trimmed == "" {
			return domain.Provider{}, domain.ErrInvalidBaseURL
		}
		if trimmed != "" {
			if err := llmx.ValidateBaseURL(ctx, trimmed, s.allowPrivate); err != nil {
				return domain.Provider{}, fmt.Errorf("%w: %w", domain.ErrInvalidBaseURL, err)
			}
		}
		if trimmed != existing.BaseURL && (req.APIKey == nil || strings.TrimSpace(*req.APIKey) == "") {
			return domain.Provider{}, domain.ErrProviderKeyRequired
		}
		u.BaseURL = &trimmed
	}

	if req.APIKey != nil && strings.TrimSpace(*req.APIKey) != "" {
		apiKey := strings.TrimSpace(*req.APIKey)
		enc, err := s.cipher.Encrypt(apiKey, strconv.FormatInt(existing.WorkspaceID, 10))
		if err != nil {
			return domain.Provider{}, err
		}
		hint := secretx.KeyHint(apiKey)
		u.APIKeyEnc = &enc
		u.KeyHint = &hint
	}

	updated, err := s.repo.Update(ctx, id, u)
	if err != nil {
		return domain.Provider{}, err
	}
	return toDomain(updated), nil
}

func (s *providerService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *providerService) SetDefault(ctx context.Context, id int64) (domain.Provider, error) {
	updated, err := s.repo.SetDefault(ctx, id)
	if err != nil {
		return domain.Provider{}, err
	}
	return toDomain(updated), nil
}

func (s *providerService) ResolveAdapter(ctx context.Context, providerID *int64) (llmx.Adapter, error) {
	if s.cipher == nil || !s.cipher.Enabled() {
		return nil, domain.ErrSecretKeyMissing
	}

	var row *ent.AgentProvider
	var err error
	if providerID != nil {
		row, err = s.repo.Get(ctx, *providerID)
	} else {
		row, err = s.repo.GetDefault(ctx)
	}
	if err != nil {
		return nil, err
	}

	apiKey, err := s.cipher.Decrypt(row.APIKeyEnc, strconv.FormatInt(row.WorkspaceID, 10))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrProviderKeyUnreadable, err)
	}

	return s.adapterFactory(llmx.Config{
		Kind:             llmx.Kind(row.Kind),
		BaseURL:          row.BaseURL,
		APIKey:           apiKey,
		Model:            row.Model,
		Capabilities:     row.Capabilities,
		AllowPrivateURLs: s.allowPrivate,
	})
}

func mapTestErrorCode(err error) string {
	switch {
	case errors.Is(err, llmx.ErrAuth):
		return "AUTH_FAILED"
	case errors.Is(err, llmx.ErrModelNotFound):
		return "MODEL_NOT_FOUND"
	case errors.Is(err, llmx.ErrRateLimit):
		return "RATE_LIMITED"
	case errors.Is(err, llmx.ErrBadRequest):
		return "BAD_REQUEST"
	case errors.Is(err, llmx.ErrUnavailable), errors.Is(err, context.DeadlineExceeded):
		return "UNAVAILABLE"
	case errors.Is(err, llmx.ErrBlockedURL):
		return "BLOCKED_URL"
	default:
		return "UNKNOWN_ERROR"
	}
}

func (s *providerService) Test(ctx context.Context, id int64) (model.TestResult, error) {
	if !s.limiter.Allow(base.UserIDFromContext(ctx)) {
		return model.TestResult{}, domain.ErrTestRateLimited
	}

	adapter, err := s.ResolveAdapter(ctx, &id)
	if err != nil {
		if errors.Is(err, domain.ErrProviderNotFound) || errors.Is(err, domain.ErrSecretKeyMissing) || errors.Is(err, domain.ErrProviderKeyUnreadable) {
			return model.TestResult{}, err
		}
		return model.TestResult{OK: false, Code: mapTestErrorCode(err)}, nil
	}

	return s.runTest(ctx, adapter)
}

func (s *providerService) TestConfig(ctx context.Context, req model.TestProviderRequest) (model.TestResult, error) {
	if !s.limiter.Allow(base.UserIDFromContext(ctx)) {
		return model.TestResult{}, domain.ErrTestRateLimited
	}

	baseURL := strings.TrimSpace(req.BaseURL)
	if baseURL != "" {
		if err := llmx.ValidateBaseURL(ctx, baseURL, s.allowPrivate); err != nil {
			return model.TestResult{OK: false, Code: mapTestErrorCode(err)}, nil
		}
	}

	adapter, err := s.adapterFactory(llmx.Config{
		Kind:             llmx.Kind(req.Kind),
		BaseURL:          baseURL,
		APIKey:           strings.TrimSpace(req.APIKey),
		Model:            strings.TrimSpace(req.Model),
		Capabilities:     req.Capabilities,
		AllowPrivateURLs: s.allowPrivate,
	})
	if err != nil {
		return model.TestResult{OK: false, Code: mapTestErrorCode(err)}, nil
	}

	return s.runTest(ctx, adapter)
}

func (s *providerService) runTest(ctx context.Context, adapter llmx.Adapter) (model.TestResult, error) {
	testCtx, cancel := context.WithTimeout(ctx, s.testTimeout)
	defer cancel()

	start := s.now()
	_, err := adapter.Chat(testCtx, llmx.Request{
		Messages:  []llmx.Message{{Role: llmx.RoleUser, Content: "ping"}},
		MaxTokens: 1,
	})
	latencyMs := s.now().Sub(start).Milliseconds()

	if err != nil {
		return model.TestResult{
			OK:        false,
			Code:      mapTestErrorCode(err),
			LatencyMs: latencyMs,
		}, nil
	}

	return model.TestResult{
		OK:        true,
		LatencyMs: latencyMs,
	}, nil
}
