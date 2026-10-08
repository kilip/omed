package repository

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/ent/user"
	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared/util"
)

type UserRepository struct {
	entClient *ent.Client
	logger    *slog.Logger
}

func NewUserRepository(entClient *ent.Client, logger *slog.Logger) UserRepository {
	return UserRepository{
		entClient,
		logger,
	}
}

func (r UserRepository) GetById(ctx context.Context, id uuid.UUID) (*model.UserSnapshot, error) {
	var snapshot model.UserSnapshot
	existing, err := r.entClient.User.Get(ctx, id)

	if err != nil && ent.IsNotFound(err) {
		return nil, core.ErrItemNotFound
	}

	if err = util.ToValue(existing, &snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

func (r UserRepository) Create(ctx context.Context, user model.UserSnapshot) error {
	return WithTx(ctx, r.entClient, func(tx *ent.Tx) error {
		_, err := tx.User.Create().
			SetID(user.ID).
			SetName(user.Name).
			SetAvatar(user.Avatar).
			SetSyncedAt(time.Now()).
			Save(ctx)
		return err
	})
}

func (r UserRepository) Update(ctx context.Context, user model.UserSnapshot) error {
	return WithTx(ctx, r.entClient, func(tx *ent.Tx) error {
		_, err := tx.User.UpdateOneID(user.ID).
			SetName(user.Name).
			SetAvatar(user.Avatar).
			SetSyncedAt(user.SyncedAt).
			Save(ctx)
		return err
	})
}

func (r UserRepository) Upsert(ctx context.Context, u model.UserSnapshot) error {
	return WithTx(ctx, r.entClient, func(tx *ent.Tx) error {
		return tx.User.Create().
			SetID(u.ID).SetName(u.Name).SetAvatar(u.Avatar).
			OnConflictColumns(user.FieldID).
			UpdateNewValues().
			Exec(ctx)
	})
}

func (r UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return WithTx(ctx, r.entClient, func(tx *ent.Tx) error {
		return tx.User.DeleteOneID(id).Exec(ctx)
	})
}
