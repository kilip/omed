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
	"github.com/shopspring/decimal"
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
		SetExchangeRate(dec(1)).
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

	amt := dec(10000000)
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
		AsOfDate:                    model.Date(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)),
		IncludePerCurrencyBreakdown: &includeCurrencies,
	})
	s.Require().NoError(err)
	s.True(res.IsBalanced)
	s.True(res.TotalDebit.Equal(dec(10000000)))
	s.True(res.TotalCredit.Equal(dec(10000000)))
	s.Equal("IDR", res.BaseCurrency)

	var foundBank, foundCap bool
	for _, item := range res.Accounts {
		if item.AccountID == bankAcc.ID {
			foundBank = true
			s.True(item.Debit.Equal(dec(10000000)))
			s.True(item.Credit.Equal(decimal.Zero))
			s.True(item.NetBalance.Equal(dec(10000000)))
			s.NotEmpty(item.Currencies)
			s.Equal("IDR", item.Currencies[0].Currency)
		}
		if item.AccountID == capAcc.ID {
			foundCap = true
			s.True(item.Debit.Equal(decimal.Zero))
			s.True(item.Credit.Equal(dec(10000000)))
			s.True(item.NetBalance.Equal(dec(10000000)))
		}
	}
	s.True(foundBank)
	s.True(foundCap)
}

func (s *ReportServiceSuite) Test_Summary() {
	ctx := s.ctx()

	// Seed COA
	_, err := s.accountRepo.Seed(ctx, "freelancer", "en")
	s.Require().NoError(err)

	bankAcc, err := s.accountRepo.FindByCode(ctx, "1111")
	s.Require().NoError(err)
	revAcc, err := s.accountRepo.FindByCode(ctx, "4140")
	s.Require().NoError(err)
	expAcc, err := s.accountRepo.FindByCode(ctx, "5100")
	s.Require().NoError(err)

	now := time.Now()
	// Create transaction
	trx, err := s.client.Transaction.Create().
		SetNumber("TRX-TEST-000001").
		SetTransactionDate(now).
		SetDescription("Income and Expense Transaction").
		SetSource("MANUAL").
		SetCurrency("IDR").
		SetExchangeRate(dec(1)).
		Save(ctx)
	s.Require().NoError(err)

	// Entry 1: Revenue 10,000,000
	e1, err := s.client.Entry.Create().
		SetWorkspaceID(s.workspaceID).
		SetTransactionID(trx.ID).
		SetPeriodID(s.periodID).
		SetNumber("JE/TEST/0001").
		SetEntryDate(now).
		SetStatus("POSTED").
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.client.Posting.Create().
		SetWorkspaceID(s.workspaceID).
		SetEntryID(e1.ID).
		SetAccountID(bankAcc.ID).
		SetDirection("DEBIT").
		SetAmount(dec(10000000)).
		SetBaseAmount(dec(10000000)).
		SetPosition(0).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.client.Posting.Create().
		SetWorkspaceID(s.workspaceID).
		SetEntryID(e1.ID).
		SetAccountID(revAcc.ID).
		SetDirection("CREDIT").
		SetAmount(dec(10000000)).
		SetBaseAmount(dec(10000000)).
		SetPosition(1).
		Save(ctx)
	s.Require().NoError(err)

	// Entry 2: Expense 4,000,000
	e2, err := s.client.Entry.Create().
		SetWorkspaceID(s.workspaceID).
		SetTransactionID(trx.ID).
		SetPeriodID(s.periodID).
		SetNumber("JE/TEST/0002").
		SetEntryDate(now).
		SetStatus("POSTED").
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.client.Posting.Create().
		SetWorkspaceID(s.workspaceID).
		SetEntryID(e2.ID).
		SetAccountID(expAcc.ID).
		SetDirection("DEBIT").
		SetAmount(dec(4000000)).
		SetBaseAmount(dec(4000000)).
		SetPosition(0).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.client.Posting.Create().
		SetWorkspaceID(s.workspaceID).
		SetEntryID(e2.ID).
		SetAccountID(bankAcc.ID).
		SetDirection("CREDIT").
		SetAmount(dec(4000000)).
		SetBaseAmount(dec(4000000)).
		SetPosition(1).
		Save(ctx)
	s.Require().NoError(err)

	summary, err := s.svc.Summary(ctx, 6)
	s.Require().NoError(err)

	s.Equal("IDR", summary.BaseCurrency)
	s.True(summary.Totals.Assets.Equal(dec(6000000)))     // 10m debit - 4m credit
	s.True(summary.Totals.Revenue.Equal(dec(10000000)))  // 10m credit
	s.True(summary.Totals.Expense.Equal(dec(4000000)))   // 4m debit
	s.True(summary.Totals.NetIncome.Equal(dec(6000000))) // 10m - 4m
	s.Len(summary.Monthly, 6)

	currentMonthStr := now.Format("2006-01")
	var foundCurrentMonth bool
	for _, m := range summary.Monthly {
		if m.Month == currentMonthStr {
			foundCurrentMonth = true
			s.True(m.Revenue.Equal(dec(10000000)))
			s.True(m.Expense.Equal(dec(4000000)))
		}
	}
	s.True(foundCurrentMonth)
}

