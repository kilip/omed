package repository_test

import (
	"context"
	"testing"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/platform/entx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type NumberingSuite struct {
	suite.Suite
	client      *ent.Client
	workspaceID int64
	userID      int64
}

func TestNumberingSuite(t *testing.T) {
	suite.Run(t, new(NumberingSuite))
}

func (s *NumberingSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:numbering_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
}

func (s *NumberingSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *NumberingSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(base.WithSkipWorkspaceFilter(context.Background()))
	_, _ = s.client.Posting.Delete().Exec(ctx)
	_, _ = s.client.Entry.Delete().Exec(ctx)
	_, _ = s.client.Transaction.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)

	u, err := s.client.User.Create().
		SetName("Numbering User").
		SetEmail("numbering@example.com").
		Save(ctx)
	s.Require().NoError(err)
	s.userID = u.ID

	userCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:            u.ID,
		Name:          u.Name,
		EmailVerified: true,
		Roles:         []base.UserRole{base.UserRoleUser},
	})

	w, err := s.client.Workspace.Create().
		SetName("Numbering WS").
		SetOwnerID(u.ID).
		Save(userCtx)
	s.Require().NoError(err)
	s.workspaceID = w.ID
}

func (s *NumberingSuite) Test_GenerateTransactionNumber_FailClosedWhenNoWorkspaceContext() {
	err := entx.WithTx(context.Background(), s.client, func(tx *ent.Tx) error {
		_, err := repository.GenerateTransactionNumber(context.Background(), tx, 2026)
		return err
	})
	s.Require().ErrorIs(err, base.ErrWorkspaceRequired)
}

func (s *NumberingSuite) Test_GenerateEntryNumber_FailClosedWhenNoWorkspaceContext() {
	err := entx.WithTx(context.Background(), s.client, func(tx *ent.Tx) error {
		_, err := repository.GenerateEntryNumber(context.Background(), tx, 2026)
		return err
	})
	s.Require().ErrorIs(err, base.ErrWorkspaceRequired)
}

func (s *NumberingSuite) Test_GenerateTransactionNumber_WithWorkspaceContext() {
	ctx := base.WithWorkspaceID(context.Background(), s.workspaceID)
	var num string
	err := entx.WithTx(ctx, s.client, func(tx *ent.Tx) error {
		var err error
		num, err = repository.GenerateTransactionNumber(ctx, tx, 2026)
		return err
	})
	s.Require().NoError(err)
	s.Equal("TRX-2026-000001", num)
}

func (s *NumberingSuite) Test_GenerateEntryNumber_WithWorkspaceContext() {
	ctx := base.WithWorkspaceID(context.Background(), s.workspaceID)
	var num string
	err := entx.WithTx(ctx, s.client, func(tx *ent.Tx) error {
		var err error
		num, err = repository.GenerateEntryNumber(ctx, tx, 2026)
		return err
	})
	s.Require().NoError(err)
	s.Equal("JE-2026-000001", num)
}

func (s *NumberingSuite) Test_GenerateNumbering_WithSkipWorkspaceFilter() {
	ctx := base.WithSkipWorkspaceFilter(context.Background())
	err := entx.WithTx(ctx, s.client, func(tx *ent.Tx) error {
		trxNum, err := repository.GenerateTransactionNumber(ctx, tx, 2026)
		s.Require().NoError(err)
		s.Equal("TRX-2026-000001", trxNum)

		jeNum, err := repository.GenerateEntryNumber(ctx, tx, 2026)
		s.Require().NoError(err)
		s.Equal("JE-2026-000001", jeNum)
		return nil
	})
	s.Require().NoError(err)
}
