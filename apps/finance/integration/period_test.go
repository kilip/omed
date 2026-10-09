package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared/util"
	"github.com/kilip/omed/finance/testutil"
)

type PeriodSuite struct {
	testutil.ApiTestSuite[model.Period]
}

func (s *PeriodSuite) SetupTest() {
	s.ApiTestSuite.SetupTest()
}

func (s *PeriodSuite) TearDownTest() {
	if s.User != nil {
		ctx := core.ContextWithUser(context.Background(), *s.User)
		_, _ = testutil.GetState().EntClient.Entry.Delete().Exec(ctx)
		_, _ = testutil.GetState().EntClient.Period.Delete().Exec(ctx)
	}
}

func (s *PeriodSuite) createPeriod(req model.CreatePeriodRequest) model.Period {
	s.T().Helper()
	s.Request("/periods", http.MethodPost, req)
	s.Created()
	resp := s.GetResponse()
	return resp.Data
}

func (s *PeriodSuite) decodeErrorResponse() model.ErrorResponse {
	s.T().Helper()
	var errResp model.ErrorResponse
	defer s.HttpResponse.Body.Close()
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&errResp))
	return errResp
}

// ---------------------------------------------------------------------------
// List Periods Tests (GET /periods)
// ---------------------------------------------------------------------------

func (s *PeriodSuite) TestList_Empty() {
	s.Request("/periods", http.MethodGet, nil)
	s.OK()
	resp := s.PagedResponse()
	s.Empty(resp.Data)
}

func (s *PeriodSuite) TestList_Success() {
	p1 := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})
	p2 := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
	})

	s.Request("/periods", http.MethodGet, nil)
	s.OK()
	resp := s.PagedResponse()
	s.Len(resp.Data, 2)

	ids := []uuid.UUID{resp.Data[0].ID, resp.Data[1].ID}
	s.Contains(ids, p1.ID)
	s.Contains(ids, p2.ID)
}

func (s *PeriodSuite) TestList_FilterByStatus() {
	p1 := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})
	p2 := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
	})

	s.Request(fmt.Sprintf("/periods/%s", p2.ID), http.MethodPut, model.UpdatePeriodRequest{
		Status: model.PeriodStatusClosed,
	})
	s.OK()

	s.Request("/periods?status=open", http.MethodGet, nil)
	s.OK()
	openList := s.PagedResponse()
	s.Len(openList.Data, 1)
	s.Equal(p1.ID, openList.Data[0].ID)

	s.Request("/periods?status=closed", http.MethodGet, nil)
	s.OK()
	closedList := s.PagedResponse()
	s.Len(closedList.Data, 1)
	s.Equal(p2.ID, closedList.Data[0].ID)
}

func (s *PeriodSuite) TestList_FilterByDate() {
	p1 := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})
	s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC),
	})

	s.Request("/periods?date=2026-01-15", http.MethodGet, nil)
	s.OK()
	resp := s.PagedResponse()
	s.Len(resp.Data, 1)
	s.Equal(p1.ID, resp.Data[0].ID)
}

func (s *PeriodSuite) TestList_InvalidFilter() {
	s.Request("/periods?status=invalid_status", http.MethodGet, nil)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
	errResp := s.decodeErrorResponse()
	s.Equal("VALIDATION_FAILED", errResp.Error.Code)
}

func (s *PeriodSuite) TestList_WorkspaceIsolation() {
	p1 := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})

	// Switch to a new workspace user
	s.User = &core.AuthenticatedUser{
		Name:           "User Two",
		ID:             util.GenerateID(),
		WorkspaceID:    util.GenerateID(),
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
		WorkspaceName:  "Workspace Two",
	}

	s.Request("/periods", http.MethodGet, nil)
	s.OK()
	resp := s.PagedResponse()
	s.Empty(resp.Data)

	p2 := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})

	s.Request("/periods", http.MethodGet, nil)
	s.OK()
	resp = s.PagedResponse()
	s.Len(resp.Data, 1)
	s.Equal(p2.ID, resp.Data[0].ID)
	s.NotEqual(p1.ID, resp.Data[0].ID)
}

func (s *PeriodSuite) TestList_Forbidden() {
	s.User.WorkspaceRoles = []core.WorkspaceRole{"guest"}
	s.Request("/periods", http.MethodGet, nil)
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *PeriodSuite) TestList_Unauthorized() {
	s.RequestWithToken("/periods", http.MethodGet, nil, "")
	s.AssertStatus(fiber.StatusUnauthorized)
}

