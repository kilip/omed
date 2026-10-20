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
	ctx := base.WithSkipAuditMutator(context.Background())
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

	userCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   u.ID,
		Name: "Bob",
	})
	w, err := s.client.Workspace.Create().
		SetName("Finance WS").
		SetOwnerID(u.ID).
		Save(userCtx)
	s.Require().NoError(err)
	s.workspaceID = w.ID
}

func (s *AccountServiceSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Bob",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func (s *AccountServiceSuite) Test_Create_InvalidType() {
	ctx := s.ctx()

	_, err := s.svc.Create(ctx, model.CreateAccountRequest{
		Code: "1000",
		Name: "Invalid",
		Type: "NON_EXISTENT",
	})
	s.Equal(domain.ErrAccountInvalidType, err)
}

func (s *AccountServiceSuite) Test_Create_DefaultNormalBalance() {
	ctx := s.ctx()

	// Asset defaults to DEBIT
	asset, err := s.svc.Create(ctx, model.CreateAccountRequest{
		Code: "1000",
		Name: "Asset",
		Type: domain.AccountTypeAsset,
	})
	s.Require().NoError(err)
	s.Equal(domain.NormalBalanceDebit, asset.NormalBalance)

	// Revenue defaults to CREDIT
	rev, err := s.svc.Create(ctx, model.CreateAccountRequest{
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
	_, err := s.svc.Create(ctx, model.CreateAccountRequest{
		Code:     "1100",
		Name:     "Child",
		Type:     domain.AccountTypeAsset,
		ParentID: invalidParentID,
	})
	s.Equal(domain.ErrAccountParentNotFound, err)
}

func (s *AccountServiceSuite) Test_Create_DuplicateCode() {
	ctx := s.ctx()

	_, err := s.svc.Create(ctx, model.CreateAccountRequest{
		Code: "1000",
		Name: "Cash",
		Type: domain.AccountTypeAsset,
	})
	s.Require().NoError(err)

	_, err = s.svc.Create(ctx, model.CreateAccountRequest{
		Code: "1000",
		Name: "Cash Duplicate",
		Type: domain.AccountTypeAsset,
	})
	s.Equal(domain.ErrAccountCodeTaken, err)
}

func (s *AccountServiceSuite) Test_Update_CyclePrevention() {
	ctx := s.ctx()

	root, err := s.svc.Create(ctx, model.CreateAccountRequest{
		Code: "1000",
		Name: "Assets",
		Type: domain.AccountTypeAsset,
	})
	s.Require().NoError(err)

	child, err := s.svc.Create(ctx, model.CreateAccountRequest{
		Code:     "1100",
		Name:     "Current Assets",
		Type:     domain.AccountTypeAsset,
		ParentID: root.ID,
	})
	s.Require().NoError(err)

	grandchild, err := s.svc.Create(ctx, model.CreateAccountRequest{
		Code:     "1110",
		Name:     "Cash and Bank",
		Type:     domain.AccountTypeAsset,
		ParentID: child.ID,
	})
	s.Require().NoError(err)

	// Attempting to set root's parent to itself fails
	_, err = s.svc.Update(ctx, root.ID, model.UpdateAccountRequest{
		Name:     "Assets",
		ParentID: root.ID,
	})
	s.Equal(domain.ErrAccountCyclicParent, err)

	// Attempting to set root's parent to grandchild fails (creates cycle)
	_, err = s.svc.Update(ctx, root.ID, model.UpdateAccountRequest{
		Name:     "Assets",
		ParentID: grandchild.ID,
	})
	s.Equal(domain.ErrAccountCyclicParent, err)
}

func (s *AccountServiceSuite) Test_Delete_BlockedWithChildren() {
	ctx := s.ctx()

	root, err := s.svc.Create(ctx, model.CreateAccountRequest{
		Code: "1000",
		Name: "Assets",
		Type: domain.AccountTypeAsset,
	})
	s.Require().NoError(err)

	child, err := s.svc.Create(ctx, model.CreateAccountRequest{
		Code:     "1100",
		Name:     "Current Assets",
		Type:     domain.AccountTypeAsset,
		ParentID: root.ID,
	})
	s.Require().NoError(err)

	// Attempting to delete root should fail because child exists
	err = s.svc.Delete(ctx, root.ID)
	s.Equal(domain.ErrAccountHasChildren, err)

	// Deleting child first succeeds
	err = s.svc.Delete(ctx, child.ID)
	s.Require().NoError(err)

	// Now deleting root succeeds
	err = s.svc.Delete(ctx, root.ID)
	s.Require().NoError(err)
}

func (s *AccountServiceSuite) Test_Seed_SuccessAndErrors() {
	ctx := s.ctx()

	// Seed invalid template
	_, err := s.svc.Seed(ctx, model.SeedAccountsRequest{
		Template: "invalid",
		Lang:     "id",
	})
	s.Require().ErrorIs(err, domain.ErrAccountTemplateNotFound)

	// Seed valid freelancer id template
	res, err := s.svc.Seed(ctx, model.SeedAccountsRequest{
		Template: "freelancer",
		Lang:     "id",
	})
	s.Require().NoError(err)
	s.Greater(res.Count, 40)

	// Second seed should fail with ErrAccountWorkspaceNotEmpty
	_, err = s.svc.Seed(ctx, model.SeedAccountsRequest{
		Template: "freelancer",
		Lang:     "id",
	})
	s.Require().ErrorIs(err, domain.ErrAccountWorkspaceNotEmpty)
}

func (s *AccountServiceSuite) Test_PreviewTemplate() {
	ctx := s.ctx()

	// Invalid template
	_, err := s.svc.PreviewTemplate(ctx, "nonexistent", "id")
	s.Require().ErrorIs(err, domain.ErrAccountTemplateNotFound)

	// Valid freelancer ID preview
	resID, err := s.svc.PreviewTemplate(ctx, "freelancer", "id")
	s.Require().NoError(err)
	s.Equal("freelancer", resID.Template)
	s.Equal("id", resID.Lang)
	s.Greater(len(resID.Accounts), 40)

	// Verify specific personal expense and liability accounts exist
	has6700 := false
	has2160 := false
	for _, acc := range resID.Accounts {
		if acc.Code == "6700" {
			has6700 = true
			s.False(acc.IsPostable)
			s.Equal("EXPENSE", acc.Type)
		}
		if acc.Code == "2160" {
			has2160 = true
			s.True(acc.IsPostable)
			s.Equal("LIABILITY", acc.Type)
		}
	}
	s.True(has6700, "Should contain 6700 Beban Hidup & Pribadi")
	s.True(has2160, "Should contain 2160 Utang PayLater")

	// Valid freelancer EN preview
	resEN, err := s.svc.PreviewTemplate(ctx, "freelancer", "en")
	s.Require().NoError(err)
	s.Equal("en", resEN.Lang)
	s.Greater(len(resEN.Accounts), 30)
}

func TestAccountServiceSuite(t *testing.T) {
	suite.Run(t, new(AccountServiceSuite))
}
