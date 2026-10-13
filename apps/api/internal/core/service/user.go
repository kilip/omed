// Package service provides business logic and orchestration for user entities.
package service

import (
	"context"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
)

// UserRepository defines the persistence contract required by UserService.
type UserRepository interface {
	base.ServiceRepository[int64, domain.User, model.CreateUserRequest, model.UpdateUserRequest, model.SearchRequest]
	FindByEmail(c context.Context, email string) (domain.User, error)
}

// UserService provides business operations and lifecycle management for user accounts.
type UserService struct {
	base.Service[int64, domain.User, model.CreateUserRequest, model.UpdateUserRequest, model.SearchRequest]
	userRepo UserRepository
}

// FindByEmail finds a user by email.
func (s UserService) FindByEmail(c context.Context, email string) (domain.User, error) {
	return s.userRepo.FindByEmail(c, email)
}

// NewUserService constructs a new UserService instance using the provided repository.
func NewUserService(repo UserRepository) UserService {
	return UserService{
		Service:  base.NewService(repo),
		userRepo: repo,
	}
}
