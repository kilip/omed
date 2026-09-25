package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/model"
)

type UserRepository struct {
	client *ent.Client
}

func NewUserRepository(client *ent.Client) UserRepository {
	return UserRepository{
		client,
	}
}

func toUserModel(user ent.User) *model.UserResponse {
	return &model.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Avatar:    user.Avatar,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func (r UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.UserResponse, error) {
	user, err := r.client.User.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return toUserModel(*user), nil
}

func (r UserRepository) Create(ctx context.Context, auth model.AuthenticatedUser) (*model.UserResponse, error) {
	user, err := r.client.User.Create().
		SetID(auth.ID).
		SetName(auth.Name).
		SetAvatar(auth.Avatar).
		Save(ctx)

	if err != nil {
		return nil, err
	}

	return toUserModel(*user), nil
}
