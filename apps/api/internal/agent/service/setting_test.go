package service_test

import (
	"context"
	"errors"
	"iter"
	"testing"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/kilip/omed/api/internal/agent/service"
	"github.com/kilip/omed/api/internal/agent/service/mock"
	"github.com/kilip/omed/api/internal/platform/llmx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type stubCapabilitiesAdapter struct {
	caps llmx.Capabilities
}

func (s *stubCapabilitiesAdapter) Chat(ctx context.Context, req llmx.Request) (*llmx.Response, error) {
	return nil, nil
}

func (s *stubCapabilitiesAdapter) Stream(ctx context.Context, req llmx.Request) iter.Seq2[llmx.Event, error] {
	return func(yield func(llmx.Event, error) bool) {}
}

func (s *stubCapabilitiesAdapter) Capabilities() llmx.Capabilities {
	return s.caps
}

func TestSettingGet_NoRowReturnsDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockSettingRepository(ctrl)
	resolver := mock.NewMockAdapterResolver(ctrl)
	svc := service.NewSettingService(repo, resolver)

	ctx := newContextWithUserWorkspace(1, 42)
	repo.EXPECT().Get(ctx).Return(nil, &ent.NotFoundError{})

	res, err := svc.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, domain.Setting{
		WorkspaceID:         42,
		FinancialEnabled:    false,
		FinancialProviderID: nil,
	}, res)
}

func TestSettingGet_ExistingRow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockSettingRepository(ctrl)
	resolver := mock.NewMockAdapterResolver(ctrl)
	svc := service.NewSettingService(repo, resolver)

	ctx := newContextWithUserWorkspace(1, 42)
	provID := int64(10)
	repo.EXPECT().Get(ctx).Return(&ent.AgentSetting{
		ID:                  1,
		WorkspaceID:         42,
		FinancialEnabled:    true,
		FinancialProviderID: &provID,
	}, nil)

	res, err := svc.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, domain.Setting{
		WorkspaceID:         42,
		FinancialEnabled:    true,
		FinancialProviderID: &provID,
	}, res)
}

func TestSettingGet_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockSettingRepository(ctrl)
	resolver := mock.NewMockAdapterResolver(ctrl)
	svc := service.NewSettingService(repo, resolver)

	ctx := newContextWithUserWorkspace(1, 42)
	dbErr := errors.New("db failure")
	repo.EXPECT().Get(ctx).Return(nil, dbErr)

	_, err := svc.Get(ctx)
	require.ErrorIs(t, err, dbErr)
}

func TestSettingUpdate_EnableWithoutProviderFails(t *testing.T) {
	t.Run("default provider missing returns ErrFinancialAgentNoProvider and skips upsert", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mock.NewMockSettingRepository(ctrl)
		resolver := mock.NewMockAdapterResolver(ctrl)
		svc := service.NewSettingService(repo, resolver)

		ctx := newContextWithUserWorkspace(1, 42)
		enabled := true
		req := model.UpdateSettingRequest{
			FinancialEnabled:    &enabled,
			FinancialProviderID: nil,
		}

		resolver.EXPECT().ResolveAdapter(ctx, (*int64)(nil)).Return(nil, domain.ErrNoDefaultProvider)
		repo.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		_, err := svc.Update(ctx, req)
		require.ErrorIs(t, err, domain.ErrFinancialAgentNoProvider)
	})

	t.Run("explicit provider missing returns ErrFinancialAgentNoProvider and skips upsert", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mock.NewMockSettingRepository(ctrl)
		resolver := mock.NewMockAdapterResolver(ctrl)
		svc := service.NewSettingService(repo, resolver)

		ctx := newContextWithUserWorkspace(1, 42)
		enabled := true
		provID := int64(99)
		req := model.UpdateSettingRequest{
			FinancialEnabled:    &enabled,
			FinancialProviderID: &provID,
		}

		resolver.EXPECT().ResolveAdapter(ctx, &provID).Return(nil, domain.ErrProviderNotFound)
		repo.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

		_, err := svc.Update(ctx, req)
		require.ErrorIs(t, err, domain.ErrFinancialAgentNoProvider)
	})
}

