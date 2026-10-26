package repository_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/entry"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type faultKey struct{}

type EntryRepoSuite struct {
	suite.Suite
	client        *ent.Client
	repo          repository.EntryRepository
	txRepo        repository.TransactionRepository
	workspaceID   int64
	userID        int64
	accountID1    int64
	accountID2    int64
	periodID      int64
	transactionID int64
}

func (s *EntryRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:entry_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.client.Entry.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if val, ok := ctx.Value(faultKey{}).(string); ok {
				if val == "entry_update" && m.Op().Is(ent.OpUpdateOne) {
					return nil, errors.New("simulated entry update failure")
				}
				if val == "entry_create" && m.Op().Is(ent.OpCreate) {
					return nil, errors.New("simulated entry create failure")
				}
			}
			return next.Mutate(ctx, m)
		})
	})
	s.client.Posting.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if val, ok := ctx.Value(faultKey{}).(string); ok {
				if val == "posting_create" && m.Op().Is(ent.OpCreate) {
					return nil, errors.New("simulated posting create failure")
				}
			}
			return next.Mutate(ctx, m)
		})
	})
	s.repo = repository.NewEntryRepository(s.client)
	s.txRepo = repository.NewTransactionRepository(s.client)
}

func (s *EntryRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *EntryRepoSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(base.WithSkipWorkspaceFilter(context.Background()))
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
		SetName("FY2026").
		SetStartDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)).
		SetFiscalYear(2026).
		Save(wsCtx)
	s.Require().NoError(err)
	s.periodID = p.ID

	t, err := s.txRepo.Create(wsCtx, model.CreateTransactionRequest{
		TransactionDate: model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)),
		Description:     "Main TX",
	})
	s.Require().NoError(err)
	s.transactionID = t.ID
}

func (s *EntryRepoSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Alice",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func (s *EntryRepoSuite) Test_CreateAndGetByID() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:    s.periodID,
		EntryDate:   eDate,
		Description: "Entry description",
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 250000, Description: "Debit"},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 250000, Description: "Credit"},
		},
	})
	s.Require().NoError(err)
	s.Equal("JE-2026-000001", e.Number)
	s.Equal(domain.EntryStatusDraft, e.Status)
	s.Equal(s.transactionID, e.TransactionID)
	s.Equal(s.periodID, e.PeriodID)
	s.Require().Len(e.Postings, 2)

	fetched, err := s.repo.GetByID(ctx, e.ID)
	s.Require().NoError(err)
	s.Equal(e.ID, fetched.ID)
	s.Equal("JE-2026-000001", fetched.Number)
	s.Len(fetched.Postings, 2)
}

func (s *EntryRepoSuite) Test_Update_DraftOnly() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	// Update draft entry
	newDesc := "Updated entry"
	updated, err := s.repo.Update(ctx, e.ID, model.UpdateEntryRequest{
		Description: &newDesc,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 200},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 200},
		},
	})
	s.Require().NoError(err)
	s.Equal("Updated entry", updated.Description)
	s.Require().Len(updated.Postings, 2)
	s.Equal(200.0, updated.Postings[0].Amount)

	// Post the entry
	posted, err := s.repo.Post(ctx, e.ID, s.userID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusPosted, posted.Status)
	s.NotNil(posted.PostedAt)
	s.Equal(&s.userID, posted.PostedBy)

	// Update on posted entry should fail
	_, err = s.repo.Update(ctx, e.ID, model.UpdateEntryRequest{
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 300},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 300},
		},
	})
	s.Require().ErrorIs(err, domain.ErrEntryNotDraft)
}

