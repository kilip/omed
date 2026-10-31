package service_test

import (
	"context"
	"encoding/json"
	"errors"
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

type InsightServiceSuite struct {
	suite.Suite
	client      *ent.Client
	reportSvc   service.ReportService
	accountSvc  service.AccountService
	cashflowSvc service.CashFlowService
	accountRepo repository.AccountRepository
	entryRepo   repository.EntryRepository
	periodRepo  repository.PeriodRepository
	workspaceID int64
	userID      int64
	periodID    int64
	fixedTime   time.Time
}

func (s *InsightServiceSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:insight_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.entryRepo = repository.NewEntryRepository(s.client)
	s.accountRepo = repository.NewAccountRepository(s.client)
	s.periodRepo = repository.NewPeriodRepository(s.client)
	currencyRepo := repository.NewCurrencyRepository(s.client)
	rateRepo := repository.NewExchangeRateRepository(s.client)
	cfRepo := repository.NewCashFlowRepository(s.client)

	s.reportSvc = service.NewReportService(s.entryRepo, s.accountRepo, currencyRepo)
	s.accountSvc = service.NewAccountService(s.accountRepo)
	s.cashflowSvc = service.NewCashFlowService(cfRepo, s.periodRepo, s.accountRepo, rateRepo, currencyRepo, nil)
	s.fixedTime = time.Date(2026, 5, 15, 10, 0, 0, 0, time.UTC)
}

func (s *InsightServiceSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *InsightServiceSuite) SetupTest() {
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
		Name:       "FY 2026",
		StartDate:  model.Date(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:    model.Date(time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)
	s.periodID = p.ID
}

func (s *InsightServiceSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Charlie",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func (s *InsightServiceSuite) newService() service.InsightService {
	return service.NewInsightService(
		s.reportSvc,
		s.accountSvc,
		s.cashflowSvc,
		service.WithInsightClock(func() time.Time { return s.fixedTime }),
	)
}

func (s *InsightServiceSuite) seedEntries() {
	ctx := s.ctx()
	_, err := s.accountRepo.Seed(ctx, "freelancer", "en")
	s.Require().NoError(err)

	bankAcc, err := s.accountRepo.FindByCode(ctx, "1111")
	s.Require().NoError(err)
	revAcc, err := s.accountRepo.FindByCode(ctx, "4140")
	s.Require().NoError(err)
	expAcc, err := s.accountRepo.FindByCode(ctx, "5100")
	s.Require().NoError(err)

	// Seed entries across months:
	// 2025-12: Rev 1m, Exp 500k
	// 2026-01: Rev 2m, Exp 800k
	// 2026-02: Rev 3m, Exp 1m
	// 2026-03: Rev 4m, Exp 1.5m
	// 2026-04: Rev 5m, Exp 2m
	// 2026-05: Rev 6m, Exp 2.5m
	entries := []struct {
		date   time.Time
		revAmt decimal.Decimal
		expAmt decimal.Decimal
		suffix string
	}{
		{date: time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC), revAmt: dec(1000000), expAmt: dec(500000), suffix: "2025-12"},
		{date: time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC), revAmt: dec(2000000), expAmt: dec(800000), suffix: "2026-01"},
		{date: time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC), revAmt: dec(3000000), expAmt: dec(1000000), suffix: "2026-02"},
		{date: time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC), revAmt: dec(4000000), expAmt: dec(1500000), suffix: "2026-03"},
		{date: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC), revAmt: dec(5000000), expAmt: dec(2000000), suffix: "2026-04"},
		{date: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC), revAmt: dec(6000000), expAmt: dec(2500000), suffix: "2026-05"},
	}

	for _, e := range entries {
		trx, err := s.client.Transaction.Create().
			SetNumber("TRX-" + e.suffix).
			SetTransactionDate(e.date).
			SetDescription("Trx " + e.suffix).
			SetSource("MANUAL").
			SetCurrency("IDR").
			SetExchangeRate(dec(1)).
			Save(ctx)
		s.Require().NoError(err)

		// Revenue entry
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

		// Expense entry
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
}

