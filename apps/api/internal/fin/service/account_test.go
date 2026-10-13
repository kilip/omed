package service_test

import (
	"context"
	"testing"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/fin/service"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type AccountServiceSuite struct {
	suite.Suite
	client      *ent.Client
	svc         service.AccountService
	workspaceID int64
	userID      int64
}

func (s *AccountServiceSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:account_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	repo := repository.NewAccountRepository(s.client)
	s.svc = service.NewAccountService(repo)
}

func (s *AccountServiceSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *AccountServiceSuite) SetupTest() {
	ctx := context.Background()
	_, _ = s.client.Account.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)

	u, err := s.client.User.Create().
		SetName("Bob").
		SetEmail("bob@example.com").
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

func (s *AccountServiceSuite) ctx() context.Context {
	return context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Bob",
	})
}

func (s *AccountServiceSuite) Test_Create_InvalidType() {
	ctx := s.ctx()

	_, err := s.svc.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code: "1000",
		Name: "Invalid",
		Type: "NON_EXISTENT",
	})
	s.Equal(domain.ErrAccountInvalidType, err)
}

func (s *AccountServiceSuite) Test_Create_DefaultNormalBalance() {
	ctx := s.ctx()

	// Asset defaults to DEBIT
	asset, err := s.svc.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code: "1000",
		Name: "Asset",
		Type: domain.AccountTypeAsset,
	})
	s.Require().NoError(err)
	s.Equal(domain.NormalBalanceDebit, asset.NormalBalance)

	// Revenue defaults to CREDIT
	rev, err := s.svc.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code: "4000",
		Name: "Sales",
		Type: domain.AccountTypeRevenue,
	})
	s.Require().NoError(err)
	s.Equal(domain.NormalBalanceCredit, rev.NormalBalance)
}

func (s *AccountServiceSuite) Test_Create_ParentNotFound() {
	ctx := s.ctx()

	invalidParentID := int64(999999)
	_, err := s.svc.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code:     "1100",
		Name:     "Child",
		Type:     domain.AccountTypeAsset,
		ParentID: invalidParentID,
	})
	s.Equal(domain.ErrAccountParentNotFound, err)
}

func (s *AccountServiceSuite) Test_Create_DuplicateCode() {
	ctx := s.ctx()

	_, err := s.svc.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code: "1000",
		Name: "Cash",
		Type: domain.AccountTypeAsset,
	})
	s.Require().NoError(err)

	_, err = s.svc.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code: "1000",
		Name: "Cash Duplicate",
		Type: domain.AccountTypeAsset,
	})
	s.Equal(domain.ErrAccountCodeTaken, err)
}

func (s *AccountServiceSuite) Test_Update_CyclePrevention() {
	ctx := s.ctx()

	root, err := s.svc.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code: "1000",
		Name: "Assets",
		Type: domain.AccountTypeAsset,
	})
	s.Require().NoError(err)

	child, err := s.svc.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code:     "1100",
		Name:     "Current Assets",
		Type:     domain.AccountTypeAsset,
		ParentID: root.ID,
	})
	s.Require().NoError(err)

	grandchild, err := s.svc.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code:     "1110",
		Name:     "Cash and Bank",
		Type:     domain.AccountTypeAsset,
		ParentID: child.ID,
	})
	s.Require().NoError(err)

	// Attempting to set root's parent to itself fails
	_, err = s.svc.Update(ctx, s.workspaceID, root.ID, model.UpdateAccountRequest{
		Name:     "Assets",
		ParentID: root.ID,
	})
	s.Equal(domain.ErrAccountCyclicParent, err)

	// Attempting to set root's parent to grandchild fails (creates cycle)
	_, err = s.svc.Update(ctx, s.workspaceID, root.ID, model.UpdateAccountRequest{
		Name:     "Assets",
		ParentID: grandchild.ID,
	})
	s.Equal(domain.ErrAccountCyclicParent, err)
}

func (s *AccountServiceSuite) Test_Delete_BlockedWithChildren() {
	ctx := s.ctx()

	root, err := s.svc.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code: "1000",
		Name: "Assets",
		Type: domain.AccountTypeAsset,
	})
	s.Require().NoError(err)

	child, err := s.svc.Create(ctx, s.workspaceID, model.CreateAccountRequest{
		Code:     "1100",
		Name:     "Current Assets",
		Type:     domain.AccountTypeAsset,
		ParentID: root.ID,
	})
	s.Require().NoError(err)

	// Attempting to delete root should fail because child exists
	err = s.svc.Delete(ctx, s.workspaceID, root.ID)
	s.Equal(domain.ErrAccountHasChildren, err)

	// Deleting child first succeeds
	err = s.svc.Delete(ctx, s.workspaceID, child.ID)
	s.Require().NoError(err)

	// Now deleting root succeeds
	err = s.svc.Delete(ctx, s.workspaceID, root.ID)
	s.Require().NoError(err)
}

func TestAccountServiceSuite(t *testing.T) {
	suite.Run(t, new(AccountServiceSuite))
}
