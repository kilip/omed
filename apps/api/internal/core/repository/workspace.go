// Package repository provides database access implementations for user and workspace entities using Ent.
package repository

import (
	"context"
	"strconv"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/predicate"
	"github.com/kilip/omed/api/ent/workspace"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/entx"
)

// WorkspaceRepository handles workspace persistence operations against PostgreSQL via Ent ORM.
type WorkspaceRepository struct {
	base.ServiceRepository[int64, domain.Workspace, model.CreateWorkspaceRequest, model.UpdateWorkspaceRequest, model.SearchWorkspaceRequest]
	cli *ent.Client
}

// NewWorkspaceRepository constructs a new WorkspaceRepository instance backed by the given Ent client.
func NewWorkspaceRepository(cli *ent.Client) WorkspaceRepository {
	return WorkspaceRepository{
		cli: cli,
	}
}

// GetByID retrieves a workspace domain entity by ID. Returns base.ErrNotFound if the workspace does not exist.
func (r WorkspaceRepository) GetByID(c context.Context, id int64) (domain.Workspace, error) {
	var ws domain.Workspace

	w, err := r.cli.Workspace.Query().
		Where(workspace.ID(id)).
		WithOwner().
		WithCreator().
		WithUpdater().
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return ws, base.ErrNotFound
		}
		return ws, err
	}

	return toWorkspaceDomain(w), nil
}

// Search searches workspaces matching the request filter with cursor pagination.
func (r WorkspaceRepository) Search(c context.Context, req model.SearchWorkspaceRequest) ([]domain.Workspace, string, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	var predicates []predicate.Workspace
	if req.Name != "" {
		predicates = append(predicates, workspace.NameContainsFold(req.Name))
	}
	if req.OwnerID != 0 {
		predicates = append(predicates, workspace.OwnerIDEQ(req.OwnerID))
	}

	if req.Cursor != "" {
		cursorID, err := strconv.ParseInt(req.Cursor, 10, 64)
		if err != nil || cursorID < 0 {
			return nil, "", base.ErrInvalidCursor
		}
		predicates = append(predicates, workspace.IDLT(cursorID))
	}

	workspaces, err := r.cli.Workspace.Query().
		Where(predicates...).
		WithOwner().
		WithCreator().
		WithUpdater().
		Order(ent.Desc(workspace.FieldID)).
		Limit(limit + 1).
		All(c)
	if err != nil {
		return nil, "", err
	}

	var cursor string
	if len(workspaces) > limit {
		cursor = strconv.FormatInt(workspaces[limit-1].ID, 10)
		workspaces = workspaces[:limit]
	}

	res := make([]domain.Workspace, len(workspaces))
	for i, w := range workspaces {
		res[i] = toWorkspaceDomain(w)
	}

	return res, cursor, nil
}

// Create persists a new workspace in a database transaction.
func (r WorkspaceRepository) Create(c context.Context, req model.CreateWorkspaceRequest) (domain.Workspace, error) {
	var w *ent.Workspace
	var dom domain.Workspace

	ownerID := req.OwnerID
	var userID int64
	if u, ok := base.UserFromContext(c); ok && u.ID != 0 {
		userID = u.ID
	}

	if ownerID == 0 {
		ownerID = userID
	}
	if ownerID == 0 {
		return dom, base.ErrUnauthorized
	}

	createdByID := userID
	if createdByID == 0 {
		createdByID = ownerID
	}

	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		var err error
		builder := tx.Workspace.Create().
			SetName(req.Name).
			SetOwnerID(ownerID).
			SetCreatedByID(createdByID).
			SetUpdatedByID(createdByID)

		if req.Description != "" {
			builder.SetDescription(req.Description)
		}

		w, err = builder.Save(c)
		return err
	})

	if err != nil {
		return dom, err
	}

	return r.GetByID(c, w.ID)
}

// Update modifies an existing workspace by ID within a transaction.
func (r WorkspaceRepository) Update(c context.Context, id int64, req model.UpdateWorkspaceRequest) (domain.Workspace, error) {
	var dom domain.Workspace
	var updatedByID *int64
	if u, ok := base.UserFromContext(c); ok && u.ID != 0 {
		updatedByID = &u.ID
	}

	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		builder := tx.Workspace.UpdateOneID(id).
			SetName(req.Name)
		if req.Description != "" {
			builder.SetDescription(req.Description)
		}
		if updatedByID != nil {
			builder.SetUpdatedByID(*updatedByID)
		}
		_, err := builder.Save(c)
		if err != nil {
			if ent.IsNotFound(err) {
				return base.ErrNotFound
			}
			return err
		}
		return nil
	})

	if err != nil {
		return dom, err
	}

	return r.GetByID(c, id)
}

// Delete removes a workspace by ID within a transaction.
func (r WorkspaceRepository) Delete(c context.Context, id int64) error {
	return entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		err := tx.Workspace.DeleteOneID(id).Exec(c)
		if err != nil {
			if ent.IsNotFound(err) {
				return base.ErrNotFound
			}
			return err
		}
		return nil
	})
}

func toWorkspaceDomain(w *ent.Workspace) domain.Workspace {
	dom := domain.Workspace{
		ID:          w.ID,
		Name:        w.Name,
		Description: w.Description,
		OwnerID:     w.OwnerID,
		CreatedByID: w.CreatedByID,
		UpdatedByID: w.UpdatedByID,
		CreatedAt:   w.CreatedAt,
		UpdatedAt:   w.UpdatedAt,
	}

	if w.Edges.Owner != nil {
		dom.Owner = domain.User{
			ID:        w.Edges.Owner.ID,
			Name:      w.Edges.Owner.Name,
			Email:     w.Edges.Owner.Email,
			Avatar:    w.Edges.Owner.Avatar,
			CreatedAt: w.Edges.Owner.CreatedAt,
			UpdatedAt: w.Edges.Owner.UpdatedAt,
		}
	}

	if w.Edges.Creator != nil {
		dom.CreatedBy = base.ByUser{
			Name:   w.Edges.Creator.Name,
			Avatar: w.Edges.Creator.Avatar,
		}
	} else if dom.Owner.ID != 0 {
		dom.CreatedBy = base.ByUser{
			Name:   dom.Owner.Name,
			Avatar: dom.Owner.Avatar,
		}
	}

	if w.Edges.Updater != nil {
		dom.UpdatedBy = base.ByUser{
			Name:   w.Edges.Updater.Name,
			Avatar: w.Edges.Updater.Avatar,
		}
	} else if dom.Owner.ID != 0 {
		dom.UpdatedBy = base.ByUser{
			Name:   dom.Owner.Name,
			Avatar: dom.Owner.Avatar,
		}
	}

	return dom
}