func (s *InsightServiceSuite) Test_PeriodOverview_DefaultsToCurrentMonthWithPrevious() {
	s.seedEntries()
	svc := s.newService()
	ctx := s.ctx()

	res, err := svc.PeriodOverview(ctx, model.PeriodOverviewRequest{
		WorkspaceID: s.workspaceID,
	})
	s.Require().NoError(err)
	s.Equal("2026-05", res.From)
	s.Equal("2026-05", res.To)
	s.Equal("IDR", res.BaseCurrency)
	s.True(res.Revenue.Equal(dec(6000000)))
	s.True(res.Expense.Equal(dec(2500000)))
	s.True(res.Net.Equal(dec(3500000)))
	s.Require().Len(res.Monthly, 1)
	s.Equal("2026-05", res.Monthly[0].Month)

	s.Require().NotNil(res.Previous)
	s.Equal("2026-04", res.Previous.From)
	s.Equal("2026-04", res.Previous.To)
	s.True(res.Previous.Revenue.Equal(dec(5000000)))
	s.True(res.Previous.Expense.Equal(dec(2000000)))
	s.True(res.Previous.Net.Equal(dec(3000000)))
	s.Require().Len(res.Previous.Monthly, 1)
	s.Equal("2026-04", res.Previous.Monthly[0].Month)

	// Delta = current (2026-05) minus previous (2026-04)
	s.True(res.Previous.Delta.Revenue.Equal(dec(1000000))) // 6m - 5m
	s.True(res.Previous.Delta.Expense.Equal(dec(500000)))  // 2.5m - 2m
	s.True(res.Previous.Delta.Net.Equal(dec(500000)))      // 3.5m - 3m
}

func (s *InsightServiceSuite) Test_PeriodOverview_RangeEqualsSummaryMonthly() {
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
		{date: m2, revAmt: dec(2000000), expAmt: dec(1000000), suffix: "sm-m2"},
		{date: m1, revAmt: dec(5000000), expAmt: dec(2000000), suffix: "sm-m1"},
		{date: m0, revAmt: dec(10000000), expAmt: dec(4000000), suffix: "sm-m0"},
	}

	for _, e := range entries {
		trx, err := s.client.Transaction.Create().
			SetNumber("TRX-" + e.suffix).
			SetTransactionDate(e.date).
			SetDescription("Trx " + e.suffix).
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

	svc := s.newService()
	res, err := svc.PeriodOverview(ctx, model.PeriodOverviewRequest{
		WorkspaceID: s.workspaceID,
		FromMonth:   m2.Format("2006-01"),
		ToMonth:     m0.Format("2006-01"),
	})
	s.Require().NoError(err)
	s.Require().Len(res.Monthly, 3)

	summary, err := s.reportSvc.Summary(ctx, 3)
	s.Require().NoError(err)
	s.Require().Len(summary.Monthly, 3)

	for i, m := range res.Monthly {
		s.Equal(summary.Monthly[i].Month, m.Month)
		s.True(summary.Monthly[i].Revenue.Equal(m.Revenue), "revenue mismatch at month %s", m.Month)
		s.True(summary.Monthly[i].Expense.Equal(m.Expense), "expense mismatch at month %s", m.Month)
		s.True(summary.Monthly[i].Revenue.Sub(summary.Monthly[i].Expense).Equal(m.Net), "net mismatch at month %s", m.Month)
	}
}