func (s *EntryRepoSuite) Test_Delete_DraftOnly() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	// Post entry
	_, err = s.repo.Post(ctx, e.ID, s.userID)
	s.Require().NoError(err)

	// Delete should fail
	err = s.repo.Delete(ctx, e.ID)
	s.Require().ErrorIs(err, domain.ErrEntryNotDraft)

	// Reset to draft
	_, err = s.client.Entry.UpdateOneID(e.ID).SetStatus("DRAFT").Save(ctx)
	s.Require().NoError(err)

	// Delete draft
	err = s.repo.Delete(ctx, e.ID)
	s.Require().NoError(err)

	// Verify not found
	_, err = s.repo.GetByID(ctx, e.ID)
	s.Require().ErrorIs(err, domain.ErrEntryNotFound)

	// Verify postings were deleted
	count, err := s.client.Posting.Query().Where().Count(ctx)
	s.Require().NoError(err)
	s.Equal(0, count)
}

func (s *EntryRepoSuite) Test_Post_DraftOnly() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	posted, err := s.repo.Post(ctx, e.ID, s.userID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusPosted, posted.Status)

	// Post again should fail with ErrEntryNotDraft
	_, err = s.repo.Post(ctx, e.ID, s.userID)
	s.Require().ErrorIs(err, domain.ErrEntryNotDraft)
}

func (s *EntryRepoSuite) Test_Reverse() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 500, Description: "Orig Debit"},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 500, Description: "Orig Credit"},
		},
	})
	s.Require().NoError(err)

	// Post the original entry
	_, err = s.repo.Post(ctx, e.ID, s.userID)
	s.Require().NoError(err)

	revDate := time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC)
	rev, err := s.repo.Reverse(ctx, e.ID, s.userID, revDate, "Correction")
	s.Require().NoError(err)
	s.Equal("JE-2026-000002", rev.Number)
	s.Equal(domain.EntryStatusPosted, rev.Status)
	s.Equal(&e.ID, rev.ReversalOfID)
	s.Equal("Correction", rev.Description)
	s.Require().Len(rev.Postings, 2)

	// Check direction swapped
	p1 := rev.Postings[0]
	s.Equal(s.accountID1, p1.AccountID)
	s.Equal("CREDIT", p1.Direction)
	s.Equal(500.0, p1.Amount)

	p2 := rev.Postings[1]
	s.Equal(s.accountID2, p2.AccountID)
	s.Equal("DEBIT", p2.Direction)
	s.Equal(500.0, p2.Amount)

	// Reverse again should fail with ErrEntryAlreadyReversed
	_, err = s.repo.Reverse(ctx, e.ID, s.userID, revDate, "Second reversal")
	s.Require().ErrorIs(err, domain.ErrEntryAlreadyReversed)
}

func (s *EntryRepoSuite) Test_Reverse_StatusAndPeriodValidations() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	eDraft, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 200},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 200},
		},
	})
	s.Require().NoError(err)

	revDate := time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC)

	// 1. Attempt to reverse DRAFT entry fails with ErrEntryNotPosted
	_, err = s.repo.Reverse(ctx, eDraft.ID, s.userID, revDate, "Reverse draft")
	s.Require().ErrorIs(err, domain.ErrEntryNotPosted)

	// 2. Void the draft entry and attempt to reverse VOID entry -> fails with ErrEntryNotPosted
	voided, err := s.repo.Void(ctx, eDraft.ID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusVoid, voided.Status)

	_, err = s.repo.Reverse(ctx, eDraft.ID, s.userID, revDate, "Reverse voided")
	s.Require().ErrorIs(err, domain.ErrEntryNotPosted)

	// Create and post a new entry for period tests
	ePosted, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 300},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 300},
		},
	})
	s.Require().NoError(err)

	_, err = s.repo.Post(ctx, ePosted.ID, s.userID)
	s.Require().NoError(err)

	// 3. Attempt to reverse with date outside any period -> fails with ErrPeriodNotFound
	outDate := time.Date(2028, 5, 1, 0, 0, 0, 0, time.UTC)
	_, err = s.repo.Reverse(ctx, ePosted.ID, s.userID, outDate, "Reverse outside period")
	s.Require().ErrorIs(err, domain.ErrPeriodNotFound)

	// Create a closed period in 2027
	closedP, err := s.client.Period.Create().
		SetName("Jan 2027 Closed").
		SetStartDate(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC)).
		SetFiscalYear(2027).
		SetStatus("CLOSED").
		Save(ctx)
	s.Require().NoError(err)

	// 4. Attempt to reverse with date in a CLOSED period -> fails with ErrPeriodClosed
	jan27Date := time.Date(2027, 1, 10, 0, 0, 0, 0, time.UTC)
	_, err = s.repo.Reverse(ctx, ePosted.ID, s.userID, jan27Date, "Reverse in closed period")
	s.Require().ErrorIs(err, domain.ErrPeriodClosed)

	// Create a new OPEN period (Feb 2027)
	feb27P, err := s.client.Period.Create().
		SetName("Feb 2027 Open").
		SetStartDate(time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2027, 2, 28, 0, 0, 0, 0, time.UTC)).
		SetFiscalYear(2027).
		SetStatus("OPEN").
		Save(ctx)
	s.Require().NoError(err)

	// 5. Successful reversal in a different open period from original entry
	feb27Date := time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC)
	revFeb27, err := s.repo.Reverse(ctx, ePosted.ID, s.userID, feb27Date, "Reverse in Feb 2027")
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusPosted, revFeb27.Status)
	s.Equal(feb27P.ID, revFeb27.PeriodID)
	s.NotEqual(s.periodID, revFeb27.PeriodID)
	s.Equal(&ePosted.ID, revFeb27.ReversalOfID)
	_ = closedP
}

