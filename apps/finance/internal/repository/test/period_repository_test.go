package test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/suite"

	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/config"
	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/repository"
)

type PeriodRepositorySuite struct {
	suite.Suite
	client *ent.Client
	repo   repository.PeriodRepository
	ctx    context.Context
	user   core.AuthenticatedUser
}

func (s *PeriodRepositorySuite) SetupSuite() {
	dbName := fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", uuid.New().String())
	client, err := ent.Open("sqlite3", dbName)
	s.Require().NoError(err)

	err = client.Schema.Create(context.Background())
	s.Require().NoError(err)

	config.ConfigureDBClient(client)
	s.client = client

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s.repo = repository.NewPeriodRepository(client, logger)
}

func (s *PeriodRepositorySuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *PeriodRepositorySuite) createUser(u core.AuthenticatedUser) {
	_, err := s.client.User.Create().
		SetID(u.ID).
		SetName(u.Name).
		SetAvatar("").
		Save(context.Background())
	s.Require().NoError(err)
}

func (s *PeriodRepositorySuite) SetupTest() {
	s.user = core.AuthenticatedUser{
		ID:             uuid.New(),
		Name:           "Test User",
		WorkspaceID:    uuid.New(),
		WorkspaceName:  "Test Workspace",
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
	}
	s.ctx = core.ContextWithUser(context.Background(), s.user)
	s.createUser(s.user)
}

func (s *PeriodRepositorySuite) TearDownTest() {
	_, _ = s.client.Entry.Delete().Exec(s.ctx)
	_, _ = s.client.Period.Delete().Exec(s.ctx)
	_, _ = s.client.User.Delete().Exec(context.Background())
}

// ---------------------------------------------------------------------------
// Create Tests
// ---------------------------------------------------------------------------

func (s *PeriodRepositorySuite) TestCreate_Success() {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)

	req := model.CreatePeriodRequest{
		StartDate: start,
		EndDate:   end,
	}

	res, err := s.repo.Create(s.ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(res)

	s.NotEqual(uuid.Nil, res.ID)
	s.Equal(s.user.WorkspaceID, res.WorkspaceID)
	s.True(start.Equal(res.StartDate))
	s.True(end.Equal(res.EndDate))
	s.Equal(model.PeriodStatusOpen, res.Status)
	s.Equal(s.user.ID, res.CreatedBy)
	s.Equal(s.user.Name, res.CreatedByName)
	s.Equal(s.user.ID, res.UpdatedBy)
	s.Equal(s.user.Name, res.UpdatedByName)
	s.False(res.CreatedAt.IsZero())
	s.False(res.UpdatedAt.IsZero())
}

func (s *PeriodRepositorySuite) TestCreate_ExplicitStatus() {
	start := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)

	req := model.CreatePeriodRequest{
		StartDate: start,
		EndDate:   end,
		Status:    model.PeriodStatusClosed,
	}

	res, err := s.repo.Create(s.ctx, req)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(model.PeriodStatusClosed, res.Status)
}

// ---------------------------------------------------------------------------
// GetByID Tests
// ---------------------------------------------------------------------------

func (s *PeriodRepositorySuite) TestGetByID_Success() {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)

	created, err := s.repo.Create(s.ctx, model.CreatePeriodRequest{
		StartDate: start,
		EndDate:   end,
	})
	s.Require().NoError(err)

	res, err := s.repo.GetByID(s.ctx, created.ID)
	s.Require().NoError(err)
	s.Require().NotNil(res)
	s.Equal(created.ID, res.ID)
	s.True(start.Equal(res.StartDate))
	s.True(end.Equal(res.EndDate))
	s.Equal(model.PeriodStatusOpen, res.Status)
	s.Equal(s.user.Name, res.CreatedByName)
}

func (s *PeriodRepositorySuite) TestGetByID_NotFound() {
	res, err := s.repo.GetByID(s.ctx, uuid.New())
	s.Error(err)
	s.True(errors.Is(err, core.ErrItemNotFound))
	s.Nil(res)
}

