package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
)

type WorkspaceRepository struct {
	client *ent.Client
	log    *slog.Logger
}

func NewWorkspaceRepository(client *ent.Client, log *slog.Logger) WorkspaceRepository {
	return WorkspaceRepository{
		client,
		log,
	}
}

func (r WorkspaceRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.WorkspaceSnapshot, error) {
	user, err := r.client.Workspace.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(shared.ErrItemNotFound, err)
		}

		return nil, err
	}

	var snapshot model.WorkspaceSnapshot
	if err = shared.ToValue(user, snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

func (r WorkspaceRepository) Create(ctx context.Context, snapshot model.WorkspaceSnapshot) error {
	return WithTx(ctx, r.client, func(tx *ent.Tx) error {
		_, err := tx.User.Create().
			SetID(snapshot.ID).
			SetName(snapshot.Name).
			SetSyncedAt(snapshot.SyncedAt).
			Save(ctx)

		return err
	})
}

func (r WorkspaceRepository) Update(ctx context.Context, snapshot model.WorkspaceSnapshot) error {
	return WithTx(ctx, r.client, func(tx *ent.Tx) error {
		_, err := tx.User.Create().
			SetID(snapshot.ID).
			SetName(snapshot.Name).
			SetSyncedAt(snapshot.SyncedAt).
			Save(ctx)
		return err
	})
}