func (s *EntryRepoSuite) Test_Post_Atomic_WithBalancedPostings() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	diff := 1.0
	balanced := []model.CreatePostingRequest{
		{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
		{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		{AccountID: s.accountID2, Direction: "CREDIT", Amount: 0, BaseAmount: &diff, Description: "Rounding"},
	}

	posted, err := s.repo.Post(ctx, e.ID, s.userID, balanced)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusPosted, posted.Status)
	s.Len(posted.Postings, 3)
}

func (s *EntryRepoSuite) Test_Post_Atomic_RollbackOnFailure() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	diff := 1.0
	balanced := []model.CreatePostingRequest{
		{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
		{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		{AccountID: s.accountID2, Direction: "CREDIT", Amount: 0, BaseAmount: &diff, Description: "Rounding"},
	}

	// Inject fault on entry update (simulating failure during status update)
	failCtx := context.WithValue(ctx, faultKey{}, "entry_update")
	_, err = s.repo.Post(failCtx, e.ID, s.userID, balanced)
	s.Require().Error(err)
	s.Contains(err.Error(), "simulated entry update failure")

	// Verify original entry remains DRAFT and postings remain unchanged (2 postings, not 3)
	curr, err := s.repo.GetByID(ctx, e.ID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusDraft, curr.Status)
	s.Len(curr.Postings, 2)
}

func (s *EntryRepoSuite) Test_Post_Concurrent() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	const concurrency = 5
	var wg sync.WaitGroup
	var successCount int
	var conflictCount int
	var mu sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.repo.Post(ctx, e.ID, s.userID)
			mu.Lock()
			defer mu.Unlock()
			if err == nil {
				successCount++
			} else if errors.Is(err, domain.ErrEntryNotDraft) {
				conflictCount++
			}
		}()
	}
	wg.Wait()

	s.Equal(1, successCount)
	s.Equal(concurrency-1, conflictCount)

	curr, err := s.repo.GetByID(ctx, e.ID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusPosted, curr.Status)
}

func (s *EntryRepoSuite) Test_Void_DraftEntry() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	voided, err := s.repo.Void(ctx, e.ID, s.userID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusVoid, voided.Status)

	// No reversal entry created
	res, _, err := s.repo.Search(ctx, model.SearchEntryRequest{})
	s.Require().NoError(err)
	s.Len(res, 1)
}

