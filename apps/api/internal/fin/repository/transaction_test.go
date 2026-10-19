package repository_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type TransactionRepoSuite struct {
	suite.Suite
	client      *ent.Client
	repo        repository.TransactionRepository
	workspaceID int64
	userID      int64
	accountID1  int64
	accountID2  int64
	periodID    int64
}

func (s *TransactionRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:transaction_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.repo = repository.NewTransactionRepository(s.client)
}

func (s *TransactionRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *TransactionRepoSuite) SetupTest() {
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

	// Create test accounts
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

	// Create test period
	p, err := s.client.Period.Create().
		SetName("FY2026").
		SetStartDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)).
		SetFiscalYear(2026).
		Save(wsCtx)
	s.Require().NoError(err)
	s.periodID = p.ID
}

func (s *TransactionRepoSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Alice",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func (s *TransactionRepoSuite) Test_CreateAndGetByID() {
	ctx := s.ctx()

	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	rate := 1.0

	t, err := s.repo.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Initial investment",
		Reference:       "REF-001",
		Currency:        "IDR",
		ExchangeRate:    &rate,
	})
	s.Require().NoError(err)
	s.Equal("TRX-2026-000001", t.Number)
	s.Equal("Initial investment", t.Description)
	s.Equal("REF-001", t.Reference)
	s.Equal("IDR", t.Currency)
	s.Equal(1.0, t.ExchangeRate)
	s.Equal(domain.TransactionSourceManual, t.Source)
	s.Empty(t.Entries)

	// Fetch by ID
	fetched, err := s.repo.GetByID(ctx, t.ID)
	s.Require().NoError(err)
	s.Equal(t.ID, fetched.ID)
	s.Equal(t.Number, fetched.Number)
	s.Equal("Alice", fetched.AuditMeta.CreatedBy.Name)
}

func (s *TransactionRepoSuite) Test_CreateWithInlineEntryAndPostings() {
	ctx := s.ctx()

	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	rate := 1.0

	t, err := s.repo.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Invoice payment",
		Reference:       "INV-001",
		Currency:        "IDR",
		ExchangeRate:    &rate,
		Entry: &model.CreateEntryInlineRequest{
			PeriodID:    s.periodID,
			EntryDate:   tDate,
			Description: "Invoice payment entry",
			Postings: []model.CreatePostingRequest{
				{
					AccountID:   s.accountID1,
					Direction:   "DEBIT",
					Amount:      500000,
					Description: "Debit cash",
				},
				{
					AccountID:   s.accountID2,
					Direction:   "CREDIT",
					Amount:      500000,
					Description: "Credit revenue",
				},
			},
		},
	})
	s.Require().NoError(err)
	s.Equal("TRX-2026-000001", t.Number)
	s.Require().Len(t.Entries, 1)

	e := t.Entries[0]
	s.Equal("JE-2026-000001", e.Number)
	s.Equal(domain.EntryStatusDraft, e.Status)
	s.Equal(s.periodID, e.PeriodID)
	s.Require().Len(e.Postings, 2)

	p1 := e.Postings[0]
	s.Equal(s.accountID1, p1.AccountID)
	s.Equal("DEBIT", p1.Direction)
	s.Equal(500000.0, p1.Amount)
	s.Equal(500000.0, p1.BaseAmount)
	s.Equal(0, p1.Position)

	p2 := e.Postings[1]
	s.Equal(s.accountID2, p2.AccountID)
	s.Equal("CREDIT", p2.Direction)
	s.Equal(500000.0, p2.Amount)
	s.Equal(500000.0, p2.BaseAmount)
	s.Equal(1, p2.Position)
}

func (s *TransactionRepoSuite) Test_NumberingSequential() {
	ctx := s.ctx()

	tDate := model.Date(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))

	t1, err := s.repo.Create(ctx, model.CreateTransactionRequest{TransactionDate: tDate})
	s.Require().NoError(err)
	s.Equal("TRX-2026-000001", t1.Number)

	t2, err := s.repo.Create(ctx, model.CreateTransactionRequest{TransactionDate: tDate})
	s.Require().NoError(err)
	s.Equal("TRX-2026-000002", t2.Number)

	// Different year
	tDate2027 := model.Date(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
	t3, err := s.repo.Create(ctx, model.CreateTransactionRequest{TransactionDate: tDate2027})
	s.Require().NoError(err)
	s.Equal("TRX-2027-000001", t3.Number)
}

func (s *TransactionRepoSuite) Test_Update_Success() {
	ctx := s.ctx()

	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	t, err := s.repo.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Original desc",
		Reference:       "REF-1",
	})
	s.Require().NoError(err)

	newDesc := "Updated desc"
	newRef := "REF-2"
	updated, err := s.repo.Update(ctx, t.ID, model.UpdateTransactionRequest{
		Description: &newDesc,
		Reference:   &newRef,
	})
	s.Require().NoError(err)
	s.Equal("Updated desc", updated.Description)
	s.Equal("REF-2", updated.Reference)
}

