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
	"github.com/stretchr/testify/suite"
)

type CurrencyServiceSuite struct {
	suite.Suite
	client      *ent.Client
	svc         service.CurrencyService
	entryRepo   repository.EntryRepository
	workspaceID int64
	userID      int64
	accountID1  int64
	accountID2  int64
	periodID    int64
}

func TestCurrencyServiceSuite(t *testing.T) {
	suite.Run(t, new(CurrencyServiceSuite))
}

func (s *CurrencyServiceSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:currency_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	core.InitEntCli(s.client)
	currRepo := repository.NewCurrencyRepository(s.client)
	s.entryRepo = repository.NewEntryRepository(s.client)
	s.svc = service.NewCurrencyService(currRepo, s.entryRepo)
}

func (s *CurrencyServiceSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *CurrencyServiceSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.Posting.Delete().Exec(ctx)
	_, _ = s.client.Entry.Delete().Exec(ctx)
	_, _ = s.client.Transaction.Delete().Exec(ctx)
	_, _ = s.client.Period.Delete().Exec(ctx)
	_, _ = s.client.Account.Delete().Exec(ctx)
	_, _ = s.client.Currency.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)

	u, err := s.client.User.Create().
		SetName("Charlie").
		SetEmail("charlie@example.com").
		Save(ctx)
	s.Require().NoError(err)
	s.userID = u.ID

	userCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   u.ID,
		Name: "Charlie",
	})
	w, err := s.client.Workspace.Create().
		SetName("Currency Workspace").
		SetOwnerID(u.ID).
		SetBaseCurrency("IDR").
		Save(userCtx)
	s.Require().NoError(err)
	s.workspaceID = w.ID

	err = seed.SeedCurrencies(ctx, s.client)
	s.Require().NoError(err)

	// Seed period and accounts
	p, err := s.client.Period.Create().
		SetWorkspaceID(w.ID).
		SetName("FY2026").
		SetStartDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)).
		SetFiscalYear(2026).
		Save(userCtx)
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
		Save(userCtx)
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
		Save(userCtx)
	s.Require().NoError(err)
	s.accountID2 = a2.ID
}

func (s *CurrencyServiceSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Charlie",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func (s *CurrencyServiceSuite) Test_ListCurrencies() {
	currencies, err := s.svc.ListCurrencies(s.ctx())
	s.Require().NoError(err)
	s.Assert().NotEmpty(currencies)
}

func (s *CurrencyServiceSuite) Test_UpdateBaseCurrency_Success() {
	resp, err := s.svc.UpdateBaseCurrency(s.ctx(), s.workspaceID, model.UpdateBaseCurrencyRequest{
		BaseCurrency: "USD",
	})
	s.Require().NoError(err)
	s.Assert().Equal("USD", resp.BaseCurrency)
}

func (s *CurrencyServiceSuite) Test_UpdateBaseCurrency_InvalidCurrency() {
	_, err := s.svc.UpdateBaseCurrency(s.ctx(), s.workspaceID, model.UpdateBaseCurrencyRequest{
		BaseCurrency: "XYZ",
	})
	s.Assert().ErrorIs(err, domain.ErrCurrencyNotFound)
}

func (s *CurrencyServiceSuite) Test_UpdateBaseCurrency_LockedWhenPosted() {
	ctx := s.ctx()

	// Create a transaction and a POSTED entry
	tRepo := repository.NewTransactionRepository(s.client)
	tx, err := tRepo.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Description:     "Test Transaction",
		Currency:        "IDR",
	})
	s.Require().NoError(err)

	entry, err := s.entryRepo.Create(ctx, tx.ID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: domain.PostingDirectionDebit, Amount: 1000},
			{AccountID: s.accountID2, Direction: domain.PostingDirectionCredit, Amount: 1000},
		},
	})
	s.Require().NoError(err)

	// Post the entry
	_, err = s.entryRepo.Post(ctx, entry.ID, s.userID)
	s.Require().NoError(err)

	// Updating base currency must now fail with ErrBaseCurrencyLocked
	_, err = s.svc.UpdateBaseCurrency(ctx, s.workspaceID, model.UpdateBaseCurrencyRequest{
		BaseCurrency: "USD",
	})
	s.Assert().ErrorIs(err, domain.ErrBaseCurrencyLocked)
}
