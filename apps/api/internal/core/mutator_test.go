package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/account"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type MutatorSuite struct {
	suite.Suite
	client *ent.Client
	user   *ent.User
	ctx    context.Context
}

func TestMutatorSuite(t *testing.T) {
	suite.Run(t, new(MutatorSuite))
}

func (s *MutatorSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:mutator_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
}

func (s *MutatorSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *MutatorSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.File.Delete().Exec(ctx)
	_, _ = s.client.Account.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)

	u, err := s.client.User.Create().
		SetName("Audit User").
		SetEmail("audit@example.com").
		Save(ctx)
	s.Require().NoError(err)
	s.user = u

	s.ctx = context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:            u.ID,
		Name:          u.Name,
		EmailVerified: true,
		Roles:         []base.UserRole{base.UserRoleUser},
	})
}

func (s *MutatorSuite) Test_CreateAndAuditFields() {
	ws, err := s.client.Workspace.Create().
		SetName("Test WS").
		SetOwnerID(s.user.ID).
		Save(s.ctx)
	s.Require().NoError(err)

	s.Equal(s.user.ID, ws.CreatedBy)
	s.Equal(s.user.ID, ws.UpdatedBy)
}

func (s *MutatorSuite) Test_UpdateAuditFields() {
	ws, err := s.client.Workspace.Create().
		SetName("Initial WS").
		SetOwnerID(s.user.ID).
		Save(s.ctx)
	s.Require().NoError(err)

	// Switch to another user in context
	u2, err := s.client.User.Create().
		SetName("Second User").
		SetEmail("second@example.com").
		Save(context.Background())
	s.Require().NoError(err)

	ctx2 := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   u2.ID,
		Name: u2.Name,
	})

	updated, err := s.client.Workspace.UpdateOneID(ws.ID).
		SetName("Renamed WS").
		Save(ctx2)
	s.Require().NoError(err)

	s.Equal(s.user.ID, updated.CreatedBy)
	s.Equal(u2.ID, updated.UpdatedBy)
}

func (s *MutatorSuite) Test_SoftDeleteAuditFields_AutoSetDeleter() {
	ws, err := s.client.Workspace.Create().
		SetName("Finance WS").
		SetOwnerID(s.user.ID).
		Save(s.ctx)
	s.Require().NoError(err)

	acc, err := s.client.Account.Create().
		SetWorkspaceID(ws.ID).
		SetCode("1001").
		SetName("Cash").
		SetType(account.TypeASSET).
		SetNormalBalance(account.NormalBalanceDEBIT).
		Save(s.ctx)
	s.Require().NoError(err)
	s.Nil(acc.DeletedAt)
	s.Nil(acc.DeletedByID)

	// Soft delete via UpdateOne with SetDeletedAt
	now := time.Now()
	deleted, err := s.client.Account.UpdateOneID(acc.ID).
		SetDeletedAt(now).
		Save(s.ctx)
	s.Require().NoError(err)
	s.NotNil(deleted.DeletedAt)
	s.Require().NotNil(deleted.DeletedByID)
	s.Equal(s.user.ID, *deleted.DeletedByID)
}

