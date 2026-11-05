package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/cmspost"
	"github.com/kilip/omed/api/ent/cmsposttranslation"
	"github.com/kilip/omed/api/ent/predicate"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
)

// PostRepository defines persistence operations for CMS posts and their translations.
type PostRepository struct {
	cli *ent.Client
}

// NewPostRepository constructs a PostRepository backed by the given Ent client.
func NewPostRepository(cli *ent.Client) PostRepository {
	return PostRepository{cli: cli}
}

// VisibleTranslationPredicate returns an Ent predicate matching publicly visible translations at time now.
func VisibleTranslationPredicate(now time.Time) predicate.CmsPostTranslation {
	return cmsposttranslation.And(
		cmsposttranslation.DeletedAtIsNil(),
		cmsposttranslation.StatusEQ(domain.StatusPublished),
		cmsposttranslation.PublishAtLTE(now.UTC()),
	)
}

// GetByID retrieves a post by ID within the caller's workspace context with translations.
func (r PostRepository) GetByID(c context.Context, id int64, requestedLocale, defaultLocale string, now time.Time) (domain.Post, error) {
	var dom domain.Post

	p, err := r.cli.CmsPost.Query().
		Where(
			cmspost.ID(id),
			cmspost.DeletedAtIsNil(),
		).
		WithTranslations(func(tq *ent.CmsPostTranslationQuery) {
			tq.Where(cmsposttranslation.DeletedAtIsNil())
		}).
		WithCoverMedia(func(mq *ent.CmsMediaQuery) {
			mq.WithFile(func(fq *ent.FileQuery) {
				fq.WithCreator().WithUpdater()
			}).WithCreator().WithUpdater().WithDeleter()
		}).
		WithCreator().
		WithUpdater().
		WithDeleter().
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom, domain.ErrPostNotFound
		}
		return dom, err
	}

	return toPostDomain(p, requestedLocale, defaultLocale, now), nil
}

// SlugExistsInLocale checks if an active post translation with the given slug exists in workspace and locale.
func (r PostRepository) SlugExistsInLocale(c context.Context, workspaceID int64, locale, slug string, excludePostID *int64) (bool, error) {
	q := r.cli.CmsPostTranslation.Query().
		Where(
			cmsposttranslation.WorkspaceIDEQ(workspaceID),
			cmsposttranslation.LocaleEQ(locale),
			cmsposttranslation.SlugEQ(slug),
			cmsposttranslation.DeletedAtIsNil(),
		)
	if excludePostID != nil && *excludePostID > 0 {
		q = q.Where(cmsposttranslation.PostIDNEQ(*excludePostID))
	}
	return q.Exist(c)
}

// Create inserts a new cms_post and its initial translation record.
func (r PostRepository) Create(c context.Context, req model.CreatePostRecord, defaultLocale string, now time.Time) (domain.Post, error) {
	tx, err := r.cli.Tx(c)
	if err != nil {
		return domain.Post{}, err
	}
	defer func() { _ = tx.Rollback() }()

	builder := tx.CmsPost.Create()
	if req.WorkspaceID > 0 {
		builder.SetWorkspaceID(req.WorkspaceID)
	}
	if req.CoverMediaID != nil {
		builder.SetCoverMediaID(*req.CoverMediaID)
	}

	post, err := builder.Save(c)
	if err != nil {
		return domain.Post{}, err
	}

	tBuilder := tx.CmsPostTranslation.Create().
		SetPostID(post.ID).
		SetWorkspaceID(post.WorkspaceID).
		SetLocale(req.Locale).
		SetTitle(req.Title).
		SetSlug(req.Slug).
		SetBody(req.Body).
		SetStatus(cmsposttranslation.Status(req.Status))

	if req.Excerpt != "" {
		tBuilder.SetExcerpt(req.Excerpt)
	}
	if req.PublishAt != nil {
		tBuilder.SetPublishAt(req.PublishAt.UTC())
	}

	_, err = tBuilder.Save(c)
	if err != nil {
		if ent.IsConstraintError(err) {
			return domain.Post{}, domain.ErrSlugTaken
		}
		return domain.Post{}, err
	}

	if err := tx.Commit(); err != nil {
		return domain.Post{}, err
	}

	return r.GetByID(c, post.ID, req.Locale, defaultLocale, now)
}

