package service_test

import (
	"context"
	"encoding/base64"
	"errors"
	"iter"
	"strings"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/agentprovider"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/kilip/omed/api/internal/agent/repository"
	"github.com/kilip/omed/api/internal/agent/service"
	"github.com/kilip/omed/api/internal/agent/service/mock"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/llmx"
	"github.com/kilip/omed/api/internal/platform/secretx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestCipher(t *testing.T) *secretx.Cipher {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	c, err := secretx.New(base64.StdEncoding.EncodeToString(key))
	require.NoError(t, err)
	return c
}

func newContextWithUserWorkspace(userID, workspaceID int64) context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{ID: userID})
	return base.WithWorkspaceID(ctx, workspaceID)
}

func TestCreateEncryptsKeyAndStoresHint(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)
	svc := service.NewProviderService(repo, cipher, false)

	ctx := newContextWithUserWorkspace(1, 42)
	req := model.CreateProviderRequest{
		Name:    "OpenAI Main",
		Kind:    string(llmx.KindOpenAICompat),
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4o",
		APIKey:  "sk-test-secret-12345678abcd",
	}

	repo.EXPECT().List(ctx).Return([]*ent.AgentProvider{{ID: 99}}, nil).AnyTimes()
	repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, p repository.NewProvider) (*ent.AgentProvider, error) {
		assert.True(t, strings.HasPrefix(p.APIKeyEnc, "v1."), "expected ciphertext to have v1. prefix")
		assert.Equal(t, "abcd", p.KeyHint)
		assert.Equal(t, "OpenAI Main", p.Name)

		return &ent.AgentProvider{
			ID:          1,
			WorkspaceID: 42,
			Name:        p.Name,
			Kind:        agentprovider.Kind(p.Kind),
			BaseURL:     p.BaseURL,
			Model:       p.Model,
			KeyHint:     p.KeyHint,
			APIKeyEnc:   p.APIKeyEnc,
			IsDefault:   false,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	})

	res, err := svc.Create(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, int64(1), res.ID)
	assert.True(t, res.HasKey)
	assert.Equal(t, "abcd", res.KeyHint)
}

func TestCreateValidatesBaseURL(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)
	svc := service.NewProviderService(repo, cipher, false)

	ctx := newContextWithUserWorkspace(1, 42)
	req := model.CreateProviderRequest{
		Name:    "Invalid Base",
		Kind:    string(llmx.KindOpenAICompat),
		BaseURL: "http://127.0.0.1:8080/v1",
		Model:   "gpt-4o",
		APIKey:  "sk-test-secret-1234",
	}

	_, err := svc.Create(ctx, req)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidBaseURL))
}

func TestCreateRejectsBlankOrOverlongName(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)
	svc := service.NewProviderService(repo, cipher, false)

	ctx := newContextWithUserWorkspace(1, 42)

	// Blank name
	_, err := svc.Create(ctx, model.CreateProviderRequest{
		Name:    "   ",
		Kind:    string(llmx.KindOpenAICompat),
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4o",
		APIKey:  "sk-test-secret-1234",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidProviderName))

	// Overlong name (> 100 characters)
	_, err = svc.Create(ctx, model.CreateProviderRequest{
		Name:    strings.Repeat("a", 101),
		Kind:    string(llmx.KindOpenAICompat),
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4o",
		APIKey:  "sk-test-secret-1234",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidProviderName))

	// Invalid model
	_, err = svc.Create(ctx, model.CreateProviderRequest{
		Name:    "Valid Name",
		Kind:    string(llmx.KindOpenAICompat),
		BaseURL: "https://api.openai.com/v1",
		Model:   "   ",
		APIKey:  "sk-test-secret-1234",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidModel))

	// Invalid provider kind
	_, err = svc.Create(ctx, model.CreateProviderRequest{
		Name:    "Valid Name",
		Kind:    "unsupported-kind",
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4o",
		APIKey:  "sk-test-secret-1234",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidProviderKind))
}

