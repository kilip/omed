package test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/service"
	"github.com/kilip/omed/finance/internal/service/test/mocks"
)

//go:generate go run go.uber.org/mock/mockgen@latest -source=../period_service.go -destination=mocks/mock_period_service.go -package=mocks

type PeriodServiceSuite struct {
	suite.Suite
	ctrl        *gomock.Controller
	mockPeriods *mocks.MockPeriodRepository
	service     service.PeriodService
	ctx         context.Context
}

func (s *PeriodServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockPeriods = mocks.NewMockPeriodRepository(s.ctrl)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s.service = service.NewPeriodService(s.mockPeriods, logger)
	s.ctx = context.Background()
}

func (s *PeriodServiceSuite) TearDownTest() {
	s.ctrl.Finish()
}

// ---------------------------------------------------------------------------
// List Tests
// ---------------------------------------------------------------------------

func (s *PeriodServiceSuite) TestList_Success() {
	req := model.ListPeriodRequest{
		Status: model.PeriodStatusOpen,
		Date:   "2026-01-15",
	}
	expected := []model.Period{
		{
			ID:        uuid.New(),
			StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
			Status:    model.PeriodStatusOpen,
		},
	}

	s.mockPeriods.EXPECT().List(s.ctx, req).Return(expected, nil)

	res, err := s.service.List(s.ctx, req)
	s.NoError(err)
	s.Equal(expected, res)
}

func (s *PeriodServiceSuite) TestList_Error() {
	req := model.ListPeriodRequest{}
	expectedErr := errors.New("db error")

	s.mockPeriods.EXPECT().List(s.ctx, req).Return(nil, expectedErr)

	res, err := s.service.List(s.ctx, req)
	s.Error(err)
	s.Equal(expectedErr, err)
	s.Nil(res)
}

// ---------------------------------------------------------------------------
// GetByID Tests
// ---------------------------------------------------------------------------

func (s *PeriodServiceSuite) TestGetByID_Success() {
	id := uuid.New()
	expected := &model.Period{
		ID:        id,
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}

	s.mockPeriods.EXPECT().GetByID(s.ctx, id).Return(expected, nil)

	res, err := s.service.GetByID(s.ctx, id)
	s.NoError(err)
	s.Equal(expected, res)
}

func (s *PeriodServiceSuite) TestGetByID_NotFound() {
	id := uuid.New()
	s.mockPeriods.EXPECT().GetByID(s.ctx, id).Return(nil, core.ErrItemNotFound)

	res, err := s.service.GetByID(s.ctx, id)
	s.Error(err)
	s.ErrorIs(err, core.ErrItemNotFound)
	s.Nil(res)
}

// ---------------------------------------------------------------------------
// Create Tests
// ---------------------------------------------------------------------------

func (s *PeriodServiceSuite) TestCreate_InvalidDateRange() {
	req := model.CreatePeriodRequest{
		StartDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), // StartDate > EndDate
		Status:    model.PeriodStatusOpen,
	}

	res, err := s.service.Create(s.ctx, req)
	s.Error(err)
	s.ErrorIs(err, core.ErrInvalidDateRange)
	s.Nil(res)
}

func (s *PeriodServiceSuite) TestCreate_FindOverlappingError() {
	req := model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	expectedErr := errors.New("query error")

	s.mockPeriods.EXPECT().FindOverlapping(s.ctx, req.StartDate, req.EndDate, nil).Return(nil, expectedErr)

	res, err := s.service.Create(s.ctx, req)
	s.Error(err)
	s.Equal(expectedErr, err)
	s.Nil(res)
}

func (s *PeriodServiceSuite) TestCreate_PeriodOverlap() {
	req := model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	overlapping := []*ent.Period{
		{ID: uuid.New()},
	}

	s.mockPeriods.EXPECT().FindOverlapping(s.ctx, req.StartDate, req.EndDate, nil).Return(overlapping, nil)

	res, err := s.service.Create(s.ctx, req)
	s.Error(err)
	s.ErrorIs(err, core.ErrPeriodOverlap)
	s.Nil(res)
}

