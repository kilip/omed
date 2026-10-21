package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/cmsmedia"
	"github.com/kilip/omed/api/ent/cmspost"
	"github.com/kilip/omed/api/ent/file"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	coredomain "github.com/kilip/omed/api/internal/core/domain"
)

// MediaRepository defines operations for persisting and querying cms media.
type MediaRepository struct {
	cli *ent.Client
}

// NewMediaRepository constructs a new MediaRepository backed by the given Ent client.
func NewMediaRepository(cli *ent.Client) MediaRepository {
	return MediaRepository{cli: cli}
}

// GetByID retrieves active media item by ID within the current workspace context.
func (r MediaRepository) GetByID(c context.Context, id int64) (domain.Media, error) {
	var dom domain.Media

	m, err := r.cli.CmsMedia.Query().
		Where(
			cmsmedia.ID(id),
			cmsmedia.DeletedAtIsNil(),
		).
		WithFile(func(fq *ent.FileQuery) {
			fq.WithCreator().WithUpdater()
		}).
		WithCreator().
		WithUpdater().
		WithDeleter().
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom, domain.ErrMediaNotFound
		}
		return dom, err
	}

	return toMediaDomain(m), nil
}

// GetPublicByID retrieves a media item by ID and workspace, only if is_public and not deleted.
// This method ignores the workspace interceptor by explicitly filtering workspace_id.
func (r MediaRepository) GetPublicByID(c context.Context, workspaceID, id int64) (domain.Media, error) {
	var dom domain.Media

	m, err := r.cli.CmsMedia.Query().
		Where(
			cmsmedia.ID(id),
			cmsmedia.WorkspaceIDEQ(workspaceID),
			cmsmedia.IsPublic(true),
			cmsmedia.DeletedAtIsNil(),
		).
		WithFile(func(fq *ent.FileQuery) {
			fq.WithCreator().WithUpdater()
		}).
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom, domain.ErrMediaNotFound
		}
		return dom, err
	}

	return toMediaDomain(m), nil
}

// Create inserts a new cms_media record.
func (r MediaRepository) Create(c context.Context, req model.CreateMediaRecordRequest) (domain.Media, error) {
	builder := r.cli.CmsMedia.Create().
		SetFileID(req.FileID).
		SetIsPublic(req.IsPublic)

	if req.WorkspaceID > 0 {
		builder.SetWorkspaceID(req.WorkspaceID)
	}
	if req.Title != "" {
		builder.SetTitle(req.Title)
	}
	if req.AltText != "" {
		builder.SetAltText(req.AltText)
	}
	if req.Caption != "" {
		builder.SetCaption(req.Caption)
	}

	created, err := builder.Save(c)
	if err != nil {
		return domain.Media{}, err
	}

	return r.GetByID(c, created.ID)
}

// Update modifies metadata fields on an active media item.
func (r MediaRepository) Update(c context.Context, id int64, req model.UpdateMediaRequest) (domain.Media, error) {
	exists, err := r.cli.CmsMedia.Query().
		Where(
			cmsmedia.ID(id),
			cmsmedia.DeletedAtIsNil(),
		).
		Exist(c)
	if err != nil {
		return domain.Media{}, err
	}
	if !exists {
		return domain.Media{}, domain.ErrMediaNotFound
	}

	builder := r.cli.CmsMedia.UpdateOneID(id)
	if req.Title != nil {
		builder.SetTitle(*req.Title)
	}
	if req.AltText != nil {
		builder.SetAltText(*req.AltText)
	}
	if req.Caption != nil {
		builder.SetCaption(*req.Caption)
	}
	if req.IsPublic != nil {
		builder.SetIsPublic(*req.IsPublic)
	}

	if err := builder.Exec(c); err != nil {
		if ent.IsNotFound(err) {
			return domain.Media{}, domain.ErrMediaNotFound
		}
		return domain.Media{}, err
	}

	return r.GetByID(c, id)
}

// Delete marks an existing media record as soft-deleted.
func (r MediaRepository) Delete(c context.Context, id int64) error {
	exists, err := r.cli.CmsMedia.Query().
		Where(
			cmsmedia.ID(id),
			cmsmedia.DeletedAtIsNil(),
		).
		Exist(c)
	if err != nil {
		return err
	}
	if !exists {
		return domain.ErrMediaNotFound
	}

	return r.cli.CmsMedia.UpdateOneID(id).
		SetDeletedAt(time.Now()).
		Exec(c)
}