func (s *PeriodRepositorySuite) TestGetByID_WorkspaceIsolation() {
	created, err := s.repo.Create(s.ctx, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})
	s.Require().NoError(err)

	user2 := core.AuthenticatedUser{
		ID:             uuid.New(),
		Name:           "User Two",
		WorkspaceID:    uuid.New(),
		WorkspaceName:  "Workspace Two",
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
	}
	s.createUser(user2)
	ctx2 := core.ContextWithUser(context.Background(), user2)

	res, err := s.repo.GetByID(ctx2, created.ID)
	s.Error(err)
	s.True(errors.Is(err, core.ErrItemNotFound))
	s.Nil(res)
}

// ---------------------------------------------------------------------------
// List Tests
// ---------------------------------------------------------------------------

func (s *PeriodRepositorySuite) TestList_Empty() {
	res, err := s.repo.List(s.ctx, model.ListPeriodRequest{})
	s.Require().NoError(err)
	s.Empty(res)
}

func (s *PeriodRepositorySuite) TestList_All() {
	_, err := s.repo.Create(s.ctx, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})
	s.Require().NoError(err)

	_, err = s.repo.Create(s.ctx, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
	})
	s.Require().NoError(err)

	res, err := s.repo.List(s.ctx, model.ListPeriodRequest{})
	s.Require().NoError(err)
	s.Len(res, 2)
}

func (s *PeriodRepositorySuite) TestList_FilterByStatus() {
	p1, err := s.repo.Create(s.ctx, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	})
	s.Require().NoError(err)

	p2, err := s.repo.Create(s.ctx, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusClosed,
	})
	s.Require().NoError(err)

	openList, err := s.repo.List(s.ctx, model.ListPeriodRequest{Status: model.PeriodStatusOpen})
	s.Require().NoError(err)
	s.Len(openList, 1)
	s.Equal(p1.ID, openList[0].ID)

	closedList, err := s.repo.List(s.ctx, model.ListPeriodRequest{Status: model.PeriodStatusClosed})
	s.Require().NoError(err)
	s.Len(closedList, 1)
	s.Equal(p2.ID, closedList[0].ID)
}

func (s *PeriodRepositorySuite) TestList_FilterByDate_YYYYMMDD() {
	p1, err := s.repo.Create(s.ctx, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})
	s.Require().NoError(err)

	_, err = s.repo.Create(s.ctx, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
	})
	s.Require().NoError(err)

	res, err := s.repo.List(s.ctx, model.ListPeriodRequest{Date: "2026-01-15"})
	s.Require().NoError(err)
	s.Len(res, 1)
	s.Equal(p1.ID, res[0].ID)
}

func (s *PeriodRepositorySuite) TestList_FilterByDate_RFC3339() {
	p1, err := s.repo.Create(s.ctx, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})
	s.Require().NoError(err)

	res, err := s.repo.List(s.ctx, model.ListPeriodRequest{Date: "2026-01-15T12:00:00Z"})
	s.Require().NoError(err)
	s.Len(res, 1)
	s.Equal(p1.ID, res[0].ID)
}

func (s *PeriodRepositorySuite) TestList_FilterByDate_InvalidFormat() {
	res, err := s.repo.List(s.ctx, model.ListPeriodRequest{Date: "invalid-date"})
	s.Error(err)
	s.Nil(res)
}

func (s *PeriodRepositorySuite) TestList_WorkspaceIsolation() {
	_, err := s.repo.Create(s.ctx, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})
	s.Require().NoError(err)

	user2 := core.AuthenticatedUser{
		ID:             uuid.New(),
		Name:           "User Two",
		WorkspaceID:    uuid.New(),
		WorkspaceName:  "Workspace Two",
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
	}
	s.createUser(user2)
	ctx2 := core.ContextWithUser(context.Background(), user2)
	defer func() {
		_, _ = s.client.Period.Delete().Exec(ctx2)
	}()

	_, err = s.repo.Create(ctx2, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
	})
	s.Require().NoError(err)

	list1, err := s.repo.List(s.ctx, model.ListPeriodRequest{})
	s.Require().NoError(err)
	s.Len(list1, 1)

	list2, err := s.repo.List(ctx2, model.ListPeriodRequest{})
	s.Require().NoError(err)
	s.Len(list2, 1)
	s.NotEqual(list1[0].ID, list2[0].ID)
}

// ---------------------------------------------------------------------------
// Update Tests
// ---------------------------------------------------------------------------