func (s *PeriodServiceSuite) TestCreate_Success() {
	req := model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	expected := &model.Period{
		ID:        uuid.New(),
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Status:    model.PeriodStatusOpen,
	}

	s.mockPeriods.EXPECT().FindOverlapping(s.ctx, req.StartDate, req.EndDate, nil).Return([]*ent.Period{}, nil)
	s.mockPeriods.EXPECT().Create(s.ctx, req).Return(expected, nil)

	res, err := s.service.Create(s.ctx, req)
	s.NoError(err)
	s.Equal(expected, res)
}

func (s *PeriodServiceSuite) TestCreate_RepoError() {
	req := model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
		Status:    model.PeriodStatusOpen,
	}
	expectedErr := errors.New("insert failed")

	s.mockPeriods.EXPECT().FindOverlapping(s.ctx, req.StartDate, req.EndDate, nil).Return([]*ent.Period{}, nil)
	s.mockPeriods.EXPECT().Create(s.ctx, req).Return(nil, expectedErr)

	res, err := s.service.Create(s.ctx, req)
	s.Error(err)
	s.Equal(expectedErr, err)
	s.Nil(res)
}

// ---------------------------------------------------------------------------
// Update Tests
// ---------------------------------------------------------------------------

func (s *PeriodServiceSuite) TestUpdate_GetByIDError() {
	id := uuid.New()
	req := model.UpdatePeriodRequest{Status: model.PeriodStatusClosed}
	expectedErr := errors.New("not found")

	s.mockPeriods.EXPECT().GetByID(s.ctx, id).Return(nil, expectedErr)

	res, err := s.service.Update(s.ctx, id, req)
	s.Error(err)
	s.Equal(expectedErr, err)
	s.Nil(res)
}

func (s *PeriodServiceSuite) TestUpdate_FromLocked_Fails() {
	id := uuid.New()
	existing := &model.Period{
		ID:     id,
		Status: model.PeriodStatusLocked,
	}
	s.mockPeriods.EXPECT().GetByID(s.ctx, id).Return(existing, nil)

	// Attempting to change locked period to open
	res, err := s.service.Update(s.ctx, id, model.UpdatePeriodRequest{Status: model.PeriodStatusOpen})
	s.Error(err)
	s.ErrorIs(err, core.ErrInvalidStatusTransition)
	s.Nil(res)
}

func (s *PeriodServiceSuite) TestUpdate_OpenToLocked_Fails() {
	id := uuid.New()
	existing := &model.Period{
		ID:     id,
		Status: model.PeriodStatusOpen,
	}
	s.mockPeriods.EXPECT().GetByID(s.ctx, id).Return(existing, nil)

	// Open -> Locked directly is not allowed (must be closed first)
	res, err := s.service.Update(s.ctx, id, model.UpdatePeriodRequest{Status: model.PeriodStatusLocked})
	s.Error(err)
	s.ErrorIs(err, core.ErrInvalidStatusTransition)
	s.Nil(res)
}

func (s *PeriodServiceSuite) TestUpdate_OpenToClosed_Success() {
	id := uuid.New()
	existing := &model.Period{
		ID:     id,
		Status: model.PeriodStatusOpen,
	}
	req := model.UpdatePeriodRequest{Status: model.PeriodStatusClosed}
	expected := &model.Period{
		ID:     id,
		Status: model.PeriodStatusClosed,
	}

	s.mockPeriods.EXPECT().GetByID(s.ctx, id).Return(existing, nil)
	s.mockPeriods.EXPECT().Update(s.ctx, id, req).Return(expected, nil)

	res, err := s.service.Update(s.ctx, id, req)
	s.NoError(err)
	s.Equal(expected, res)
}

func (s *PeriodServiceSuite) TestUpdate_ClosedToOpen_Success() {
	id := uuid.New()
	existing := &model.Period{
		ID:     id,
		Status: model.PeriodStatusClosed,
	}
	req := model.UpdatePeriodRequest{Status: model.PeriodStatusOpen}
	expected := &model.Period{
		ID:     id,
		Status: model.PeriodStatusOpen,
	}

	s.mockPeriods.EXPECT().GetByID(s.ctx, id).Return(existing, nil)
	s.mockPeriods.EXPECT().Update(s.ctx, id, req).Return(expected, nil)

	res, err := s.service.Update(s.ctx, id, req)
	s.NoError(err)
	s.Equal(expected, res)
}

