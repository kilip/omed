package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/fin/service"
	"github.com/kilip/omed/api/testutil"
	"github.com/shopspring/decimal"
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
	accountRepo := repository.NewAccountRepository(s.client)
	entryRepo := repository.NewEntryRepository(s.client)
	rateRepo := repository.NewExchangeRateRepository(s.client)
	currRepo := repository.NewCurrencyRepository(s.client)
	settingRepo := repository.NewSettingRepository(s.client)

	s.svc = service.NewPeriodService(
		repo,
		service.WithPeriodDependencies(accountRepo, entryRepo, rateRepo, currRepo, settingRepo),
	)
}

func (s *PeriodServiceSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *PeriodServiceSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(base.WithSkipWorkspaceFilter(context.Background()))
	_, _ = s.client.Posting.Delete().Exec(ctx)
	_, _ = s.client.Entry.Delete().Exec(ctx)
	_, _ = s.client.Transaction.Delete().Exec(ctx)
	_, _ = s.client.Period.Delete().Exec(ctx)
	_, _ = s.client.Account.Delete().Exec(ctx)
	_, _ = s.client.Session.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)

	u, err := s.client.User.Create().
		SetName("Bob").
		SetEmail(fmt.Sprintf("bob-%d@example.com", time.Now().UnixNano())).
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
		SetExchangeRate(decimal.NewFromInt(1)).
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

func (s *PeriodServiceSuite) Test_Revalue() {
	ctx := s.ctx()

	// Seed COA template
	accountRepo := repository.NewAccountRepository(s.client)
	_, err := accountRepo.Seed(ctx, "freelancer", "en")
	s.Require().NoError(err)

	// Create USD Bank account
	bankParent, err := accountRepo.FindByCode(ctx, "1110")
	s.Require().NoError(err)
	usdCurr := "USD"
	usdAcc, err := accountRepo.Create(ctx, model.CreateAccountRequest{
		Code:     "1115",
		Name:     "Wise USD",
		Type:     "ASSET",
		ParentID: bankParent.ID,
		Currency: usdCurr,
	}, "DEBIT")
	s.Require().NoError(err)

	capitalAcc, err := accountRepo.FindByCode(ctx, "3100")
	s.Require().NoError(err)

	// Setup Open Period for Jan 2026
	p, err := s.svc.Create(ctx, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseDate("2026-01-01"),
		EndDate:    parseDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)

	// Seed rate on 2026-01-01: 1 USD = 15,000 IDR
	rateRepo := repository.NewExchangeRateRepository(s.client)
	_, err = rateRepo.Create(ctx, model.CreateExchangeRateRequest{
		FromCurrency:  "USD",
		ToCurrency:    "IDR",
		Rate:          decimal.NewFromInt(15000),
		EffectiveDate: model.Date(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
	})
	s.Require().NoError(err)

	// Post an initial transaction on 2026-01-01 of 100 USD
	trx, err := s.client.Transaction.Create().
		SetNumber("TRX-2026-000001").
		SetTransactionDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetCurrency("USD").
		SetExchangeRate(decimal.NewFromInt(15000)).
		SetSource("MANUAL").
		Save(ctx)
	s.Require().NoError(err)

	e, err := s.client.Entry.Create().
		SetTransactionID(trx.ID).
		SetPeriodID(p.ID).
		SetNumber("JE-2026-000001").
		SetEntryDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetStatus("POSTED").
		SetPostedAt(time.Now()).
		SetPostedBy(s.userID).
		Save(ctx)
	s.Require().NoError(err)

	// 100 USD = 1,500,000 IDR
	_, err = s.client.Posting.Create().
		SetEntryID(e.ID).
		SetAccountID(usdAcc.ID).
		SetDirection("DEBIT").
		SetAmount(decimal.NewFromInt(100)).
		SetBaseAmount(decimal.NewFromInt(1500000)).
		SetPosition(0).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.client.Posting.Create().
		SetEntryID(e.ID).
		SetAccountID(capitalAcc.ID).
		SetDirection("CREDIT").
		SetAmount(decimal.NewFromInt(100)).
		SetBaseAmount(decimal.NewFromInt(1500000)).
		SetPosition(1).
		Save(ctx)
	s.Require().NoError(err)

	// Set rate on period.EndDate (2026-01-31): 1 USD = 16,000 IDR
	// New value = 100 * 16000 = 1,600,000 IDR. Diff = +100,000 IDR (Gain)
	_, err = rateRepo.Create(ctx, model.CreateExchangeRateRequest{
		FromCurrency:  "USD",
		ToCurrency:    "IDR",
		Rate:          decimal.NewFromInt(16000),
		EffectiveDate: model.Date(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)),
	})
	s.Require().NoError(err)

	revalEntry, err := s.svc.Revalue(ctx, p.ID, s.userID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusPosted, revalEntry.Status)
	s.Equal(p.EndDate.Format("2006-01-02"), revalEntry.EntryDate.Format("2006-01-02"))
	s.Len(revalEntry.Postings, 2)

	// Postings:
	// DEBIT to usdAcc: amount = 0, base_amount = 100,000
	// CREDIT to 7220 (Unrealized FX Gain): amount = 0, base_amount = 100,000
	gainAcc, err := accountRepo.FindByCode(ctx, "7220")
	s.Require().NoError(err)

	var foundForeign, foundGain bool
	for _, posting := range revalEntry.Postings {
		if posting.AccountID == usdAcc.ID {
			foundForeign = true
			s.Equal("DEBIT", posting.Direction)
			s.True(posting.Amount.Equal(decimal.Zero))
			s.True(posting.BaseAmount.Equal(decimal.NewFromInt(100000)))
		} else if posting.AccountID == gainAcc.ID {
			foundGain = true
			s.Equal("CREDIT", posting.Direction)
			s.True(posting.Amount.Equal(decimal.Zero))
			s.True(posting.BaseAmount.Equal(decimal.NewFromInt(100000)))
		}
	}
	s.True(foundForeign)
	s.True(foundGain)

	// Idempotent test: Re-running revalue should fail with domain.ErrPeriodAlreadyRevalued
	_, err = s.svc.Revalue(ctx, p.ID, s.userID)
	s.ErrorIs(err, domain.ErrPeriodAlreadyRevalued)
}

func TestPeriodServiceSuite(t *testing.T) {
	suite.Run(t, new(PeriodServiceSuite))
}
