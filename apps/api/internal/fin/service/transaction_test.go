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

type TransactionServiceSuite struct {
	suite.Suite
	client      *ent.Client
	svc         service.TransactionService
	entryRepo   repository.EntryRepository
	workspaceID int64
	userID      int64
	accountID1  int64
	accountID2  int64
	periodID    int64
}

func (s *TransactionServiceSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:tx_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	txRepo := repository.NewTransactionRepository(s.client)
	periodRepo := repository.NewPeriodRepository(s.client)
	accountRepo := repository.NewAccountRepository(s.client)
	s.svc = service.NewTransactionService(txRepo, periodRepo, accountRepo)
	s.entryRepo = repository.NewEntryRepository(s.client)
}

func (s *TransactionServiceSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *TransactionServiceSuite) SetupTest() {
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
}

func (s *TransactionServiceSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Alice",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func (s *TransactionServiceSuite) Test_Create_WithoutEntry() {
	ctx := s.ctx()
	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	tx, err := s.svc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Header only transaction",
		Reference:       "INV-001",
		Currency:        "IDR",
	})
	s.Require().NoError(err)
	s.Equal("TRX-2026-000001", tx.Number)
	s.Equal("Header only transaction", tx.Description)
	s.Equal("INV-001", tx.Reference)
	s.Equal("IDR", tx.Currency)
	s.Equal(1.0, tx.ExchangeRate)
	s.Empty(tx.Entries)
}

func (s *TransactionServiceSuite) Test_Create_WithValidInlineEntry() {
	ctx := s.ctx()
	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	rate := 1.0
	tx, err := s.svc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Transaction with inline entry",
		Reference:       "INV-002",
		Currency:        "IDR",
		ExchangeRate:    &rate,
		Entry: &model.CreateEntryInlineRequest{
			PeriodID:  s.periodID,
			EntryDate: tDate,
			Postings: []model.CreatePostingRequest{
				{AccountID: s.accountID1, Direction: "DEBIT", Amount: 1000},
				{AccountID: s.accountID2, Direction: "CREDIT", Amount: 1000},
			},
		},
	})
	s.Require().NoError(err)
	s.Equal("TRX-2026-000001", tx.Number)
	s.Require().Len(tx.Entries, 1)

	entry := tx.Entries[0]
	s.Equal("JE-2026-000001", entry.Number)
	s.Equal(domain.EntryStatusDraft, entry.Status)
	s.Require().Len(entry.Postings, 2)
}

func (s *TransactionServiceSuite) Test_Create_WithInvalidInlineEntry_Validations() {
	ctx := s.ctx()
	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	// Outside period
	outDate := model.Date(time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC))
	_, err := s.svc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Entry: &model.CreateEntryInlineRequest{
			PeriodID:  s.periodID,
			EntryDate: outDate,
			Postings: []model.CreatePostingRequest{
				{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
				{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
			},
		},
	})
	s.ErrorIs(err, domain.ErrEntryDateOutsidePeriod)

	// Unbalanced
	_, err = s.svc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Entry: &model.CreateEntryInlineRequest{
			PeriodID:  s.periodID,
			EntryDate: tDate,
			Postings: []model.CreatePostingRequest{
				{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
				{AccountID: s.accountID2, Direction: "CREDIT", Amount: 200},
			},
		},
	})
	s.ErrorIs(err, domain.ErrEntryNotBalanced)

	// Non-postable account
	nonPostable, err := s.client.Account.Create().
		SetCode("1000").
		SetName("Current Assets").
		SetType("ASSET").
		SetNormalBalance("DEBIT").
		SetIsPostable(false).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.svc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Entry: &model.CreateEntryInlineRequest{
			PeriodID:  s.periodID,
			EntryDate: tDate,
			Postings: []model.CreatePostingRequest{
				{AccountID: nonPostable.ID, Direction: "DEBIT", Amount: 100},
				{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
			},
		},
	})
	s.ErrorIs(err, domain.ErrPostingAccountNotPostable)

	// Inactive account
	inactive, err := s.client.Account.Create().
		SetCode("1102").
		SetName("Old Bank").
		SetType("ASSET").
		SetNormalBalance("DEBIT").
		SetIsActive(false).
		Save(ctx)
	s.Require().NoError(err)

	_, err = s.svc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Entry: &model.CreateEntryInlineRequest{
			PeriodID:  s.periodID,
			EntryDate: tDate,
			Postings: []model.CreatePostingRequest{
				{AccountID: inactive.ID, Direction: "DEBIT", Amount: 100},
				{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
			},
		},
	})
	s.ErrorIs(err, domain.ErrPostingAccountInactive)
}