// IsReferenced checks if any non-deleted post references this media as cover_media_id.
func (r MediaRepository) IsReferenced(c context.Context, id int64) (bool, error) {
	return r.cli.CmsPost.Query().
		Where(
			cmspost.CoverMediaID(id),
			cmspost.DeletedAtIsNil(),
		).
		Exist(c)
}

// Search searches active media records matching the filter criteria with cursor pagination.
func (r MediaRepository) Search(c context.Context, req model.SearchMediaRequest) ([]domain.Media, string, error) {
	limit := 20
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
		if limit > 100 {
			limit = 100
		}
	}

	q := r.cli.CmsMedia.Query().
		Where(
			cmsmedia.DeletedAtIsNil(),
		)

	if req.Query != nil && *req.Query != "" {
		q = q.Where(cmsmedia.Or(
			cmsmedia.TitleContainsFold(*req.Query),
			cmsmedia.AltTextContainsFold(*req.Query),
			cmsmedia.CaptionContainsFold(*req.Query),
			cmsmedia.HasFileWith(file.NameContainsFold(*req.Query)),
		))
	}
	if req.MimeType != nil && *req.MimeType != "" {
		q = q.Where(cmsmedia.HasFileWith(file.MimeTypeContainsFold(*req.MimeType)))
	}
	if req.Cursor != nil && *req.Cursor != "" {
		cursorID, err := strconv.ParseInt(*req.Cursor, 10, 64)
		if err != nil || cursorID < 0 {
			return nil, "", base.ErrInvalidCursor
		}
		q = q.Where(cmsmedia.IDLT(cursorID))
	}

	items, err := q.
		Order(ent.Desc(cmsmedia.FieldID)).
		Limit(limit + 1).
		WithFile(func(fq *ent.FileQuery) {
			fq.WithCreator().WithUpdater()
		}).
		WithCreator().
		WithUpdater().
		WithDeleter().
		All(c)
	if err != nil {
		return nil, "", err
	}

	var cursor string
	if len(items) > limit {
		cursor = strconv.FormatInt(items[limit-1].ID, 10)
		items = items[:limit]
	}

	res := make([]domain.Media, len(items))
	for i, it := range items {
		res[i] = toMediaDomain(it)
	}

	return res, cursor, nil
}

func toMediaDomain(m *ent.CmsMedia) domain.Media {
	dom := domain.Media{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		FileID:      m.FileID,
		Title:       m.Title,
		AltText:     m.AltText,
		Caption:     m.Caption,
		IsPublic:    m.IsPublic,
		AuditMeta: base.AuditMeta{
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
			DeletedAt: m.DeletedAt,
		},
	}

	if m.Edges.Creator != nil {
		dom.AuditMeta.CreatedBy = base.ByUser{
			Name:   m.Edges.Creator.Name,
			Avatar: m.Edges.Creator.Avatar,
		}
	}
	if m.Edges.Updater != nil {
		dom.AuditMeta.UpdatedBy = base.ByUser{
			Name:   m.Edges.Updater.Name,
			Avatar: m.Edges.Updater.Avatar,
		}
	}
	if m.Edges.Deleter != nil {
		dom.AuditMeta.DeletedBy = &base.ByUser{
			Name:   m.Edges.Deleter.Name,
			Avatar: m.Edges.Deleter.Avatar,
		}
	}

	if m.Edges.File != nil {
		f := m.Edges.File
		dom.File = &coredomain.File{
			ID:          f.ID,
			WorkspaceID: f.WorkspaceID,
			Name:        f.Name,
			Bucket:      f.Bucket,
			Key:         f.Key,
			MimeType:    f.MimeType,
			Size:        f.Size,
			CreatedByID: f.CreatedBy,
			UpdatedByID: f.UpdatedBy,
			CreatedAt:   f.CreatedAt,
			UpdatedAt:   f.UpdatedAt,
		}
		if f.EntityType != nil {
			dom.File.EntityType = *f.EntityType
		}
		if f.EntityID != nil {
			dom.File.EntityID = *f.EntityID
		}
		if f.Edges.Creator != nil {
			dom.File.CreatedBy = base.ByUser{
				Name:   f.Edges.Creator.Name,
				Avatar: f.Edges.Creator.Avatar,
			}
		}
		if f.Edges.Updater != nil {
			dom.File.UpdatedBy = base.ByUser{
				Name:   f.Edges.Updater.Name,
				Avatar: f.Edges.Updater.Avatar,
			}
		}
	}

	return dom
}
