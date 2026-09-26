package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/model"
)

type UserRepository interface {
	GetByID(context.Context, uuid.UUID) (*ent.User, error)
	Create(context.Context, model.AuthenticatedUser) (*ent.User, error)
	Update(context.Context, model.AuthenticatedUser) (*ent.User, error)
}

type UserService struct {
	repository UserRepository
}

func NewUserService(repository UserRepository) UserService {
	return UserService{
		repository,
	}
}

func (s UserService) Create(ctx context.Context, auth model.AuthenticatedUser) error {
	_, err := s.repository.Create(ctx, auth)

	return err
}

func (s UserService) Update(ctx context.Context, auth model.AuthenticatedUser) error {
	_, err := s.repository.Update(ctx, auth)
	return err
}

func (s UserService) Ensure(ctx context.Context, auth model.AuthenticatedUser) error {
	existing, err := s.repository.GetByID(ctx, auth.ID)

	if err != nil {
		if model.IsNotFound(err) {
			return s.Create(ctx, auth)
		}
		return err
	}

	if auth.UpdatedAt.After(existing.SyncedAt) {
		return s.Update(ctx, auth)
	}

	return nil
}
