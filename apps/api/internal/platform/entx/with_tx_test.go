package entx_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/platform/entx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type WithTxSuite struct {
	suite.Suite
	client *ent.Client
}

func (s *WithTxSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:entx_with_tx_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
}

func (s *WithTxSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *WithTxSuite) SetupTest() {
	ctx := context.Background()
	_, err := s.client.User.Delete().Exec(ctx)
	s.Require().NoError(err)
}

func (s *WithTxSuite) Test_WithTx_Success() {
	ctx := context.Background()

	err := entx.WithTx(ctx, s.client, func(tx *ent.Tx) error {
		_, err := tx.User.Create().
			SetName("Alice").
			SetEmail("alice@example.com").
			Save(ctx)
		return err
	})

	s.Require().NoError(err)

	count, err := s.client.User.Query().Count(ctx)
	s.Require().NoError(err)
	s.Equal(1, count)

	user, err := s.client.User.Query().Only(ctx)
	s.Require().NoError(err)
	s.Equal("Alice", user.Name)
	s.Equal("alice@example.com", user.Email)
}

func (s *WithTxSuite) Test_WithTx_FunctionError() {
	ctx := context.Background()
	expectedErr := errors.New("custom operation failed")

	err := entx.WithTx(ctx, s.client, func(tx *ent.Tx) error {
		_, err := tx.User.Create().
			SetName("Bob").
			SetEmail("bob@example.com").
			Save(ctx)
		s.Require().NoError(err)
		return expectedErr
	})

	s.Require().Error(err)
	s.ErrorIs(err, expectedErr)

	count, err := s.client.User.Query().Count(ctx)
	s.Require().NoError(err)
	s.Equal(0, count)
}

func (s *WithTxSuite) Test_WithTx_Panic() {
	ctx := context.Background()
	panicVal := "unexpected panic occurred"

	s.PanicsWithValue(panicVal, func() {
		_ = entx.WithTx(ctx, s.client, func(tx *ent.Tx) error {
			_, err := tx.User.Create().
				SetName("Charlie").
				SetEmail("charlie@example.com").
				Save(ctx)
			s.Require().NoError(err)
			panic(panicVal)
		})
	})

	count, err := s.client.User.Query().Count(ctx)
	s.Require().NoError(err)
	s.Equal(0, count)
}

func (s *WithTxSuite) Test_WithTx_TxError() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	called := false
	err := entx.WithTx(ctx, s.client, func(tx *ent.Tx) error {
		called = true
		return nil
	})

	s.Require().Error(err)
	s.False(called)
}

func (s *WithTxSuite) Test_WithTx_RollbackError() {
	ctx := context.Background()
	expectedErr := errors.New("operation failure")

	err := entx.WithTx(ctx, s.client, func(tx *ent.Tx) error {
		_ = tx.Rollback() // Force transaction to already be closed
		return expectedErr
	})

	s.Require().Error(err)
	s.ErrorIs(err, expectedErr)
	s.Contains(err.Error(), "rolling back transaction")
}

func (s *WithTxSuite) Test_WithTx_CommitError() {
	ctx := context.Background()

	err := entx.WithTx(ctx, s.client, func(tx *ent.Tx) error {
		_ = tx.Rollback() // Force transaction to already be closed
		return nil
	})

	s.Require().Error(err)
	s.Contains(err.Error(), "committing transaction")
}

func TestWithTxSuite(t *testing.T) {
	suite.Run(t, new(WithTxSuite))
}
