package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/platform/money"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type ExchangeRateRepoSuite struct {
	suite.Suite
	client      *ent.Client
	repo        repository.ExchangeRateRepository
	workspaceID int64
	userID      int64
}

func TestExchangeRateRepoSuite(t *testing.T) {
	suite.Run(t, new(ExchangeRateRepoSuite))
}

func (s *ExchangeRateRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:exchangerate_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	core.InitEntCli(s.client)
	s.repo = repository.NewExchangeRateRepository(s.client)
}

func (s *ExchangeRateRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *ExchangeRateRepoSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(base.WithSkipWorkspaceFilter(context.Background()))
	_, _ = s.client.ExchangeRate.Delete().Exec(ctx)
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
		SetName("FX Workspace").
		SetOwnerID(u.ID).
		Save(userCtx)
	s.Require().NoError(err)
	s.workspaceID = w.ID
}

func (s *ExchangeRateRepoSuite) ctx() context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   s.userID,
		Name: "Bob",
	})
	return base.WithWorkspaceID(ctx, s.workspaceID)
}

func (s *ExchangeRateRepoSuite) Test_Create_And_GetByID() {
	ctx := s.ctx()
	d := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	req := model.CreateExchangeRateRequest{
		FromCurrency:  "USD",
		ToCurrency:    "IDR",
		Rate:          16200.0,
		EffectiveDate: model.Date(d),
	}

	created, err := s.repo.Create(ctx, req)
	s.Require().NoError(err)
	s.Assert().Equal("USD", created.FromCurrency)
	s.Assert().Equal("IDR", created.ToCurrency)
	s.Assert().Equal(16200.0, created.Rate)
	s.Assert().Equal(domain.ExchangeRateSourceManual, created.Source)

	// Fetch by ID
	fetched, err := s.repo.GetByID(ctx, created.ID)
	s.Require().NoError(err)
	s.Assert().Equal(created.ID, fetched.ID)

	// Duplicate create fails
	_, err = s.repo.Create(ctx, req)
	s.Assert().ErrorIs(err, domain.ErrExchangeRateDuplicate)

	// Non-positive rate fails
	reqInvalid := req
	reqInvalid.Rate = -1
	_, err = s.repo.Create(ctx, reqInvalid)
	s.Assert().ErrorIs(err, domain.ErrExchangeRateInvalid)
}

func (s *ExchangeRateRepoSuite) Test_Update_And_Delete() {
	ctx := s.ctx()
	d1 := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)

	created, err := s.repo.Create(ctx, model.CreateExchangeRateRequest{
		FromCurrency:  "USD",
		ToCurrency:    "IDR",
		Rate:          16000.0,
		EffectiveDate: model.Date(d1),
	})
	s.Require().NoError(err)

	// Update rate and effective date
	updated, err := s.repo.Update(ctx, created.ID, model.UpdateExchangeRateRequest{
		Rate:          16100.0,
		EffectiveDate: model.Date(d2),
	})
	s.Require().NoError(err)
	s.Assert().Equal(16100.0, updated.Rate)

	// Delete
	err = s.repo.Delete(ctx, created.ID)
	s.Require().NoError(err)

	// Get after delete returns not found
	_, err = s.repo.GetByID(ctx, created.ID)
	s.Assert().ErrorIs(err, base.ErrNotFound)
}

func (s *ExchangeRateRepoSuite) Test_Lookup() {
	ctx := s.ctx()
	d1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	// 1. Same currency returns 1.0
	r, err := s.repo.Lookup(ctx, "USD", "USD", time.Now())
	s.Require().NoError(err)
	s.Assert().Equal(1.0, r)

	// 2. No rates yet -> ErrExchangeRateNotFound
	_, err = s.repo.Lookup(ctx, "USD", "IDR", d2)
	s.Assert().ErrorIs(err, domain.ErrExchangeRateNotFound)

	// 3. Create rate effective on 2026-01-01 (16000) and 2026-01-15 (16500)
	_, err = s.repo.Create(ctx, model.CreateExchangeRateRequest{
		FromCurrency:  "USD",
		ToCurrency:    "IDR",
		Rate:          16000.0,
		EffectiveDate: model.Date(d1),
	})
	s.Require().NoError(err)

	_, err = s.repo.Create(ctx, model.CreateExchangeRateRequest{
		FromCurrency:  "USD",
		ToCurrency:    "IDR",
		Rate:          16500.0,
		EffectiveDate: model.Date(d2),
	})
	s.Require().NoError(err)

	// On 2026-01-05 -> should find d1 rate (16000)
	rate05, err := s.repo.Lookup(ctx, "USD", "IDR", time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC))
	s.Require().NoError(err)
	s.Assert().Equal(16000.0, rate05)

	// On 2026-01-20 -> should find d2 rate (16500)
	rate20, err := s.repo.Lookup(ctx, "USD", "IDR", time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))
	s.Require().NoError(err)
	s.Assert().Equal(16500.0, rate20)

	// 4. Inverse lookup: IDR -> USD on 2026-01-20 should be 1.0 / 16500.0
	invRate, err := s.repo.Lookup(ctx, "IDR", "USD", time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))
	s.Require().NoError(err)
	s.Assert().True(money.Equal(invRate, 1.0/16500.0, 8))
}

func (s *ExchangeRateRepoSuite) Test_Search() {
	ctx := s.ctx()
	d1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	_, err := s.repo.Create(ctx, model.CreateExchangeRateRequest{
		FromCurrency:  "USD",
		ToCurrency:    "IDR",
		Rate:          16000.0,
		EffectiveDate: model.Date(d1),
	})
	s.Require().NoError(err)

	_, err = s.repo.Create(ctx, model.CreateExchangeRateRequest{
		FromCurrency:  "EUR",
		ToCurrency:    "IDR",
		Rate:          17500.0,
		EffectiveDate: model.Date(d2),
	})
	s.Require().NoError(err)

	// Filter by fromCurrency USD
	usd := "USD"
	results, _, err := s.repo.Search(ctx, model.SearchExchangeRateRequest{
		FromCurrency: &usd,
	})
	s.Require().NoError(err)
	s.Assert().Len(results, 1)
	s.Assert().Equal("USD", results[0].FromCurrency)
}
