package test

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilip/omed/internal/domain/user"
	"github.com/kilip/omed/internal/dto"
	"github.com/stretchr/testify/mock"
)

type UserRepositoryMock struct {
	mock.Mock
}

func (u *UserRepositoryMock) List(ctx context.Context, req dto.UserListRequest) ([]*user.User, error) {
	args := u.Called(ctx, req)
	return args.Get(0).([]*user.User), args.Error(1)
}

func (u *UserRepositoryMock) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := u.Called(ctx, id)
	return args.Get(0).(*user.User), args.Error(1)
}

func (u *UserRepositoryMock) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	args := u.Called(ctx, email)
	return args.Get(0).(*user.User), args.Error(1)
}

func (u *UserRepositoryMock) Create(ctx context.Context, data *user.User) error {
	args := u.Called(ctx, data)
	return args.Error(0)
}

func (u *UserRepositoryMock) Update(ctx context.Context, data *user.User) error {
	args := u.Called(ctx, data)
	return args.Error(0)
}

func (u *UserRepositoryMock) Delete(ctx context.Context, id uuid.UUID) error {
	args := u.Called(ctx, id)
	return args.Error(0)
}
