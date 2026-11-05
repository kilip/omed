package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/cmspage"
	"github.com/kilip/omed/api/ent/cmspagetranslation"
	"github.com/kilip/omed/api/ent/predicate"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
)

// PageRepository defines persistence operations for CMS pages and their translations.
type PageRepository struct {
	cli *ent.Client
}

// NewPageRepository constructs a PageRepository backed by the given Ent client.
func NewPageRepository(cli *ent.Client) PageRepository {
	return PageRepository{cli: cli}
}

// VisiblePageTranslationPredicate returns an Ent predicate matching publicly visible page translations at time now.
func VisiblePageTranslationPredicate(now time.Time) predicate.CmsPageTranslation {
	return cmspagetranslation.And(
		cmspagetranslation.DeletedAtIsNil(),
		cmspagetranslation.StatusEQ(domain.StatusPublished),
		cmspagetranslation.PublishAtLTE(now.UTC()),
	)
}

// GetByID retrieves a page by ID within the caller's workspace context with translations.
func (r PageRepository) GetByID(c context.Context, id int64, requestedLocale, defaultLocale string, now time.Time) (domain.Page, error) {
	var dom domain.Page

	p, err := r.cli.CmsPage.Query().
		Where(
			cmspage.ID(id),
			cmspage.DeletedAtIsNil(),
		).
		WithTranslations(func(tq *ent.CmsPageTranslationQuery) {
			tq.Where(cmspagetranslation.DeletedAtIsNil())
		}).
		WithCreator().
		WithUpdater().
		WithDeleter().
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom, domain.ErrPageNotFound
		}
		return dom, err
	}

	return toPageDomain(p, requestedLocale, defaultLocale, now), nil
}

// SlugExistsInLocale checks if an active page translation with the given slug exists in workspace and locale.
func (r PageRepository) SlugExistsInLocale(c context.Context, workspaceID int64, locale, slug string, excludePageID *int64) (bool, error) {
	q := r.cli.CmsPageTranslation.Query().
		Where(
			cmspagetranslation.WorkspaceIDEQ(workspaceID),
			cmspagetranslation.LocaleEQ(locale),
			cmspagetranslation.SlugEQ(slug),
			cmspagetranslation.DeletedAtIsNil(),
		)
	if excludePageID != nil && *excludePageID > 0 {
		q = q.Where(cmspagetranslation.PageIDNEQ(*excludePageID))
	}
	return q.Exist(c)
}