func (s *EntryRepoSuite) Test_Void_PostedEntry() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	_, err = s.repo.Post(ctx, e.ID, s.userID)
	s.Require().NoError(err)

	voided, err := s.repo.Void(ctx, e.ID, s.userID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusVoid, voided.Status)

	// Reversal entry created
	res, _, err := s.repo.Search(ctx, model.SearchEntryRequest{})
	s.Require().NoError(err)
	s.Len(res, 2)

	var revEntry *domain.Entry
	for _, it := range res {
		if it.ID != e.ID {
			revEntry = &it
			break
		}
	}
	s.Require().NotNil(revEntry)
	s.Equal(domain.EntryStatusPosted, revEntry.Status)
	s.Equal(&e.ID, revEntry.ReversalOfID)
}

func (s *EntryRepoSuite) Test_Void_AlreadyVoid_Idempotent() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	_, err = s.repo.Post(ctx, e.ID, s.userID)
	s.Require().NoError(err)

	// Void once
	voided1, err := s.repo.Void(ctx, e.ID, s.userID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusVoid, voided1.Status)

	// Void again -> idempotent, no additional reversal entry
	voided2, err := s.repo.Void(ctx, e.ID, s.userID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusVoid, voided2.Status)

	res, _, err := s.repo.Search(ctx, model.SearchEntryRequest{})
	s.Require().NoError(err)
	s.Len(res, 2) // exactly 1 original + 1 reversal
}

func (s *EntryRepoSuite) Test_Void_Atomic_RollbackOnPostingFailure() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	_, err = s.repo.Post(ctx, e.ID, s.userID)
	s.Require().NoError(err)

	// Inject fault when creating reversal postings
	failCtx := context.WithValue(ctx, faultKey{}, "posting_create")
	_, err = s.repo.Void(failCtx, e.ID, s.userID)
	s.Require().Error(err)
	s.Contains(err.Error(), "simulated posting create failure")

	// Verify original entry is STILL POSTED and no reversal entry exists
	curr, err := s.repo.GetByID(ctx, e.ID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusPosted, curr.Status)

	count, err := s.client.Entry.Query().Where(entry.ReversalOfIDEQ(e.ID)).Count(ctx)
	s.Require().NoError(err)
	s.Equal(0, count)

	totalEntries, err := s.client.Entry.Query().Count(ctx)
	s.Require().NoError(err)
	s.Equal(1, totalEntries)
}

func (s *EntryRepoSuite) Test_Void_Atomic_RollbackOnEntryUpdateFailure() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	_, err = s.repo.Post(ctx, e.ID, s.userID)
	s.Require().NoError(err)

	// Inject fault when updating original entry status to VOID
	failCtx := context.WithValue(ctx, faultKey{}, "entry_update")
	_, err = s.repo.Void(failCtx, e.ID, s.userID)
	s.Require().Error(err)
	s.Contains(err.Error(), "simulated entry update failure")

	// Verify original entry is STILL POSTED and no orphan reversal entry was committed
	curr, err := s.repo.GetByID(ctx, e.ID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusPosted, curr.Status)

	count, err := s.client.Entry.Query().Where(entry.ReversalOfIDEQ(e.ID)).Count(ctx)
	s.Require().NoError(err)
	s.Equal(0, count)

	totalEntries, err := s.client.Entry.Query().Count(ctx)
	s.Require().NoError(err)
	s.Equal(1, totalEntries)
}

func (s *EntryRepoSuite) Test_Void_Atomic_RollbackOnClosedPeriod() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	_, err = s.repo.Post(ctx, e.ID, s.userID)
	s.Require().NoError(err)

	// Close the period
	err = s.client.Period.UpdateOneID(s.periodID).SetStatus("CLOSED").Exec(ctx)
	s.Require().NoError(err)

	// Voiding should fail with ErrPeriodClosed
	_, err = s.repo.Void(ctx, e.ID, s.userID)
	s.Require().ErrorIs(err, domain.ErrPeriodClosed)

	// Original entry remains POSTED and no reversal was created
	curr, err := s.repo.GetByID(ctx, e.ID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusPosted, curr.Status)

	count, err := s.client.Entry.Query().Where(entry.ReversalOfIDEQ(e.ID)).Count(ctx)
	s.Require().NoError(err)
	s.Equal(0, count)
}

