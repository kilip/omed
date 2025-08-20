package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kilip/omed/internal/domain/user"
	"github.com/kilip/omed/internal/dto"
	"github.com/kilip/omed/internal/service"
	"github.com/kilip/omed/test"
	"github.com/stretchr/testify/assert"
)

var req = dto.UserRequest{
	ID:       "405A7AC9-BCBC-4DDA-9C70-B865FE8F686B",
	Email:    "test@example.com",
	Password: "test",
	Name:     "Test user",
}
var res = &user.User{
	Email: req.Email,
	Name:  req.Name,
}

var reqList = dto.UserListRequest{}

func TestList(t *testing.T) {

	ctx := context.Background()
	repo := new(test.UserRepositoryMock)
	repo.On("List", ctx, reqList).Return([]*user.User{res}, nil)

	svc := service.NewUserService(repo)
	users, err := svc.List(ctx, reqList)

	assert.Nil(t, err)
	repo.AssertExpectations(t)
	assert.Len(t, users, 1)
}

func TestListError(t *testing.T) {
	ctx := context.Background()
	repo := new(test.UserRepositoryMock)
	repo.On("List", ctx, reqList).Return([]*user.User{}, errors.New("failed"))

	svc := service.NewUserService(repo)
	users, err := svc.List(ctx, reqList)

	assert.NotNil(t, err)
	repo.AssertExpectations(t)
	assert.Len(t, users, 0)
}

func TestCreate(t *testing.T) {
	ctx := context.Background()
	repo := new(test.UserRepositoryMock)
	repo.On("Create", ctx, res).Return(nil)

	svc := service.NewUserService(repo)
	user, err := svc.Create(ctx, req)

	assert.Nil(t, err)
	assert.NotNil(t, user)
	repo.AssertExpectations(t)
}

func TestCreateError(t *testing.T) {
	ctx := context.Background()
	repo := new(test.UserRepositoryMock)
	repo.On("Create", ctx, res).Return(errors.New("failed"))

	svc := service.NewUserService(repo)
	user, err := svc.Create(ctx, req)

	repo.AssertExpectations(t)
	assert.NotNil(t, err)
	assert.Nil(t, user)
}

func TestUpdateSuccess(t *testing.T) {
	ctx := context.Background()
	repo := new(test.UserRepositoryMock)
	repo.On("FindByID", ctx, req.ID).Return(res, nil)
	repo.On("Update", ctx, res).Return(nil)

	svc := service.NewUserService(repo)
	user, err := svc.Update(ctx, req)

	repo.AssertExpectations(t)
	assert.Nil(t, err)
	assert.NotNil(t, user)
}

func TestUpdateWithInvalidId(t *testing.T) {
	ctx := context.Background()
	repo := new(test.UserRepositoryMock)
	repo.On("FindByID", ctx, req.ID).Return(&user.User{}, errors.New("failed"))

	svc := service.NewUserService(repo)
	_, err := svc.Update(ctx, req)

	repo.AssertExpectations(t)
	assert.NotNil(t, err)
}

func TestDeleteSuccessfull(t *testing.T) {
	ctx := context.Background()
	repo := new(test.UserRepositoryMock)
	repo.On("FindByID", ctx, req.ID).Return(res, nil)
	repo.On("Delete", ctx, req.ID).Return(nil)

	svc := service.NewUserService(repo)
	err := svc.Delete(ctx, req.ID)

	repo.AssertExpectations(t)
	assert.Nil(t, err)

}

func TestDeleteWithInvalidID(t *testing.T) {
	ctx := context.Background()
	repo := new(test.UserRepositoryMock)
	repo.On("FindByID", ctx, req.ID).Return(&user.User{}, errors.New("failed"))
	// repo.On("Delete", ctx, req.ID).Return(nil)

	svc := service.NewUserService(repo)
	err := svc.Delete(ctx, req.ID)

	repo.AssertExpectations(t)
	assert.NotNil(t, err)
}

func TestDeleteFailed(t *testing.T) {
	ctx := context.Background()
	repo := new(test.UserRepositoryMock)
	repo.On("FindByID", ctx, req.ID).Return(res, nil)
	repo.On("Delete", ctx, req.ID).Return(errors.New("failed"))

	svc := service.NewUserService(repo)
	err := svc.Delete(ctx, req.ID)

	repo.AssertExpectations(t)
	assert.NotNil(t, err)
}
