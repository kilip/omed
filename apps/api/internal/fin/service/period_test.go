package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/fin/service"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type PeriodServiceSuite struct {
	suite.Suite
	client      *ent.Client
	svc         service.PeriodService
	workspaceID int64
	userID      int64
}

func (s *PeriodServiceSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:period_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	repo := repository.NewPeriodRepository(s.client)
	s.svc = service.NewPeriodService(repo)
}

func (s *PeriodServiceSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *PeriodServiceSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.Posting.Delete().Exec(ctx)
	_, _ = s.client.Entry.Delete().Exec(ctx)
	_, _ = s.client.Transaction.Delete().Exec(ctx)
	_, _ = s.client.Period.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)

	u, err := s.client.User.Create().
		SetName("Bob").
		SetEmail("bob@example.com").
		Save(ctx)
	s.Require().NoError(err)
	s.userID = u.ID

	userCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   u.ID,
		Name: "Bob",
	})
	w, err := s.client.Workspace.Create().
		SetName("Finance WS").
		SetOwnerID(u.ID).
		Save(userCtx)
	s.Require().NoError(err)
	s.workspaceID = w.ID
}

func (s *PeriodServiceSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Bob",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func parseDate(str string) model.Date {
	t, _ := time.Parse("2006-01-02", str)
	return model.Date(t)
}

func (s *PeriodServiceSuite) Test_Create_Validations() {
	ctx := s.ctx()

	// Invalid fiscal year
	_, err := s.svc.Create(ctx, model.CreatePeriodRequest{
		Name:       "Period 0",
		StartDate:  parseDate("2026-01-01"),
		EndDate:    parseDate("2026-01-31"),
		FiscalYear: 0,
	})
	s.ErrorIs(err, domain.ErrPeriodInvalidFiscalYear)

	// Invalid dates (start > end)
	_, err = s.svc.Create(ctx, model.CreatePeriodRequest{
		Name:       "Period Inverted",
		StartDate:  parseDate("2026-02-01"),
		EndDate:    parseDate("2026-01-01"),
		FiscalYear: 2026,
	})
	s.ErrorIs(err, domain.ErrPeriodInvalidDates)

	// Valid creation
	p1, err := s.svc.Create(ctx, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseDate("2026-01-01"),
		EndDate:    parseDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)
	s.Equal(domain.PeriodStatusOpen, p1.Status)

	// Overlapping creation
	_, err = s.svc.Create(ctx, model.CreatePeriodRequest{
		Name:       "Overlapping Period",
		StartDate:  parseDate("2026-01-15"),
		EndDate:    parseDate("2026-02-15"),
		FiscalYear: 2026,
	})
	s.ErrorIs(err, domain.ErrPeriodOverlap)
}

func (s *PeriodServiceSuite) Test_Update_ClosedPeriodFails() {
	ctx := s.ctx()

	p, err := s.svc.Create(ctx, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseDate("2026-01-01"),
		EndDate:    parseDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)

	// Close the period
	_, err = s.svc.Close(ctx, p.ID)
	s.Require().NoError(err)

	// Updating closed period must fail
	_, err = s.svc.Update(ctx, p.ID, model.UpdatePeriodRequest{
		Name:       "Try Update Closed",
		StartDate:  parseDate("2026-01-01"),
		EndDate:    parseDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.ErrorIs(err, domain.ErrPeriodClosed)

	// Deleting closed period must fail
	err = s.svc.Delete(ctx, p.ID)
	s.ErrorIs(err, domain.ErrPeriodClosed)

	// Reopen the period
	reopened, err := s.svc.Reopen(ctx, p.ID)
	s.Require().NoError(err)
	s.Equal(domain.PeriodStatusOpen, reopened.Status)

	// Now update succeeds
	updated, err := s.svc.Update(ctx, p.ID, model.UpdatePeriodRequest{
		Name:       "Reopened Jan 2026",
		StartDate:  parseDate("2026-01-01"),
		EndDate:    parseDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)
	s.Equal("Reopened Jan 2026", updated.Name)
}

func (s *PeriodServiceSuite) Test_CloseAndReopen_IdempotencyChecks() {
	ctx := s.ctx()

	p, err := s.svc.Create(ctx, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseDate("2026-01-01"),
		EndDate:    parseDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)

	// Reopening already open period fails
	_, err = s.svc.Reopen(ctx, p.ID)
	s.ErrorIs(err, domain.ErrPeriodAlreadyOpen)

	// Close succeeds
	_, err = s.svc.Close(ctx, p.ID)
	s.Require().NoError(err)

	// Closing already closed period fails
	_, err = s.svc.Close(ctx, p.ID)
	s.ErrorIs(err, domain.ErrPeriodAlreadyClosed)
}

func (s *PeriodServiceSuite) Test_Close_PeriodWithDraftEntriesFails() {
	ctx := s.ctx()

	p, err := s.svc.Create(ctx, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseDate("2026-01-01"),
		EndDate:    parseDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)

	trx, err := s.client.Transaction.Create().
		SetNumber("TRX-2026-000001").
		SetTransactionDate(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)).
		SetCurrency("IDR").
		SetExchangeRate(1.0).
		SetSource("MANUAL").
		Save(ctx)
	s.Require().NoError(err)

	e, err := s.client.Entry.Create().
		SetTransactionID(trx.ID).
		SetPeriodID(p.ID).
		SetNumber("JE-2026-000001").
		SetEntryDate(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)).
		SetStatus("DRAFT").
		Save(ctx)
	s.Require().NoError(err)

	// Trying to close period with draft entries must fail
	_, err = s.svc.Close(ctx, p.ID)
	s.ErrorIs(err, domain.ErrPeriodHasDraftEntries)

	// Update entry status to POSTED
	_, err = s.client.Entry.UpdateOneID(e.ID).SetStatus("POSTED").Save(ctx)
	s.Require().NoError(err)

	// Now close succeeds
	closed, err := s.svc.Close(ctx, p.ID)
	s.Require().NoError(err)
	s.Equal(domain.PeriodStatusClosed, closed.Status)
}

func TestPeriodServiceSuite(t *testing.T) {
	suite.Run(t, new(PeriodServiceSuite))
}
