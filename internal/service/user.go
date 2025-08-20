package service

import (
	"context"

	"github.com/kilip/omed/internal/contracts"
	"github.com/kilip/omed/internal/dto"
	"github.com/kilip/omed/internal/entity"
)

type UserService struct {
	repository contracts.UserRepository
}

func NewUserService(repository contracts.UserRepository) *UserService {
	return &UserService{repository}
}

func (s *UserService) List(ctx context.Context, req dto.UserListRequest) ([]*entity.User, error) {
	return s.repository.List(ctx, req)
}

func (s *UserService) Create(ctx context.Context, req dto.UserRequest) (*entity.User, error) {
	user := &entity.User{
		Email: req.Email,
		Name:  req.Name,
	}

	if err := s.repository.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) Update(ctx context.Context, req dto.UserRequest) (*entity.User, error) {
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

func (s *UserService) Delete(ctx context.Context, id string) error {
	_, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return err
	}

	return s.repository.Delete(ctx, id)
}
