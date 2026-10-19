package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/fin/service"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type ReportServiceSuite struct {
	suite.Suite
	client      *ent.Client
	svc         service.ReportService
	accountRepo repository.AccountRepository
	entryRepo   repository.EntryRepository
	periodRepo  repository.PeriodRepository
	workspaceID int64
	userID      int64
	periodID    int64
}

func (s *ReportServiceSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:report_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.entryRepo = repository.NewEntryRepository(s.client)
	s.accountRepo = repository.NewAccountRepository(s.client)
	s.periodRepo = repository.NewPeriodRepository(s.client)
	currencyRepo := repository.NewCurrencyRepository(s.client)
	s.svc = service.NewReportService(s.entryRepo, s.accountRepo, currencyRepo)
}

func (s *ReportServiceSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *ReportServiceSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
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
		SetName("Charlie").
		SetEmail(fmt.Sprintf("charlie-%d@example.com", time.Now().UnixNano())).
		Save(ctx)
	s.Require().NoError(err)
	s.userID = u.ID

	userCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   u.ID,
		Name: "Charlie",
	})
	w, err := s.client.Workspace.Create().
		SetName("Finance WS").
		SetOwnerID(u.ID).
		Save(userCtx)
	s.Require().NoError(err)
	s.workspaceID = w.ID

	p, err := s.periodRepo.Create(s.ctx(), model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseDate("2026-01-01"),
		EndDate:    parseDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)
	s.periodID = p.ID
}

func (s *ReportServiceSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Charlie",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func (s *ReportServiceSuite) Test_TrialBalance() {
	ctx := s.ctx()

	// Seed COA
	_, err := s.accountRepo.Seed(ctx, "freelancer", "en")
	s.Require().NoError(err)

	bankAcc, err := s.accountRepo.FindByCode(ctx, "1111")
	s.Require().NoError(err)

	capAcc, err := s.accountRepo.FindByCode(ctx, "3100")
	s.Require().NoError(err)

	// Post a transaction: 10,000,000 IDR capital injection
	eDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	trx, err := s.client.Transaction.Create().
		SetNumber("TRX-2026-000001").
		SetTransactionDate(eDate).
		SetCurrency("IDR").
		SetExchangeRate(1.0).
		SetSource("MANUAL").
		Save(ctx)
	s.Require().NoError(err)

	e, err := s.client.Entry.Create().
		SetTransactionID(trx.ID).
		SetPeriodID(s.periodID).
		SetNumber("JE-2026-000001").
		SetEntryDate(eDate).
		SetStatus("POSTED").
		SetPostedAt(time.Now()).
		SetPostedBy(s.userID).
		Save(ctx)
	s.Require().NoError(err)

	amt := 10000000.0
	_, err = s.client.Posting.Create().
		SetEntryID(e.ID).
		SetAccountID(bankAcc.ID).
		SetDirection("DEBIT").
		SetAmount(amt).
		SetBaseAmount(amt).
		SetPosition(0).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.client.Posting.Create().
		SetEntryID(e.ID).
		SetAccountID(capAcc.ID).
		SetDirection("CREDIT").
		SetAmount(amt).
		SetBaseAmount(amt).
		SetPosition(1).
		Save(ctx)
	s.Require().NoError(err)

	// Generate trial balance as of 2026-01-31
	includeCurrencies := true
	res, err := s.svc.TrialBalance(ctx, model.TrialBalanceRequest{
		AsOfDate:                   model.Date(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)),
		IncludePerCurrencyBreakdown: &includeCurrencies,
	})
	s.Require().NoError(err)
	s.True(res.IsBalanced)
	s.Equal(10000000.0, res.TotalDebit)
	s.Equal(10000000.0, res.TotalCredit)
	s.Equal("IDR", res.BaseCurrency)

	var foundBank, foundCap bool
	for _, item := range res.Accounts {
		if item.AccountID == bankAcc.ID {
			foundBank = true
			s.Equal(10000000.0, item.Debit)
			s.Equal(0.0, item.Credit)
			s.Equal(10000000.0, item.NetBalance)
			s.NotEmpty(item.Currencies)
			s.Equal("IDR", item.Currencies[0].Currency)
		}
		if item.AccountID == capAcc.ID {
			foundCap = true
			s.Equal(0.0, item.Debit)
			s.Equal(10000000.0, item.Credit)
			s.Equal(10000000.0, item.NetBalance)
		}
	}
	s.True(foundBank)
	s.True(foundCap)
}

func TestReportServiceSuite(t *testing.T) {
	suite.Run(t, new(ReportServiceSuite))
}
