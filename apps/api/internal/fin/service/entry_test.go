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

type EntryServiceSuite struct {
	suite.Suite
	client        *ent.Client
	svc           service.EntryService
	txRepo        repository.TransactionRepository
	periodRepo    repository.PeriodRepository
	accountRepo   repository.AccountRepository
	workspaceID   int64
	userID        int64
	accountID1    int64
	accountID2    int64
	periodID      int64
	transactionID int64
}

func (s *EntryServiceSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:entry_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	entryRepo := repository.NewEntryRepository(s.client)
	s.txRepo = repository.NewTransactionRepository(s.client)
	s.periodRepo = repository.NewPeriodRepository(s.client)
	s.accountRepo = repository.NewAccountRepository(s.client)
	s.svc = service.NewEntryService(entryRepo, s.txRepo, s.periodRepo, s.accountRepo)
}

func (s *EntryServiceSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *EntryServiceSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.Posting.Delete().Exec(ctx)
	_, _ = s.client.Entry.Delete().Exec(ctx)
	_, _ = s.client.Transaction.Delete().Exec(ctx)
	_, _ = s.client.Period.Delete().Exec(ctx)
	_, _ = s.client.Account.Delete().Exec(ctx)
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

	wsCtx := base.WithWorkspaceID(userCtx, w.ID)

	acc1, err := s.client.Account.Create().
		SetCode("1101").
		SetName("Cash").
		SetType("ASSET").
		SetNormalBalance("DEBIT").
		Save(wsCtx)
	s.Require().NoError(err)
	s.accountID1 = acc1.ID

	acc2, err := s.client.Account.Create().
		SetCode("4101").
		SetName("Revenue").
		SetType("REVENUE").
		SetNormalBalance("CREDIT").
		Save(wsCtx)
	s.Require().NoError(err)
	s.accountID2 = acc2.ID

	p, err := s.client.Period.Create().
		SetName("Jan 2026").
		SetStartDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)).
		SetFiscalYear(2026).
		Save(wsCtx)
	s.Require().NoError(err)
	s.periodID = p.ID

	trx, err := s.client.Transaction.Create().
		SetNumber("TRX-2026-000001").
		SetTransactionDate(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)).
		SetCurrency("IDR").
		SetExchangeRate(1.0).
		SetSource("MANUAL").
		Save(wsCtx)
	s.Require().NoError(err)
	s.transactionID = trx.ID
}

func (s *EntryServiceSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Alice",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func (s *EntryServiceSuite) Test_Create_Validations() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	// 1. Period not found
	_, err := s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  999999,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.ErrorIs(err, domain.ErrPeriodNotFound)

	// 1. Period closed
	closedP, err := s.client.Period.Create().
		SetName("Closed Period").
		SetStartDate(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)).
		SetFiscalYear(2026).
		SetStatus("CLOSED").
		Save(ctx)
	s.Require().NoError(err)

	febDate := model.Date(time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC))
	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  closedP.ID,
		EntryDate: febDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.ErrorIs(err, domain.ErrPeriodClosed)

	// 1. Date outside period (before start date)
	beforeDate := model.Date(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC))
	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: beforeDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.ErrorIs(err, domain.ErrEntryDateOutsidePeriod)

	// 1. Date outside period (after end date)
	afterDate := model.Date(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))
	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: afterDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.ErrorIs(err, domain.ErrEntryDateOutsidePeriod)

	// 2. Minimum 2 postings
	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
		},
	})
	s.ErrorIs(err, domain.ErrEntryMinPostings)

	// 3. Amount <= 0
	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 0},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 0},
		},
	})
	s.ErrorIs(err, domain.ErrPostingInvalidAmount)

	// 3. Non-postable account
	nonPostable, err := s.client.Account.Create().
		SetCode("1000").
		SetName("Parent Asset").
		SetType("ASSET").
		SetNormalBalance("DEBIT").
		SetIsPostable(false).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: nonPostable.ID, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.ErrorIs(err, domain.ErrPostingAccountNotPostable)

	// 3. Inactive account
	inactive, err := s.client.Account.Create().
		SetCode("1001").
		SetName("Old Account").
		SetType("ASSET").
		SetNormalBalance("DEBIT").
		SetIsActive(false).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: inactive.ID, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.ErrorIs(err, domain.ErrPostingAccountInactive)

	// 4. Currency mismatch: account restricted to USD, but transaction is IDR
	usdAcc, err := s.client.Account.Create().
		SetCode("1103").
		SetName("USD Cash").
		SetType("ASSET").
		SetNormalBalance("DEBIT").
		SetCurrency("USD").
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: usdAcc.ID, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.ErrorIs(err, domain.ErrPostingCurrencyMismatch)

	// 5. Not balanced in transaction currency
	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 200},
		},
	})
	s.ErrorIs(err, domain.ErrEntryNotBalanced)

	// 6. Not balanced in base currency
	baseDebit := 100.0
	baseCredit := 150.0
	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100, BaseAmount: &baseDebit},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100, BaseAmount: &baseCredit},
		},
	})
	s.ErrorIs(err, domain.ErrEntryBaseNotBalanced)

	// Valid creation
	entry, err := s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 500},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 500},
		},
	})
	s.Require().NoError(err)
	s.Equal("JE-2026-000001", entry.Number)
	s.Equal(domain.EntryStatusDraft, entry.Status)
	s.Require().Len(entry.Postings, 2)
}