// ---------------------------------------------------------------------------
// Create Period Tests (POST /periods)
// ---------------------------------------------------------------------------

func (s *PeriodSuite) TestCreate_Success() {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)

	req := model.CreatePeriodRequest{
		StartDate: start,
		EndDate:   end,
	}

	s.Request("/periods", http.MethodPost, req)
	s.Created()
	resp := s.GetResponse()

	s.NotEqual(uuid.Nil, resp.Data.ID)
	s.Equal(s.User.WorkspaceID, resp.Data.WorkspaceID)
	s.True(start.Equal(resp.Data.StartDate))
	s.True(end.Equal(resp.Data.EndDate))
	s.Equal(model.PeriodStatusOpen, resp.Data.Status)
	s.Equal(s.User.ID, resp.Data.CreatedBy)
	s.Equal(s.User.ID, resp.Data.UpdatedBy)
}

func (s *PeriodSuite) TestCreate_InvalidDateRange() {
	req := model.CreatePeriodRequest{
		StartDate: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	s.Request("/periods", http.MethodPost, req)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
	errResp := s.decodeErrorResponse()
	s.Equal("INVALID_DATE_RANGE", errResp.Error.Code)
}

func (s *PeriodSuite) TestCreate_PeriodOverlap() {
	s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})

	// Overlapping period
	reqOverlap := model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 2, 15, 23, 59, 59, 0, time.UTC),
	}

	s.Request("/periods", http.MethodPost, reqOverlap)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
	errResp := s.decodeErrorResponse()
	s.Equal("PERIOD_OVERLAP", errResp.Error.Code)
}

func (s *PeriodSuite) TestCreate_ValidationError() {
	reqMissing := map[string]any{}
	s.Request("/periods", http.MethodPost, reqMissing)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
	errResp := s.decodeErrorResponse()
	s.Equal("VALIDATION_FAILED", errResp.Error.Code)
}

func (s *PeriodSuite) TestCreate_Forbidden() {
	s.User.WorkspaceRoles = []core.WorkspaceRole{core.WorkspaceRoleMember}
	s.Request("/periods", http.MethodPost, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *PeriodSuite) TestCreate_Unauthorized() {
	s.RequestWithToken("/periods", http.MethodPost, model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	}, "")
	s.AssertStatus(fiber.StatusUnauthorized)
}

// ---------------------------------------------------------------------------
// Get Period By ID Tests (GET /periods/:id)
// ---------------------------------------------------------------------------

func (s *PeriodSuite) TestGetByID_Success() {
	created := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})

	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodGet, nil)
	s.OK()
	resp := s.GetResponse()

	s.Equal(created.ID, resp.Data.ID)
	s.True(created.StartDate.Equal(resp.Data.StartDate))
	s.True(created.EndDate.Equal(resp.Data.EndDate))
	s.Equal(created.Status, resp.Data.Status)
}

func (s *PeriodSuite) TestGetByID_NotFound() {
	randomID := uuid.New()
	s.Request(fmt.Sprintf("/periods/%s", randomID), http.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
	errResp := s.decodeErrorResponse()
	s.Equal("NOT_FOUND", errResp.Error.Code)
}

func (s *PeriodSuite) TestGetByID_InvalidID() {
	s.Request("/periods/invalid-uuid-format", http.MethodGet, nil)
	s.AssertStatus(fiber.StatusBadRequest)
	errResp := s.decodeErrorResponse()
	s.Equal("INVALID_ID", errResp.Error.Code)
}

func (s *PeriodSuite) TestGetByID_WorkspaceIsolation() {
	created := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})

	s.User = &core.AuthenticatedUser{
		Name:           "User Two",
		ID:             util.GenerateID(),
		WorkspaceID:    util.GenerateID(),
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
		WorkspaceName:  "Workspace Two",
	}

	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *PeriodSuite) TestGetByID_Forbidden() {
	created := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})

	s.User.WorkspaceRoles = []core.WorkspaceRole{"guest"}
	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodGet, nil)
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *PeriodSuite) TestGetByID_Unauthorized() {
	s.RequestWithToken(fmt.Sprintf("/periods/%s", uuid.New()), http.MethodGet, nil, "")
	s.AssertStatus(fiber.StatusUnauthorized)
}

