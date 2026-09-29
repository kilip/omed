package test

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/service"
	"github.com/kilip/omed/finance/internal/service/mocks"
	"github.com/kilip/omed/finance/internal/shared"
	"github.com/kilip/omed/finance/testutil/cmocks"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type AuthServiceTest struct {
	suite.Suite
	users   *mocks.MockUserRepository
	works   *mocks.MockWorkspaceRepository
	svc     service.AuthService
	handler *cmocks.MockHandler
}

func (s *AuthServiceTest) SetupTest() {
	ctl := gomock.NewController(s.T())
	s.users = mocks.NewMockUserRepository(ctl)
	s.works = mocks.NewMockWorkspaceRepository(ctl)
	s.handler = cmocks.NewMockHandler(ctl)
	s.svc = service.NewAuthService(s.users, s.works, slog.New(s.handler))
}

func (s *AuthServiceTest) TestCreateSnapshot() {
	ctx := s.T().Context()
	id := shared.GenerateID()
	s.users.EXPECT().GetByID(ctx, id).Times(1).Return(nil, errors.Join(shared.ErrItemNotFound))
	s.users.EXPECT().Create(ctx, gomock.Any()).Times(1).Return(nil)
	s.works.EXPECT().GetByID(ctx, id).Times(1).Return(nil, errors.Join(shared.ErrItemNotFound))
	s.works.EXPECT().Create(ctx, gomock.Any()).Times(1).Return(nil)

	err := s.svc.CheckSnapshot(ctx, shared.AuthenticatedUser{
		ID:            id,
		Name:          "Name",
		WorkspaceID:   id,
		WorkspaceName: "test",
	})

	require.NoError(s.T(), err)
}

func (s *AuthServiceTest) TestUpdateSnapshot() {
	ctx := s.T().Context()
	eightDaysAgo := time.Now().AddDate(0, 0, -8)
	user := model.UserSnapshot{
		ID:       shared.GenerateID(),
		Name:     "Test User",
		SyncedAt: eightDaysAgo,
	}
	ws := model.WorkspaceSnapshot{
		ID:       shared.GenerateID(),
		Name:     "Test Workspace",
		SyncedAt: eightDaysAgo,
	}

	s.users.EXPECT().GetByID(ctx, user.ID).Times(1).Return(&user, nil)
	s.users.EXPECT().Update(ctx, gomock.Any()).Times(1).Return(nil)
	s.works.EXPECT().GetByID(ctx, ws.ID).Times(1).Return(&ws, nil)
	s.works.EXPECT().Update(ctx, gomock.Any()).Times(1).Return(nil)

	s.svc.CheckSnapshot(ctx, shared.AuthenticatedUser{
		ID:            user.ID,
		Name:          user.Name,
		WorkspaceID:   ws.ID,
		WorkspaceName: ws.Name,
	})
}

func TestAuhService(t *testing.T) {
	suite.Run(t, new(AuthServiceTest))
}