func TestSettingUpdate_EnableRequiresVisionAndStructuredOutput(t *testing.T) {
	tests := []struct {
		name             string
		vision           bool
		structuredOutput bool
	}{
		{
			name:             "lacks vision",
			vision:           false,
			structuredOutput: true,
		},
		{
			name:             "lacks structured output",
			vision:           true,
			structuredOutput: false,
		},
		{
			name:             "lacks both",
			vision:           false,
			structuredOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mock.NewMockSettingRepository(ctrl)
			resolver := mock.NewMockAdapterResolver(ctrl)
			svc := service.NewSettingService(repo, resolver)

			ctx := newContextWithUserWorkspace(1, 42)
			enabled := true
			provID := int64(10)
			req := model.UpdateSettingRequest{
				FinancialEnabled:    &enabled,
				FinancialProviderID: &provID,
			}

			adapter := &stubCapabilitiesAdapter{
				caps: llmx.Capabilities{
					Vision:           tt.vision,
					StructuredOutput: tt.structuredOutput,
				},
			}

			resolver.EXPECT().ResolveAdapter(ctx, &provID).Return(adapter, nil)
			repo.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			_, err := svc.Update(ctx, req)
			require.ErrorIs(t, err, domain.ErrFinancialAgentNoProvider)
		})
	}
}

func TestSettingUpdate_EnableOK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockSettingRepository(ctrl)
	resolver := mock.NewMockAdapterResolver(ctrl)
	svc := service.NewSettingService(repo, resolver)

	ctx := newContextWithUserWorkspace(1, 42)
	enabled := true
	provID := int64(10)
	req := model.UpdateSettingRequest{
		FinancialEnabled:    &enabled,
		FinancialProviderID: &provID,
	}

	adapter := &stubCapabilitiesAdapter{
		caps: llmx.Capabilities{
			Vision:           true,
			StructuredOutput: true,
		},
	}

	resolver.EXPECT().ResolveAdapter(ctx, &provID).Return(adapter, nil)
	repo.EXPECT().Upsert(ctx, true, &provID).Return(&ent.AgentSetting{
		ID:                  1,
		WorkspaceID:         42,
		FinancialEnabled:    true,
		FinancialProviderID: &provID,
	}, nil)

	res, err := svc.Update(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, domain.Setting{
		WorkspaceID:         42,
		FinancialEnabled:    true,
		FinancialProviderID: &provID,
	}, res)
}

func TestSettingUpdate_DisableSkipsResolution(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockSettingRepository(ctrl)
	resolver := mock.NewMockAdapterResolver(ctrl)
	svc := service.NewSettingService(repo, resolver)

	ctx := newContextWithUserWorkspace(1, 42)
	enabled := false
	provID := int64(999) // points to non-existent or broken provider
	req := model.UpdateSettingRequest{
		FinancialEnabled:    &enabled,
		FinancialProviderID: &provID,
	}

	resolver.EXPECT().ResolveAdapter(gomock.Any(), gomock.Any()).Times(0)
	repo.EXPECT().Upsert(ctx, false, &provID).Return(&ent.AgentSetting{
		ID:                  1,
		WorkspaceID:         42,
		FinancialEnabled:    false,
		FinancialProviderID: &provID,
	}, nil)

	res, err := svc.Update(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, domain.Setting{
		WorkspaceID:         42,
		FinancialEnabled:    false,
		FinancialProviderID: &provID,
	}, res)
}