// Update modifies an existing post and/or its translation.
func (r PostRepository) Update(c context.Context, id int64, req model.UpdatePostRecord, defaultLocale string, now time.Time) (domain.Post, error) {
	p, err := r.cli.CmsPost.Query().
		Where(
			cmspost.ID(id),
			cmspost.DeletedAtIsNil(),
		).
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.Post{}, domain.ErrPostNotFound
		}
		return domain.Post{}, err
	}

	if req.ClearCoverMedia {
		if err := r.cli.CmsPost.UpdateOneID(id).ClearCoverMediaID().Exec(c); err != nil {
			return domain.Post{}, err
		}
	} else if req.CoverMediaID != nil {
		if err := r.cli.CmsPost.UpdateOneID(id).SetCoverMediaID(*req.CoverMediaID).Exec(c); err != nil {
			return domain.Post{}, err
		}
	}

	locale := req.Locale
	if locale == "" {
		locale = defaultLocale
	}

	hasTranslationUpdate := req.Title != nil || req.Slug != nil || req.Excerpt != nil || req.Body != nil || req.Status != nil || req.PublishAt != nil || req.ClearPublishAt
	if hasTranslationUpdate {
		t, err := r.cli.CmsPostTranslation.Query().
			Where(
				cmsposttranslation.PostIDEQ(id),
				cmsposttranslation.LocaleEQ(locale),
				cmsposttranslation.DeletedAtIsNil(),
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
				excerpt := ""
				if req.Excerpt != nil {
					excerpt = *req.Excerpt
				}
				status := domain.StatusDraft
				if req.Status != nil {
					status = *req.Status
				}

				tBuilder := r.cli.CmsPostTranslation.Create().
					SetPostID(id).
					SetWorkspaceID(p.WorkspaceID).
					SetLocale(locale).
					SetTitle(title).
					SetSlug(slug).
					SetBody(body).
					SetStatus(cmsposttranslation.Status(status))

				if excerpt != "" {
					tBuilder.SetExcerpt(excerpt)
				}
				if req.PublishAt != nil {
					tBuilder.SetPublishAt(req.PublishAt.UTC())
				}

				if _, err := tBuilder.Save(c); err != nil {
					if ent.IsConstraintError(err) {
						return domain.Post{}, domain.ErrSlugTaken
					}
					return domain.Post{}, err
				}
			} else {
				return domain.Post{}, err
			}
		} else {
			tBuilder := r.cli.CmsPostTranslation.UpdateOneID(t.ID)
			if req.Title != nil {
				tBuilder.SetTitle(*req.Title)
			}
			if req.Slug != nil {
				tBuilder.SetSlug(*req.Slug)
			}
			if req.Excerpt != nil {
				tBuilder.SetExcerpt(*req.Excerpt)
			}
			if req.Body != nil {
				tBuilder.SetBody(*req.Body)
			}
			if req.Status != nil {
				tBuilder.SetStatus(cmsposttranslation.Status(*req.Status))
			}
			if req.ClearPublishAt {
				tBuilder.ClearPublishAt()
			} else if req.PublishAt != nil {
				tBuilder.SetPublishAt(req.PublishAt.UTC())
			}

			if err := tBuilder.Exec(c); err != nil {
				if ent.IsConstraintError(err) {
					return domain.Post{}, domain.ErrSlugTaken
				}
				return domain.Post{}, err
			}
		}
	}

	return r.GetByID(c, id, locale, defaultLocale, now)
}