func (s *EntryRepoSuite) Test_Void_Concurrent() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	_, err = s.repo.Post(ctx, e.ID, s.userID)
	s.Require().NoError(err)

	const concurrency = 5
	var wg sync.WaitGroup
	var errCount int
	var mu sync.Mutex

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.repo.Void(ctx, e.ID, s.userID)
			if err != nil {
				mu.Lock()
				errCount++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	s.Equal(0, errCount)

	curr, err := s.repo.GetByID(ctx, e.ID)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusVoid, curr.Status)

	// Exactly 1 reversal entry created despite 5 concurrent calls
	count, err := s.client.Entry.Query().Where(entry.ReversalOfIDEQ(e.ID)).Count(ctx)
	s.Require().NoError(err)
	s.Equal(1, count)
}

func (s *EntryRepoSuite) Test_Search() {
	ctx := s.ctx()
	d1 := model.Date(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	d2 := model.Date(time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC))

	e1, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: d1,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
		},
	})
	s.Require().NoError(err)

	e2, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: d2,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 200},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 200},
		},
	})
	s.Require().NoError(err)

	_, err = s.repo.Post(ctx, e1.ID, s.userID)
	s.Require().NoError(err)

	// Search all
	all, _, err := s.repo.Search(ctx, model.SearchEntryRequest{})
	s.Require().NoError(err)
	s.Len(all, 2)

	// Filter by status POSTED
	postedStatus := domain.EntryStatusPosted
	postedItems, _, err := s.repo.Search(ctx, model.SearchEntryRequest{
		Status: &postedStatus,
	})
	s.Require().NoError(err)
	s.Len(postedItems, 1)
	s.Equal(e1.ID, postedItems[0].ID)

	// Filter by account
	accItems, _, err := s.repo.Search(ctx, model.SearchEntryRequest{
		AccountID: &s.accountID1,
	})
	s.Require().NoError(err)
	s.Len(accItems, 2)

	// Filter by TransactionID
	txItems, _, err := s.repo.Search(ctx, model.SearchEntryRequest{
		TransactionID: &s.transactionID,
	})
	s.Require().NoError(err)
	s.Len(txItems, 2)

	nonExistentTx := int64(999999)
	emptyTxItems, _, err := s.repo.Search(ctx, model.SearchEntryRequest{
		TransactionID: &nonExistentTx,
	})
	s.Require().NoError(err)
	s.Empty(emptyTxItems)

	// Pagination with Limit and Cursor
	limitOne := 1
	page1, cursor1, err := s.repo.Search(ctx, model.SearchEntryRequest{
		Limit: &limitOne,
	})
	s.Require().NoError(err)
	s.Len(page1, 1)
	s.NotEmpty(cursor1)
	s.Equal(e2.ID, page1[0].ID)

	page2, cursor2, err := s.repo.Search(ctx, model.SearchEntryRequest{
		Limit:  &limitOne,
		Cursor: &cursor1,
	})
	s.Require().NoError(err)
	s.Len(page2, 1)
	s.Empty(cursor2)
	s.Equal(e1.ID, page2[0].ID)

	// Invalid cursor
	invalidCursor := "invalid-cursor"
	_, _, err = s.repo.Search(ctx, model.SearchEntryRequest{
		Cursor: &invalidCursor,
	})
	s.Require().ErrorIs(err, base.ErrInvalidCursor)

	// Delete draft e2
	err = s.repo.Delete(ctx, e2.ID)
	s.Require().NoError(err)

	allAfterDelete, _, err := s.repo.Search(ctx, model.SearchEntryRequest{})
	s.Require().NoError(err)
	s.Len(allAfterDelete, 1)
}