func (s *EntryServiceSuite) Test_ValidationOrder() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	// Closed period + only 1 posting: step 1 (Period) should trigger first -> ErrPeriodClosed
	closedP, err := s.client.Period.Create().
		SetName("Closed Period").
		SetStartDate(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)).
		SetFiscalYear(2026).
		SetStatus("CLOSED").
		Save(ctx)
	s.Require().NoError(err)

	marDate := model.Date(time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC))
	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  closedP.ID,
		EntryDate: marDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
		},
	})
	s.ErrorIs(err, domain.ErrPeriodClosed)

	// Outside period + only 1 posting: step 1 (Period range) should trigger first -> ErrEntryDateOutsidePeriod
	outDate := model.Date(time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC))
	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: outDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
		},
	})
	s.ErrorIs(err, domain.ErrEntryDateOutsidePeriod)

	// Valid period + 1 posting + inactive account: step 2 (Min postings) should trigger first -> ErrEntryMinPostings
	inactive, err := s.client.Account.Create().
		SetCode("9999").
		SetName("Inactive Acc").
		SetType("ASSET").
		SetNormalBalance("DEBIT").
		SetIsActive(false).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: inactive.ID, Direction: "DEBIT", Amount: 100},
		},
	})
	s.ErrorIs(err, domain.ErrEntryMinPostings)

	// Valid period + 2 postings + amount <= 0 + unbalanced: step 3 (Amount > 0) should trigger first -> ErrPostingInvalidAmount
	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 0},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 200},
		},
	})
	s.ErrorIs(err, domain.ErrPostingInvalidAmount)

	// Valid period + 2 postings + non-postable account + unbalanced: step 3 (Postable) should trigger first -> ErrPostingAccountNotPostable
	nonPostable, err := s.client.Account.Create().
		SetCode("9998").
		SetName("Parent").
		SetType("ASSET").
		SetNormalBalance("DEBIT").
		SetIsPostable(false).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: nonPostable.ID, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 200},
		},
	})
	s.ErrorIs(err, domain.ErrPostingAccountNotPostable)

	// Valid period + 2 postings + inactive account + unbalanced: step 3 (Active) should trigger first -> ErrPostingAccountInactive
	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: inactive.ID, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 200},
		},
	})
	s.ErrorIs(err, domain.ErrPostingAccountInactive)

	// Valid period + 2 postings + currency mismatch + unbalanced: step 4 (Currency mismatch) should trigger first -> ErrPostingCurrencyMismatch
	usdAcc, err := s.client.Account.Create().
		SetCode("9997").
		SetName("USD Account").
		SetType("ASSET").
		SetNormalBalance("DEBIT").
		SetCurrency("USD").
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: usdAcc.ID, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 200},
		},
	})
	s.ErrorIs(err, domain.ErrPostingCurrencyMismatch)
}

func (s *EntryServiceSuite) Test_Update_DraftOnlyAndValidation() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	// Update draft with valid postings succeeds
	newDesc := "Updated draft entry"
	updated, err := s.svc.Update(ctx, e.ID, model.UpdateEntryRequest{
		Description: &newDesc,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 300},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 300},
		},
	})
	s.Require().NoError(err)
	s.Equal("Updated draft entry", updated.Description)
	s.Require().Len(updated.Postings, 2)
	s.Equal(300.0, updated.Postings[0].Amount)

	// Update with unbalanced postings fails
	_, err = s.svc.Update(ctx, e.ID, model.UpdateEntryRequest{
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 300},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 400},
		},
	})
	s.ErrorIs(err, domain.ErrEntryNotBalanced)

	// Post the entry
	_, err = s.svc.Post(ctx, e.ID, s.userID)
	s.Require().NoError(err)

	// Updating POSTED entry must fail with ErrEntryNotDraft
	_, err = s.svc.Update(ctx, e.ID, model.UpdateEntryRequest{
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 500},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 500},
		},
	})
	s.ErrorIs(err, domain.ErrEntryNotDraft)
}

func (s *EntryServiceSuite) Test_Delete_DraftOnly() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	// Post the entry
	_, err = s.svc.Post(ctx, e.ID, s.userID)
	s.Require().NoError(err)

	// Delete on POSTED must fail
	err = s.svc.Delete(ctx, e.ID)
	s.ErrorIs(err, domain.ErrEntryNotDraft)

	// Create another draft entry
	e2, err := s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 200},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 200},
		},
	})
	s.Require().NoError(err)

	// Delete draft succeeds
	err = s.svc.Delete(ctx, e2.ID)
	s.Require().NoError(err)

	_, err = s.svc.GetByID(ctx, e2.ID)
	s.ErrorIs(err, domain.ErrEntryNotFound)
}