func (s *ReportServiceSuite) seedMonthlyEntries() (time.Time, time.Time, time.Time) {
	ctx := s.ctx()
	_, err := s.accountRepo.Seed(ctx, "freelancer", "en")
	s.Require().NoError(err)

	bankAcc, err := s.accountRepo.FindByCode(ctx, "1111")
	s.Require().NoError(err)
	revAcc, err := s.accountRepo.FindByCode(ctx, "4140")
	s.Require().NoError(err)
	expAcc, err := s.accountRepo.FindByCode(ctx, "5100")
	s.Require().NoError(err)

	now := time.Now()
	m0 := now
	m1 := now.AddDate(0, -1, 0)
	m2 := now.AddDate(0, -2, 0)

	entries := []struct {
		date   time.Time
		revAmt decimal.Decimal
		expAmt decimal.Decimal
		suffix string
	}{
		{date: m2, revAmt: dec(2000000), expAmt: dec(1000000), suffix: "M2"},
		{date: m1, revAmt: dec(5000000), expAmt: dec(2000000), suffix: "M1"},
		{date: m0, revAmt: dec(10000000), expAmt: dec(4000000), suffix: "M0"},
	}

	for _, e := range entries {
		trx, err := s.client.Transaction.Create().
			SetNumber("TRX-TEST-" + e.suffix).
			SetTransactionDate(e.date).
			SetDescription("Test Transaction " + e.suffix).
			SetSource("MANUAL").
			SetCurrency("IDR").
			SetExchangeRate(dec(1)).
			Save(ctx)
		s.Require().NoError(err)

		jeRev, err := s.client.Entry.Create().
			SetWorkspaceID(s.workspaceID).
			SetTransactionID(trx.ID).
			SetPeriodID(s.periodID).
			SetNumber("JE-REV-" + e.suffix).
			SetEntryDate(e.date).
			SetStatus("POSTED").
			Save(ctx)
		s.Require().NoError(err)

		_, err = s.client.Posting.Create().
			SetWorkspaceID(s.workspaceID).
			SetEntryID(jeRev.ID).
			SetAccountID(bankAcc.ID).
			SetDirection("DEBIT").
			SetAmount(e.revAmt).
			SetBaseAmount(e.revAmt).
			SetPosition(0).
			Save(ctx)
		s.Require().NoError(err)

		_, err = s.client.Posting.Create().
			SetWorkspaceID(s.workspaceID).
			SetEntryID(jeRev.ID).
			SetAccountID(revAcc.ID).
			SetDirection("CREDIT").
			SetAmount(e.revAmt).
			SetBaseAmount(e.revAmt).
			SetPosition(1).
			Save(ctx)
		s.Require().NoError(err)

		jeExp, err := s.client.Entry.Create().
			SetWorkspaceID(s.workspaceID).
			SetTransactionID(trx.ID).
			SetPeriodID(s.periodID).
			SetNumber("JE-EXP-" + e.suffix).
			SetEntryDate(e.date).
			SetStatus("POSTED").
			Save(ctx)
		s.Require().NoError(err)

		_, err = s.client.Posting.Create().
			SetWorkspaceID(s.workspaceID).
			SetEntryID(jeExp.ID).
			SetAccountID(expAcc.ID).
			SetDirection("DEBIT").
			SetAmount(e.expAmt).
			SetBaseAmount(e.expAmt).
			SetPosition(0).
			Save(ctx)
		s.Require().NoError(err)

		_, err = s.client.Posting.Create().
			SetWorkspaceID(s.workspaceID).
			SetEntryID(jeExp.ID).
			SetAccountID(bankAcc.ID).
			SetDirection("CREDIT").
			SetAmount(e.expAmt).
			SetBaseAmount(e.expAmt).
			SetPosition(1).
			Save(ctx)
		s.Require().NoError(err)
	}

	return m2, m1, m0
}

func (s *ReportServiceSuite) Test_MonthlyRevenueExpense_EqualsSummaryMonthly() {
	ctx := s.ctx()
	m2, _, _ := s.seedMonthlyEntries()

	summary, err := s.svc.Summary(ctx, 3)
	s.Require().NoError(err)
	s.Len(summary.Monthly, 3)

	monthly, err := s.svc.MonthlyRevenueExpense(ctx, m2, 3)
	s.Require().NoError(err)
	s.Len(monthly, 3)

	s.Equal(summary.Monthly, monthly)
}

func (s *ReportServiceSuite) Test_MonthlyRevenueExpense_EmptyMonthsAreZeroRows() {
	ctx := s.ctx()
	s.seedMonthlyEntries()

	firstMonth := time.Date(2020, 1, 15, 10, 0, 0, 0, time.UTC)
	monthly, err := s.svc.MonthlyRevenueExpense(ctx, firstMonth, 4)
	s.Require().NoError(err)
	s.Require().Len(monthly, 4)

	expectedMonths := []string{"2020-01", "2020-02", "2020-03", "2020-04"}
	for i, item := range monthly {
		s.Equal(expectedMonths[i], item.Month)
		s.True(item.Revenue.IsZero(), "revenue for %s should be zero", item.Month)
		s.True(item.Expense.IsZero(), "expense for %s should be zero", item.Month)
	}
}

func (s *ReportServiceSuite) Test_MonthlyRevenueExpense_RangeNotEndingToday() {
	ctx := s.ctx()
	m2, _, _ := s.seedMonthlyEntries()

	m3 := m2.AddDate(0, -1, 0)
	monthly, err := s.svc.MonthlyRevenueExpense(ctx, m3, 2)
	s.Require().NoError(err)
	s.Require().Len(monthly, 2)

	m3Str := m3.Format("2006-01")
	m2Str := m2.Format("2006-01")
	s.Equal(m3Str, monthly[0].Month)
	s.True(monthly[0].Revenue.IsZero())
	s.True(monthly[0].Expense.IsZero())

	s.Equal(m2Str, monthly[1].Month)
	s.True(monthly[1].Revenue.Equal(dec(2000000)))
	s.True(monthly[1].Expense.Equal(dec(1000000)))
}

func TestReportServiceSuite(t *testing.T) {
	suite.Run(t, new(ReportServiceSuite))
}
