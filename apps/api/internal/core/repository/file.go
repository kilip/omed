package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/file"
	"github.com/kilip/omed/api/ent/predicate"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/entx"
)

// FileRepository handles file metadata persistence operations against PostgreSQL via Ent ORM.
type FileRepository struct {
	cli *ent.Client
}

// NewFileRepository constructs a new FileRepository instance backed by the given Ent client.
func NewFileRepository(cli *ent.Client) FileRepository {
	return FileRepository{cli: cli}
}

// GetByID retrieves an active file metadata record by ID within the current workspace context.
func (r FileRepository) GetByID(c context.Context, id int64) (domain.File, error) {
	var dom domain.File

	f, err := r.cli.File.Query().
		Where(
			file.ID(id),
			file.DeletedAtIsNil(),
		).
		WithCreator().
		WithUpdater().
		WithDeleter().
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom, base.ErrNotFound
		}
		return dom, err
	}

	return toFileDomain(f), nil
}

// Search searches active files matching the request filter with cursor pagination.
func (r FileRepository) Search(c context.Context, req model.SearchFileRequest) ([]domain.File, string, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	predicates := []predicate.File{
		file.DeletedAtIsNil(),
	}
	if req.Name != "" {
		predicates = append(predicates, file.NameContainsFold(req.Name))
	}
	if req.MimeType != "" {
		predicates = append(predicates, file.MimeTypeContainsFold(req.MimeType))
	}
	if req.EntityType != "" {
		predicates = append(predicates, file.EntityTypeEQ(req.EntityType))
	}
	if req.EntityID > 0 {
		predicates = append(predicates, file.EntityIDEQ(req.EntityID))
	}

	if req.Cursor != "" {
		cursorID, err := strconv.ParseInt(req.Cursor, 10, 64)
		if err != nil || cursorID < 0 {
			return nil, "", base.ErrInvalidCursor
		}
		predicates = append(predicates, file.IDLT(cursorID))
	}

	files, err := r.cli.File.Query().
		Where(predicates...).
		WithCreator().
		WithUpdater().
		Order(ent.Desc(file.FieldID)).
		Limit(limit + 1).
		All(c)
	if err != nil {
		return nil, "", err
	}

	var cursor string
	if len(files) > limit {
		cursor = strconv.FormatInt(files[limit-1].ID, 10)
		files = files[:limit]
	}

	res := make([]domain.File, len(files))
	for i, f := range files {
		res[i] = toFileDomain(f)
	}

	return res, cursor, nil
}

// Create persists a new file metadata record in a database transaction.
func (r FileRepository) Create(c context.Context, req model.CreateFileRecordRequest) (domain.File, error) {
	var dom domain.File
	var createdID int64

	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		builder := tx.File.Create().
			SetName(req.Name).
			SetBucket(req.Bucket).
			SetKey(req.Key).
			SetMimeType(req.MimeType).
			SetSize(req.Size)

		if req.WorkspaceID > 0 {
			builder.SetWorkspaceID(req.WorkspaceID)
		}
		if req.EntityType != "" {
			builder.SetEntityType(req.EntityType)
		}
		if req.EntityID > 0 {
			builder.SetEntityID(req.EntityID)
		}

		res, err := builder.Save(c)
		if err != nil {
			return err
		}
		createdID = res.ID
		return nil
	})
	if err != nil {
		return dom, err
	}

	return r.GetByID(c, createdID)
}

// Update modifies an existing file metadata record by ID within a transaction.
func (r FileRepository) Update(c context.Context, id int64, req model.UpdateFileRequest) (domain.File, error) {
	var dom domain.File

	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		exists, err := tx.File.Query().
			Where(
				file.ID(id),
				file.DeletedAtIsNil(),
			).
			Exist(c)
		if err != nil {
			return err
		}
		if !exists {
			return base.ErrNotFound
		}

		builder := tx.File.UpdateOneID(id).
			SetName(req.Name)
		if req.EntityType != "" {
			builder.SetEntityType(req.EntityType)
		}
		if req.EntityID > 0 {
			builder.SetEntityID(req.EntityID)
		}

		_, err = builder.Save(c)
		return err
	})
	if err != nil {
		return dom, err
	}

	return r.GetByID(c, id)
}

// Delete soft-deletes a file metadata record by setting deleted_at within a transaction.
func (r FileRepository) Delete(c context.Context, id int64) error {
	return entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		exists, err := tx.File.Query().
			Where(
				file.ID(id),
				file.DeletedAtIsNil(),
			).
			Exist(c)
		if err != nil {
			return err
		}
		if !exists {
			return base.ErrNotFound
		}

		now := time.Now()
		return tx.File.UpdateOneID(id).
			SetDeletedAt(now).
			Exec(c)
	})
}

func toFileDomain(f *ent.File) domain.File {
	dom := domain.File{
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
		dom.EntityType = *f.EntityType
	}
	if f.EntityID != nil {
		dom.EntityID = *f.EntityID
	}

	if f.Edges.Creator != nil {
		dom.CreatedBy = base.ByUser{
			Name:   f.Edges.Creator.Name,
			Avatar: f.Edges.Creator.Avatar,
		}
	}
	if f.Edges.Updater != nil {
		dom.UpdatedBy = base.ByUser{
			Name:   f.Edges.Updater.Name,
			Avatar: f.Edges.Updater.Avatar,
		}
	}

	return dom
}