func (s *PeriodRepositorySuite) TestUpdate_Success() {
	created, err := s.repo.Create(s.ctx, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})
	s.Require().NoError(err)
	s.Equal(model.PeriodStatusOpen, created.Status)

	updated, err := s.repo.Update(s.ctx, created.ID, model.UpdatePeriodRequest{
		Status: model.PeriodStatusClosed,
	})
	s.Require().NoError(err)
	s.Require().NotNil(updated)
	s.Equal(model.PeriodStatusClosed, updated.Status)

	fetched, err := s.repo.GetByID(s.ctx, created.ID)
	s.Require().NoError(err)
	s.Equal(model.PeriodStatusClosed, fetched.Status)
}

func (s *PeriodRepositorySuite) TestUpdate_NotFound() {
	res, err := s.repo.Update(s.ctx, uuid.New(), model.UpdatePeriodRequest{
		Status: model.PeriodStatusClosed,
	})
	s.Error(err)
	s.True(errors.Is(err, core.ErrItemNotFound))
	s.Nil(res)
}

func (s *PeriodRepositorySuite) TestUpdate_WorkspaceIsolation() {
	created, err := s.repo.Create(s.ctx, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})
	s.Require().NoError(err)

	user2 := core.AuthenticatedUser{
		ID:             uuid.New(),
		Name:           "User Two",
		WorkspaceID:    uuid.New(),
		WorkspaceName:  "Workspace Two",
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
	}
	s.createUser(user2)
	ctx2 := core.ContextWithUser(context.Background(), user2)

	res, err := s.repo.Update(ctx2, created.ID, model.UpdatePeriodRequest{
		Status: model.PeriodStatusClosed,
	})
	s.Error(err)
	s.True(errors.Is(err, core.ErrItemNotFound))
	s.Nil(res)
}

// ---------------------------------------------------------------------------
// Delete Tests
// ---------------------------------------------------------------------------

func (s *PeriodRepositorySuite) TestDelete_Success() {
	created, err := s.repo.Create(s.ctx, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})
	s.Require().NoError(err)

	err = s.repo.Delete(s.ctx, created.ID)
	s.Require().NoError(err)

	res, err := s.repo.GetByID(s.ctx, created.ID)
	s.Error(err)
	s.True(errors.Is(err, core.ErrItemNotFound))
	s.Nil(res)
}

func (s *PeriodRepositorySuite) TestDelete_NotFound() {
	err := s.repo.Delete(s.ctx, uuid.New())
	s.Error(err)
	s.True(errors.Is(err, core.ErrItemNotFound))
}

func (s *PeriodRepositorySuite) TestDelete_WorkspaceIsolation() {
	created, err := s.repo.Create(s.ctx, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})
	s.Require().NoError(err)

	user2 := core.AuthenticatedUser{
		ID:             uuid.New(),
		Name:           "User Two",
		WorkspaceID:    uuid.New(),
		WorkspaceName:  "Workspace Two",
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
	}
	s.createUser(user2)
	ctx2 := core.ContextWithUser(context.Background(), user2)

	err = s.repo.Delete(ctx2, created.ID)
	s.Error(err)
	s.True(errors.Is(err, core.ErrItemNotFound))
}

// ---------------------------------------------------------------------------
// FindOverlapping Tests
// ---------------------------------------------------------------------------

func (s *PeriodRepositorySuite) TestFindOverlapping() {
	p1, err := s.repo.Create(s.ctx, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 20, 23, 59, 59, 0, time.UTC),
	})
	s.Require().NoError(err)

	// Overlapping: range [1 Jan, 15 Jan] intersects [10 Jan, 20 Jan]
	overlaps, err := s.repo.FindOverlapping(s.ctx,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		nil,
	)
	s.Require().NoError(err)
	s.Len(overlaps, 1)
	s.Equal(p1.ID, overlaps[0].ID)

	// Non-overlapping: range [21 Jan, 31 Jan] does not intersect [10 Jan, 20 Jan]
	overlaps, err = s.repo.FindOverlapping(s.ctx,
		time.Date(2026, 1, 21, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC),
		nil,
	)
	s.Require().NoError(err)
	s.Empty(overlaps)

	// Exclude ID: should ignore p1
	overlaps, err = s.repo.FindOverlapping(s.ctx,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		&p1.ID,
	)
	s.Require().NoError(err)
	s.Empty(overlaps)
}

func TestPeriodRepositorySuite(t *testing.T) {
	suite.Run(t, new(PeriodRepositorySuite))
}

