package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type PeriodRepoSuite struct {
	suite.Suite
	client      *ent.Client
	repo        repository.PeriodRepository
	workspaceID int64
	userID      int64
}

func (s *PeriodRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:period_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.repo = repository.NewPeriodRepository(s.client)
}

func (s *PeriodRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *PeriodRepoSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(base.WithSkipWorkspaceFilter(context.Background()))
	_, _ = s.client.Posting.Delete().Exec(ctx)
	_, _ = s.client.Entry.Delete().Exec(ctx)
	_, _ = s.client.Transaction.Delete().Exec(ctx)
	_, _ = s.client.Period.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)

	u, err := s.client.User.Create().
		SetName("Alice").
		SetEmail("alice@example.com").
		Save(ctx)
	s.Require().NoError(err)
	s.userID = u.ID

	userCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   u.ID,
		Name: "Alice",
	})
	w, err := s.client.Workspace.Create().
		SetName("Finance WS").
		SetOwnerID(u.ID).
		Save(userCtx)
	s.Require().NoError(err)
	s.workspaceID = w.ID
}

func (s *PeriodRepoSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Alice",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func parseDate(str string) model.Date {
	t, _ := time.Parse("2006-01-02", str)
	return model.Date(t)
}

func (s *PeriodRepoSuite) Test_CreateAndGetByID() {
	ctx := s.ctx()

	p, err := s.repo.Create(ctx, model.CreatePeriodRequest{
		Name:        "January 2026",
		StartDate:   parseDate("2026-01-01"),
		EndDate:     parseDate("2026-01-31"),
		FiscalYear:  2026,
		Description: "First month",
	})
	s.Require().NoError(err)
	s.Equal("January 2026", p.Name)
	s.Equal(domain.PeriodStatusOpen, p.Status)
	s.Equal(2026, p.FiscalYear)
	s.Equal("First month", p.Description)

	fetched, err := s.repo.GetByID(ctx, p.ID)
	s.Require().NoError(err)
	s.Equal(p.ID, fetched.ID)
	s.Equal("January 2026", fetched.Name)
}

