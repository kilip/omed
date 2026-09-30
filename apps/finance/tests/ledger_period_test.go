package tests

import (
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/kilip/omed/finance/internal/http/controller"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/repository"
	"github.com/kilip/omed/finance/internal/service"
	"github.com/kilip/omed/finance/internal/shared"
	"github.com/kilip/omed/finance/testutil"
)

var periodSeq atomic.Int64

func newLedgerPeriodReq() model.CreateLedgerPeriodRequest {
	seq := periodSeq.Add(1)
	now := time.Now().Truncate(time.Hour * 24).Add(time.Duration(seq*35*24) * time.Hour)
	return model.CreateLedgerPeriodRequest{
		StartDate: now,
		EndDate:   now.AddDate(0, 1, 0),
	}
}

type LedgerPeriodTestSuite struct {
	testutil.ApiTestSuite[model.LedgerPeriod]
}

func (s *LedgerPeriodTestSuite) SetupSuite() {
	state := testutil.GetState()

	repo := repository.NewLedgerPeriodRepository(state.EntClient, state.Log)
	svc := service.NewLedgerPeriodService(repo, state.Log)

	controller.NewLedgerPeriodController(svc).Register(state.Api)
}

// ---------------------------------------------------------------- helpers

func (s *LedgerPeriodTestSuite) withRoles(roles ...shared.WorkspaceRole) {
	s.User.WorkspaceRoles = roles
}

func (s *LedgerPeriodTestSuite) create(req model.CreateLedgerPeriodRequest) model.LedgerPeriod {
	s.T().Helper()
	s.Request("/ledger-periods", fiber.MethodPost, req)
	s.Created()
	return s.GetResponse().Data
}

func ledgerPeriodPath(id uuid.UUID) string {
	return "/ledger-periods/" + id.String()
}

// ---------------------------------------------------------------- create

func (s *LedgerPeriodTestSuite) TestCreate() {
	req := newLedgerPeriodReq()

	got := s.create(req)

	s.NotEqual(uuid.Nil, got.ID)
	s.Equal(s.User.WorkspaceID, got.WorkspaceID)
	s.Equal(model.LedgerPeriodStatusOpen, got.Status)
	s.Equal(s.User.ID, got.CreatedBy)
	s.Equal(s.User.Name, got.CreatedByName)
	s.Equal(s.User.ID, got.UpdatedBy)
}

func (s *LedgerPeriodTestSuite) TestCreateValidation() {
	cases := map[string]func(r *model.CreateLedgerPeriodRequest){
		"start date zero": func(r *model.CreateLedgerPeriodRequest) { r.StartDate = time.Time{} },
		"end date zero":   func(r *model.CreateLedgerPeriodRequest) { r.EndDate = time.Time{} },
		"end date <= start date": func(r *model.CreateLedgerPeriodRequest) {
			r.EndDate = r.StartDate
		},
	}

	for name, mutate := range cases {
		s.Run(name, func() {
			req := newLedgerPeriodReq()
			mutate(&req)
			s.Request("/ledger-periods", fiber.MethodPost, req)
			s.AssertStatus(fiber.StatusUnprocessableEntity)
		})
	}
}

// ---------------------------------------------------------------- list

func (s *LedgerPeriodTestSuite) TestListEmpty() {
	s.Request("/ledger-periods", fiber.MethodGet, nil)
	s.OK()
	s.Empty(s.PagedResponse().Data)
}

