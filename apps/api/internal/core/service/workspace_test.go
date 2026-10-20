package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/core/service"
	"github.com/kilip/omed/api/internal/core/service/mock"
	"github.com/kilip/omed/api/internal/platform/s3x"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

//go:generate go run go.uber.org/mock/mockgen -source workspace.go -destination mock/mock_workspace.go -package mock

const callerID int64 = 10

type WorkspaceSuite struct {
	suite.Suite
	mockRepo *mock.MockWorkspaceRepository
	service  service.WorkspaceService
	ctx      context.Context
}

func (s *WorkspaceSuite) SetupTest() {
	ctl := gomock.NewController(s.T())
	s.mockRepo = mock.NewMockWorkspaceRepository(ctl)
	s.service = service.NewWorkspaceService(s.mockRepo)
	s.ctx = context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:    callerID,
		Roles: []base.UserRole{base.UserRoleUser},
	})
}

func (s *WorkspaceSuite) Test_GetByID_Success() {
	ctx := context.Background()
	expected := domain.Workspace{
		ID:          1,
		Name:        "Test Workspace",
		Description: "A sample workspace",
		OwnerID:     10,
		Owner: domain.User{
			ID:    10,
			Name:  "Alice",
			Email: "alice@example.com",
		},
		CreatedBy: base.ByUser{Name: "Alice"},
		UpdatedBy: base.ByUser{Name: "Alice"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.mockRepo.EXPECT().GetByID(ctx, int64(1)).Return(expected, nil)

	actual, err := s.service.GetByID(ctx, 1)
	s.NoError(err)
	s.Equal(expected, actual)
}

func (s *WorkspaceSuite) Test_GetByID_NotFound() {
	ctx := context.Background()
	s.mockRepo.EXPECT().GetByID(ctx, int64(999)).Return(domain.Workspace{}, errors.New("workspace not found"))

	actual, err := s.service.GetByID(ctx, 999)
	s.Error(err)
	s.Equal(domain.Workspace{}, actual)
}

func (s *WorkspaceSuite) Test_Search_Success() {
	ctx := s.ctx
	req := model.SearchWorkspaceRequest{
		Name:  "Test",
		Limit: 10,
	}
	expectedWorkspaces := []domain.Workspace{
		{
			ID:          1,
			Name:        "Test Workspace",
			Description: "Desc",
			OwnerID:     10,
		},
	}
	nextCursor := "cursor-123"

	scoped := req
	scoped.MemberID = callerID
	s.mockRepo.EXPECT().Search(ctx, scoped).Return(expectedWorkspaces, nextCursor, nil)

	workspaces, cursor, err := s.service.Search(ctx, req)
	s.NoError(err)
	s.Equal(expectedWorkspaces, workspaces)
	s.Equal(nextCursor, cursor)
}

func (s *WorkspaceSuite) Test_Search_Error() {
	ctx := s.ctx
	req := model.SearchWorkspaceRequest{
		Name: "NonExistent",
	}

	scoped := req
	scoped.MemberID = callerID
	s.mockRepo.EXPECT().Search(ctx, scoped).Return(nil, "", errors.New("database query error"))

	workspaces, cursor, err := s.service.Search(ctx, req)
	s.Error(err)
	s.Nil(workspaces)
	s.Empty(cursor)
}

func (s *WorkspaceSuite) Test_Create_Success() {
	ctx := s.ctx
	req := model.CreateWorkspaceRequest{
		Name:        "New Workspace",
		Description: "New Description",
	}
	expected := domain.Workspace{
		ID:          2,
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     10,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	owned := req
	owned.OwnerID = callerID
	s.mockRepo.EXPECT().Create(ctx, owned).Return(expected, nil)

	actual, err := s.service.Create(ctx, req)
	s.NoError(err)
	s.Equal(expected, actual)
}

func (s *WorkspaceSuite) Test_Create_AssignsCallerAsOwner() {
	ctx := s.ctx
	req := model.CreateWorkspaceRequest{
		Name:    "Owned By Caller",
		OwnerID: 99,
	}
	owned := req
	owned.OwnerID = callerID

	s.mockRepo.EXPECT().Create(ctx, owned).Return(domain.Workspace{ID: 3, OwnerID: callerID}, nil)

	actual, err := s.service.Create(ctx, req)
	s.NoError(err)
	s.Equal(callerID, actual.OwnerID)
}

func (s *WorkspaceSuite) Test_Create_Error() {
	ctx := s.ctx
	req := model.CreateWorkspaceRequest{
		Name: "New Workspace",
	}

	owned := req
	owned.OwnerID = callerID
	s.mockRepo.EXPECT().Create(ctx, owned).Return(domain.Workspace{}, errors.New("cannot create workspace"))

	actual, err := s.service.Create(ctx, req)
	s.Error(err)
	s.Equal(domain.Workspace{}, actual)
}

func (s *WorkspaceSuite) Test_Update_Success() {
	ctx := context.Background()
	id := int64(1)
	req := model.UpdateWorkspaceRequest{
		Name:        "Updated Name",
		Description: "Updated Description",
	}
	existing := domain.Workspace{
		ID:          id,
		Name:        "Original Name",
		Description: "Original Description",
		OwnerID:     10,
	}
	updated := domain.Workspace{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     10,
	}

	s.mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
	s.mockRepo.EXPECT().Update(ctx, id, req).Return(updated, nil)

	actual, err := s.service.Update(ctx, id, req)
	s.NoError(err)
	s.Equal(updated, actual)
}

func (s *WorkspaceSuite) Test_Update_NotFound() {
	ctx := context.Background()
	id := int64(999)
	req := model.UpdateWorkspaceRequest{
		Name: "NonExistent",
	}

	s.mockRepo.EXPECT().GetByID(ctx, id).Return(domain.Workspace{}, errors.New("workspace not found"))

	actual, err := s.service.Update(ctx, id, req)
	s.Error(err)
	s.Equal(domain.Workspace{}, actual)
}

func (s *WorkspaceSuite) Test_Update_RepositoryError() {
	ctx := context.Background()
	id := int64(1)
	req := model.UpdateWorkspaceRequest{
		Name: "Updated Name",
	}
	existing := domain.Workspace{
		ID:   id,
		Name: "Original",
	}

	s.mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
	s.mockRepo.EXPECT().Update(ctx, id, req).Return(domain.Workspace{}, errors.New("database update error"))

	actual, err := s.service.Update(ctx, id, req)
	s.Error(err)
	s.Equal(domain.Workspace{}, actual)
}

func (s *WorkspaceSuite) Test_Delete_Success() {
	ctx := context.Background()
	id := int64(1)
	existing := domain.Workspace{
		ID:   id,
		Name: "Workspace 1",
	}

	s.mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
	s.mockRepo.EXPECT().Delete(ctx, id).Return(nil)

	err := s.service.Delete(ctx, id)
	s.NoError(err)
}

func (s *WorkspaceSuite) Test_Delete_NotFound() {
	ctx := context.Background()
	id := int64(999)

	s.mockRepo.EXPECT().GetByID(ctx, id).Return(domain.Workspace{}, errors.New("workspace not found"))

	err := s.service.Delete(ctx, id)
	s.Error(err)
}

func (s *WorkspaceSuite) Test_Delete_RepositoryError() {
	ctx := context.Background()
	id := int64(1)
	existing := domain.Workspace{
		ID:   id,
		Name: "Workspace 1",
	}

	s.mockRepo.EXPECT().GetByID(ctx, id).Return(existing, nil)
	s.mockRepo.EXPECT().Delete(ctx, id).Return(errors.New("database delete error"))

	err := s.service.Delete(ctx, id)
	s.Error(err)
}

func (s *WorkspaceSuite) Test_SetRepository() {
	ctx := context.Background()
	ctl := gomock.NewController(s.T())
	newMockRepo := mock.NewMockWorkspaceRepository(ctl)

	s.service.SetRepository(newMockRepo)

	expected := domain.Workspace{
		ID:   10,
		Name: "New Workspace",
	}
	newMockRepo.EXPECT().GetByID(ctx, int64(10)).Return(expected, nil)

	actual, err := s.service.GetByID(ctx, 10)
	s.NoError(err)
	s.Equal(expected, actual)
}

func (s *WorkspaceSuite) Test_CreateAndDelete_ProvisionsAndDeletesS3Bucket() {
	ctx := s.ctx
	st, err := s3x.New(s3x.Config{Driver: "memory"})
	s.Require().NoError(err)

	svcWithStorage := service.NewWorkspaceService(s.mockRepo, st)

	req := model.CreateWorkspaceRequest{Name: "Bucket WS"}
	owned := req
	owned.OwnerID = callerID
	createdWS := domain.Workspace{ID: 88, Name: "Bucket WS", OwnerID: callerID}

	s.mockRepo.EXPECT().Create(ctx, owned).Return(createdWS, nil)
	ws, err := svcWithStorage.Create(ctx, req)
	s.Require().NoError(err)
	s.Equal(int64(88), ws.ID)

	exists, err := st.BucketExists(ctx, "omed-ws-88")
	s.Require().NoError(err)
	s.True(exists)

	s.mockRepo.EXPECT().GetByID(ctx, int64(88)).Return(createdWS, nil)
	s.mockRepo.EXPECT().Delete(ctx, int64(88)).Return(nil)
	s.Require().NoError(svcWithStorage.Delete(ctx, 88))

	exists, err = st.BucketExists(ctx, "omed-ws-88")
	s.Require().NoError(err)
	s.False(exists)
}

func TestWorkspaceSuite(t *testing.T) {
	suite.Run(t, new(WorkspaceSuite))
}
