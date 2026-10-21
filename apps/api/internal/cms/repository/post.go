package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/cmspost"
	"github.com/kilip/omed/api/ent/predicate"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
)

// PostRepository defines persistence operations for CMS posts.
type PostRepository struct {
	cli *ent.Client
}

// NewPostRepository constructs a PostRepository backed by the given Ent client.
func NewPostRepository(cli *ent.Client) PostRepository {
	return PostRepository{cli: cli}
}

// VisiblePredicate returns an Ent predicate matching publicly visible posts at time now.
func VisiblePredicate(now time.Time) predicate.CmsPost {
	return cmspost.And(
		cmspost.DeletedAtIsNil(),
		cmspost.StatusEQ(domain.StatusPublished),
		cmspost.PublishAtLTE(now),
	)
}

// GetByID retrieves a post by ID within the caller's workspace context.
func (r PostRepository) GetByID(c context.Context, id int64, now time.Time) (domain.Post, error) {
	var dom domain.Post

	p, err := r.cli.CmsPost.Query().
		Where(
			cmspost.ID(id),
			cmspost.DeletedAtIsNil(),
		).
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

	return toPostDomain(p, now), nil
}

// SlugExists checks if an active post with the given slug exists in the workspace.
func (r PostRepository) SlugExists(c context.Context, slug string, excludeID *int64) (bool, error) {
	q := r.cli.CmsPost.Query().
		Where(
			cmspost.SlugEQ(slug),
			cmspost.DeletedAtIsNil(),
		)
	if excludeID != nil && *excludeID > 0 {
		q = q.Where(cmspost.IDNEQ(*excludeID))
	}
	return q.Exist(c)
}

// Create inserts a new cms_post record.
func (r PostRepository) Create(c context.Context, req model.CreatePostRecord, now time.Time) (domain.Post, error) {
	builder := r.cli.CmsPost.Create().
		SetTitle(req.Title).
		SetSlug(req.Slug).
		SetBody(req.Body).
		SetStatus(cmspost.Status(req.Status))

	if req.WorkspaceID > 0 {
		builder.SetWorkspaceID(req.WorkspaceID)
	}
	if req.Excerpt != "" {
		builder.SetExcerpt(req.Excerpt)
	}
	if req.CoverMediaID != nil {
		builder.SetCoverMediaID(*req.CoverMediaID)
	}
	if req.PublishAt != nil {
		builder.SetPublishAt(*req.PublishAt)
	}

	created, err := builder.Save(c)
	if err != nil {
		if ent.IsConstraintError(err) {
			return domain.Post{}, domain.ErrSlugTaken
		}
		return domain.Post{}, err
	}

	return r.GetByID(c, created.ID, now)
}

// Update modifies an existing post record.
func (r PostRepository) Update(c context.Context, id int64, req model.UpdatePostRecord, now time.Time) (domain.Post, error) {
	exists, err := r.cli.CmsPost.Query().
		Where(
			cmspost.ID(id),
			cmspost.DeletedAtIsNil(),
		).
		Exist(c)
	if err != nil {
		return domain.Post{}, err
	}
	if !exists {
		return domain.Post{}, domain.ErrPostNotFound
	}

	builder := r.cli.CmsPost.UpdateOneID(id)
	if req.Title != nil {
		builder.SetTitle(*req.Title)
	}
	if req.Slug != nil {
		builder.SetSlug(*req.Slug)
	}
	if req.Excerpt != nil {
		builder.SetExcerpt(*req.Excerpt)
	}
	if req.Body != nil {
		builder.SetBody(*req.Body)
	}
	if req.ClearCoverMedia {
		builder.ClearCoverMediaID()
	} else if req.CoverMediaID != nil {
		builder.SetCoverMediaID(*req.CoverMediaID)
	}
	if req.Status != nil {
		builder.SetStatus(cmspost.Status(*req.Status))
	}
	if req.ClearPublishAt {
		builder.ClearPublishAt()
	} else if req.PublishAt != nil {
		builder.SetPublishAt(*req.PublishAt)
	}

	if err := builder.Exec(c); err != nil {
		if ent.IsConstraintError(err) {
			return domain.Post{}, domain.ErrSlugTaken
		}
		if ent.IsNotFound(err) {
			return domain.Post{}, domain.ErrPostNotFound
		}
		return domain.Post{}, err
	}

	return r.GetByID(c, id, now)
}