// SaveTranslation creates or updates a translation for a post.
func (r PostRepository) SaveTranslation(c context.Context, req model.SavePostTranslationRecord, defaultLocale string, now time.Time) (domain.Post, error) {
	p, err := r.cli.CmsPost.Query().
		Where(
			cmspost.ID(req.PostID),
			cmspost.DeletedAtIsNil(),
		).
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.Post{}, domain.ErrPostNotFound
		}
		return domain.Post{}, err
	}

	t, err := r.cli.CmsPostTranslation.Query().
		Where(
			cmsposttranslation.PostIDEQ(req.PostID),
			cmsposttranslation.LocaleEQ(req.Locale),
			cmsposttranslation.DeletedAtIsNil(),
		).
		Only(c)

	if err != nil {
		if ent.IsNotFound(err) {
			status := req.Status
			if status == "" {
				status = domain.StatusDraft
			}
			builder := r.cli.CmsPostTranslation.Create().
				SetPostID(req.PostID).
				SetWorkspaceID(p.WorkspaceID).
				SetLocale(req.Locale).
				SetTitle(req.Title).
				SetSlug(req.Slug).
				SetBody(req.Body).
				SetStatus(cmsposttranslation.Status(status))

			if req.Excerpt != "" {
				builder.SetExcerpt(req.Excerpt)
			}
			if req.PublishAt != nil {
				builder.SetPublishAt(req.PublishAt.UTC())
			}

			if _, err := builder.Save(c); err != nil {
				if ent.IsConstraintError(err) {
					return domain.Post{}, domain.ErrSlugTaken
				}
				return domain.Post{}, err
			}
		} else {
			return domain.Post{}, err
		}
	} else {
		builder := r.cli.CmsPostTranslation.UpdateOneID(t.ID).
			SetTitle(req.Title).
			SetSlug(req.Slug).
			SetBody(req.Body).
			SetExcerpt(req.Excerpt)

		if err := builder.Exec(c); err != nil {
			if ent.IsConstraintError(err) {
				return domain.Post{}, domain.ErrSlugTaken
			}
			return domain.Post{}, err
		}
	}

	return r.GetByID(c, req.PostID, req.Locale, defaultLocale, now)
}

// DeleteTranslation soft-deletes a single translation for a post.
func (r PostRepository) DeleteTranslation(c context.Context, postID int64, locale string) error {
	t, err := r.cli.CmsPostTranslation.Query().
		Where(
			cmsposttranslation.PostIDEQ(postID),
			cmsposttranslation.LocaleEQ(locale),
			cmsposttranslation.DeletedAtIsNil(),
		).
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.ErrTranslationNotFound
		}
		return err
	}

	return r.cli.CmsPostTranslation.UpdateOneID(t.ID).
		SetDeletedAt(time.Now()).
		Exec(c)
}

// Delete marks a post and all its translations as soft-deleted.
func (r PostRepository) Delete(c context.Context, id int64) error {
	exists, err := r.cli.CmsPost.Query().
		Where(
			cmspost.ID(id),
			cmspost.DeletedAtIsNil(),
		).
		Exist(c)
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrPostNotFound
	}

	tx, err := r.cli.Tx(c)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	if err := tx.CmsPost.UpdateOneID(id).SetDeletedAt(now).Exec(c); err != nil {
		return err
	}

	_, err = tx.CmsPostTranslation.Update().
		Where(
			cmsposttranslation.PostIDEQ(id),
			cmsposttranslation.DeletedAtIsNil(),
		).
		SetDeletedAt(now).
		Save(c)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Search searches active posts matching the criteria with admin cursor pagination.