func TestFirstProviderBecomesDefault(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)
	svc := service.NewProviderService(repo, cipher, false)

	ctx := newContextWithUserWorkspace(1, 42)
	req := model.CreateProviderRequest{
		Name:      "First Provider",
		Kind:      string(llmx.KindOpenAICompat),
		BaseURL:   "https://api.openai.com/v1",
		Model:     "gpt-4o",
		APIKey:    "sk-test-1234",
		IsDefault: false, // caller passed false, but if no provider exists it should become default
	}

	repo.EXPECT().List(ctx).Return([]*ent.AgentProvider{}, nil)
	repo.EXPECT().Create(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, p repository.NewProvider) (*ent.AgentProvider, error) {
		assert.True(t, p.IsDefault, "first provider must be marked as default")
		return &ent.AgentProvider{
			ID:          1,
			WorkspaceID: 42,
			Name:        p.Name,
			Kind:        agentprovider.Kind(p.Kind),
			BaseURL:     p.BaseURL,
			Model:       p.Model,
			KeyHint:     p.KeyHint,
			APIKeyEnc:   p.APIKeyEnc,
			IsDefault:   true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}, nil
	})

	res, err := svc.Create(ctx, req)
	require.NoError(t, err)
	assert.True(t, res.IsDefault)
}

func TestCreateWithoutSecretKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	disabledCipher, err := secretx.New("")
	require.NoError(t, err)
	svc := service.NewProviderService(repo, disabledCipher, false)

	ctx := newContextWithUserWorkspace(1, 42)

	// Create should fail with ErrSecretKeyMissing
	_, err = svc.Create(ctx, model.CreateProviderRequest{
		Name:    "Disabled Cipher Provider",
		Kind:    string(llmx.KindOpenAICompat),
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4o",
		APIKey:  "sk-test-1234",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrSecretKeyMissing))

	// List and Delete must still work
	repo.EXPECT().List(ctx).Return([]*ent.AgentProvider{}, nil)
	list, err := svc.List(ctx)
	require.NoError(t, err)
	assert.Empty(t, list)

	repo.EXPECT().Delete(ctx, int64(1)).Return(nil)
	err = svc.Delete(ctx, 1)
	require.NoError(t, err)
}

func TestUpdateKeepsKeyWhenAPIKeyNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)
	svc := service.NewProviderService(repo, cipher, false)

	ctx := newContextWithUserWorkspace(1, 42)
	newName := "Updated Name"

	repo.EXPECT().Get(ctx, int64(1)).Return(&ent.AgentProvider{
		ID:          1,
		WorkspaceID: 42,
		Name:        "Old Name",
		Kind:        agentprovider.Kind(llmx.KindOpenAICompat),
		BaseURL:     "https://api.openai.com/v1",
		Model:       "gpt-4o",
		KeyHint:     "1234",
		APIKeyEnc:   "v1.existingkey",
	}, nil)

	repo.EXPECT().Update(ctx, int64(1), gomock.Any()).DoAndReturn(func(_ context.Context, id int64, u repository.ProviderUpdate) (*ent.AgentProvider, error) {
		assert.Equal(t, &newName, u.Name)
		assert.Nil(t, u.APIKeyEnc, "APIKeyEnc should be nil when APIKey is not provided in update")
		assert.Nil(t, u.KeyHint)

		return &ent.AgentProvider{
			ID:          1,
			WorkspaceID: 42,
			Name:        newName,
			Kind:        agentprovider.Kind(llmx.KindOpenAICompat),
			BaseURL:     "https://api.openai.com/v1",
			Model:       "gpt-4o",
			KeyHint:     "1234",
			APIKeyEnc:   "v1.existingkey",
		}, nil
	})

	res, err := svc.Update(ctx, 1, model.UpdateProviderRequest{
		Name: &newName,
	})
	require.NoError(t, err)
	assert.Equal(t, newName, res.Name)
	assert.True(t, res.HasKey)
	assert.Equal(t, "1234", res.KeyHint)
}

func TestUpdateBaseURLWithoutKeyRejected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)
	svc := service.NewProviderService(repo, cipher, false)

	ctx := newContextWithUserWorkspace(1, 42)
	newBaseURL := "https://newapi.openai.com/v1"

	// Provider existing without key
	repo.EXPECT().Get(ctx, int64(1)).Return(&ent.AgentProvider{
		ID:          1,
		WorkspaceID: 42,
		Name:        "Provider No Key",
		Kind:        agentprovider.Kind(llmx.KindOpenAICompat),
		BaseURL:     "https://api.openai.com/v1",
		Model:       "gpt-4o",
		APIKeyEnc:   "",
	}, nil)

	_, err := svc.Update(ctx, 1, model.UpdateProviderRequest{
		BaseURL: &newBaseURL,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrProviderKeyRequired))
}

