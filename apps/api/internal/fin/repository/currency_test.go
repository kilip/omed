package repository_test

import (
	"context"
	"testing"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/fin/seed"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type CurrencyRepoSuite struct {
	suite.Suite
	client      *ent.Client
	repo        repository.CurrencyRepository
	workspaceID int64
	userID      int64
}

func TestCurrencyRepoSuite(t *testing.T) {
	suite.Run(t, new(CurrencyRepoSuite))
}

func (s *CurrencyRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:currency_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	core.InitEntCli(s.client)
	s.repo = repository.NewCurrencyRepository(s.client)
}

func (s *CurrencyRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *CurrencyRepoSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.Currency.Delete().Exec(ctx)
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
		SetName("Test Workspace").
		SetOwnerID(u.ID).
		SetBaseCurrency("IDR").
		Save(userCtx)
	s.Require().NoError(err)
	s.workspaceID = w.ID

	err = seed.SeedCurrencies(ctx, s.client)
	s.Require().NoError(err)
}

func (s *CurrencyRepoSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Alice",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func (s *CurrencyRepoSuite) Test_List() {
	currencies, err := s.repo.List(s.ctx())
	s.Require().NoError(err)
	s.Require().NotEmpty(currencies)
	s.Assert().Equal("EUR", currencies[0].Code) // alphabetical order
}

func (s *CurrencyRepoSuite) Test_GetByCode() {
	ctx := base.WithWorkspaceID(context.Background(), s.workspaceID)

	c, err := s.repo.GetByCode(ctx, "USD")
	s.Require().NoError(err)
	s.Assert().Equal("USD", c.Code)
	s.Assert().Equal(2, c.Decimals)

	_, err = s.repo.GetByCode(ctx, "XYZ")
	s.Assert().ErrorIs(err, domain.ErrCurrencyNotFound)
}

func (s *CurrencyRepoSuite) Test_WorkspaceBaseCurrency() {
	ctx := s.ctx()

	bc, err := s.repo.GetBaseCurrency(ctx, s.workspaceID)
	s.Require().NoError(err)
	s.Assert().Equal("IDR", bc)

	resp, err := s.repo.UpdateWorkspaceBaseCurrency(ctx, s.workspaceID, "USD")
	s.Require().NoError(err)
	s.Assert().Equal("USD", resp.BaseCurrency)

	bc, err = s.repo.GetBaseCurrency(ctx, s.workspaceID)
	s.Require().NoError(err)
	s.Assert().Equal("USD", bc)
}