func (s *InsightServiceSuite) Test_PeriodOverview_PreviousIsSameLengthEndingBeforeFrom() {
	s.seedEntries()
	svc := s.newService()
	ctx := s.ctx()

	res, err := svc.PeriodOverview(ctx, model.PeriodOverviewRequest{
		WorkspaceID: s.workspaceID,
		FromMonth:   "2026-03",
		ToMonth:     "2026-05",
	})
	s.Require().NoError(err)
	s.Equal("2026-03", res.From)
	s.Equal("2026-05", res.To)
	// Current totals: 2026-03 (4m/1.5m) + 2026-04 (5m/2m) + 2026-05 (6m/2.5m) = Rev: 15m, Exp: 6m, Net: 9m
	s.True(res.Revenue.Equal(dec(15000000)))
	s.True(res.Expense.Equal(dec(6000000)))
	s.True(res.Net.Equal(dec(9000000)))

	// Previous period: 2025-12 to 2026-02 (3 months)
	s.Require().NotNil(res.Previous)
	s.Equal("2025-12", res.Previous.From)
	s.Equal("2026-02", res.Previous.To)
	s.Require().Len(res.Previous.Monthly, 3)
	// Prev totals: 2025-12 (1m/500k) + 2026-01 (2m/800k) + 2026-02 (3m/1m) = Rev: 6m, Exp: 2.3m, Net: 3.7m
	s.True(res.Previous.Revenue.Equal(dec(6000000)))
	s.True(res.Previous.Expense.Equal(dec(2300000)))
	s.True(res.Previous.Net.Equal(dec(3700000)))

	// Delta = current - prev: Rev: 15m - 6m = 9m, Exp: 6m - 2.3m = 3.7m, Net: 9m - 3.7m = 5.3m
	s.True(res.Previous.Delta.Revenue.Equal(dec(9000000)))
	s.True(res.Previous.Delta.Expense.Equal(dec(3700000)))
	s.True(res.Previous.Delta.Net.Equal(dec(5300000)))
}

func (s *InsightServiceSuite) Test_PeriodOverview_CompareFalse_OmitsPrevious() {
	s.seedEntries()
	svc := s.newService()
	ctx := s.ctx()
	compare := false

	res, err := svc.PeriodOverview(ctx, model.PeriodOverviewRequest{
		WorkspaceID:     s.workspaceID,
		FromMonth:       "2026-03",
		ToMonth:         "2026-05",
		ComparePrevious: &compare,
	})
	s.Require().NoError(err)
	s.Nil(res.Previous)
	s.Equal("2026-03", res.From)
	s.Equal("2026-05", res.To)
	s.Len(res.Monthly, 3)
}

func (s *InsightServiceSuite) Test_PeriodOverview_EmptyMonthsStillListed() {
	ctx := s.ctx()
	_, err := s.accountRepo.Seed(ctx, "freelancer", "en")
	s.Require().NoError(err)

	svc := s.newService()
	res, err := svc.PeriodOverview(ctx, model.PeriodOverviewRequest{
		WorkspaceID: s.workspaceID,
		FromMonth:   "2024-01",
		ToMonth:     "2024-03",
	})
	s.Require().NoError(err)
	s.Require().Len(res.Monthly, 3)
	s.True(res.Revenue.IsZero())
	s.True(res.Expense.IsZero())
	s.True(res.Net.IsZero())
	for _, m := range res.Monthly {
		s.True(m.Revenue.IsZero())
		s.True(m.Expense.IsZero())
		s.True(m.Net.IsZero())

		// Verify JSON marshaling produces string numbers
		b, err := json.Marshal(m)
		s.Require().NoError(err)
		s.Contains(string(b), `"revenue":"0"`)
		s.Contains(string(b), `"expense":"0"`)
		s.Contains(string(b), `"net":"0"`)
	}
}