// Create inserts a new cms_page and its initial translation record.
func (r PageRepository) Create(c context.Context, req model.CreatePageRecord, defaultLocale string, now time.Time) (domain.Page, error) {
	tx, err := r.cli.Tx(c)
	if err != nil {
		return domain.Page{}, err
	}
	defer func() { _ = tx.Rollback() }()

	builder := tx.CmsPage.Create()
	if req.WorkspaceID > 0 {
		builder.SetWorkspaceID(req.WorkspaceID)
	}

	page, err := builder.Save(c)
	if err != nil {
		return domain.Page{}, err
	}

	tBuilder := tx.CmsPageTranslation.Create().
		SetPageID(page.ID).
		SetWorkspaceID(page.WorkspaceID).
		SetLocale(req.Locale).
		SetTitle(req.Title).
		SetSlug(req.Slug).
		SetBody(req.Body).
		SetStatus(cmspagetranslation.Status(req.Status))

	if req.PublishAt != nil {
		tBuilder.SetPublishAt(req.PublishAt.UTC())
	}

	_, err = tBuilder.Save(c)
	if err != nil {
		if ent.IsConstraintError(err) {
			return domain.Page{}, domain.ErrSlugTaken
		}
		return domain.Page{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Page{}, err
	}

	return r.GetByID(c, page.ID, req.Locale, defaultLocale, now)
}

// Update modifies an existing page and/or its translation.
func (r PageRepository) Update(c context.Context, id int64, req model.UpdatePageRecord, defaultLocale string, now time.Time) (domain.Page, error) {
	p, err := r.cli.CmsPage.Query().
		Where(
			cmspage.ID(id),
			cmspage.DeletedAtIsNil(),
		).
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.Page{}, domain.ErrPageNotFound
		}
		return domain.Page{}, err
	}

	locale := req.Locale
	if locale == "" {
		locale = defaultLocale
	}

	hasTranslationUpdate := req.Title != nil || req.Slug != nil || req.Body != nil || req.Status != nil || req.PublishAt != nil || req.ClearPublishAt
	if hasTranslationUpdate {
		t, err := r.cli.CmsPageTranslation.Query().
			Where(
				cmspagetranslation.PageIDEQ(id),
				cmspagetranslation.LocaleEQ(locale),
				cmspagetranslation.DeletedAtIsNil(),
			).
			Only(c)

		if err != nil {
			if ent.IsNotFound(err) {
				title := ""
				if req.Title != nil {
					title = *req.Title
				}
				slug := ""
				if req.Slug != nil {
					slug = *req.Slug
				}
				body := ""
				if req.Body != nil {
					body = *req.Body
				}
				status := domain.StatusDraft
				if req.Status != nil {
					status = *req.Status
				}

				tBuilder := r.cli.CmsPageTranslation.Create().
					SetPageID(id).
					SetWorkspaceID(p.WorkspaceID).
					SetLocale(locale).
					SetTitle(title).
					SetSlug(slug).
					SetBody(body).
					SetStatus(cmspagetranslation.Status(status))

				if req.PublishAt != nil {
					tBuilder.SetPublishAt(req.PublishAt.UTC())
				}

				if _, err := tBuilder.Save(c); err != nil {
					if ent.IsConstraintError(err) {
						return domain.Page{}, domain.ErrSlugTaken
					}
					return domain.Page{}, err
				}
			} else {
				return domain.Page{}, err
			}
		} else {
			tBuilder := r.cli.CmsPageTranslation.UpdateOneID(t.ID)
			if req.Title != nil {
				tBuilder.SetTitle(*req.Title)
			}
			if req.Slug != nil {
				tBuilder.SetSlug(*req.Slug)
			}
			if req.Body != nil {
				tBuilder.SetBody(*req.Body)
			}
			if req.Status != nil {
				tBuilder.SetStatus(cmspagetranslation.Status(*req.Status))
			}
			if req.ClearPublishAt {
				tBuilder.ClearPublishAt()
			} else if req.PublishAt != nil {
				tBuilder.SetPublishAt(req.PublishAt.UTC())
			}

			if err := tBuilder.Exec(c); err != nil {
				if ent.IsConstraintError(err) {
					return domain.Page{}, domain.ErrSlugTaken
				}
				return domain.Page{}, err
			}
		}
	}

	return r.GetByID(c, id, locale, defaultLocale, now)
}

// SaveTranslation creates or updates a translation for a page.
func (r PageRepository) SaveTranslation(c context.Context, req model.SavePageTranslationRecord, defaultLocale string, now time.Time) (domain.Page, error) {
	p, err := r.cli.CmsPage.Query().
		Where(
			cmspage.ID(req.PageID),
			cmspage.DeletedAtIsNil(),
		).
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.Page{}, domain.ErrPageNotFound
		}
		return domain.Page{}, err
	}

	t, err := r.cli.CmsPageTranslation.Query().
		Where(
			cmspagetranslation.PageIDEQ(req.PageID),
			cmspagetranslation.LocaleEQ(req.Locale),
			cmspagetranslation.DeletedAtIsNil(),
		).
		Only(c)

	if err != nil {
		if ent.IsNotFound(err) {
			status := req.Status
			if status == "" {
				status = domain.StatusDraft
			}
			builder := r.cli.CmsPageTranslation.Create().
				SetPageID(req.PageID).
				SetWorkspaceID(p.WorkspaceID).
				SetLocale(req.Locale).
				SetTitle(req.Title).
				SetSlug(req.Slug).
				SetBody(req.Body).
				SetStatus(cmspagetranslation.Status(status))

			if req.PublishAt != nil {
				builder.SetPublishAt(req.PublishAt.UTC())
			}

			if _, err := builder.Save(c); err != nil {
				if ent.IsConstraintError(err) {
					return domain.Page{}, domain.ErrSlugTaken
				}
				return domain.Page{}, err
			}
		} else {
			return domain.Page{}, err
		}
	} else {
		builder := r.cli.CmsPageTranslation.UpdateOneID(t.ID).
			SetTitle(req.Title).
			SetSlug(req.Slug).
			SetBody(req.Body)

		if err := builder.Exec(c); err != nil {
			if ent.IsConstraintError(err) {
				return domain.Page{}, domain.ErrSlugTaken
			}
			return domain.Page{}, err
		}
	}

	return r.GetByID(c, req.PageID, req.Locale, defaultLocale, now)
}

