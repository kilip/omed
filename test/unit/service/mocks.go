package service_test

import (
	"context"

	"github.com/kilip/omed/internal/dto"
	"github.com/kilip/omed/internal/entity"
	"github.com/stretchr/testify/mock"
)

type UserRepositoryMock struct {
	mock.Mock
}

func (u *UserRepositoryMock) List(ctx context.Context, req dto.UserListRequest) ([]*entity.User, error) {
	args := u.Called(ctx, req)
	return args.Get(0).([]*entity.User), args.Error(1)
}

func (u *UserRepositoryMock) FindByID(ctx context.Context, id string) (*entity.User, error) {
	args := u.Called(ctx, id)
	return args.Get(0).(*entity.User), args.Error(1)
}

func (u *UserRepositoryMock) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	args := u.Called(ctx, email)
	return args.Get(0).(*entity.User), args.Error(1)
}

func (u *UserRepositoryMock) Create(ctx context.Context, data *entity.User) error {
	args := u.Called(ctx, data)
	return args.Error(0)
}

func (u *UserRepositoryMock) Update(ctx context.Context, data *entity.User) error {
	args := u.Called(ctx, data)
	return args.Error(0)
}

func (u *UserRepositoryMock) Delete(ctx context.Context, id string) error {
	args := u.Called(ctx, id)
	return args.Error(0)
}
