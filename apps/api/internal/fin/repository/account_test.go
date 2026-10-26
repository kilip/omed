package repository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type AccountRepoSuite struct {
	suite.Suite
	client      *ent.Client
	repo        repository.AccountRepository
	workspaceID int64
	userID      int64
}

func (s *AccountRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:account_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.repo = repository.NewAccountRepository(s.client)
}

func (s *AccountRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *AccountRepoSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(base.WithSkipWorkspaceFilter(context.Background()))
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
}

func (s *AccountRepoSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Alice",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func ptr[T any](v T) *T {
	return &v
}

func (s *AccountRepoSuite) Test_CreateAndGetByID() {
	ctx := s.ctx()

	desc := "Cash account"
	acc, err := s.repo.Create(ctx, model.CreateAccountRequest{
		Code:        "1000",
		Name:        "Cash",
		Type:        domain.AccountTypeAsset,
		Description: desc,
		IsActive:    ptr(true),
		IsPostable:  ptr(true),
	}, domain.NormalBalanceDebit)
	s.Require().NoError(err)
	s.Equal("1000", acc.Code)
	s.Equal("Cash", acc.Name)
	s.Equal(domain.AccountTypeAsset, acc.Type)
	s.Equal(domain.NormalBalanceDebit, acc.NormalBalance)
	s.True(acc.IsActive)
	s.True(acc.IsPostable)
	s.Nil(acc.AuditMeta.DeletedAt)

	// Fetch via GetByID
	fetched, err := s.repo.GetByID(ctx, acc.ID)
	s.Require().NoError(err)
	s.Equal(acc.ID, fetched.ID)
}

func (s *AccountRepoSuite) Test_Search() {
	ctx := s.ctx()

	for i := 1; i <= 5; i++ {
		code := fmt.Sprintf("100%d", i)
		name := fmt.Sprintf("Account %d", i)
		_, err := s.repo.Create(ctx, model.CreateAccountRequest{
			Code: code,
			Name: name,
			Type: domain.AccountTypeAsset,
		}, domain.NormalBalanceDebit)
		s.Require().NoError(err)
	}

	items, _, err := s.repo.Search(ctx, model.SearchAccountRequest{})
	s.Require().NoError(err)
	s.Len(items, 5)
}

func (s *AccountRepoSuite) Test_Update() {
	ctx := s.ctx()

	acc, err := s.repo.Create(ctx, model.CreateAccountRequest{
		Code:       "2000",
		Name:       "Accounts Payable",
		Type:       domain.AccountTypeLiability,
		IsActive:   ptr(true),
		IsPostable: ptr(true),
	}, domain.NormalBalanceCredit)
	s.Require().NoError(err)

	updated, err := s.repo.Update(ctx, acc.ID, model.UpdateAccountRequest{
		Name:       "AP Trade",
		IsPostable: ptr(false),
	})
	s.Require().NoError(err)
	s.Equal("AP Trade", updated.Name)
	s.False(updated.IsPostable)
}

func (s *AccountRepoSuite) Test_SoftDeleteAndCodeReuse() {
	ctx := s.ctx()

	acc, err := s.repo.Create(ctx, model.CreateAccountRequest{
		Code:       "3000",
		Name:       "Retained Earnings",
		Type:       domain.AccountTypeEquity,
		IsActive:   ptr(true),
		IsPostable: ptr(true),
	}, domain.NormalBalanceCredit)
	s.Require().NoError(err)

	// Soft delete
	err = s.repo.Delete(ctx, acc.ID)
	s.Require().NoError(err)

	// GetByID should return ErrNotFound
	_, err = s.repo.GetByID(ctx, acc.ID)
	s.Equal(base.ErrNotFound, err)

	// Search should not return soft deleted accounts by default
	items, _, err := s.repo.Search(ctx, model.SearchAccountRequest{})
	s.Require().NoError(err)
	s.Empty(items)

	// Re-creating an account with the same code succeeds because the old one is soft-deleted
	newAcc, err := s.repo.Create(ctx, model.CreateAccountRequest{
		Code:       "3000",
		Name:       "New Retained Earnings",
		Type:       domain.AccountTypeEquity,
		IsActive:   ptr(true),
		IsPostable: ptr(true),
	}, domain.NormalBalanceCredit)
	s.Require().NoError(err)
	s.Equal("3000", newAcc.Code)
}

func (s *AccountRepoSuite) Test_HasActiveChildren() {
	ctx := s.ctx()

	root, err := s.repo.Create(ctx, model.CreateAccountRequest{
		Code:       "1000",
		Name:       "Assets Header",
		Type:       domain.AccountTypeAsset,
		IsActive:   ptr(true),
		IsPostable: ptr(false),
	}, domain.NormalBalanceDebit)
	s.Require().NoError(err)

	child, err := s.repo.Create(ctx, model.CreateAccountRequest{
		Code:       "1100",
		Name:       "Current Assets",
		Type:       domain.AccountTypeAsset,
		ParentID:   root.ID,
		IsActive:   ptr(true),
		IsPostable: ptr(true),
	}, domain.NormalBalanceDebit)
	s.Require().NoError(err)
	s.NotZero(child.ID)

	hasChildren, err := s.repo.HasActiveChildren(ctx, root.ID)
	s.Require().NoError(err)
	s.True(hasChildren)
}

func TestAccountRepoSuite(t *testing.T) {
	suite.Run(t, new(AccountRepoSuite))
}