// DeleteTranslation soft-deletes a single translation for a page.
func (r PageRepository) DeleteTranslation(c context.Context, pageID int64, locale string) error {
	t, err := r.cli.CmsPageTranslation.Query().
		Where(
			cmspagetranslation.PageIDEQ(pageID),
			cmspagetranslation.LocaleEQ(locale),
			cmspagetranslation.DeletedAtIsNil(),
		).
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.ErrTranslationNotFound
		}
		return err
	}

	return r.cli.CmsPageTranslation.UpdateOneID(t.ID).
		SetDeletedAt(time.Now()).
		Exec(c)
}

// Delete marks a page and all its translations as soft-deleted.
func (r PageRepository) Delete(c context.Context, id int64) error {
	exists, err := r.cli.CmsPage.Query().
		Where(
			cmspage.ID(id),
			cmspage.DeletedAtIsNil(),
		).
		Exist(c)
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrPageNotFound
	}

	tx, err := r.cli.Tx(c)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	if err := tx.CmsPage.UpdateOneID(id).SetDeletedAt(now).Exec(c); err != nil {
		return err
	}

	_, err = tx.CmsPageTranslation.Update().
		Where(
			cmspagetranslation.PageIDEQ(id),
			cmspagetranslation.DeletedAtIsNil(),
		).
		SetDeletedAt(now).
		Save(c)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Search searches active pages matching the criteria with admin cursor pagination.
func (r PageRepository) Search(c context.Context, req model.SearchPageRequest, defaultLocale string, now time.Time) ([]domain.Page, string, error) {
	limit := 20
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
		if limit > 100 {
			limit = 100
		}
	}

	q := r.cli.CmsPage.Query().
		Where(cmspage.DeletedAtIsNil())

	targetLocale := defaultLocale
	if req.Locale != nil && *req.Locale != "" {
		targetLocale = *req.Locale
	}

	if req.Query != nil && *req.Query != "" {
		q = q.Where(cmspage.HasTranslationsWith(
			cmspagetranslation.LocaleEQ(targetLocale),
			cmspagetranslation.DeletedAtIsNil(),
			cmspagetranslation.Or(
				cmspagetranslation.TitleContainsFold(*req.Query),
				cmspagetranslation.SlugContainsFold(*req.Query),
			),
		))
	}

	if req.Status != nil && *req.Status != "" {
		switch *req.Status {
		case domain.EffectiveStatusDraft:
			q = q.Where(cmspage.HasTranslationsWith(
				cmspagetranslation.LocaleEQ(targetLocale),
				cmspagetranslation.DeletedAtIsNil(),
				cmspagetranslation.StatusEQ(domain.StatusDraft),
			))
		case domain.EffectiveStatusScheduled:
			q = q.Where(cmspage.HasTranslationsWith(
				cmspagetranslation.LocaleEQ(targetLocale),
				cmspagetranslation.DeletedAtIsNil(),
				cmspagetranslation.StatusEQ(domain.StatusPublished),
				cmspagetranslation.PublishAtGT(now.UTC()),
			))
		case domain.EffectiveStatusPublished:
			q = q.Where(cmspage.HasTranslationsWith(
				cmspagetranslation.LocaleEQ(targetLocale),
				cmspagetranslation.DeletedAtIsNil(),
				cmspagetranslation.StatusEQ(domain.StatusPublished),
				cmspagetranslation.PublishAtLTE(now.UTC()),
			))
		}
	}

	if req.Cursor != nil && *req.Cursor != "" {
		cursorID, err := strconv.ParseInt(*req.Cursor, 10, 64)
		if err != nil || cursorID < 0 {
			return nil, "", base.ErrInvalidCursor
		}
		q = q.Where(cmspage.IDLT(cursorID))
	}

	items, err := q.
		Order(ent.Desc(cmspage.FieldID)).
		Limit(limit + 1).
		WithTranslations(func(tq *ent.CmsPageTranslationQuery) {
			tq.Where(cmspagetranslation.DeletedAtIsNil())
		}).
		WithCreator().
		WithUpdater().
		WithDeleter().
		All(c)
	if err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(items) > limit {
		nextCursor = strconv.FormatInt(items[limit-1].ID, 10)
		items = items[:limit]
	}

	res := make([]domain.Page, len(items))
	for i, it := range items {
		res[i] = toPageDomain(it, targetLocale, defaultLocale, now)
	}

	return res, nextCursor, nil
}