func (s *LedgerPeriodTestSuite) TestListAndFilter() {
	period1 := s.create(newLedgerPeriodReq())
	period2 := s.create(newLedgerPeriodReq())

	// Close period2 to test status filter
	closedStatus := model.LedgerPeriodStatusClosed
	s.Request(ledgerPeriodPath(period2.ID), fiber.MethodPut, model.UpdateLedgerPeriodRequest{
		Status: closedStatus,
	})
	s.OK()

	s.Run("tanpa filter", func() {
		s.Request("/ledger-periods", fiber.MethodGet, nil)
		s.OK()
		s.Len(s.PagedResponse().Data, 2)
	})

	s.Run("filter status open", func() {
		s.Request("/ledger-periods?status=open", fiber.MethodGet, nil)
		s.OK()
		data := s.PagedResponse().Data
		s.Require().Len(data, 1)
		s.Equal(period1.ID, data[0].ID)
	})

	s.Run("filter status closed", func() {
		s.Request("/ledger-periods?status=closed", fiber.MethodGet, nil)
		s.OK()
		data := s.PagedResponse().Data
		s.Require().Len(data, 1)
		s.Equal(period2.ID, data[0].ID)
	})

	s.Run("filter date", func() {
		dateStr := url.QueryEscape(period1.StartDate.Add(time.Hour * 12).Format(time.RFC3339))
		s.Request("/ledger-periods?date="+dateStr, fiber.MethodGet, nil)
		s.OK()
		data := s.PagedResponse().Data
		s.Require().Len(data, 1)
		s.Equal(period1.ID, data[0].ID)
	})

	s.Run("filter status tidak valid", func() {
		s.Request("/ledger-periods?status=bogus", fiber.MethodGet, nil)
		s.AssertStatus(fiber.StatusUnprocessableEntity)
	})
}

// ---------------------------------------------------------------- get

func (s *LedgerPeriodTestSuite) TestGet() {
	created := s.create(newLedgerPeriodReq())

	s.Request(ledgerPeriodPath(created.ID), fiber.MethodGet, nil)
	s.OK()

	got := s.GetResponse().Data
	s.Equal(created.ID, got.ID)
	s.Equal(model.LedgerPeriodStatusOpen, got.Status)
}