func (s *TransactionRepoSuite) Test_Update_BlockedIfPostedEntries() {
	ctx := s.ctx()

	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	t, err := s.repo.Create(ctx, model.CreateTransactionRequest{
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
	s.Require().Len(t.Entries, 1)

	// Post the entry directly via client
	now := time.Now()
	_, err = s.client.Entry.UpdateOneID(t.Entries[0].ID).
		SetStatus("POSTED").
		SetPostedAt(now).
		SetPostedBy(s.userID).
		Save(ctx)
	s.Require().NoError(err)

	newDesc := "Cannot update"
	_, err = s.repo.Update(ctx, t.ID, model.UpdateTransactionRequest{
		Description: &newDesc,
	})
	s.Require().ErrorIs(err, domain.ErrTransactionHasPostedEntries)
}

func (s *TransactionRepoSuite) Test_Delete_SuccessAndBlocked() {
	ctx := s.ctx()

	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	t, err := s.repo.Create(ctx, model.CreateTransactionRequest{
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

	// Blocked if posted
	now := time.Now()
	_, err = s.client.Entry.UpdateOneID(t.Entries[0].ID).
		SetStatus("POSTED").
		SetPostedAt(now).
		SetPostedBy(s.userID).
		Save(ctx)
	s.Require().NoError(err)

	err = s.repo.Delete(ctx, t.ID)
	s.Require().ErrorIs(err, domain.ErrTransactionHasPostedEntries)

	// Reset entry to DRAFT
	_, err = s.client.Entry.UpdateOneID(t.Entries[0].ID).
		SetStatus("DRAFT").
		Save(ctx)
	s.Require().NoError(err)

	// Can delete when draft
	err = s.repo.Delete(ctx, t.ID)
	s.Require().NoError(err)

	// GetByID should return ErrTransactionNotFound
	_, err = s.repo.GetByID(ctx, t.ID)
	s.Require().ErrorIs(err, domain.ErrTransactionNotFound)
}

func (s *TransactionRepoSuite) Test_Search() {
	ctx := s.ctx()

	d1 := model.Date(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	d2 := model.Date(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))
	d3 := model.Date(time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC))

	ref1 := "REF-A"
	ref2 := "REF-B"

	_, err := s.repo.Create(ctx, model.CreateTransactionRequest{TransactionDate: d1, Reference: ref1})
	s.Require().NoError(err)
	_, err = s.repo.Create(ctx, model.CreateTransactionRequest{TransactionDate: d2, Reference: ref2})
	s.Require().NoError(err)
	t3, err := s.repo.Create(ctx, model.CreateTransactionRequest{TransactionDate: d3, Reference: ref1})
	s.Require().NoError(err)

	// Search all
	all, _, err := s.repo.Search(ctx, model.SearchTransactionRequest{})
	s.Require().NoError(err)
	s.Len(all, 3)

	// Filter by Date Range (Jan only)
	janEnd := model.Date(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC))
	janItems, _, err := s.repo.Search(ctx, model.SearchTransactionRequest{
		DateFrom: &d1,
		DateTo:   &janEnd,
	})
	s.Require().NoError(err)
	s.Len(janItems, 2)

	// Filter by Reference
	refItems, _, err := s.repo.Search(ctx, model.SearchTransactionRequest{
		Reference: &ref1,
	})
	s.Require().NoError(err)
	s.Len(refItems, 2)

	// Filter by Query (matches number or description)
	tQueryDesc, err := s.repo.Create(ctx, model.CreateTransactionRequest{
		TransactionDate: d1,
		Description:     "Special office supplies",
	})
	s.Require().NoError(err)

	qDesc := "office supplies"
	foundByDesc, _, err := s.repo.Search(ctx, model.SearchTransactionRequest{
		Query: &qDesc,
	})
	s.Require().NoError(err)
	s.Len(foundByDesc, 1)
	s.Equal(tQueryDesc.ID, foundByDesc[0].ID)

	qNum := tQueryDesc.Number
	foundByNum, _, err := s.repo.Search(ctx, model.SearchTransactionRequest{
		Query: &qNum,
	})
	s.Require().NoError(err)
	s.Len(foundByNum, 1)
	s.Equal(tQueryDesc.ID, foundByNum[0].ID)

	// Pagination with Limit and Cursor
	limitTwo := 2
	page1, cursor1, err := s.repo.Search(ctx, model.SearchTransactionRequest{
		Limit: &limitTwo,
	})
	s.Require().NoError(err)
	s.Len(page1, 2)
	s.NotEmpty(cursor1)

	page2, cursor2, err := s.repo.Search(ctx, model.SearchTransactionRequest{
		Limit:  &limitTwo,
		Cursor: &cursor1,
	})
	s.Require().NoError(err)
	s.Len(page2, 2)
	s.Empty(cursor2)

	// Invalid cursor
	invalidCursor := "not-a-valid-cursor"
	_, _, err = s.repo.Search(ctx, model.SearchTransactionRequest{
		Cursor: &invalidCursor,
	})
	s.Require().ErrorIs(err, base.ErrInvalidCursor)

	// Soft delete t3 and verify search excludes it
	err = s.repo.Delete(ctx, t3.ID)
	s.Require().NoError(err)

	allAfterDelete, _, err := s.repo.Search(ctx, model.SearchTransactionRequest{})
	s.Require().NoError(err)
	s.Len(allAfterDelete, 3)
}

func (s *TransactionRepoSuite) Test_ConcurrentNumbering() {
	ctx := s.ctx()
	tDate := model.Date(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))

	concurrency := 5
	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)
	numbers := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			t, err := s.repo.Create(ctx, model.CreateTransactionRequest{
				TransactionDate: tDate,
				Description:     fmt.Sprintf("Concurrent %d", idx),
			})
			if err != nil {
				errCh <- err
				return
			}
			numbers <- t.Number
		}(i)
	}

	wg.Wait()
	close(errCh)
	close(numbers)

	for err := range errCh {
		s.Require().NoError(err)
	}

	seen := make(map[string]bool)
	for num := range numbers {
		s.False(seen[num], "duplicate transaction number: %s", num)
		seen[num] = true
	}
	s.Len(seen, concurrency)
}

func TestTransactionRepoSuite(t *testing.T) {
	suite.Run(t, new(TransactionRepoSuite))
}
