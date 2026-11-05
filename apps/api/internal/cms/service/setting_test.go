package service_test

import (
	"context"
	"testing"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	"github.com/kilip/omed/api/internal/cms/service"
	"github.com/kilip/omed/api/internal/cms/service/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSettingService_Get(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockSettingRepository(ctrl)
	svc := service.NewSettingService(repo)

	ctx := base.WithWorkspaceID(context.Background(), 10)
	expected := domain.Setting{
		WorkspaceID:      10,
		DefaultLocale:    "en",
		SupportedLocales: []string{"en", "id"},
	}

	repo.EXPECT().Get(gomock.Any(), int64(10)).Return(expected, nil)

	res, err := svc.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, expected, res)
}

func TestSettingService_Update(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := mock.NewMockSettingRepository(ctrl)
	svc := service.NewSettingService(repo)

	ctx := base.WithWorkspaceID(context.Background(), 10)
	current := domain.Setting{
		WorkspaceID:      10,
		DefaultLocale:    "en",
		SupportedLocales: []string{"en", "id"},
	}

	t.Run("valid update", func(t *testing.T) {
		repo.EXPECT().Get(gomock.Any(), int64(10)).Return(current, nil)
		repo.EXPECT().CountMissingTranslations(gomock.Any(), int64(10), "id").Return(0, 0, nil)

		newDef := "id"
		newSupp := []string{"id", "en", "ja"}
		expected := domain.Setting{
			WorkspaceID:      10,
			DefaultLocale:    "id",
			SupportedLocales: newSupp,
		}
		repo.EXPECT().Upsert(gomock.Any(), int64(10), "id", newSupp).Return(expected, nil)

		res, err := svc.Update(ctx, model.UpdateCmsSettingRequest{
			DefaultLocale:    &newDef,
			SupportedLocales: newSupp,
		})
		require.NoError(t, err)
		assert.Equal(t, "id", res.DefaultLocale)
	})

	t.Run("default locale not in supported locales", func(t *testing.T) {
		repo.EXPECT().Get(gomock.Any(), int64(10)).Return(current, nil)

		newDef := "fr"
		_, err := svc.Update(ctx, model.UpdateCmsSettingRequest{
			DefaultLocale: &newDef,
		})
		assert.ErrorIs(t, err, domain.ErrLocaleNotSupported)
	})

	t.Run("default locale change blocked if missing translations", func(t *testing.T) {
		repo.EXPECT().Get(gomock.Any(), int64(10)).Return(current, nil)
		repo.EXPECT().CountMissingTranslations(gomock.Any(), int64(10), "id").Return(2, 0, nil)

		newDef := "id"
		_, err := svc.Update(ctx, model.UpdateCmsSettingRequest{
			DefaultLocale: &newDef,
		})
		assert.ErrorIs(t, err, domain.ErrDefaultLocaleChangeBlocked)
	})

	t.Run("locale removal blocked if translations exist", func(t *testing.T) {
		repo.EXPECT().Get(gomock.Any(), int64(10)).Return(current, nil)
		repo.EXPECT().CountTranslationsForLocale(gomock.Any(), int64(10), "id").Return(5, nil)

		newSupp := []string{"en"} // removing "id"
		_, err := svc.Update(ctx, model.UpdateCmsSettingRequest{
			SupportedLocales: newSupp,
		})
		assert.ErrorIs(t, err, domain.ErrLocaleInUse)
	})
}
