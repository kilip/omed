package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/model"
)

type UserRepository struct {
	cl  *ent.Client
	log *slog.Logger
}

func NewUserRepository(cl *ent.Client, logger *slog.Logger) UserRepository {
	return UserRepository{
		cl,
		logger,
	}
}

func (r UserRepository) Create(ctx context.Context, auth model.AuthenticatedUser) (*ent.User, error) {
	var created *ent.User

	err := WithTx(ctx, r.cl, func(tx *ent.Tx) error {
		var err error
		q := tx.User.Create().
			SetID(auth.ID).
			SetName(auth.Name).
			SetSyncedAt(auth.UpdatedAt)
		if auth.Avatar != "" {
			q.SetAvatar(auth.Avatar)
		}
		created, err = q.Save(ctx)
		return err
	})

	if err != nil {
		r.log.Error("User create error", "error", err)
	}

	return created, err
}

func (r UserRepository) Update(ctx context.Context, auth model.AuthenticatedUser) (*ent.User, error) {
	var updated *ent.User
	err := WithTx(ctx, r.cl, func(tx *ent.Tx) error {
		var err error
		updated, err = tx.User.UpdateOneID(auth.ID).
			SetName(auth.Name).
			SetAvatar(auth.Avatar).
			Save(ctx)

		return err
	})

	if err != nil {
		return nil, errors.Join(model.ErrUpdateFailed, err)
	}

	return updated, err
}

func (r UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.User, error) {
	user, err := r.cl.User.Get(ctx, id)
	if ent.IsNotFound(err) {
		return nil, errors.Join(model.ErrItemNotFound, err)
	}
	return user, err
}
