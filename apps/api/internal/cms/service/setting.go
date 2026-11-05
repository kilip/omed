//go:generate go run go.uber.org/mock/mockgen -source=setting.go -destination=mock/mock_setting.go -package=mock

package service

import (
	"context"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
)

// SettingRepository defines persistence operations for CMS workspace settings.
type SettingRepository interface {
	Get(ctx context.Context, workspaceID int64) (domain.Setting, error)
	Upsert(ctx context.Context, workspaceID int64, defaultLocale string, supportedLocales []string) (domain.Setting, error)
	CountMissingTranslations(ctx context.Context, workspaceID int64, targetLocale string) (int, int, error)
	CountTranslationsForLocale(ctx context.Context, workspaceID int64, locale string) (int, error)
}

// SettingService provides business logic for CMS settings.
type SettingService struct {
	repo SettingRepository
}

// NewSettingService constructs a new SettingService instance.
func NewSettingService(repo SettingRepository) SettingService {
	return SettingService{repo: repo}
}

// Get retrieves CMS settings for the current workspace context.
func (s SettingService) Get(ctx context.Context) (domain.Setting, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID <= 0 {
		return domain.Setting{}, base.ErrWorkspaceRequired
	}
	return s.repo.Get(ctx, wsID)
}

// Update validates and updates CMS settings for the current workspace context.
func (s SettingService) Update(ctx context.Context, req model.UpdateCmsSettingRequest) (domain.Setting, error) {
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID <= 0 {
		return domain.Setting{}, base.ErrWorkspaceRequired
	}

	current, err := s.repo.Get(ctx, wsID)
	if err != nil {
		return domain.Setting{}, err
	}

	targetDefault := current.DefaultLocale
	if req.DefaultLocale != nil {
		norm, err := domain.NormalizeLocale(*req.DefaultLocale)
		if err != nil {
			return domain.Setting{}, domain.ErrInvalidLocale
		}
		targetDefault = norm
	}

	targetSupported := current.SupportedLocales
	if req.SupportedLocales != nil {
		if len(req.SupportedLocales) == 0 {
			return domain.Setting{}, domain.ErrInvalidLocale
		}
		seen := make(map[string]bool)
		normalizedList := make([]string, 0, len(req.SupportedLocales))
		for _, l := range req.SupportedLocales {
			norm, err := domain.NormalizeLocale(l)
			if err != nil {
				return domain.Setting{}, domain.ErrInvalidLocale
			}
			if !seen[norm] {
				seen[norm] = true
				normalizedList = append(normalizedList, norm)
			}
		}
		targetSupported = normalizedList
	}

	// Invariant: default_locale must be in supported_locales
	if !domain.IsLocaleSupported(targetDefault, targetSupported) {
		return domain.Setting{}, domain.ErrLocaleNotSupported
	}

	// Invariant: If default_locale changed, ensure all existing non-deleted posts & pages have a translation
	if targetDefault != current.DefaultLocale {
		missingPosts, missingPages, err := s.repo.CountMissingTranslations(ctx, wsID, targetDefault)
		if err != nil {
			return domain.Setting{}, err
		}
		if missingPosts > 0 || missingPages > 0 {
			return domain.Setting{}, domain.ErrDefaultLocaleChangeBlocked
		}
	}

	// Invariant: If a locale was removed from supported_locales, ensure no translations exist for it
	for _, oldLoc := range current.SupportedLocales {
		if !domain.IsLocaleSupported(oldLoc, targetSupported) {
			count, err := s.repo.CountTranslationsForLocale(ctx, wsID, oldLoc)
			if err != nil {
				return domain.Setting{}, err
			}
			if count > 0 {
				return domain.Setting{}, domain.ErrLocaleInUse
			}
		}
	}

	return s.repo.Upsert(ctx, wsID, targetDefault, targetSupported)
}