func (s *PeriodRepoSuite) Test_SearchAndFilter() {
	ctx := s.ctx()

	_, err := s.repo.Create(ctx, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseDate("2026-01-01"),
		EndDate:    parseDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)

	p2, err := s.repo.Create(ctx, model.CreatePeriodRequest{
		Name:       "Feb 2026",
		StartDate:  parseDate("2026-02-01"),
		EndDate:    parseDate("2026-02-28"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)

	// Close p2
	_, err = s.repo.UpdateStatus(ctx, p2.ID, domain.PeriodStatusClosed)
	s.Require().NoError(err)

	// Search all
	all, _, err := s.repo.Search(ctx, model.SearchPeriodRequest{})
	s.Require().NoError(err)
	s.Len(all, 2)

	// Search by status CLOSED
	statusClosed := domain.PeriodStatusClosed
	closedItems, _, err := s.repo.Search(ctx, model.SearchPeriodRequest{
		Status: &statusClosed,
	})
	s.Require().NoError(err)
	s.Len(closedItems, 1)
	s.Equal("Feb 2026", closedItems[0].Name)

	// Search by fiscal year 2027
	year2027 := 2027
	y2027Items, _, err := s.repo.Search(ctx, model.SearchPeriodRequest{
		FiscalYear: &year2027,
	})
	s.Require().NoError(err)
	s.Empty(y2027Items)
}

func (s *PeriodRepoSuite) Test_HasOverlapping() {
	ctx := s.ctx()

	p1, err := s.repo.Create(ctx, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseDate("2026-01-01"),
		EndDate:    parseDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)

	// Exactly overlapping
	has, err := s.repo.HasOverlapping(ctx, parseDate("2026-01-10").Time(), parseDate("2026-01-20").Time(), 0)
	s.Require().NoError(err)
	s.True(has)

	// Non-overlapping (adjacent next month)
	has, err = s.repo.HasOverlapping(ctx, parseDate("2026-02-01").Time(), parseDate("2026-02-28").Time(), 0)
	s.Require().NoError(err)
	s.False(has)

	// Excluding p1 should not report overlap
	has, err = s.repo.HasOverlapping(ctx, parseDate("2026-01-01").Time(), parseDate("2026-01-31").Time(), p1.ID)
	s.Require().NoError(err)
	s.False(has)
}

func (s *PeriodRepoSuite) Test_Update() {
	ctx := s.ctx()

	p, err := s.repo.Create(ctx, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseDate("2026-01-01"),
		EndDate:    parseDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)

	updated, err := s.repo.Update(ctx, p.ID, model.UpdatePeriodRequest{
		Name:        "January 2026 Renamed",
		StartDate:   parseDate("2026-01-01"),
		EndDate:     parseDate("2026-01-31"),
		FiscalYear:  2026,
		Description: "Updated desc",
	})
	s.Require().NoError(err)
	s.Equal("January 2026 Renamed", updated.Name)
	s.Equal("Updated desc", updated.Description)
}

func (s *PeriodRepoSuite) Test_Delete() {
	ctx := s.ctx()

	p, err := s.repo.Create(ctx, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseDate("2026-01-01"),
		EndDate:    parseDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)

	err = s.repo.Delete(ctx, p.ID)
	s.Require().NoError(err)

	_, err = s.repo.GetByID(ctx, p.ID)
	s.ErrorIs(err, base.ErrNotFound)
}

func (s *PeriodRepoSuite) Test_HasDraftEntries() {
	ctx := s.ctx()

	p, err := s.repo.Create(ctx, model.CreatePeriodRequest{
		Name:       "Feb 2026",
		StartDate:  parseDate("2026-02-01"),
		EndDate:    parseDate("2026-02-28"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)

	hasDraft, err := s.repo.HasDraftEntries(ctx, p.ID)
	s.Require().NoError(err)
	s.False(hasDraft)

	trx, err := s.client.Transaction.Create().
		SetNumber("TRX-2026-000002").
		SetTransactionDate(time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)).
		SetCurrency("IDR").
		SetExchangeRate(decimal.NewFromInt(1)).
		SetSource("MANUAL").
		Save(ctx)
	s.Require().NoError(err)

	e, err := s.client.Entry.Create().
		SetTransactionID(trx.ID).
		SetPeriodID(p.ID).
		SetNumber("JE-2026-000002").
		SetEntryDate(time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)).
		SetStatus("DRAFT").
		Save(ctx)
	s.Require().NoError(err)

	hasDraft, err = s.repo.HasDraftEntries(ctx, p.ID)
	s.Require().NoError(err)
	s.True(hasDraft)

	_, err = s.client.Entry.UpdateOneID(e.ID).SetStatus("POSTED").Save(ctx)
	s.Require().NoError(err)

	hasDraft, err = s.repo.HasDraftEntries(ctx, p.ID)
	s.Require().NoError(err)
	s.False(hasDraft)
}

func (s *PeriodRepoSuite) Test_HasRevaluationEntry() {
	ctx := s.ctx()

	p, err := s.repo.Create(ctx, model.CreatePeriodRequest{
		Name:       "March 2026",
		StartDate:  parseDate("2026-03-01"),
		EndDate:    parseDate("2026-03-31"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)

	hasReval, err := s.repo.HasRevaluationEntry(ctx, p.ID)
	s.Require().NoError(err)
	s.False(hasReval)

	ref := domain.PeriodRevaluationReference(p.ID)
	trx, err := s.client.Transaction.Create().
		SetNumber("TRX-2026-000003").
		SetTransactionDate(time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)).
		SetCurrency("IDR").
		SetExchangeRate(decimal.NewFromInt(1)).
		SetReference(ref).
		SetSource("SYSTEM").
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.client.Entry.Create().
		SetTransactionID(trx.ID).
		SetPeriodID(p.ID).
		SetNumber("JE-2026-000003").
		SetEntryDate(time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)).
		SetStatus("POSTED").
		Save(ctx)
	s.Require().NoError(err)

	hasReval, err = s.repo.HasRevaluationEntry(ctx, p.ID)
	s.Require().NoError(err)
	s.True(hasReval)
}

func (s *PeriodRepoSuite) Test_FindByDate() {
	ctx := s.ctx()

	p, err := s.repo.Create(ctx, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseDate("2026-01-01"),
		EndDate:    parseDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)

	// Within period
	found, err := s.repo.FindByDate(ctx, time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC))
	s.Require().NoError(err)
	s.Equal(p.ID, found.ID)

	// First day of period
	found, err = s.repo.FindByDate(ctx, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	s.Require().NoError(err)
	s.Equal(p.ID, found.ID)

	// Last day of period
	found, err = s.repo.FindByDate(ctx, time.Date(2026, 1, 31, 23, 59, 0, 0, time.UTC))
	s.Require().NoError(err)
	s.Equal(p.ID, found.ID)

	// Outside period
	_, err = s.repo.FindByDate(ctx, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))
	s.Require().ErrorIs(err, base.ErrNotFound)
}

func TestPeriodRepoSuite(t *testing.T) {
	suite.Run(t, new(PeriodRepoSuite))
}