// PublicList retrieves publicly visible pages for a workspace with cursor pagination.
func (r PageRepository) PublicList(c context.Context, workspaceID int64, locale, defaultLocale string, cursor *string, limit int, now time.Time) ([]domain.Page, string, error) {
	if base.WorkspaceIDFromContext(c) <= 0 && workspaceID > 0 {
		c = base.WithWorkspaceID(c, workspaceID)
	}
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	q := r.cli.CmsPage.Query().
		Where(
			cmspage.WorkspaceIDEQ(workspaceID),
			cmspage.DeletedAtIsNil(),
			cmspage.Or(
				cmspage.HasTranslationsWith(
					cmspagetranslation.LocaleEQ(locale),
					VisiblePageTranslationPredicate(now),
				),
				cmspage.HasTranslationsWith(
					cmspagetranslation.LocaleEQ(defaultLocale),
					VisiblePageTranslationPredicate(now),
				),
			),
		)

	if cursor != nil && *cursor != "" {
		cursorID, err := strconv.ParseInt(*cursor, 10, 64)
		if err != nil || cursorID < 0 {
			return nil, "", base.ErrInvalidCursor
		}
		q = q.Where(cmspage.IDLT(cursorID))
	}

	items, err := q.
		Order(ent.Desc(cmspage.FieldID)).
		Limit(limit + 1).
		WithTranslations(func(tq *ent.CmsPageTranslationQuery) {
			tq.Where(cmspagetranslation.DeletedAtIsNil())
		}).
		WithCreator().
		WithUpdater().
		WithDeleter().
		All(c)
	if err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(items) > limit {
		nextCursor = strconv.FormatInt(items[limit-1].ID, 10)
		items = items[:limit]
	}

	res := make([]domain.Page, len(items))
	for i, it := range items {
		res[i] = toPageDomain(it, locale, defaultLocale, now)
	}

	return res, nextCursor, nil
}