func TestResolveAdapterUsesDefault(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)

	ctx := newContextWithUserWorkspace(1, 42)
	encKey, err := cipher.Encrypt("sk-valid-key-1234", "42")
	require.NoError(t, err)

	repo.EXPECT().GetDefault(ctx).Return(&ent.AgentProvider{
		ID:          10,
		WorkspaceID: 42,
		Name:        "Default Provider",
		Kind:        agentprovider.Kind(llmx.KindOpenAICompat),
		BaseURL:     "https://api.openai.com/v1",
		Model:       "gpt-4o",
		APIKeyEnc:   encKey,
		IsDefault:   true,
	}, nil)

	factoryCalled := false
	customFactory := func(cfg llmx.Config) (llmx.Adapter, error) {
		factoryCalled = true
		assert.Equal(t, llmx.KindOpenAICompat, cfg.Kind)
		assert.Equal(t, "sk-valid-key-1234", cfg.APIKey)
		return llmx.NewMemory(), nil
	}

	svc := service.NewProviderService(repo, cipher, false, service.WithAdapterFactory(customFactory))
	adapter, err := svc.ResolveAdapter(ctx, nil)
	require.NoError(t, err)
	assert.NotNil(t, adapter)
	assert.True(t, factoryCalled)
}

func TestResolveAdapterNoDefault(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)
	svc := service.NewProviderService(repo, cipher, false)

	ctx := newContextWithUserWorkspace(1, 42)
	repo.EXPECT().GetDefault(ctx).Return(nil, domain.ErrNoDefaultProvider)

	_, err := svc.ResolveAdapter(ctx, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrNoDefaultProvider))
}

func TestResolveAdapterUnreadableKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher1 := newTestCipher(t)

	// Create a second cipher with different key
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = byte(i + 99)
	}
	cipher2, err := secretx.New(base64.StdEncoding.EncodeToString(key2))
	require.NoError(t, err)

	ctx := newContextWithUserWorkspace(1, 42)
	encKey, err := cipher1.Encrypt("sk-valid-key-1234", "42")
	require.NoError(t, err)

	provID := int64(5)
	repo.EXPECT().Get(ctx, provID).Return(&ent.AgentProvider{
		ID:          5,
		WorkspaceID: 42,
		Name:        "Unreadable Key Provider",
		Kind:        agentprovider.Kind(llmx.KindOpenAICompat),
		BaseURL:     "https://api.openai.com/v1",
		Model:       "gpt-4o",
		APIKeyEnc:   encKey,
	}, nil)

	// Using cipher2 which cannot decrypt encKey
	svc := service.NewProviderService(repo, cipher2, false)
	_, err = svc.ResolveAdapter(ctx, &provID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrProviderKeyUnreadable))
}

func TestResolveAdapterMergesCapabilities(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)

	ctx := newContextWithUserWorkspace(1, 42)
	encKey, err := cipher.Encrypt("sk-valid-key-1234", "42")
	require.NoError(t, err)

	provID := int64(7)
	repo.EXPECT().Get(ctx, provID).Return(&ent.AgentProvider{
		ID:          7,
		WorkspaceID: 42,
		Name:        "Caps Provider",
		Kind:        agentprovider.Kind(llmx.KindOpenAICompat),
		BaseURL:     "https://api.openai.com/v1",
		Model:       "gpt-4o",
		APIKeyEnc:   encKey,
		Capabilities: &llmx.Capabilities{
			Vision:      true,
			ToolCalling: true,
		},
	}, nil)

	factoryCalled := false
	customFactory := func(cfg llmx.Config) (llmx.Adapter, error) {
		factoryCalled = true
		require.NotNil(t, cfg.Capabilities)
		assert.True(t, cfg.Capabilities.Vision)
		assert.True(t, cfg.Capabilities.ToolCalling)
		return llmx.NewMemory(), nil
	}

	svc := service.NewProviderService(repo, cipher, false, service.WithAdapterFactory(customFactory))
	_, err = svc.ResolveAdapter(ctx, &provID)
	require.NoError(t, err)
	assert.True(t, factoryCalled)
}

func TestTestMapsSentinelsToCodes(t *testing.T) {
	sentinelCases := []struct {
		err          error
		expectedCode string
	}{
		{llmx.ErrAuth, "AUTH_FAILED"},
		{llmx.ErrModelNotFound, "MODEL_NOT_FOUND"},
		{llmx.ErrRateLimit, "RATE_LIMITED"},
		{llmx.ErrBadRequest, "BAD_REQUEST"},
		{llmx.ErrUnavailable, "UNAVAILABLE"},
		{llmx.ErrBlockedURL, "BLOCKED_URL"},
	}

	for _, tc := range sentinelCases {
		t.Run(tc.expectedCode, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mock.NewMockProviderRepository(ctrl)
			cipher := newTestCipher(t)

			stubFactory := func(cfg llmx.Config) (llmx.Adapter, error) {
				return &stubAdapter{err: tc.err}, nil
			}

			svc := service.NewProviderService(repo, cipher, false, service.WithAdapterFactory(stubFactory))
			ctx := newContextWithUserWorkspace(1, 42)

			res, err := svc.TestConfig(ctx, model.TestProviderRequest{
				Kind:    string(llmx.KindOpenAICompat),
				BaseURL: "https://api.openai.com/v1",
				Model:   "gpt-4o",
				APIKey:  "sk-test",
			})
			require.NoError(t, err)
			assert.False(t, res.OK)
			assert.Equal(t, tc.expectedCode, res.Code)
		})
	}
}