func (r PostRepository) Search(c context.Context, req model.SearchPostRequest, defaultLocale string, now time.Time) ([]domain.Post, string, error) {
	limit := 20
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
		if limit > 100 {
			limit = 100
		}
	}

	q := r.cli.CmsPost.Query().
		Where(cmspost.DeletedAtIsNil())

	targetLocale := defaultLocale
	if req.Locale != nil && *req.Locale != "" {
		targetLocale = *req.Locale
	}

	if req.Query != nil && *req.Query != "" {
		q = q.Where(cmspost.HasTranslationsWith(
			cmsposttranslation.LocaleEQ(targetLocale),
			cmsposttranslation.DeletedAtIsNil(),
			cmsposttranslation.Or(
				cmsposttranslation.TitleContainsFold(*req.Query),
				cmsposttranslation.SlugContainsFold(*req.Query),
			),
		))
	}

	if req.Status != nil && *req.Status != "" {
		switch *req.Status {
		case domain.EffectiveStatusDraft:
			q = q.Where(cmspost.HasTranslationsWith(
				cmsposttranslation.LocaleEQ(targetLocale),
				cmsposttranslation.DeletedAtIsNil(),
				cmsposttranslation.StatusEQ(domain.StatusDraft),
			))
		case domain.EffectiveStatusScheduled:
			q = q.Where(cmspost.HasTranslationsWith(
				cmsposttranslation.LocaleEQ(targetLocale),
				cmsposttranslation.DeletedAtIsNil(),
				cmsposttranslation.StatusEQ(domain.StatusPublished),
				cmsposttranslation.PublishAtGT(now.UTC()),
			))
		case domain.EffectiveStatusPublished:
			q = q.Where(cmspost.HasTranslationsWith(
				cmsposttranslation.LocaleEQ(targetLocale),
				cmsposttranslation.DeletedAtIsNil(),
				cmsposttranslation.StatusEQ(domain.StatusPublished),
				cmsposttranslation.PublishAtLTE(now.UTC()),
			))
		}
	}

	if req.Cursor != nil && *req.Cursor != "" {
		cursorID, err := strconv.ParseInt(*req.Cursor, 10, 64)
		if err != nil || cursorID < 0 {
			return nil, "", base.ErrInvalidCursor
		}
		q = q.Where(cmspost.IDLT(cursorID))
	}

	items, err := q.
		Order(ent.Desc(cmspost.FieldID)).
		Limit(limit + 1).
		WithTranslations(func(tq *ent.CmsPostTranslationQuery) {
			tq.Where(cmsposttranslation.DeletedAtIsNil())
		}).
		WithCoverMedia(func(mq *ent.CmsMediaQuery) {
			mq.WithFile(func(fq *ent.FileQuery) {
				fq.WithCreator().WithUpdater()
			}).WithCreator().WithUpdater().WithDeleter()
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

	res := make([]domain.Post, len(items))
	for i, it := range items {
		res[i] = toPostDomain(it, targetLocale, defaultLocale, now)
	}

	return res, nextCursor, nil
}

// PublicList retrieves publicly visible posts for a workspace with composite cursor pagination.
func (r PostRepository) PublicList(c context.Context, workspaceID int64, locale, defaultLocale string, cursor *string, limit int, now time.Time) ([]domain.Post, string, error) {
	if base.WorkspaceIDFromContext(c) <= 0 && workspaceID > 0 {
		c = base.WithWorkspaceID(c, workspaceID)
	}
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	// Must have a published translation in `locale` OR in `defaultLocale`
	q := r.cli.CmsPost.Query().
		Where(
			cmspost.WorkspaceIDEQ(workspaceID),
			cmspost.DeletedAtIsNil(),
			cmspost.Or(
				cmspost.HasTranslationsWith(
					cmsposttranslation.LocaleEQ(locale),
					VisibleTranslationPredicate(now),
				),
				cmspost.HasTranslationsWith(
					cmsposttranslation.LocaleEQ(defaultLocale),
					VisibleTranslationPredicate(now),
				),
			),
		)

	if cursor != nil && *cursor != "" {
		cursorID, err := strconv.ParseInt(*cursor, 10, 64)
		if err != nil || cursorID < 0 {
			return nil, "", base.ErrInvalidCursor
		}
		q = q.Where(cmspost.IDLT(cursorID))
	}

	items, err := q.
		Order(ent.Desc(cmspost.FieldID)).
		Limit(limit + 1).
		WithTranslations(func(tq *ent.CmsPostTranslationQuery) {
			tq.Where(cmsposttranslation.DeletedAtIsNil())
		}).
		WithCoverMedia(func(mq *ent.CmsMediaQuery) {
			mq.WithFile(func(fq *ent.FileQuery) {
				fq.WithCreator().WithUpdater()
			}).WithCreator().WithUpdater().WithDeleter()
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

	res := make([]domain.Post, len(items))
	for i, it := range items {
		res[i] = toPostDomain(it, locale, defaultLocale, now)
	}

	return res, nextCursor, nil
}

// PublicGetBySlug retrieves a publicly visible post by slug for a workspace, returning domain.Post, fallback bool, and alternates.
func (r PostRepository) PublicGetBySlug(c context.Context, workspaceID int64, locale, defaultLocale, slug string, now time.Time) (domain.Post, bool, []model.PublicAlternate, error) {
	if base.WorkspaceIDFromContext(c) <= 0 && workspaceID > 0 {
		c = base.WithWorkspaceID(c, workspaceID)
	}
	var dom domain.Post

	// Step 1: Look for published translation in requested locale with slug
	p, err := r.cli.CmsPost.Query().
		Where(
			cmspost.WorkspaceIDEQ(workspaceID),
			cmspost.DeletedAtIsNil(),
			cmspost.HasTranslationsWith(
				cmsposttranslation.LocaleEQ(locale),
				cmsposttranslation.SlugEQ(slug),
				VisibleTranslationPredicate(now),
			),
		).
		WithTranslations(func(tq *ent.CmsPostTranslationQuery) {
			tq.Where(cmsposttranslation.DeletedAtIsNil())
		}).
		WithCoverMedia(func(mq *ent.CmsMediaQuery) {
			mq.WithFile(func(fq *ent.FileQuery) {
				fq.WithCreator().WithUpdater()
			}).WithCreator().WithUpdater().WithDeleter()
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
		p, err = r.cli.CmsPost.Query().
			Where(
				cmspost.WorkspaceIDEQ(workspaceID),
				cmspost.DeletedAtIsNil(),
				cmspost.HasTranslationsWith(
					cmsposttranslation.LocaleEQ(defaultLocale),
					cmsposttranslation.SlugEQ(slug),
					VisibleTranslationPredicate(now),
				),
			).
			WithTranslations(func(tq *ent.CmsPostTranslationQuery) {
				tq.Where(cmsposttranslation.DeletedAtIsNil())
			}).
			WithCoverMedia(func(mq *ent.CmsMediaQuery) {
				mq.WithFile(func(fq *ent.FileQuery) {
					fq.WithCreator().WithUpdater()
				}).WithCreator().WithUpdater().WithDeleter()
			}).
			WithCreator().
			WithUpdater().
			WithDeleter().
			Only(c)

		if err != nil {
			if ent.IsNotFound(err) {
				return dom, false, nil, domain.ErrPostNotFound
			}
			return dom, false, nil, err
		}
		fallback = true
	}

	resolvedLocale := locale
	if fallback {
		resolvedLocale = defaultLocale
	}

	dom = toPostDomain(p, resolvedLocale, defaultLocale, now)

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

func toPostDomain(p *ent.CmsPost, requestedLocale, defaultLocale string, now time.Time) domain.Post {
	dom := domain.Post{
		ID:           p.ID,
		WorkspaceID:  p.WorkspaceID,
		CoverMediaID: p.CoverMediaID,
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

	if p.Edges.CoverMedia != nil {
		cm := toMediaDomain(p.Edges.CoverMedia)
		dom.CoverMedia = &cm
	}

	var active []*ent.CmsPostTranslation
	for _, t := range p.Edges.Translations {
		if t.DeletedAt == nil {
			active = append(active, t)
		}
	}

	dom.Translations = make([]domain.TranslationSummary, len(active))
	var matchRequested, matchDefault, fallback *ent.CmsPostTranslation

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
		dom.Excerpt = selected.Excerpt
		dom.Body = selected.Body
		dom.Status = string(selected.Status)
		dom.EffectiveStatus = domain.EffectiveStatus(string(selected.Status), selected.PublishAt, now)
		dom.PublishAt = selected.PublishAt
	}

	return dom
}