// Delete marks a post as soft-deleted.
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

	return r.cli.CmsPost.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(c)
}

// Search searches active posts matching the criteria with admin cursor pagination.
func (r PostRepository) Search(c context.Context, req model.SearchPostRequest, now time.Time) ([]domain.Post, string, error) {
	limit := 20
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
		if limit > 100 {
			limit = 100
		}
	}

	q := r.cli.CmsPost.Query().
		Where(cmspost.DeletedAtIsNil())

	if req.Query != nil && *req.Query != "" {
		q = q.Where(cmspost.Or(
			cmspost.TitleContainsFold(*req.Query),
			cmspost.SlugContainsFold(*req.Query),
		))
	}

	if req.Status != nil && *req.Status != "" {
		switch *req.Status {
		case domain.EffectiveStatusDraft:
			q = q.Where(cmspost.StatusEQ(domain.StatusDraft))
		case domain.EffectiveStatusScheduled:
			q = q.Where(
				cmspost.StatusEQ(domain.StatusPublished),
				cmspost.PublishAtGT(now),
			)
		case domain.EffectiveStatusPublished:
			q = q.Where(
				cmspost.StatusEQ(domain.StatusPublished),
				cmspost.PublishAtLTE(now),
			)
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
		res[i] = toPostDomain(it, now)
	}

	return res, nextCursor, nil
}

// PublicList retrieves publicly visible posts for a workspace with composite cursor pagination.
func (r PostRepository) PublicList(c context.Context, workspaceID int64, cursor *string, limit int, now time.Time) ([]domain.Post, string, error) {
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	q := r.cli.CmsPost.Query().
		Where(
			cmspost.WorkspaceIDEQ(workspaceID),
			VisiblePredicate(now),
		)

	if cursor != nil && *cursor != "" {
		curTime, curID, err := ParsePublicCursor(*cursor)
		if err != nil {
			return nil, "", base.ErrInvalidCursor
		}
		q = q.Where(cmspost.Or(
			cmspost.PublishAtLT(curTime),
			cmspost.And(
				cmspost.PublishAtEQ(curTime),
				cmspost.IDLT(curID),
			),
		))
	}

	items, err := q.
		Order(ent.Desc(cmspost.FieldPublishAt), ent.Desc(cmspost.FieldID)).
		Limit(limit + 1).
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
		last := items[limit-1]
		if last.PublishAt != nil {
			nextCursor = FormatPublicCursor(*last.PublishAt, last.ID)
		}
		items = items[:limit]
	}

	res := make([]domain.Post, len(items))
	for i, it := range items {
		res[i] = toPostDomain(it, now)
	}

	return res, nextCursor, nil
}

// PublicGetBySlug retrieves a publicly visible post by slug for a workspace.
func (r PostRepository) PublicGetBySlug(c context.Context, workspaceID int64, slug string, now time.Time) (domain.Post, error) {
	var dom domain.Post

	p, err := r.cli.CmsPost.Query().
		Where(
			cmspost.WorkspaceIDEQ(workspaceID),
			cmspost.SlugEQ(slug),
			VisiblePredicate(now),
		).
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

	return toPostDomain(p, now), nil
}

// FormatPublicCursor encodes publishAt and id into composite cursor string "<publish_at unix micros>_<id>".
func FormatPublicCursor(publishAt time.Time, id int64) string {
	return fmt.Sprintf("%d_%d", publishAt.UnixMicro(), id)
}

// ParsePublicCursor decodes composite cursor string "<publish_at unix micros>_<id>".
func ParsePublicCursor(cursor string) (time.Time, int64, error) {
	parts := strings.Split(cursor, "_")
	if len(parts) != 2 {
		return time.Time{}, 0, base.ErrInvalidCursor
	}
	micros, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, 0, base.ErrInvalidCursor
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return time.Time{}, 0, base.ErrInvalidCursor
	}
	return time.UnixMicro(micros), id, nil
}

func toPostDomain(p *ent.CmsPost, now time.Time) domain.Post {
	dom := domain.Post{
		ID:              p.ID,
		WorkspaceID:     p.WorkspaceID,
		Title:           p.Title,
		Slug:            p.Slug,
		Excerpt:         p.Excerpt,
		Body:            p.Body,
		CoverMediaID:    p.CoverMediaID,
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

	if p.Edges.CoverMedia != nil {
		cm := toMediaDomain(p.Edges.CoverMedia)
		dom.CoverMedia = &cm
	}

	return dom
}