func TestTestHangingUpstreamTimesOut(t *testing.T) {
	assert.Equal(t, 20*time.Second, service.DefaultTestTimeout)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)

	blockingAdapter := &blockingStubAdapter{}
	stubFactory := func(cfg llmx.Config) (llmx.Adapter, error) {
		return blockingAdapter, nil
	}

	svc := service.NewProviderService(repo, cipher, false,
		service.WithAdapterFactory(stubFactory),
		service.WithTestTimeout(50*time.Millisecond),
	)

	ctx := newContextWithUserWorkspace(1, 42)
	start := time.Now()
	res, err := svc.TestConfig(ctx, model.TestProviderRequest{
		Kind:    string(llmx.KindOpenAICompat),
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4o",
		APIKey:  "sk-test",
	})
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.False(t, res.OK)
	assert.Equal(t, "UNAVAILABLE", res.Code)
	assert.Less(t, elapsed, 1*time.Second)
}

type stubAdapter struct {
	err error
}

func (s *stubAdapter) Chat(ctx context.Context, req llmx.Request) (*llmx.Response, error) {
	return nil, s.err
}

func (s *stubAdapter) Stream(ctx context.Context, req llmx.Request) iter.Seq2[llmx.Event, error] {
	return func(yield func(llmx.Event, error) bool) {
		yield(llmx.Event{}, s.err)
	}
}

func (s *stubAdapter) Capabilities() llmx.Capabilities {
	return llmx.Capabilities{}
}

type blockingStubAdapter struct{}

func (b *blockingStubAdapter) Chat(ctx context.Context, req llmx.Request) (*llmx.Response, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (b *blockingStubAdapter) Stream(ctx context.Context, req llmx.Request) iter.Seq2[llmx.Event, error] {
	return func(yield func(llmx.Event, error) bool) {
		<-ctx.Done()
		yield(llmx.Event{}, ctx.Err())
	}
}

func (b *blockingStubAdapter) Capabilities() llmx.Capabilities {
	return llmx.Capabilities{}
}

func TestProviderService_ListModels_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)

	ctx := newContextWithUserWorkspace(1, 42)
	encKey, err := cipher.Encrypt("sk-test-key-1234", "42")
	require.NoError(t, err)

	provID := int64(10)
	repo.EXPECT().Get(ctx, provID).Return(&ent.AgentProvider{
		ID:          10,
		WorkspaceID: 42,
		Name:        "Test Provider",
		Kind:        agentprovider.Kind(llmx.KindOpenAICompat),
		BaseURL:     "https://api.openai.com/v1",
		Model:       "gpt-4o",
		APIKeyEnc:   encKey,
	}, nil)

	listCalled := false
	customList := func(ctx context.Context, cfg llmx.Config) ([]string, error) {
		listCalled = true
		assert.Equal(t, llmx.KindOpenAICompat, cfg.Kind)
		assert.Equal(t, "https://api.openai.com/v1", cfg.BaseURL)
		assert.Equal(t, "sk-test-key-1234", cfg.APIKey)
		assert.False(t, cfg.AllowPrivateURLs)
		return []string{"gpt-4o", "gpt-4o-mini"}, nil
	}

	svc := service.NewProviderService(repo, cipher, false, service.WithListModels(customList))
	models, err := svc.ListModels(ctx, provID)
	require.NoError(t, err)
	assert.True(t, listCalled)
	assert.Equal(t, []string{"gpt-4o", "gpt-4o-mini"}, models)
}

func TestProviderService_ListModels_SecretKeyMissing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	svc := service.NewProviderService(repo, nil, false)
	ctx := newContextWithUserWorkspace(1, 42)

	_, err := svc.ListModels(ctx, 1)
	require.ErrorIs(t, err, domain.ErrSecretKeyMissing)
}

