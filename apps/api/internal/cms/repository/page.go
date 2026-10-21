package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/cmspage"
	"github.com/kilip/omed/api/ent/predicate"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
)

// PageRepository defines persistence operations for CMS pages.
type PageRepository struct {
	cli *ent.Client
}

// NewPageRepository constructs a PageRepository backed by the given Ent client.
func NewPageRepository(cli *ent.Client) PageRepository {
	return PageRepository{cli: cli}
}

// VisiblePagePredicate returns an Ent predicate matching publicly visible pages at time now.
func VisiblePagePredicate(now time.Time) predicate.CmsPage {
	return cmspage.And(
		cmspage.DeletedAtIsNil(),
		cmspage.StatusEQ(cmspage.Status(domain.StatusPublished)),
		cmspage.PublishAtLTE(now),
	)
}

// GetByID retrieves a page by ID within the caller's workspace context.
func (r PageRepository) GetByID(c context.Context, id int64, now time.Time) (domain.Page, error) {
	var dom domain.Page

	p, err := r.cli.CmsPage.Query().
		Where(
			cmspage.ID(id),
			cmspage.DeletedAtIsNil(),
		).
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

	return toPageDomain(p, now), nil
}

// SlugExists checks if an active page with the given slug exists in the workspace.
func (r PageRepository) SlugExists(c context.Context, slug string, excludeID *int64) (bool, error) {
	q := r.cli.CmsPage.Query().
		Where(
			cmspage.SlugEQ(slug),
			cmspage.DeletedAtIsNil(),
		)
	if excludeID != nil && *excludeID > 0 {
		q = q.Where(cmspage.IDNEQ(*excludeID))
	}
	return q.Exist(c)
}

// Create inserts a new cms_page record.
func (r PageRepository) Create(c context.Context, req model.CreatePageRecord, now time.Time) (domain.Page, error) {
	builder := r.cli.CmsPage.Create().
		SetTitle(req.Title).
		SetSlug(req.Slug).
		SetBody(req.Body).
		SetStatus(cmspage.Status(req.Status))

	if req.WorkspaceID > 0 {
		builder.SetWorkspaceID(req.WorkspaceID)
	}
	if req.PublishAt != nil {
		builder.SetPublishAt(*req.PublishAt)
	}

	created, err := builder.Save(c)
	if err != nil {
		if ent.IsConstraintError(err) {
			return domain.Page{}, domain.ErrSlugTaken
		}
		return domain.Page{}, err
	}

	return r.GetByID(c, created.ID, now)
}

// Update modifies an existing page record.
func (r PageRepository) Update(c context.Context, id int64, req model.UpdatePageRecord, now time.Time) (domain.Page, error) {
	exists, err := r.cli.CmsPage.Query().
		Where(
			cmspage.ID(id),
			cmspage.DeletedAtIsNil(),
		).
		Exist(c)
	if err != nil {
		return domain.Page{}, err
	}
	if !exists {
		return domain.Page{}, domain.ErrPageNotFound
	}

	builder := r.cli.CmsPage.UpdateOneID(id)
	if req.Title != nil {
		builder.SetTitle(*req.Title)
	}
	if req.Slug != nil {
		builder.SetSlug(*req.Slug)
	}
	if req.Body != nil {
		builder.SetBody(*req.Body)
	}
	if req.Status != nil {
		builder.SetStatus(cmspage.Status(*req.Status))
	}
	if req.ClearPublishAt {
		builder.ClearPublishAt()
	} else if req.PublishAt != nil {
		builder.SetPublishAt(*req.PublishAt)
	}

	if err := builder.Exec(c); err != nil {
		if ent.IsConstraintError(err) {
			return domain.Page{}, domain.ErrSlugTaken
		}
		if ent.IsNotFound(err) {
			return domain.Page{}, domain.ErrPageNotFound
		}
		return domain.Page{}, err
	}

	return r.GetByID(c, id, now)
}

// Delete marks a page as soft-deleted.
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

	return r.cli.CmsPage.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(c)
}

// Search searches active pages matching the criteria with admin cursor pagination.
func (r PageRepository) Search(c context.Context, req model.SearchPageRequest, now time.Time) ([]domain.Page, string, error) {
	limit := 20
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
		if limit > 100 {
			limit = 100
		}
	}

	q := r.cli.CmsPage.Query().
		Where(cmspage.DeletedAtIsNil())

	if req.Query != nil && *req.Query != "" {
		q = q.Where(cmspage.Or(
			cmspage.TitleContainsFold(*req.Query),
			cmspage.SlugContainsFold(*req.Query),
		))
	}

	if req.Status != nil && *req.Status != "" {
		switch *req.Status {
		case domain.EffectiveStatusDraft:
			q = q.Where(cmspage.StatusEQ(cmspage.Status(domain.StatusDraft)))
		case domain.EffectiveStatusScheduled:
			q = q.Where(
				cmspage.StatusEQ(cmspage.Status(domain.StatusPublished)),
				cmspage.PublishAtGT(now),
			)
		case domain.EffectiveStatusPublished:
			q = q.Where(
				cmspage.StatusEQ(cmspage.Status(domain.StatusPublished)),
				cmspage.PublishAtLTE(now),
			)
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
		res[i] = toPageDomain(it, now)
	}

	return res, nextCursor, nil
}

// PublicList retrieves publicly visible pages for a workspace with composite cursor pagination.
func (r PageRepository) PublicList(c context.Context, workspaceID int64, cursor *string, limit int, now time.Time) ([]domain.Page, string, error) {
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	q := r.cli.CmsPage.Query().
		Where(
			cmspage.WorkspaceIDEQ(workspaceID),
			VisiblePagePredicate(now),
		)

	if cursor != nil && *cursor != "" {
		curTime, curID, err := ParsePublicCursor(*cursor)
		if err != nil {
			return nil, "", base.ErrInvalidCursor
		}
		q = q.Where(cmspage.Or(
			cmspage.PublishAtLT(curTime),
			cmspage.And(
				cmspage.PublishAtEQ(curTime),
				cmspage.IDLT(curID),
			),
		))
	}

	items, err := q.
		Order(ent.Desc(cmspage.FieldPublishAt), ent.Desc(cmspage.FieldID)).
		Limit(limit + 1).
		WithCreator().
		WithUpdater().
		WithDeleter().
		All(c)
	if err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(items) > limit {
		last := items[limit-1]
		if last.PublishAt != nil {
			nextCursor = FormatPublicCursor(*last.PublishAt, last.ID)
		}
		items = items[:limit]
	}

	res := make([]domain.Page, len(items))
	for i, it := range items {
		res[i] = toPageDomain(it, now)
	}

	return res, nextCursor, nil
}

// PublicGetBySlug retrieves a publicly visible page by slug for a workspace.
func (r PageRepository) PublicGetBySlug(c context.Context, workspaceID int64, slug string, now time.Time) (domain.Page, error) {
	var dom domain.Page

	p, err := r.cli.CmsPage.Query().
		Where(
			cmspage.WorkspaceIDEQ(workspaceID),
			cmspage.SlugEQ(slug),
			VisiblePagePredicate(now),
		).
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

	return toPageDomain(p, now), nil
}

func toPageDomain(p *ent.CmsPage, now time.Time) domain.Page {
	dom := domain.Page{
		ID:              p.ID,
		WorkspaceID:     p.WorkspaceID,
		Title:           p.Title,
		Slug:            p.Slug,
		Body:            p.Body,
		Status:          string(p.Status),
		EffectiveStatus: domain.EffectiveStatus(string(p.Status), p.PublishAt, now),
		PublishAt:       p.PublishAt,
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

	return dom
}