func TestSettingUpdate_ClearProviderToDefault(t *testing.T) {
	t.Run("disabled with nil provider", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mock.NewMockSettingRepository(ctrl)
		resolver := mock.NewMockAdapterResolver(ctrl)
		svc := service.NewSettingService(repo, resolver)

		ctx := newContextWithUserWorkspace(1, 42)
		enabled := false
		req := model.UpdateSettingRequest{
			FinancialEnabled:    &enabled,
			FinancialProviderID: nil,
		}

		resolver.EXPECT().ResolveAdapter(gomock.Any(), gomock.Any()).Times(0)
		repo.EXPECT().Upsert(ctx, false, (*int64)(nil)).Return(&ent.AgentSetting{
			ID:                  1,
			WorkspaceID:         42,
			FinancialEnabled:    false,
			FinancialProviderID: nil,
		}, nil)

		res, err := svc.Update(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, domain.Setting{
			WorkspaceID:         42,
			FinancialEnabled:    false,
			FinancialProviderID: nil,
		}, res)
	})

	t.Run("enabled with nil provider using default provider", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := mock.NewMockSettingRepository(ctrl)
		resolver := mock.NewMockAdapterResolver(ctrl)
		svc := service.NewSettingService(repo, resolver)

		ctx := newContextWithUserWorkspace(1, 42)
		enabled := true
		req := model.UpdateSettingRequest{
			FinancialEnabled:    &enabled,
			FinancialProviderID: nil,
		}

		adapter := &stubCapabilitiesAdapter{
			caps: llmx.Capabilities{
				Vision:           true,
				StructuredOutput: true,
			},
		}

		resolver.EXPECT().ResolveAdapter(ctx, (*int64)(nil)).Return(adapter, nil)
		repo.EXPECT().Upsert(ctx, true, (*int64)(nil)).Return(&ent.AgentSetting{
			ID:                  1,
			WorkspaceID:         42,
			FinancialEnabled:    true,
			FinancialProviderID: nil,
		}, nil)

		res, err := svc.Update(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, domain.Setting{
			WorkspaceID:         42,
			FinancialEnabled:    true,
			FinancialProviderID: nil,
		}, res)
	})
}

func TestSettingUpdate_ResolverPassThroughErrors(t *testing.T) {
	passThroughErrors := []struct {
		name string
		err  error
	}{
		{
			name: "secret key missing",
			err:  domain.ErrSecretKeyMissing,
		},
		{
			name: "provider key unreadable",
			err:  domain.ErrProviderKeyUnreadable,
		},
	}

	for _, tt := range passThroughErrors {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			repo := mock.NewMockSettingRepository(ctrl)
			resolver := mock.NewMockAdapterResolver(ctrl)
			svc := service.NewSettingService(repo, resolver)

			ctx := newContextWithUserWorkspace(1, 42)
			enabled := true
			provID := int64(10)
			req := model.UpdateSettingRequest{
				FinancialEnabled:    &enabled,
				FinancialProviderID: &provID,
			}

			resolver.EXPECT().ResolveAdapter(ctx, &provID).Return(nil, tt.err)
			repo.EXPECT().Upsert(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			_, err := svc.Update(ctx, req)
			require.ErrorIs(t, err, tt.err)
			assert.NotErrorIs(t, err, domain.ErrFinancialAgentNoProvider)
		})
	}
}

func TestSettingUpdate_RepoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockSettingRepository(ctrl)
	resolver := mock.NewMockAdapterResolver(ctrl)
	svc := service.NewSettingService(repo, resolver)

	ctx := newContextWithUserWorkspace(1, 42)
	enabled := true
	provID := int64(10)
	req := model.UpdateSettingRequest{
		FinancialEnabled:    &enabled,
		FinancialProviderID: &provID,
	}

	adapter := &stubCapabilitiesAdapter{
		caps: llmx.Capabilities{
			Vision:           true,
			StructuredOutput: true,
		},
	}

	resolver.EXPECT().ResolveAdapter(ctx, &provID).Return(adapter, nil)
	dbErr := errors.New("upsert failed")
	repo.EXPECT().Upsert(ctx, true, &provID).Return(nil, dbErr)

	_, err := svc.Update(ctx, req)
	require.ErrorIs(t, err, dbErr)
}