func (s *PeriodServiceSuite) TestUpdate_ClosedToLocked_Success() {
	id := uuid.New()
	existing := &model.Period{
		ID:     id,
		Status: model.PeriodStatusClosed,
	}
	req := model.UpdatePeriodRequest{Status: model.PeriodStatusLocked}
	expected := &model.Period{
		ID:     id,
		Status: model.PeriodStatusLocked,
	}

	s.mockPeriods.EXPECT().GetByID(s.ctx, id).Return(existing, nil)
	s.mockPeriods.EXPECT().Update(s.ctx, id, req).Return(expected, nil)

	res, err := s.service.Update(s.ctx, id, req)
	s.NoError(err)
	s.Equal(expected, res)
}

// ---------------------------------------------------------------------------
// Delete Tests
// ---------------------------------------------------------------------------

func (s *PeriodServiceSuite) TestDelete_GetByIDError() {
	id := uuid.New()
	expectedErr := errors.New("not found")

	s.mockPeriods.EXPECT().GetByID(s.ctx, id).Return(nil, expectedErr)

	err := s.service.Delete(s.ctx, id)
	s.Error(err)
	s.Equal(expectedErr, err)
}

func (s *PeriodServiceSuite) TestDelete_Closed_Fails() {
	id := uuid.New()
	existing := &model.Period{
		ID:     id,
		Status: model.PeriodStatusClosed,
	}

	s.mockPeriods.EXPECT().GetByID(s.ctx, id).Return(existing, nil)

	err := s.service.Delete(s.ctx, id)
	s.Error(err)
	s.ErrorIs(err, core.ErrPeriodClosed)
}

func (s *PeriodServiceSuite) TestDelete_Locked_Fails() {
	id := uuid.New()
	existing := &model.Period{
		ID:     id,
		Status: model.PeriodStatusLocked,
	}

	s.mockPeriods.EXPECT().GetByID(s.ctx, id).Return(existing, nil)

	err := s.service.Delete(s.ctx, id)
	s.Error(err)
	s.ErrorIs(err, core.ErrPeriodLocked)
}

func (s *PeriodServiceSuite) TestDelete_CountEntriesError() {
	id := uuid.New()
	existing := &model.Period{
		ID:     id,
		Status: model.PeriodStatusOpen,
	}
	expectedErr := errors.New("count error")

	s.mockPeriods.EXPECT().GetByID(s.ctx, id).Return(existing, nil)
	s.mockPeriods.EXPECT().CountEntries(s.ctx, id).Return(0, expectedErr)

	err := s.service.Delete(s.ctx, id)
	s.Error(err)
	s.Equal(expectedErr, err)
}

func (s *PeriodServiceSuite) TestDelete_HasEntries_Fails() {
	id := uuid.New()
	existing := &model.Period{
		ID:     id,
		Status: model.PeriodStatusOpen,
	}

	s.mockPeriods.EXPECT().GetByID(s.ctx, id).Return(existing, nil)
	s.mockPeriods.EXPECT().CountEntries(s.ctx, id).Return(3, nil)

	err := s.service.Delete(s.ctx, id)
	s.Error(err)
	s.ErrorIs(err, core.ErrEntriesNotEmpty)
}

func (s *PeriodServiceSuite) TestDelete_Success() {
	id := uuid.New()
	existing := &model.Period{
		ID:     id,
		Status: model.PeriodStatusOpen,
	}

	s.mockPeriods.EXPECT().GetByID(s.ctx, id).Return(existing, nil)
	s.mockPeriods.EXPECT().CountEntries(s.ctx, id).Return(0, nil)
	s.mockPeriods.EXPECT().Delete(s.ctx, id).Return(nil)

	err := s.service.Delete(s.ctx, id)
	s.NoError(err)
}

func TestPeriodServiceSuite(t *testing.T) {
	suite.Run(t, new(PeriodServiceSuite))
}