func (s *LedgerPeriodTestSuite) TestGetNotFound() {
	s.Request(ledgerPeriodPath(shared.GenerateID()), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *LedgerPeriodTestSuite) TestGetInvalidID() {
	s.Request("/ledger-periods/not-a-uuid", fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusBadRequest)
}

// ---------------------------------------------------------------- update

func (s *LedgerPeriodTestSuite) TestUpdate() {
	created := s.create(newLedgerPeriodReq())

	status := model.LedgerPeriodStatusClosed
	s.Request(ledgerPeriodPath(created.ID), fiber.MethodPut, model.UpdateLedgerPeriodRequest{
		Status: status,
	})
	s.OK()

	got := s.GetResponse().Data
	s.Equal(model.LedgerPeriodStatusClosed, got.Status)
	s.Equal(s.User.ID, got.UpdatedBy)

	// Transition to locked
	statusLocked := model.LedgerPeriodStatusLocked
	s.Request(ledgerPeriodPath(created.ID), fiber.MethodPut, model.UpdateLedgerPeriodRequest{
		Status: statusLocked,
	})
	s.OK()

	gotLocked := s.GetResponse().Data
	s.Equal(model.LedgerPeriodStatusLocked, gotLocked.Status)
}

func (s *LedgerPeriodTestSuite) TestUpdateValidation() {
	created := s.create(newLedgerPeriodReq())

	badStatus := model.LedgerPeriodStatus("bogus")
	s.Request(ledgerPeriodPath(created.ID), fiber.MethodPut, model.UpdateLedgerPeriodRequest{
		Status: badStatus,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *LedgerPeriodTestSuite) TestUpdateNotFound() {
	s.Request(ledgerPeriodPath(shared.GenerateID()), fiber.MethodPut, model.UpdateLedgerPeriodRequest{
		Status: model.LedgerPeriodStatusClosed,
	})
	s.AssertStatus(fiber.StatusNotFound)
}

// ---------------------------------------------------------------- delete

func (s *LedgerPeriodTestSuite) TestDelete() {
	created := s.create(newLedgerPeriodReq())

	s.Request(ledgerPeriodPath(created.ID), fiber.MethodDelete, nil)
	s.NoContent()

	s.Request(ledgerPeriodPath(created.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *LedgerPeriodTestSuite) TestDeleteNotFound() {
	s.Request(ledgerPeriodPath(shared.GenerateID()), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

// ---------------------------------------------------------------- RBAC

func (s *LedgerPeriodTestSuite) TestRBAC() {
	type op struct {
		name   string
		method string
		path   func(id uuid.UUID) string
		body   func() any
		write  bool
	}
	ops := []op{
		{"list", fiber.MethodGet, func(uuid.UUID) string { return "/ledger-periods" }, func() any { return nil }, false},
		{"get", fiber.MethodGet, ledgerPeriodPath, func() any { return nil }, false},
		{"create", fiber.MethodPost, func(uuid.UUID) string { return "/ledger-periods" },
			func() any { return newLedgerPeriodReq() }, true},
		{"update", fiber.MethodPut, ledgerPeriodPath,
			func() any { return model.UpdateLedgerPeriodRequest{Status: model.LedgerPeriodStatusClosed} }, true},
		{"delete", fiber.MethodDelete, ledgerPeriodPath, func() any { return nil }, true},
	}

	successStatus := map[string]int{
		"list":   fiber.StatusOK,
		"get":    fiber.StatusOK,
		"create": fiber.StatusCreated,
		"update": fiber.StatusOK,
		"delete": fiber.StatusNoContent,
	}

	roles := []struct {
		name  string
		roles []shared.WorkspaceRole
		read  bool
		write bool
	}{
		{"owner", []shared.WorkspaceRole{shared.WorkspaceRoleOwner}, true, true},
		{"admin", []shared.WorkspaceRole{shared.WorkspaceRoleAdmin}, true, true},
		{"member", []shared.WorkspaceRole{shared.WorkspaceRoleMember}, true, false},
		{"user", []shared.WorkspaceRole{shared.WorkspaceRole(shared.UserRoleUser)}, false, false},
		{"tanpa role", nil, false, false},
	}

	for _, r := range roles {
		for _, o := range ops {
			s.Run(r.name+"/"+o.name, func() {
				s.withRoles(shared.WorkspaceRoleOwner)
				seed := s.create(newLedgerPeriodReq())

				s.withRoles(r.roles...)
				s.Request(o.path(seed.ID), o.method, o.body())

				allowed := (o.write && r.write) || (!o.write && r.read)
				if allowed {
					s.AssertStatus(successStatus[o.name])
				} else {
					s.AssertStatus(fiber.StatusForbidden)
				}
			})
		}
	}
}

// ---------------------------------------------------------------- workspace isolation

func (s *LedgerPeriodTestSuite) TestWorkspaceIsolation() {
	wsA := s.User.WorkspaceID
	created := s.create(newLedgerPeriodReq())

	s.User.WorkspaceID = shared.GenerateID()
	s.User.WorkspaceName = "Other Workspace"

	s.Run("get tidak terlihat", func() {
		s.Request(ledgerPeriodPath(created.ID), fiber.MethodGet, nil)
		s.AssertStatus(fiber.StatusNotFound)
	})

	s.Run("list kosong", func() {
		s.Request("/ledger-periods", fiber.MethodGet, nil)
		s.OK()
		s.Empty(s.PagedResponse().Data)
	})

	s.Run("update ditolak", func() {
		s.Request(ledgerPeriodPath(created.ID), fiber.MethodPut, model.UpdateLedgerPeriodRequest{
			Status: model.LedgerPeriodStatusClosed,
		})
		s.AssertStatus(fiber.StatusNotFound)
	})

	s.Run("delete ditolak", func() {
		s.Request(ledgerPeriodPath(created.ID), fiber.MethodDelete, nil)
		s.AssertStatus(fiber.StatusNotFound)
	})

	// Restore workspace A
	s.User.WorkspaceID = wsA
	s.Request(ledgerPeriodPath(created.ID), fiber.MethodGet, nil)
	s.OK()
	got := s.GetResponse().Data
	s.Equal(created.ID, got.ID)
	s.Equal(wsA, got.WorkspaceID)
}

func TestLedgerPeriodSuite(t *testing.T) {
	suite.Run(t, new(LedgerPeriodTestSuite))
}
