package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilip/omed/internal/domain/user"
	"github.com/kilip/omed/internal/dto"
)

type UserService struct {
	repository user.UserRepository
}

func NewUserService(repository user.UserRepository) *UserService {
	return &UserService{repository}
}

func (s *UserService) Create(ctx context.Context, req dto.UserRequest) (*user.User, error) {
	user := &user.User{
		Email: req.Email,
		Name:  req.Name,
	}

	if err := s.repository.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Update(ctx context.Context, req dto.UserRequest) (*user.User, error) {

	existing, err := s.repository.FindByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	existing.Name = req.Name
	existing.Avatar = req.Avatar

	if err := s.repository.Update(ctx, existing); err != nil {
		return nil, err
	}

	return existing, nil
}

func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return err
	}

	return s.repository.Delete(ctx, id)
}