func TestProviderService_ListModels_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)

	ctx := newContextWithUserWorkspace(1, 42)
	provID := int64(999)
	repo.EXPECT().Get(ctx, provID).Return(nil, domain.ErrProviderNotFound)

	svc := service.NewProviderService(repo, cipher, false)
	_, err := svc.ListModels(ctx, provID)
	require.ErrorIs(t, err, domain.ErrProviderNotFound)
}

func TestProviderService_ListModels_UnreadableKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)

	ctx := newContextWithUserWorkspace(1, 42)
	provID := int64(10)
	repo.EXPECT().Get(ctx, provID).Return(&ent.AgentProvider{
		ID:          10,
		WorkspaceID: 42,
		Name:        "Test Provider",
		Kind:        agentprovider.Kind(llmx.KindOpenAICompat),
		BaseURL:     "https://api.openai.com/v1",
		Model:       "gpt-4o",
		APIKeyEnc:   "corrupt-key",
	}, nil)

	svc := service.NewProviderService(repo, cipher, false)
	_, err := svc.ListModels(ctx, provID)
	require.ErrorIs(t, err, domain.ErrProviderKeyUnreadable)
}

func TestProviderService_ListModelsConfig_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)

	listCalled := false
	customList := func(ctx context.Context, cfg llmx.Config) ([]string, error) {
		listCalled = true
		assert.Equal(t, llmx.KindOpenAICompat, cfg.Kind)
		assert.Equal(t, "https://api.openai.com/v1", cfg.BaseURL)
		assert.Equal(t, "sk-test-key", cfg.APIKey)
		assert.False(t, cfg.AllowPrivateURLs)
		return []string{"gpt-4o", "gpt-4o-mini"}, nil
	}

	svc := service.NewProviderService(repo, cipher, false, service.WithListModels(customList))
	ctx := newContextWithUserWorkspace(1, 42)

	req := model.ListModelsRequest{
		Kind:    string(llmx.KindOpenAICompat),
		BaseURL: "  https://api.openai.com/v1  ",
		APIKey:  "  sk-test-key  ",
	}

	models, err := svc.ListModelsConfig(ctx, req)
	require.NoError(t, err)
	assert.True(t, listCalled)
	assert.Equal(t, []string{"gpt-4o", "gpt-4o-mini"}, models)
}

func TestProviderService_ListModelsConfig_SSRF(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)

	svc := service.NewProviderService(repo, cipher, false)
	ctx := newContextWithUserWorkspace(1, 42)

	req := model.ListModelsRequest{
		Kind:    string(llmx.KindOpenAICompat),
		BaseURL: "http://127.0.0.1:8080/v1",
		APIKey:  "sk-test-secret",
	}

	_, err := svc.ListModelsConfig(ctx, req)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrInvalidBaseURL) || errors.Is(err, llmx.ErrBlockedURL))
}

func TestProviderService_ListModelsConfig_AllowPrivate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)

	customList := func(ctx context.Context, cfg llmx.Config) ([]string, error) {
		assert.True(t, cfg.AllowPrivateURLs)
		assert.Equal(t, "http://127.0.0.1:8080/v1", cfg.BaseURL)
		return []string{"local-model"}, nil
	}

	svc := service.NewProviderService(repo, cipher, true, service.WithListModels(customList))
	ctx := newContextWithUserWorkspace(1, 42)

	models, err := svc.ListModelsConfig(ctx, model.ListModelsRequest{
		Kind:    string(llmx.KindOpenAICompat),
		BaseURL: "http://127.0.0.1:8080/v1",
		APIKey:  "sk-local",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"local-model"}, models)
}

func TestProviderService_ListModels_RateLimited(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockProviderRepository(ctrl)
	cipher := newTestCipher(t)

	stubFactory := func(cfg llmx.Config) (llmx.Adapter, error) {
		return &stubAdapter{}, nil
	}
	svc := service.NewProviderService(repo, cipher, false, service.WithAdapterFactory(stubFactory))
	ctx := newContextWithUserWorkspace(1, 42)

	// Deplete rate limiter tokens (10 capacity)
	for i := 0; i < 10; i++ {
		_, _ = svc.TestConfig(ctx, model.TestProviderRequest{
			Kind:   string(llmx.KindOpenAICompat),
			Model:  "gpt-4o",
			APIKey: "sk-test",
		})
	}

	// Next calls should fail with ErrTestRateLimited
	_, err := svc.ListModels(ctx, 1)
	require.ErrorIs(t, err, domain.ErrTestRateLimited)

	_, err = svc.ListModelsConfig(ctx, model.ListModelsRequest{
		Kind:   string(llmx.KindOpenAICompat),
		APIKey: "sk-test",
	})
	require.ErrorIs(t, err, domain.ErrTestRateLimited)
}