// PublicGetBySlug retrieves a publicly visible page by slug for a workspace, returning domain.Page, fallback bool, and alternates.
func (r PageRepository) PublicGetBySlug(c context.Context, workspaceID int64, locale, defaultLocale, slug string, now time.Time) (domain.Page, bool, []model.PublicAlternate, error) {
	if base.WorkspaceIDFromContext(c) <= 0 && workspaceID > 0 {
		c = base.WithWorkspaceID(c, workspaceID)
	}
	var dom domain.Page

	// Step 1: Look for published translation in requested locale with slug
	p, err := r.cli.CmsPage.Query().
		Where(
			cmspage.WorkspaceIDEQ(workspaceID),
			cmspage.DeletedAtIsNil(),
			cmspage.HasTranslationsWith(
				cmspagetranslation.LocaleEQ(locale),
				cmspagetranslation.SlugEQ(slug),
				VisiblePageTranslationPredicate(now),
			),
		).
		WithTranslations(func(tq *ent.CmsPageTranslationQuery) {
			tq.Where(cmspagetranslation.DeletedAtIsNil())
		}).
		WithCreator().
		WithUpdater().
		WithDeleter().
		Only(c)

	fallback := false
	if err != nil {
		if !ent.IsNotFound(err) {
			return dom, false, nil, err
		}

		// Step 2: Try resolving slug in defaultLocale
		p, err = r.cli.CmsPage.Query().
			Where(
				cmspage.WorkspaceIDEQ(workspaceID),
				cmspage.DeletedAtIsNil(),
				cmspage.HasTranslationsWith(
					cmspagetranslation.LocaleEQ(defaultLocale),
					cmspagetranslation.SlugEQ(slug),
					VisiblePageTranslationPredicate(now),
				),
			).
			WithTranslations(func(tq *ent.CmsPageTranslationQuery) {
				tq.Where(cmspagetranslation.DeletedAtIsNil())
			}).
			WithCreator().
			WithUpdater().
			WithDeleter().
			Only(c)

		if err != nil {
			if ent.IsNotFound(err) {
				return dom, false, nil, domain.ErrPageNotFound
			}
			return dom, false, nil, err
		}
		fallback = true
	}

	resolvedLocale := locale
	if fallback {
		resolvedLocale = defaultLocale
	}

	dom = toPageDomain(p, resolvedLocale, defaultLocale, now)

	// Collect published alternates
	var alternates []model.PublicAlternate
	for _, t := range p.Edges.Translations {
		if t.DeletedAt == nil && t.Status == "PUBLISHED" && (t.PublishAt == nil || !t.PublishAt.After(now.UTC())) {
			alternates = append(alternates, model.PublicAlternate{
				Locale: t.Locale,
				Slug:   t.Slug,
				Path:   fmt.Sprintf("/%s/%s", t.Locale, t.Slug),
			})
		}
	}

	return dom, fallback, alternates, nil
}

func toPageDomain(p *ent.CmsPage, requestedLocale, defaultLocale string, now time.Time) domain.Page {
	dom := domain.Page{
		ID:          p.ID,
		WorkspaceID: p.WorkspaceID,
		AuditMeta: base.AuditMeta{
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
			DeletedAt: p.DeletedAt,
		},
	}

	if p.Edges.Creator != nil {
		dom.AuditMeta.CreatedBy = base.ByUser{
			Name:   p.Edges.Creator.Name,
			Avatar: p.Edges.Creator.Avatar,
		}
	}
	if p.Edges.Updater != nil {
		dom.AuditMeta.UpdatedBy = base.ByUser{
			Name:   p.Edges.Updater.Name,
			Avatar: p.Edges.Updater.Avatar,
		}
	}
	if p.Edges.Deleter != nil {
		dom.AuditMeta.DeletedBy = &base.ByUser{
			Name:   p.Edges.Deleter.Name,
			Avatar: p.Edges.Deleter.Avatar,
		}
	}

	var active []*ent.CmsPageTranslation
	for _, t := range p.Edges.Translations {
		if t.DeletedAt == nil {
			active = append(active, t)
		}
	}

	dom.Translations = make([]domain.TranslationSummary, len(active))
	var matchRequested, matchDefault, fallback *ent.CmsPageTranslation

	for i, t := range active {
		dom.Translations[i] = domain.TranslationSummary{
			Locale:          t.Locale,
			Slug:            t.Slug,
			Title:           t.Title,
			Status:          string(t.Status),
			EffectiveStatus: domain.EffectiveStatus(string(t.Status), t.PublishAt, now),
			PublishAt:       t.PublishAt,
		}

		if strings.EqualFold(t.Locale, requestedLocale) {
			matchRequested = t
		}
		if strings.EqualFold(t.Locale, defaultLocale) {
			matchDefault = t
		}
		if fallback == nil {
			fallback = t
		}
	}

	selected := matchRequested
	if selected == nil {
		selected = matchDefault
	}
	if selected == nil {
		selected = fallback
	}

	if selected != nil {
		dom.Locale = selected.Locale
		dom.Title = selected.Title
		dom.Slug = selected.Slug
		dom.Body = selected.Body
		dom.Status = string(selected.Status)
		dom.EffectiveStatus = domain.EffectiveStatus(string(selected.Status), selected.PublishAt, now)
		dom.PublishAt = selected.PublishAt
	}

	return dom
}
