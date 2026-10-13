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
	ctx := context.Background()
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
		Roles:         []base.UserRole{base.UserRoleAdmin},
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
