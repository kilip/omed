// Package service provides business logic and orchestration for user entities.
package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
)

// UserRepository defines the persistence contract required by UserService.
type UserRepository interface {
	base.ServiceRepository[int64, domain.User, model.CreateUserRequest, model.UpdateUserRequest, model.SearchRequest]
	FindByEmail(c context.Context, email string) (domain.User, error)
}

// ProvisionInput holds the input parameters for finding or creating a verified user.
type ProvisionInput struct {
	Email  string
	Name   string
	Avatar string
	Locale string
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

// FindOrCreateVerified looks up an existing user by email (if provided) or provisions
// a new verified user with normalized locale and currency.
func (s UserService) FindOrCreateVerified(ctx context.Context, in ProvisionInput) (domain.User, error) {
	if in.Email != "" {
		usr, err := s.userRepo.FindByEmail(ctx, in.Email)
		if err == nil {
			return usr, nil
		}
	}

	name := in.Name
	if name == "" {
		name = strings.Split(in.Email, "@")[0]
	}

	locale, currency := NormalizeLocaleAndCurrency(in.Locale)
	createdUser, err := s.userRepo.Create(ctx, model.CreateUserRequest{
		Name:          name,
		Email:         in.Email,
		Avatar:        in.Avatar,
		Locale:        locale,
		Currency:      currency,
		EmailVerified: true,
	})
	if err != nil {
		if errors.Is(err, domain.ErrEmailTaken) {
			return s.userRepo.FindByEmail(ctx, in.Email)
		}
		return domain.User{}, err
	}

	return createdUser, nil
}

// NewUserService constructs a new UserService instance using the provided repository.
func NewUserService(repo UserRepository) UserService {
	return UserService{
		Service:  base.NewService(repo),
		userRepo: repo,
	}
}
