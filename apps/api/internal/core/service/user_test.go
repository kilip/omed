package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/core/service"
	"github.com/kilip/omed/api/internal/core/service/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

//go:generate go run go.uber.org/mock/mockgen -source=user.go -destination=mock/mock_user.go -package=mock

type UserSuite struct {
	suite.Suite
	mockRepo *mock.MockUserRepository
	service  service.UserService
}

func (u *UserSuite) SetupTest() {
	ctl := gomock.NewController(u.T())
	u.mockRepo = mock.NewMockUserRepository(ctl)
	u.service = service.NewUserService(u.mockRepo)
}

func (u *UserSuite) Test_GetByID_Success() {
	ctx := context.Background()
	expected := domain.User{
		ID:        1,
		Name:      "John Doe",
		Email:     "john@example.com",
		Avatar:    "https://example.com/avatar.png",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	u.mockRepo.EXPECT().GetByID(ctx, int64(1)).Return(expected, nil)

	actual, err := u.service.GetByID(ctx, 1)
	u.NoError(err)
	u.Equal(expected, actual)
}

func (u *UserSuite) Test_GetByID_NotFound() {
	ctx := context.Background()
	u.mockRepo.EXPECT().GetByID(ctx, int64(999)).Return(domain.User{}, errors.New("user not found"))

	actual, err := u.service.GetByID(ctx, 999)
	u.Error(err)
	u.Equal(domain.User{}, actual)
}

func (u *UserSuite) Test_Search_Success() {
	ctx := context.Background()
	req := model.SearchRequest{
		Name:  "John",
		Limit: 10,
	}
	expectedUsers := []domain.User{
		{
			ID:     1,
			Name:   "John Doe",
			Email:  "john@example.com",
			Avatar: "https://example.com/avatar.png",
		},
	}
	nextCursor := "cursor-123"

	u.mockRepo.EXPECT().Search(ctx, req).Return(expectedUsers, nextCursor, nil)

	users, cursor, err := u.service.Search(ctx, req)
	u.NoError(err)
	u.Equal(expectedUsers, users)
	u.Equal(nextCursor, cursor)
}

func (u *UserSuite) Test_Search_Error() {
	ctx := context.Background()
	req := model.SearchRequest{
		Name: "NonExistent",
	}

	u.mockRepo.EXPECT().Search(ctx, req).Return(nil, "", errors.New("database query error"))

	users, cursor, err := u.service.Search(ctx, req)
	u.Error(err)
	u.Nil(users)
	u.Empty(cursor)
}

func (u *UserSuite) Test_Create_Success() {
	ctx := context.Background()
	req := model.CreateUserRequest{
		Name:   "Jane Doe",
		Email:  "jane@example.com",
		Avatar: "https://example.com/avatar2.png",
	}
	expected := domain.User{
		ID:        2,
		Name:      req.Name,
		Email:     req.Email,
		Avatar:    req.Avatar,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	u.mockRepo.EXPECT().Create(ctx, req).Return(expected, nil)

	actual, err := u.service.Create(ctx, req)
	u.NoError(err)
	u.Equal(expected, actual)
}

func (u *UserSuite) Test_Create_Error() {
	ctx := context.Background()
	req := model.CreateUserRequest{
		Name:   "Jane Doe",
		Email:  "taken@example.com",
		Avatar: "https://example.com/avatar2.png",
	}

	u.mockRepo.EXPECT().Create(ctx, req).Return(domain.User{}, domain.ErrEmailTaken)

	actual, err := u.service.Create(ctx, req)
	u.ErrorIs(err, domain.ErrEmailTaken)
	u.Equal(domain.User{}, actual)
}

func (u *UserSuite) Test_Update_Success() {
	ctx := context.Background()
	id := int64(1)
	req := model.UpdateUserRequest{
		Name:   "John Updated",
		Avatar: "https://example.com/updated.png",
	}
	existing := domain.User{
		ID:     id,
		Name:   "John Doe",
		Email:  "john@example.com",
		Avatar: "https://example.com/old.png",
	}
	updated := domain.User{
		ID:     id,
		Name:   req.Name,
		Email:  existing.Email,
		Avatar: req.Avatar,
	}

	u.mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
	u.mockRepo.EXPECT().Update(ctx, id, req).Return(updated, nil)

	actual, err := u.service.Update(ctx, id, req)
	u.NoError(err)
	u.Equal(updated, actual)
}

func (u *UserSuite) Test_Update_NotFound() {
	ctx := context.Background()
	id := int64(999)
	req := model.UpdateUserRequest{
		Name: "NonExistent",
	}

	// Should fail at GetByID and not call Update
	u.mockRepo.EXPECT().GetByID(ctx, id).Return(domain.User{}, errors.New("user not found"))

	actual, err := u.service.Update(ctx, id, req)
	u.Error(err)
	u.Equal(domain.User{}, actual)
}

func (u *UserSuite) Test_Update_RepositoryError() {
	ctx := context.Background()
	id := int64(1)
	req := model.UpdateUserRequest{
		Name: "John Updated",
	}
	existing := domain.User{
		ID:    id,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	u.mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
	u.mockRepo.EXPECT().Update(ctx, id, req).Return(domain.User{}, errors.New("database update error"))

	actual, err := u.service.Update(ctx, id, req)
	u.Error(err)
	u.Equal(domain.User{}, actual)
}

func (u *UserSuite) Test_Delete_Success() {
	ctx := context.Background()
	id := int64(1)
	existing := domain.User{
		ID:    id,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	u.mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
	u.mockRepo.EXPECT().Delete(ctx, id).Return(nil)

	err := u.service.Delete(ctx, id)
	u.NoError(err)
}

func (u *UserSuite) Test_Delete_NotFound() {
	ctx := context.Background()
	id := int64(999)

	// Should fail at GetByID and not call Delete
	u.mockRepo.EXPECT().GetByID(ctx, id).Return(domain.User{}, errors.New("user not found"))

	err := u.service.Delete(ctx, id)
	u.Error(err)
}

func (u *UserSuite) Test_Delete_RepositoryError() {
	ctx := context.Background()
	id := int64(1)
	existing := domain.User{
		ID:    id,
		Name:  "John Doe",
		Email: "john@example.com",
	}

	u.mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
	u.mockRepo.EXPECT().Delete(ctx, id).Return(errors.New("database delete error"))

	err := u.service.Delete(ctx, id)
	u.Error(err)
}

func (u *UserSuite) Test_SetRepository() {
	ctx := context.Background()
	ctl := gomock.NewController(u.T())
	newMockRepo := mock.NewMockUserRepository(ctl)

	u.service.SetRepository(newMockRepo)

	expected := domain.User{
		ID:   10,
		Name: "New User",
	}
	newMockRepo.EXPECT().GetByID(ctx, int64(10)).Return(expected, nil)

	actual, err := u.service.GetByID(ctx, 10)
	u.NoError(err)
	u.Equal(expected, actual)
}

func TestUserSuite(t *testing.T) {
	suite.Run(t, new(UserSuite))
}
