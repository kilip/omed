// Package service provides business logic and orchestration for user entities.
package service

import (
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/user/domain"
	"github.com/kilip/omed/api/internal/user/model"
)

// UserRepository defines the persistence contract required by UserService.
type UserRepository interface {
	core.ServiceRepository[int64, domain.User, model.CreateUserRequest, model.UpdateUserRequest, model.SearchRequest]
}

// UserService provides business operations and lifecycle management for user accounts.
type UserService struct {
	core.Service[int64, domain.User, model.CreateUserRequest, model.UpdateUserRequest, model.SearchRequest]
}

// NewUserService constructs a new UserService instance using the provided repository.
func NewUserService(repo UserRepository) UserService {
	return UserService{core.NewService(repo)}
}