// ---------------------------------------------------------------------------
// Update Period Tests (PUT /periods/:id)
// ---------------------------------------------------------------------------

func (s *PeriodSuite) TestUpdate_Success() {
	created := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})

	// Open -> Closed
	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodPut, model.UpdatePeriodRequest{
		Status: model.PeriodStatusClosed,
	})
	s.OK()
	resp := s.GetResponse()
	s.Equal(model.PeriodStatusClosed, resp.Data.Status)

	// Closed -> Open (reopen)
	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodPut, model.UpdatePeriodRequest{
		Status: model.PeriodStatusOpen,
	})
	s.OK()
	resp = s.GetResponse()
	s.Equal(model.PeriodStatusOpen, resp.Data.Status)

	// Open -> Closed -> Locked
	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodPut, model.UpdatePeriodRequest{
		Status: model.PeriodStatusClosed,
	})
	s.OK()
	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodPut, model.UpdatePeriodRequest{
		Status: model.PeriodStatusLocked,
	})
	s.OK()
	resp = s.GetResponse()
	s.Equal(model.PeriodStatusLocked, resp.Data.Status)
}

func (s *PeriodSuite) TestUpdate_InvalidTransition() {
	created := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})

	// Open -> Locked directly is not allowed
	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodPut, model.UpdatePeriodRequest{
		Status: model.PeriodStatusLocked,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
	errResp := s.decodeErrorResponse()
	s.Equal("INVALID_STATUS_TRANSITION", errResp.Error.Code)

	// Transition to Closed then Locked
	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodPut, model.UpdatePeriodRequest{
		Status: model.PeriodStatusClosed,
	})
	s.OK()
	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodPut, model.UpdatePeriodRequest{
		Status: model.PeriodStatusLocked,
	})
	s.OK()

	// Locked -> Open is not allowed
	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodPut, model.UpdatePeriodRequest{
		Status: model.PeriodStatusOpen,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
	errResp = s.decodeErrorResponse()
	s.Equal("INVALID_STATUS_TRANSITION", errResp.Error.Code)
}

func (s *PeriodSuite) TestUpdate_NotFound() {
	s.Request(fmt.Sprintf("/periods/%s", uuid.New()), http.MethodPut, model.UpdatePeriodRequest{
		Status: model.PeriodStatusClosed,
	})
	s.AssertStatus(fiber.StatusNotFound)
	errResp := s.decodeErrorResponse()
	s.Equal("NOT_FOUND", errResp.Error.Code)
}

func (s *PeriodSuite) TestUpdate_Forbidden() {
	created := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})

	s.User.WorkspaceRoles = []core.WorkspaceRole{core.WorkspaceRoleMember}
	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodPut, model.UpdatePeriodRequest{
		Status: model.PeriodStatusClosed,
	})
	s.AssertStatus(fiber.StatusForbidden)
}

// ---------------------------------------------------------------------------
// Delete Period Tests (DELETE /periods/:id)
// ---------------------------------------------------------------------------

func (s *PeriodSuite) TestDelete_Success() {
	created := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})

	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodDelete, nil)
	s.NoContent()

	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *PeriodSuite) TestDelete_ClosedFails() {
	created := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})

	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodPut, model.UpdatePeriodRequest{
		Status: model.PeriodStatusClosed,
	})
	s.OK()

	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodDelete, nil)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
	errResp := s.decodeErrorResponse()
	s.Equal("PERIOD_CLOSED", errResp.Error.Code)
}

func (s *PeriodSuite) TestDelete_NotFound() {
	s.Request(fmt.Sprintf("/periods/%s", uuid.New()), http.MethodDelete, nil)
	s.AssertStatus(fiber.StatusNotFound)
	errResp := s.decodeErrorResponse()
	s.Equal("NOT_FOUND", errResp.Error.Code)
}

func (s *PeriodSuite) TestDelete_Forbidden() {
	created := s.createPeriod(model.CreatePeriodRequest{
		StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
	})

	s.User.WorkspaceRoles = []core.WorkspaceRole{core.WorkspaceRoleMember}
	s.Request(fmt.Sprintf("/periods/%s", created.ID), http.MethodDelete, nil)
	s.AssertStatus(fiber.StatusForbidden)
}

func TestPeriodSuite(t *testing.T) {
	suite.Run(t, new(PeriodSuite))
}

