package repository

import (
	"context"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/cmspage"
	"github.com/kilip/omed/api/ent/cmspagetranslation"
	"github.com/kilip/omed/api/ent/cmspost"
	"github.com/kilip/omed/api/ent/cmsposttranslation"
	"github.com/kilip/omed/api/ent/cmssetting"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/domain"
)

// SettingRepository defines persistence operations for CMS workspace settings.
type SettingRepository struct {
	cli *ent.Client
}

// NewSettingRepository constructs a SettingRepository instance.
func NewSettingRepository(cli *ent.Client) SettingRepository {
	return SettingRepository{cli: cli}
}

// Get retrieves CMS settings for the given workspace, or default settings if none exist.
func (r SettingRepository) Get(ctx context.Context, workspaceID int64) (domain.Setting, error) {
	s, err := r.cli.CmsSetting.Query().
		Where(cmssetting.WorkspaceIDEQ(workspaceID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.Setting{
				WorkspaceID:      workspaceID,
				DefaultLocale:    "en",
				SupportedLocales: []string{"en", "id"},
			}, nil
		}
		return domain.Setting{}, err
	}
	return domain.Setting{
		WorkspaceID:      s.WorkspaceID,
		DefaultLocale:    s.DefaultLocale,
		SupportedLocales: s.SupportedLocales,
	}, nil
}

// Upsert creates or updates CMS settings for the given workspace.
func (r SettingRepository) Upsert(ctx context.Context, workspaceID int64, defaultLocale string, supportedLocales []string) (domain.Setting, error) {
	s, err := r.cli.CmsSetting.Query().
		Where(cmssetting.WorkspaceIDEQ(workspaceID)).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			builder := r.cli.CmsSetting.Create().
				SetWorkspaceID(workspaceID).
				SetDefaultLocale(defaultLocale).
				SetSupportedLocales(supportedLocales)

			if userID := base.UserIDFromContext(ctx); userID > 0 {
				builder.SetCreatedBy(userID).SetUpdatedBy(userID)
			} else {
				builder.SetCreatedBy(0).SetUpdatedBy(0)
			}

			created, err := builder.Save(ctx)
			if err != nil {
				return domain.Setting{}, err
			}
			return domain.Setting{
				WorkspaceID:      created.WorkspaceID,
				DefaultLocale:    created.DefaultLocale,
				SupportedLocales: created.SupportedLocales,
			}, nil
		}
		return domain.Setting{}, err
	}

	builder := r.cli.CmsSetting.UpdateOneID(s.ID).
		SetDefaultLocale(defaultLocale).
		SetSupportedLocales(supportedLocales)

	if userID := base.UserIDFromContext(ctx); userID > 0 {
		builder.SetUpdatedBy(userID)
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		return domain.Setting{}, err
	}

	return domain.Setting{
		WorkspaceID:      updated.WorkspaceID,
		DefaultLocale:    updated.DefaultLocale,
		SupportedLocales: updated.SupportedLocales,
	}, nil
}

// CountMissingTranslations returns count of non-deleted posts and pages missing translation in targetLocale.
func (r SettingRepository) CountMissingTranslations(ctx context.Context, workspaceID int64, targetLocale string) (int, int, error) {
	missingPosts, err := r.cli.CmsPost.Query().
		Where(
			cmspost.WorkspaceIDEQ(workspaceID),
			cmspost.DeletedAtIsNil(),
			cmspost.Not(
				cmspost.HasTranslationsWith(
					cmsposttranslation.LocaleEQ(targetLocale),
					cmsposttranslation.DeletedAtIsNil(),
				),
			),
		).
		Count(ctx)
	if err != nil {
		return 0, 0, err
	}

	missingPages, err := r.cli.CmsPage.Query().
		Where(
			cmspage.WorkspaceIDEQ(workspaceID),
			cmspage.DeletedAtIsNil(),
			cmspage.Not(
				cmspage.HasTranslationsWith(
					cmspagetranslation.LocaleEQ(targetLocale),
					cmspagetranslation.DeletedAtIsNil(),
				),
			),
		).
		Count(ctx)
	if err != nil {
		return 0, 0, err
	}

	return missingPosts, missingPages, nil
}

// CountTranslationsForLocale returns total translations across posts and pages for the given locale.
func (r SettingRepository) CountTranslationsForLocale(ctx context.Context, workspaceID int64, locale string) (int, error) {
	postCount, err := r.cli.CmsPostTranslation.Query().
		Where(
			cmsposttranslation.WorkspaceIDEQ(workspaceID),
			cmsposttranslation.LocaleEQ(locale),
			cmsposttranslation.DeletedAtIsNil(),
		).
		Count(ctx)
	if err != nil {
		return 0, err
	}

	pageCount, err := r.cli.CmsPageTranslation.Query().
		Where(
			cmspagetranslation.WorkspaceIDEQ(workspaceID),
			cmspagetranslation.LocaleEQ(locale),
			cmspagetranslation.DeletedAtIsNil(),
		).
		Count(ctx)
	if err != nil {
		return 0, err
	}

	return postCount + pageCount, nil
}