func (s *EntryRepoSuite) Test_GetAccountLedger() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	// Create draft entry (should NOT appear in ledger until posted)
	e1, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "DEBIT", Amount: 1000},
			{AccountID: s.accountID2, Direction: "CREDIT", Amount: 1000},
		},
	})
	s.Require().NoError(err)

	// Ledger before posting should be empty
	postings, total, err := s.repo.GetAccountLedger(ctx, s.accountID1, 1, 10)
	s.Require().NoError(err)
	s.Equal(int64(0), total)
	s.Empty(postings)

	// Post entry
	_, err = s.repo.Post(ctx, e1.ID, s.userID)
	s.Require().NoError(err)

	// Ledger after posting should have 1 item
	postings, total, err = s.repo.GetAccountLedger(ctx, s.accountID1, 1, 10)
	s.Require().NoError(err)
	s.Equal(int64(1), total)
	s.Require().Len(postings, 1)
	s.Equal(1000.0, postings[0].Amount)
	s.Equal("DEBIT", postings[0].Direction)

	// Create second entry and post it
	e2, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
		PeriodID:  s.periodID,
		EntryDate: eDate,
		Postings: []model.CreatePostingRequest{
			{AccountID: s.accountID1, Direction: "CREDIT", Amount: 300},
			{AccountID: s.accountID2, Direction: "DEBIT", Amount: 300},
		},
	})
	s.Require().NoError(err)
	_, err = s.repo.Post(ctx, e2.ID, s.userID)
	s.Require().NoError(err)

	// Check pagination: page 1 limit 1
	p1, total, err := s.repo.GetAccountLedger(ctx, s.accountID1, 1, 1)
	s.Require().NoError(err)
	s.Equal(int64(2), total)
	s.Len(p1, 1)
	s.Equal(1000.0, p1[0].Amount)

	// Page 2 limit 1
	p2, total, err := s.repo.GetAccountLedger(ctx, s.accountID1, 2, 1)
	s.Require().NoError(err)
	s.Equal(int64(2), total)
	s.Len(p2, 1)
	s.Equal(300.0, p2[0].Amount)
}

func (s *EntryRepoSuite) Test_ConcurrentNumbering() {
	ctx := s.ctx()
	eDate := model.Date(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))

	concurrency := 5
	var wg sync.WaitGroup
	errCh := make(chan error, concurrency)
	numbers := make(chan string, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			e, err := s.repo.Create(ctx, s.transactionID, model.CreateEntryRequest{
				PeriodID:    s.periodID,
				EntryDate:   eDate,
				Description: fmt.Sprintf("Concurrent %d", idx),
				Postings: []model.CreatePostingRequest{
					{AccountID: s.accountID1, Direction: "DEBIT", Amount: 100},
					{AccountID: s.accountID2, Direction: "CREDIT", Amount: 100},
				},
			})
			if err != nil {
				errCh <- err
				return
			}
			numbers <- e.Number
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
		s.False(seen[num], "duplicate entry number: %s", num)
		seen[num] = true
	}
	s.Len(seen, concurrency)
}

func (s *EntryRepoSuite) Test_CreateSystemPostedEntry() {
	ctx := s.ctx()
	eDate := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)

	postings := []model.CreatePostingRequest{
		{AccountID: s.accountID1, Direction: "DEBIT", Amount: 500},
		{AccountID: s.accountID2, Direction: "CREDIT", Amount: 500},
	}

	entry, err := s.repo.CreateSystemPostedEntry(
		ctx,
		s.periodID,
		eDate,
		"System test trx",
		"System test entry",
		"TEST-REF-001",
		postings,
		s.userID,
	)
	s.Equal(domain.EntryStatusPosted, entry.Status)
	s.Equal("System test entry", entry.Description)
	s.Require().NotNil(entry.PostedBy)
	s.Equal(s.userID, *entry.PostedBy)
	s.Len(entry.Postings, 2)

	// Check balances as of 2026-01-31
	bals, err := s.repo.GetAccountBalancesAsOf(ctx, eDate)
	s.Require().NoError(err)
	s.Equal(500.0, bals[s.accountID1].Debit)
	s.Equal(500.0, bals[s.accountID2].Credit)
}

func TestEntryRepoSuite(t *testing.T) {
	suite.Run(t, new(EntryRepoSuite))
}