func (s *InsightServiceSuite) Test_PeriodOverview_InvalidInput() {
	svc := s.newService()
	ctx := s.ctx()

	tests := []struct {
		name       string
		from       string
		to         string
		wantDetail string
	}{
		{
			name:       "invalid month 13",
			from:       "2026-13",
			to:         "2026-13",
			wantDetail: "from_month must use the format YYYY-MM",
		},
		{
			name:       "invalid slash separator",
			from:       "2026/03",
			to:         "2026/03",
			wantDetail: "from_month must use the format YYYY-MM",
		},
		{
			name:       "to before from",
			from:       "2026-05",
			to:         "2026-03",
			wantDetail: "to_month must not be before from_month",
		},
		{
			name:       "25-month range exceeding 24 months",
			from:       "2024-01",
			to:         "2026-01",
			wantDetail: "range must not exceed 24 months",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, err := svc.PeriodOverview(ctx, model.PeriodOverviewRequest{
				WorkspaceID: s.workspaceID,
				FromMonth:   tc.from,
				ToMonth:     tc.to,
			})
			s.Require().Error(err)
			s.True(errors.Is(err, domain.ErrInvalidArgument), "expected ErrInvalidArgument for %s, got %v", tc.name, err)
			var argErr domain.ArgumentError
			s.True(errors.As(err, &argErr))
			s.Equal(tc.wantDetail, argErr.ErrorDetail())
		})
	}
}

func (s *InsightServiceSuite) Test_AccountBalance_CodePrefix() {
	s.seedEntries()
	svc := s.newService()
	ctx := s.ctx()

	res, err := svc.AccountBalance(ctx, model.AccountBalanceRequest{
		WorkspaceID: s.workspaceID,
		Account:     "11",
	})
	s.Require().NoError(err)
	s.NotEmpty(res.Accounts)

	sumDebit := decimal.Zero
	sumCredit := decimal.Zero
	for _, acc := range res.Accounts {
		s.True(len(acc.Code) >= 2 && acc.Code[:2] == "11", "expected code starting with 11, got %s", acc.Code)
		sumDebit = sumDebit.Add(acc.Debit)
		sumCredit = sumCredit.Add(acc.Credit)
	}
	s.True(res.Totals.Debit.Equal(sumDebit), "totals debit should match sum over matched accounts")
	s.True(res.Totals.Credit.Equal(sumCredit), "totals credit should match sum over matched accounts")
}

func (s *InsightServiceSuite) Test_AccountBalance_DigitsOnlyNeverMatchesNames() {
	ctx := s.ctx()
	_, err := s.client.Account.Create().
		SetWorkspaceID(s.workspaceID).
		SetCode("5000").
		SetName("Fund 11").
		SetType("EXPENSE").
		SetNormalBalance("DEBIT").
		SetCurrency("IDR").
		SetIsActive(true).
		Save(ctx)
	s.Require().NoError(err)

	svc := s.newService()
	res, err := svc.AccountBalance(ctx, model.AccountBalanceRequest{
		WorkspaceID: s.workspaceID,
		Account:     "11",
	})
	s.Require().NoError(err)
	for _, acc := range res.Accounts {
		s.NotEqual("5000", acc.Code, "digits-only filter must not match names containing the digits")
	}
}

func (s *InsightServiceSuite) Test_AccountBalance_NameFragmentCaseInsensitive() {
	ctx := s.ctx()
	_, err := s.client.Account.Create().
		SetWorkspaceID(s.workspaceID).
		SetCode("1199").
		SetName("Bank Kaltim").
		SetType("ASSET").
		SetNormalBalance("DEBIT").
		SetCurrency("IDR").
		SetIsActive(true).
		Save(ctx)
	s.Require().NoError(err)

	svc := s.newService()
	res, err := svc.AccountBalance(ctx, model.AccountBalanceRequest{
		WorkspaceID: s.workspaceID,
		Account:     "BANK",
	})
	s.Require().NoError(err)
	found := false
	for _, acc := range res.Accounts {
		if acc.Code == "1199" && acc.Name == "Bank Kaltim" {
			found = true
			break
		}
	}
	s.True(found, "expected to find Bank Kaltim for case-insensitive 'BANK' search")
}

func (s *InsightServiceSuite) Test_AccountBalance_EmptyFilterMeansAll() {
	s.seedEntries()
	svc := s.newService()
	ctx := s.ctx()

	resEmpty, err := svc.AccountBalance(ctx, model.AccountBalanceRequest{
		WorkspaceID: s.workspaceID,
		Account:     "",
	})
	s.Require().NoError(err)
	s.NotEmpty(resEmpty.Accounts)

	resWhitespace, err := svc.AccountBalance(ctx, model.AccountBalanceRequest{
		WorkspaceID: s.workspaceID,
		Account:     "   ",
	})
	s.Require().NoError(err)
	s.Equal(len(resEmpty.Accounts), len(resWhitespace.Accounts))
}

