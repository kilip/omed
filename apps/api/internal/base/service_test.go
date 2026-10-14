package base_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/base/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

//go:generate go run go.uber.org/mock/mockgen -source=service.go -destination=mock/mock_service.go -package=mock

type dummyEntity struct {
	ID   int64
	Name string
}

type dummyCreateReq struct {
	Name string
}

type dummyUpdateReq struct {
	Name string
}

type dummySearchReq struct {
	Query string
}

type ServiceSuite struct {
	suite.Suite
	ctrl     *gomock.Controller
	mockRepo *mock.MockServiceRepository[int64, dummyEntity, dummyCreateReq, dummyUpdateReq, dummySearchReq]
	service  base.Service[int64, dummyEntity, dummyCreateReq, dummyUpdateReq, dummySearchReq]
}

func (s *ServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockRepo = mock.NewMockServiceRepository[int64, dummyEntity, dummyCreateReq, dummyUpdateReq, dummySearchReq](s.ctrl)
	s.service = base.NewService(s.mockRepo)
}

func (s *ServiceSuite) Test_GetByID_Success() {
	ctx := context.Background()
	expected := dummyEntity{ID: 1, Name: "Test"}
	s.mockRepo.EXPECT().GetByID(ctx, int64(1)).Return(expected, nil)

	actual, err := s.service.GetByID(ctx, 1)
	s.NoError(err)
	s.Equal(expected, actual)
}

func (s *ServiceSuite) Test_GetByID_Error() {
	ctx := context.Background()
	expectedErr := errors.New("not found")
	s.mockRepo.EXPECT().GetByID(ctx, int64(999)).Return(dummyEntity{}, expectedErr)

	actual, err := s.service.GetByID(ctx, 999)
	s.ErrorIs(err, expectedErr)
	s.Equal(dummyEntity{}, actual)
}

func (s *ServiceSuite) Test_Search_Success() {
	ctx := context.Background()
	req := dummySearchReq{Query: "test"}
	expectedEntities := []dummyEntity{{ID: 1, Name: "Test"}}
	cursor := "next-cursor"
	s.mockRepo.EXPECT().Search(ctx, req).Return(expectedEntities, cursor, nil)

	entities, nextCursor, err := s.service.Search(ctx, req)
	s.NoError(err)
	s.Equal(expectedEntities, entities)
	s.Equal(cursor, nextCursor)
}

func (s *ServiceSuite) Test_Search_Error() {
	ctx := context.Background()
	req := dummySearchReq{Query: "error"}
	expectedErr := errors.New("search error")
	s.mockRepo.EXPECT().Search(ctx, req).Return(nil, "", expectedErr)

	entities, cursor, err := s.service.Search(ctx, req)
	s.ErrorIs(err, expectedErr)
	s.Nil(entities)
	s.Empty(cursor)
}

func (s *ServiceSuite) Test_Create_Success() {
	ctx := context.Background()
	req := dummyCreateReq{Name: "New Entity"}
	expected := dummyEntity{ID: 2, Name: "New Entity"}
	s.mockRepo.EXPECT().Create(ctx, req).Return(expected, nil)

	actual, err := s.service.Create(ctx, req)
	s.NoError(err)
	s.Equal(expected, actual)
}

func (s *ServiceSuite) Test_Create_Error() {
	ctx := context.Background()
	req := dummyCreateReq{Name: "Fail"}
	expectedErr := errors.New("create error")
	s.mockRepo.EXPECT().Create(ctx, req).Return(dummyEntity{}, expectedErr)

	actual, err := s.service.Create(ctx, req)
	s.ErrorIs(err, expectedErr)
	s.Equal(dummyEntity{}, actual)
}

func (s *ServiceSuite) Test_Update_Success() {
	ctx := context.Background()
	id := int64(1)
	req := dummyUpdateReq{Name: "Updated Name"}
	existing := dummyEntity{ID: id, Name: "Old Name"}
	updated := dummyEntity{ID: id, Name: req.Name}

	s.mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
	s.mockRepo.EXPECT().Update(ctx, id, req).Return(updated, nil)

	actual, err := s.service.Update(ctx, id, req)
	s.NoError(err)
	s.Equal(updated, actual)
}

func (s *ServiceSuite) Test_Update_NotFound_Error() {
	ctx := context.Background()
	id := int64(999)
	req := dummyUpdateReq{Name: "Updated Name"}
	expectedErr := errors.New("not found")

	s.mockRepo.EXPECT().GetByID(ctx, id).Return(dummyEntity{}, expectedErr)

	actual, err := s.service.Update(ctx, id, req)
	s.ErrorIs(err, expectedErr)
	s.Equal(dummyEntity{}, actual)
}

func (s *ServiceSuite) Test_Update_RepositoryError() {
	ctx := context.Background()
	id := int64(1)
	req := dummyUpdateReq{Name: "Updated Name"}
	existing := dummyEntity{ID: id, Name: "Old Name"}
	expectedErr := errors.New("update error")

	s.mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
	s.mockRepo.EXPECT().Update(ctx, id, req).Return(dummyEntity{}, expectedErr)

	actual, err := s.service.Update(ctx, id, req)
	s.ErrorIs(err, expectedErr)
	s.Equal(dummyEntity{}, actual)
}

func (s *ServiceSuite) Test_Delete_Success() {
	ctx := context.Background()
	id := int64(1)
	existing := dummyEntity{ID: id, Name: "To Delete"}

	s.mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
	s.mockRepo.EXPECT().Delete(ctx, id).Return(nil)

	err := s.service.Delete(ctx, id)
	s.NoError(err)
}

func (s *ServiceSuite) Test_Delete_NotFound_Error() {
	ctx := context.Background()
	id := int64(999)
	expectedErr := errors.New("not found")

	s.mockRepo.EXPECT().GetByID(ctx, id).Return(dummyEntity{}, expectedErr)

	err := s.service.Delete(ctx, id)
	s.ErrorIs(err, expectedErr)
}

func (s *ServiceSuite) Test_Delete_RepositoryError() {
	ctx := context.Background()
	id := int64(1)
	existing := dummyEntity{ID: id, Name: "To Delete"}
	expectedErr := errors.New("delete error")

	s.mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
	s.mockRepo.EXPECT().Delete(ctx, id).Return(expectedErr)

	err := s.service.Delete(ctx, id)
	s.ErrorIs(err, expectedErr)
}

func (s *ServiceSuite) Test_SetRepository() {
	ctx := context.Background()
	newMockRepo := mock.NewMockServiceRepository[int64, dummyEntity, dummyCreateReq, dummyUpdateReq, dummySearchReq](s.ctrl)
	s.service.SetRepository(newMockRepo)

	expected := dummyEntity{ID: 10, Name: "New Repo Entity"}
	newMockRepo.EXPECT().GetByID(ctx, int64(10)).Return(expected, nil)

	actual, err := s.service.GetByID(ctx, 10)
	s.NoError(err)
	s.Equal(expected, actual)
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(ServiceSuite))
}
