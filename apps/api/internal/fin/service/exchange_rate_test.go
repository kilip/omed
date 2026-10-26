package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/fin/seed"
	"github.com/kilip/omed/api/internal/fin/service"
	"github.com/kilip/omed/api/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type ExchangeRateServiceSuite struct {
	suite.Suite
	client      *ent.Client
	svc         service.ExchangeRateService
	txSvc       service.TransactionService
	entrySvc    service.EntryService
	workspaceID int64
	userID      int64
	accountID1  int64
	accountID2  int64
	periodID    int64
}

func TestExchangeRateServiceSuite(t *testing.T) {
	suite.Run(t, new(ExchangeRateServiceSuite))
}

func (s *ExchangeRateServiceSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:exchangerate_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	core.InitEntCli(s.client)

	currRepo := repository.NewCurrencyRepository(s.client)
	rateRepo := repository.NewExchangeRateRepository(s.client)
	s.svc = service.NewExchangeRateService(rateRepo, currRepo)

	periodRepo := repository.NewPeriodRepository(s.client)
	accountRepo := repository.NewAccountRepository(s.client)
	txRepo := repository.NewTransactionRepository(s.client)
	entryRepo := repository.NewEntryRepository(s.client)

	s.txSvc = service.NewTransactionService(txRepo, periodRepo, accountRepo, rateRepo, currRepo)
	s.entrySvc = service.NewEntryService(entryRepo, txRepo, periodRepo, accountRepo, rateRepo, currRepo)
}

func (s *ExchangeRateServiceSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *ExchangeRateServiceSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(base.WithSkipWorkspaceFilter(context.Background()))
	_, _ = s.client.Posting.Delete().Exec(ctx)
	_, _ = s.client.Entry.Delete().Exec(ctx)
	_, _ = s.client.Transaction.Delete().Exec(ctx)
	_, _ = s.client.ExchangeRate.Delete().Exec(ctx)
	_, _ = s.client.Period.Delete().Exec(ctx)
	_, _ = s.client.Account.Delete().Exec(ctx)
	_, _ = s.client.Currency.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)

	u, err := s.client.User.Create().
		SetName("David").
		SetEmail("david@example.com").
		Save(ctx)
	s.Require().NoError(err)
	s.userID = u.ID

	userCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   u.ID,
		Name: "David",
	})
	w, err := s.client.Workspace.Create().
		SetName("FX Service Workspace").
		SetOwnerID(u.ID).
		SetBaseCurrency("IDR").
		Save(userCtx)
	s.Require().NoError(err)
	s.workspaceID = w.ID

	err = seed.SeedCurrencies(ctx, s.client)
	s.Require().NoError(err)

	wsCtx := base.WithWorkspaceID(userCtx, w.ID)

	p, err := s.client.Period.Create().
		SetWorkspaceID(w.ID).
		SetName("FY2026").
		SetStartDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)).
		SetFiscalYear(2026).
		Save(wsCtx)
	s.Require().NoError(err)
	s.periodID = p.ID

	a1, err := s.client.Account.Create().
		SetWorkspaceID(w.ID).
		SetCode("1001").
		SetName("Cash").
		SetType("ASSET").
		SetNormalBalance("DEBIT").
		SetIsPostable(true).
		SetIsActive(true).
		Save(wsCtx)
	s.Require().NoError(err)
	s.accountID1 = a1.ID

	a2, err := s.client.Account.Create().
		SetWorkspaceID(w.ID).
		SetCode("4001").
		SetName("Revenue").
		SetType("REVENUE").
		SetNormalBalance("CREDIT").
		SetIsPostable(true).
		SetIsActive(true).
		Save(wsCtx)
	s.Require().NoError(err)
	s.accountID2 = a2.ID
}

func (s *ExchangeRateServiceSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "David",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func (s *ExchangeRateServiceSuite) Test_Create_ValidatesCurrencies() {
	ctx := s.ctx()
	d := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	// Unknown from currency
	_, err := s.svc.Create(ctx, model.CreateExchangeRateRequest{
		FromCurrency:  "UNKNOWN",
		ToCurrency:    "IDR",
		Rate:          decimal.NewFromInt(16000),
		EffectiveDate: model.Date(d),
	})
	s.Assert().ErrorIs(err, domain.ErrCurrencyNotFound)

	// Valid currencies
	created, err := s.svc.Create(ctx, model.CreateExchangeRateRequest{
		FromCurrency:  "USD",
		ToCurrency:    "IDR",
		Rate:          decimal.NewFromInt(16200),
		EffectiveDate: model.Date(d),
	})
	s.Require().NoError(err)
	s.Assert().True(created.Rate.Equal(decimal.NewFromInt(16200)))
}

func (s *ExchangeRateServiceSuite) Test_TransactionRateResolution() {
	ctx := s.ctx()
	d := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	// 1. Transaction in base currency (IDR) -> rate defaults to 1.0
	tx1, err := s.txSvc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: model.Date(d),
		Currency:        "IDR",
		Description:     "Base currency tx",
	})
	s.Require().NoError(err)
	s.Assert().True(tx1.ExchangeRate.Equal(decimal.NewFromInt(1)))

	// 2. Transaction in foreign currency (USD) without explicit rate and without FX rate set -> ErrExchangeRateNotFound
	_, err = s.txSvc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: model.Date(d),
		Currency:        "USD",
		Description:     "Foreign currency tx without rate",
	})
	s.Assert().ErrorIs(err, domain.ErrExchangeRateNotFound)

	// 3. Create rate USD -> IDR
	_, err = s.svc.Create(ctx, model.CreateExchangeRateRequest{
		FromCurrency:  "USD",
		ToCurrency:    "IDR",
		Rate:          decimal.NewFromInt(16250),
		EffectiveDate: model.Date(d),
	})
	s.Require().NoError(err)

	// Now transaction in USD resolves rate from lookup
	tx2, err := s.txSvc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: model.Date(d),
		Currency:        "USD",
		Description:     "Foreign currency tx resolved from lookup",
	})
	s.Require().NoError(err)
	s.Assert().True(tx2.ExchangeRate.Equal(decimal.NewFromInt(16250)))

	// 4. Transaction with explicit rate uses explicit rate
	explicitRate := decimal.NewFromInt(16000)
	tx3, err := s.txSvc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: model.Date(d),
		Currency:        "USD",
		ExchangeRate:    &explicitRate,
		Description:     "Explicit rate tx",
	})
	s.Require().NoError(err)
	s.Assert().True(tx3.ExchangeRate.Equal(decimal.NewFromInt(16000)))
}