func (s *MutatorSuite) Test_SoftDeleteAuditFields_ExplicitDeleterPreserved() {
	ws, err := s.client.Workspace.Create().
		SetName("Finance WS").
		SetOwnerID(s.user.ID).
		Save(s.ctx)
	s.Require().NoError(err)

	acc, err := s.client.Account.Create().
		SetWorkspaceID(ws.ID).
		SetCode("1002").
		SetName("Bank").
		SetType(account.TypeASSET).
		SetNormalBalance(account.NormalBalanceDEBIT).
		Save(s.ctx)
	s.Require().NoError(err)

	u2, err := s.client.User.Create().
		SetName("Another User").
		SetEmail("another@example.com").
		Save(context.Background())
	s.Require().NoError(err)

	// Soft delete with explicit deleted_by_id specified
	now := time.Now()
	deleted, err := s.client.Account.UpdateOneID(acc.ID).
		SetDeletedAt(now).
		SetDeletedByID(u2.ID).
		Save(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(deleted.DeletedByID)
	s.Equal(u2.ID, *deleted.DeletedByID, "explicitly set deleted_by_id must be preserved")
}

func (s *MutatorSuite) Test_WorkspaceMutator_AutoInjectOnCreate() {
	ws, err := s.client.Workspace.Create().
		SetName("Tenant A").
		SetOwnerID(s.user.ID).
		Save(s.ctx)
	s.Require().NoError(err)

	ctxWithWS := base.WithWorkspaceID(s.ctx, ws.ID)

	// Create account without calling SetWorkspaceID
	acc, err := s.client.Account.Create().
		SetCode("1101").
		SetName("Petty Cash").
		SetType(account.TypeASSET).
		SetNormalBalance(account.NormalBalanceDEBIT).
		Save(ctxWithWS)
	s.Require().NoError(err)
	s.Equal(ws.ID, acc.WorkspaceID)
}

func (s *MutatorSuite) Test_WorkspaceMutator_OverrideManualWhenContextHasWorkspace() {
	wsA, err := s.client.Workspace.Create().
		SetName("Tenant A").
		SetOwnerID(s.user.ID).
		Save(s.ctx)
	s.Require().NoError(err)

	wsB, err := s.client.Workspace.Create().
		SetName("Tenant B").
		SetOwnerID(s.user.ID).
		Save(s.ctx)
	s.Require().NoError(err)

	// Context says wsA, but manual SetWorkspaceID passes wsB
	ctxWithWSA := base.WithWorkspaceID(s.ctx, wsA.ID)
	acc, err := s.client.Account.Create().
		SetWorkspaceID(wsB.ID).
		SetCode("1102").
		SetName("Account Receivable").
		SetType(account.TypeASSET).
		SetNormalBalance(account.NormalBalanceDEBIT).
		Save(ctxWithWSA)
	s.Require().NoError(err)
	s.Equal(wsA.ID, acc.WorkspaceID, "context workspace must override manual set")
}

func (s *MutatorSuite) Test_WorkspaceMutator_FallbackManualWhenContextEmpty() {
	ws, err := s.client.Workspace.Create().
		SetName("Tenant Manual").
		SetOwnerID(s.user.ID).
		Save(s.ctx)
	s.Require().NoError(err)

	// Context has no workspace ID, but builder sets it manually
	acc, err := s.client.Account.Create().
		SetWorkspaceID(ws.ID).
		SetCode("1103").
		SetName("Inventory").
		SetType(account.TypeASSET).
		SetNormalBalance(account.NormalBalanceDEBIT).
		Save(s.ctx)
	s.Require().NoError(err)
	s.Equal(ws.ID, acc.WorkspaceID)
}

func (s *MutatorSuite) Test_WorkspaceInterceptor_QueryFilteringAndBypass() {
	wsA, err := s.client.Workspace.Create().SetName("WS A").SetOwnerID(s.user.ID).Save(s.ctx)
	s.Require().NoError(err)
	wsB, err := s.client.Workspace.Create().SetName("WS B").SetOwnerID(s.user.ID).Save(s.ctx)
	s.Require().NoError(err)

	ctxA := base.WithWorkspaceID(s.ctx, wsA.ID)
	ctxB := base.WithWorkspaceID(s.ctx, wsB.ID)

	_, err = s.client.Account.Create().
		SetCode("A101").
		SetName("WS A Account").
		SetType(account.TypeASSET).
		SetNormalBalance(account.NormalBalanceDEBIT).
		Save(ctxA)
	s.Require().NoError(err)

	_, err = s.client.Account.Create().
		SetCode("B101").
		SetName("WS B Account").
		SetType(account.TypeASSET).
		SetNormalBalance(account.NormalBalanceDEBIT).
		Save(ctxB)
	s.Require().NoError(err)

	// Query with ctxA -> should only see 1 account belonging to WS A
	accountsA, err := s.client.Account.Query().All(ctxA)
	s.Require().NoError(err)
	s.Len(accountsA, 1)
	s.Equal("A101", accountsA[0].Code)

	// Query with ctxB -> should only see 1 account belonging to WS B
	accountsB, err := s.client.Account.Query().All(ctxB)
	s.Require().NoError(err)
	s.Len(accountsB, 1)
	s.Equal("B101", accountsB[0].Code)

	// Query with ctxA but with bypass -> should see both accounts
	ctxBypass := base.WithSkipWorkspaceFilter(ctxA)
	allAccounts, err := s.client.Account.Query().All(ctxBypass)
	s.Require().NoError(err)
	s.Len(allAccounts, 2)

	// Verify File entity workspace interceptor
	_, err = s.client.File.Create().
		SetName("a.txt").
		SetBucket("omed-ws-a").
		SetKey("k1").
		SetMimeType("text/plain").
		SetSize(10).
		Save(ctxA)
	s.Require().NoError(err)

	_, err = s.client.File.Create().
		SetName("b.txt").
		SetBucket("omed-ws-b").
		SetKey("k2").
		SetMimeType("text/plain").
		SetSize(20).
		Save(ctxB)
	s.Require().NoError(err)

	filesA, err := s.client.File.Query().All(ctxA)
	s.Require().NoError(err)
	s.Len(filesA, 1)
	s.Equal("a.txt", filesA[0].Name)
}

func (s *MutatorSuite) Test_SkipAuditMutator() {
	// WithSkipAuditMutator allows mutation without UserContext
	ctx := base.WithSkipAuditMutator(context.Background())
	ws, err := s.client.Workspace.Create().
		SetName("Skipped Audit WS").
		SetOwnerID(s.user.ID).
		SetCreatedBy(s.user.ID).
		SetUpdatedBy(s.user.ID).
		Save(ctx)
	s.Require().NoError(err)
	s.Equal(s.user.ID, ws.CreatedBy)
	s.Equal(s.user.ID, ws.UpdatedBy)

	// Updating with WithSkipAuditMutator does not auto-inject or modify updated_by
	updated, err := s.client.Workspace.UpdateOneID(ws.ID).
		SetName("Updated Skipped Audit WS").
		Save(ctx)
	s.Require().NoError(err)
	s.Equal(s.user.ID, updated.UpdatedBy)

	// Deleting with WithSkipAuditMutator does not panic without UserContext
	err = s.client.Workspace.DeleteOneID(ws.ID).Exec(ctx)
	s.Require().NoError(err)
}

func (s *MutatorSuite) Test_AuditMutator_PanicsWithoutUserAndWithoutSkip() {
	ctx := context.Background()
	s.Panics(func() {
		_, _ = s.client.Workspace.Create().
			SetName("Panic WS").
			SetOwnerID(s.user.ID).
			Save(ctx)
	})
}