func (s *TransactionServiceSuite) Test_GetByID() {
	ctx := s.ctx()
	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	created, err := s.svc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Test tx",
	})
	s.Require().NoError(err)

	fetched, err := s.svc.GetByID(ctx, created.ID)
	s.Require().NoError(err)
	s.Equal(created.ID, fetched.ID)
	s.Equal(created.Number, fetched.Number)

	_, err = s.svc.GetByID(ctx, 999999)
	s.ErrorIs(err, domain.ErrTransactionNotFound)
}

func (s *TransactionServiceSuite) Test_Update() {
	ctx := s.ctx()
	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	tx, err := s.svc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Initial Description",
		Reference:       "REF-1",
		Entry: &model.CreateEntryInlineRequest{
			PeriodID:  s.periodID,
			EntryDate: tDate,
			Postings: []model.CreatePostingRequest{
				{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
				{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
			},
		},
	})
	s.Require().NoError(err)

	newDesc := "Updated Description"
	newRef := "REF-2"
	updated, err := s.svc.Update(ctx, tx.ID, model.UpdateTransactionRequest{
		Description: &newDesc,
		Reference:   &newRef,
	})
	s.Require().NoError(err)
	s.Equal("Updated Description", updated.Description)
	s.Equal("REF-2", updated.Reference)

	// Post the entry
	_, err = s.entryRepo.Post(ctx, tx.Entries[0].ID, s.userID)
	s.Require().NoError(err)

	// Update should now be rejected because transaction has posted entries
	_, err = s.svc.Update(ctx, tx.ID, model.UpdateTransactionRequest{
		Description: &newDesc,
	})
	s.ErrorIs(err, domain.ErrTransactionHasPostedEntries)
}

func (s *TransactionServiceSuite) Test_Delete() {
	ctx := s.ctx()
	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	tx, err := s.svc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Entry: &model.CreateEntryInlineRequest{
			PeriodID:  s.periodID,
			EntryDate: tDate,
			Postings: []model.CreatePostingRequest{
				{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
				{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
			},
		},
	})
	s.Require().NoError(err)

	// Post the entry
	_, err = s.entryRepo.Post(ctx, tx.Entries[0].ID, s.userID)
	s.Require().NoError(err)

	// Delete must fail when transaction has posted entries
	err = s.svc.Delete(ctx, tx.ID)
	s.ErrorIs(err, domain.ErrTransactionHasPostedEntries)

	// Create another transaction with draft entry
	tx2, err := s.svc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Draft tx",
	})
	s.Require().NoError(err)

	// Delete draft transaction succeeds
	err = s.svc.Delete(ctx, tx2.ID)
	s.Require().NoError(err)

	_, err = s.svc.GetByID(ctx, tx2.ID)
	s.ErrorIs(err, domain.ErrTransactionNotFound)
}

func (s *TransactionServiceSuite) Test_Search() {
	ctx := s.ctx()
	d1 := model.Date(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	d2 := model.Date(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))

	ref1 := "INV-101"
	ref2 := "INV-102"
	_, err := s.svc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: d1,
		Reference:       ref1,
	})
	s.Require().NoError(err)

	_, err = s.svc.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: d2,
		Reference:       ref2,
	})
	s.Require().NoError(err)

	all, _, err := s.svc.Search(ctx, model.SearchTransactionRequest{})
	s.Require().NoError(err)
	s.Len(all, 2)

	filtered, _, err := s.svc.Search(ctx, model.SearchTransactionRequest{
		Reference: &ref1,
	})
	s.Require().NoError(err)
	s.Require().Len(filtered, 1)
	s.Equal(ref1, filtered[0].Reference)
}

func TestTransactionServiceSuite(t *testing.T) {
	suite.Run(t, new(TransactionServiceSuite))
}