func (s *InsightServiceSuite) Test_AccountBalance_MoreThan50_TruncatedWithTotalsOverAll() {
	ctx := s.ctx()
	svc := s.newService()

	// Create 60 active accounts under 80xx
	for i := 1; i <= 60; i++ {
		code := fmt.Sprintf("80%02d", i)
		_, err := s.client.Account.Create().
			SetWorkspaceID(s.workspaceID).
			SetCode(code).
			SetName(fmt.Sprintf("Account %s", code)).
			SetType("EXPENSE").
			SetNormalBalance("DEBIT").
			SetCurrency("IDR").
			SetIsActive(true).
			Save(ctx)
		s.Require().NoError(err)
	}

	res, err := svc.AccountBalance(ctx, model.AccountBalanceRequest{
		WorkspaceID: s.workspaceID,
		Account:     "80",
	})
	s.Require().NoError(err)
	s.Len(res.Accounts, 50)
	s.True(res.Truncated)
	s.Contains(res.Hint, "account")
}

func (s *InsightServiceSuite) Test_AccountBalance_NoAccounts_EmptyAndBalanced() {
	ctx := s.ctx()
	// Create an empty workspace with no accounts
	wEmpty, err := s.client.Workspace.Create().
		SetName("Empty WS").
		SetOwnerID(s.userID).
		Save(ctx)
	s.Require().NoError(err)

	svc := s.newService()
	res, err := svc.AccountBalance(ctx, model.AccountBalanceRequest{
		WorkspaceID: wEmpty.ID,
	})
	s.Require().NoError(err)
	s.Empty(res.Accounts)
	s.True(res.IsBalanced)
	s.NotEmpty(res.BaseCurrency)
}

func (s *InsightServiceSuite) Test_AccountBalance_IsBalancedIgnoresFilter() {
	s.seedEntries()
	svc := s.newService()
	ctx := s.ctx()

	res, err := svc.AccountBalance(ctx, model.AccountBalanceRequest{
		WorkspaceID: s.workspaceID,
		Account:     "1111",
	})
	s.Require().NoError(err)
	s.Require().Len(res.Accounts, 1)
	s.False(res.Totals.Debit.Equal(res.Totals.Credit), "single account totals should not balance")
	s.True(res.IsBalanced, "IsBalanced should reflect the whole ledger being balanced")
}

func (s *InsightServiceSuite) Test_AccountBalance_InvalidDate() {
	svc := s.newService()
	ctx := s.ctx()

	_, err := svc.AccountBalance(ctx, model.AccountBalanceRequest{
		WorkspaceID: s.workspaceID,
		AsOfDate:    "2026/05/15",
	})
	s.Require().Error(err)
	s.True(errors.Is(err, domain.ErrInvalidArgument))
	var argErr domain.ArgumentError
	s.True(errors.As(err, &argErr))
	s.Equal("as_of_date must use the format YYYY-MM-DD", argErr.ErrorDetail())
}

func (s *InsightServiceSuite) Test_AccountBalance_Shrink() {
	res := model.AccountBalanceResponse{
		Accounts: make([]model.AccountBalanceItem, 50),
	}
	s.True(res.Shrink())
	s.Len(res.Accounts, 25)
	s.True(res.Truncated)
	s.Contains(res.Hint, "account")

	// Shrink down to 0
	res.Accounts = make([]model.AccountBalanceItem, 1)
	s.True(res.Shrink())
	s.Len(res.Accounts, 0)
	s.False(res.Shrink(), "cannot shrink empty accounts")
}

func TestInsightServiceSuite(t *testing.T) {
	suite.Run(t, new(InsightServiceSuite))
}
