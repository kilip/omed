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
	ctx := context.Background()
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

	w, err := s.client.Workspace.Create().
		SetName("Finance WS").
		SetOwnerID(u.ID).
		SetCreatedByID(u.ID).
		SetUpdatedByID(u.ID).
		Save(ctx)
	s.Require().NoError(err)
	s.workspaceID = w.ID
}

func (s *AccountRepoSuite) ctx() context.Context {
	return context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Alice",
	})
}

func (s *AccountRepoSuite) Test_CreateAndGetByID() {
	ctx := s.ctx()

	desc := "Cash account"
	isPostable := true
	acc, err := s.repo.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code:        "1000",
		Name:        "Cash",
		Type:        domain.AccountTypeAsset,
		Description: desc,
		IsPostable:  &isPostable,
	}, domain.NormalBalanceDebit)
	s.Require().NoError(err)
	s.Equal("1000", acc.Code)
	s.Equal("Cash", acc.Name)
	s.Equal(domain.AccountTypeAsset, acc.Type)
	s.Equal(domain.NormalBalanceDebit, acc.NormalBalance)
	s.True(acc.IsActive)
	s.True(acc.IsPostable)
	s.Nil(acc.DeletedAt)

	// Fetch via GetByID
	fetched, err := s.repo.GetByID(ctx, s.workspaceID, acc.ID)
	s.Require().NoError(err)
	s.Equal(acc.ID, fetched.ID)
	s.Equal("Alice", fetched.AuditMeta.CreatedBy.Name)
}

func (s *AccountRepoSuite) Test_SearchAndCursorPagination() {
	ctx := s.ctx()

	for i := 1; i <= 5; i++ {
		code := fmt.Sprintf("100%d", i)
		name := fmt.Sprintf("Account %d", i)
		_, err := s.repo.Create(ctx, s.workspaceID, model.CreateAccountRequest{
			Code: code,
			Name: name,
			Type: domain.AccountTypeAsset,
		}, domain.NormalBalanceDebit)
		s.Require().NoError(err)
	}

	// Search limit 3
	items, cursor, err := s.repo.Search(ctx, s.workspaceID, model.SearchAccountRequest{Limit: 3})
	s.Require().NoError(err)
	s.Len(items, 3)
	s.NotEmpty(cursor)

	// Fetch next page
	nextItems, nextCursor, err := s.repo.Search(ctx, s.workspaceID, model.SearchAccountRequest{
		Limit:  3,
		Cursor: cursor,
	})
	s.Require().NoError(err)
	s.Len(nextItems, 2)
	s.Empty(nextCursor)
}

func (s *AccountRepoSuite) Test_Update() {
	ctx := s.ctx()

	acc, err := s.repo.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code: "2000",
		Name: "Accounts Payable",
		Type: domain.AccountTypeLiability,
	}, domain.NormalBalanceCredit)
	s.Require().NoError(err)

	isPostable := false
	updated, err := s.repo.Update(ctx, s.workspaceID, acc.ID, model.UpdateAccountRequest{
		Name:       "AP Trade",
		IsPostable: &isPostable,
	})
	s.Require().NoError(err)
	s.Equal("AP Trade", updated.Name)
	s.False(updated.IsPostable)
}

func (s *AccountRepoSuite) Test_SoftDeleteAndCodeReuse() {
	ctx := s.ctx()

	acc, err := s.repo.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code: "3000",
		Name: "Retained Earnings",
		Type: domain.AccountTypeEquity,
	}, domain.NormalBalanceCredit)
	s.Require().NoError(err)

	// Soft delete
	err = s.repo.Delete(ctx, s.workspaceID, acc.ID)
	s.Require().NoError(err)

	// GetByID should return ErrNotFound
	_, err = s.repo.GetByID(ctx, s.workspaceID, acc.ID)
	s.Equal(base.ErrNotFound, err)

	// Search should not return soft deleted accounts by default
	items, _, err := s.repo.Search(ctx, s.workspaceID, model.SearchAccountRequest{Code: "3000"})
	s.Require().NoError(err)
	s.Empty(items)

	// Re-creating an account with the same code succeeds because the old one is soft-deleted
	newAcc, err := s.repo.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code: "3000",
		Name: "New Retained Earnings",
		Type: domain.AccountTypeEquity,
	}, domain.NormalBalanceCredit)
	s.Require().NoError(err)
	s.Equal("3000", newAcc.Code)
}

func (s *AccountRepoSuite) Test_HierarchyAndGetTree() {
	ctx := s.ctx()

	headerPostable := false
	root, err := s.repo.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code:       "1000",
		Name:       "Assets Header",
		Type:       domain.AccountTypeAsset,
		IsPostable: &headerPostable,
	}, domain.NormalBalanceDebit)
	s.Require().NoError(err)

	child, err := s.repo.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code:     "1100",
		Name:     "Current Assets",
		Type:     domain.AccountTypeAsset,
		ParentID: &root.ID,
	}, domain.NormalBalanceDebit)
	s.Require().NoError(err)

	hasChildren, err := s.repo.HasActiveChildren(ctx, s.workspaceID, root.ID)
	s.Require().NoError(err)
	s.True(hasChildren)

	tree, err := s.repo.GetTree(ctx, s.workspaceID)
	s.Require().NoError(err)
	s.Len(tree, 1)
	s.Equal("1000", tree[0].Code)
	s.Len(tree[0].Children, 1)
	s.Equal("1100", tree[0].Children[0].Code)
	s.Equal(child.ID, tree[0].Children[0].ID)
}

func TestAccountRepoSuite(t *testing.T) {
	suite.Run(t, new(AccountRepoSuite))
}
