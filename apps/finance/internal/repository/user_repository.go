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

type UserRepository struct {
	client *ent.Client
	log    *slog.Logger
}

func NewUserRepository(client *ent.Client, log *slog.Logger) UserRepository {
	return UserRepository{
		client,
		log,
	}
}

func (r UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.UserSnapshot, error) {
	user, err := r.client.User.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(shared.ErrItemNotFound, err)
		}

		return nil, err
	}

	var snapshot model.UserSnapshot
	if err = shared.ToValue(user, snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

func (r UserRepository) Create(ctx context.Context, snapshot model.UserSnapshot) error {
	return WithTx(ctx, r.client, func(tx *ent.Tx) error {
		_, err := tx.User.Create().
			SetID(snapshot.ID).
			SetName(snapshot.Name).
			SetAvatar(snapshot.Avatar).
			SetSyncedAt(snapshot.SyncedAt).
			Save(ctx)

		return err
	})
}

func (r UserRepository) Update(ctx context.Context, snapshot model.UserSnapshot) error {
	return WithTx(ctx, r.client, func(tx *ent.Tx) error {
		_, err := tx.User.Create().
			SetID(snapshot.ID).
			SetName(snapshot.Name).
			SetAvatar(snapshot.Avatar).
			SetSyncedAt(snapshot.SyncedAt).
			Save(ctx)
		return err
	})
}