func (s *EntryServiceSuite) Test_Post_Validations() {
	ctx := s.ctx()

	// Create a separate open period
	febP, err := s.client.Period.Create().
		SetName("Feb 2026").
		SetStartDate(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)).
		SetFiscalYear(2026).
		Save(ctx)
	s.Require().NoError(err)

	febDate := model.Date(time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC))
	e, err := s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  febP.ID,
		EntryDate: febDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	// Close the period after draft was created
	_, err = s.client.Period.UpdateOneID(febP.ID).SetStatus("CLOSED").Save(ctx)
	s.Require().NoError(err)

	// Posting should fail because period is now closed
	_, err = s.svc.Post(ctx, e.ID, s.userID)
	s.ErrorIs(err, domain.ErrPeriodClosed)

	// Reopen period
	_, err = s.client.Period.UpdateOneID(febP.ID).SetStatus("OPEN").Save(ctx)
	s.Require().NoError(err)

	// Post succeeds
	posted, err := s.svc.Post(ctx, e.ID, s.userID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusPosted, posted.Status)
	s.NotNil(posted.PostedAt)
	s.Equal(&s.userID, posted.PostedBy)

	// Posting again fails with ErrEntryNotDraft
	_, err = s.svc.Post(ctx, e.ID, s.userID)
	s.ErrorIs(err, domain.ErrEntryNotDraft)
}

func (s *EntryServiceSuite) Test_Reverse() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 500, Description: "Original Debit"},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 500, Description: "Original Credit"},
		},
	})
	s.Require().NoError(err)

	// Post original entry
	_, err = s.svc.Post(ctx, e.ID, s.userID)
	s.Require().NoError(err)

	// Reverse entry
	revDate := time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC)
	rev, err := s.svc.Reverse(ctx, e.ID, s.userID, revDate, "Manual Reversal")
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusPosted, rev.Status)
	s.Equal(&e.ID, rev.ReversalOfID)
	s.Equal("Manual Reversal", rev.Description)
	s.Require().Len(rev.Postings, 2)

	// Verify swapped directions
	s.Equal("CREDIT", rev.Postings[0].Direction)
	s.Equal(500.0, rev.Postings[0].Amount)
	s.Equal("DEBIT", rev.Postings[1].Direction)
	s.Equal(500.0, rev.Postings[1].Amount)

	// Reverse again fails with ErrEntryAlreadyReversed
	_, err = s.svc.Reverse(ctx, e.ID, s.userID, revDate, "Second Reversal")
	s.ErrorIs(err, domain.ErrEntryAlreadyReversed)
}

func (s *EntryServiceSuite) Test_Void_DraftEntry() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	// Voiding draft entry marks original VOID without creating reversing entry
	voided, err := s.svc.Void(ctx, e.ID, s.userID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusVoid, voided.Status)

	// No reversing entry created
	searchRes, _, err := s.svc.Search(ctx, model.SearchEntryRequest{})
	s.Require().NoError(err)
	s.Len(searchRes, 1)
}

func (s *EntryServiceSuite) Test_Void_PostedEntry() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	_, err = s.svc.Post(ctx, e.ID, s.userID)
	s.Require().NoError(err)

	// Voiding POSTED entry reverses the entry and marks original VOID
	voided, err := s.svc.Void(ctx, e.ID, s.userID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusVoid, voided.Status)

	// Reversing entry was created
	searchRes, _, err := s.svc.Search(ctx, model.SearchEntryRequest{})
	s.Require().NoError(err)
	s.Len(searchRes, 2)

	var revEntry *domain.Entry
	for _, it := range searchRes {
		if it.ID != e.ID {
			revEntry = &it
			break
		}
	}
	s.Require().NotNil(revEntry)
	s.Equal(domain.EntryStatusPosted, revEntry.Status)
	s.Equal(&e.ID, revEntry.ReversalOfID)
}

func (s *EntryServiceSuite) Test_GetAccountLedger() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	// Draft entry (should NOT appear in ledger)
	_, err := s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 999},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 999},
		},
	})
	s.Require().NoError(err)

	// Posted entry (SHOULD appear in ledger)
	e2, err := s.svc.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	_, err = s.svc.Post(ctx, e2.ID, s.userID)
	s.Require().NoError(err)

	// Query ledger for account 1
	postings, total, err := s.svc.GetAccountLedger(ctx, s.accountID1, 1, 10)
	s.Require().NoError(err)
	s.Equal(int64(1), total)
	s.Require().Len(postings, 1)
	s.Equal(100.0, postings[0].Amount)
	s.Equal("DEBIT", postings[0].Direction)

	// Query ledger for non-existent account
	_, _, err = s.svc.GetAccountLedger(ctx, 999999, 1, 10)
	s.Error(err)
}

func TestEntryServiceSuite(t *testing.T) {
	suite.Run(t, new(EntryServiceSuite))
}
