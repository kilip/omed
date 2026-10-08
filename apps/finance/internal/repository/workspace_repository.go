package repository

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared/util"
)

type WorkspaceRepository struct {
	cl     *ent.Client
	logger *slog.Logger
}

func NewWorkspaceRepository(cl *ent.Client, logger *slog.Logger) WorkspaceRepository {
	return WorkspaceRepository{
		cl,
		logger,
	}
}

func (r WorkspaceRepository) GetById(ctx context.Context, id uuid.UUID) (*model.WorkspaceSnapshot, error) {
	works, err := r.cl.Workspace.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, core.ErrItemNotFound
		}
		return nil, err
	}

	var snapshot model.WorkspaceSnapshot
	if err := util.ToValue(works, &snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

func (r WorkspaceRepository) Create(ctx context.Context, snapshot model.WorkspaceSnapshot) error {
	return WithTx(ctx, r.cl, func(tx *ent.Tx) error {
		_, err := tx.Workspace.Create().
			SetID(snapshot.ID).
			SetName(snapshot.Name).
			SetSyncedAt(time.Now()).
			Save(ctx)

		return err
	})
}

func (r WorkspaceRepository) Update(ctx context.Context, snapshot model.WorkspaceSnapshot) error {
	return WithTx(ctx, r.cl, func(tx *ent.Tx) error {
		_, err := tx.Workspace.UpdateOneID(snapshot.ID).
			SetName(snapshot.Name).
			SetSyncedAt(snapshot.SyncedAt).
			Save(ctx)
		return err
	})
}
