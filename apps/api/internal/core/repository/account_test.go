package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/repository"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type AccountRepoSuite struct {
	suite.Suite
	client *ent.Client
	repo   repository.AccountRepository
	ctx    context.Context
}

func (s *AccountRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:account_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.repo = repository.NewAccountRepository(s.client)
	s.ctx = testutil.NewAdminContext()
}

func (s *AccountRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *AccountRepoSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.UserAccount.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)
}

func (s *AccountRepoSuite) createTestUser(name, email string) *ent.User {
	ctx := context.Background()
	u, err := s.client.User.Create().
		SetName(name).
		SetEmail(email).
		Save(ctx)
	s.Require().NoError(err)
	return u
}

func (s *AccountRepoSuite) Test_UpsertAndFindByProviderAndAccountID() {
	user := s.createTestUser("Alice", "alice@example.com")
	now := time.Now().Truncate(time.Second)

	acc := domain.Account{
		UserID:               user.ID,
		ProviderID:           domain.ProviderGoogle,
		AccountID:            "google-user-123",
		AccessToken:          "token-v1",
		RefreshToken:         "refresh-v1",
		AccessTokenExpiresAt: &now,
		Scope:                "openid profile email",
	}

	created, err := s.repo.Upsert(s.ctx, acc)
	s.Require().NoError(err)
	s.Equal(user.ID, created.UserID)
	s.Equal("google-user-123", created.AccountID)
	s.Equal("token-v1", created.AccessToken)

	// Test find
	found, err := s.repo.FindByProviderAndAccountID(s.ctx, domain.ProviderGoogle, "google-user-123")
	s.Require().NoError(err)
	s.Equal(created.ID, found.ID)
	s.Equal("token-v1", found.AccessToken)

	// Test update via Upsert
	acc.AccessToken = "token-v2"
	updated, err := s.repo.Upsert(s.ctx, acc)
	s.Require().NoError(err)
	s.Equal("token-v2", updated.AccessToken)

	foundUpdated, err := s.repo.FindByProviderAndAccountID(s.ctx, domain.ProviderGoogle, "google-user-123")
	s.Require().NoError(err)
	s.Equal("token-v2", foundUpdated.AccessToken)
}

func (s *AccountRepoSuite) Test_FindByUserID() {
	user := s.createTestUser("Bob", "bob@example.com")

	_, err := s.repo.Upsert(s.ctx, domain.Account{
		UserID:     user.ID,
		ProviderID: domain.ProviderGoogle,
		AccountID:  "google-bob",
	})
	s.Require().NoError(err)

	_, err = s.repo.Upsert(s.ctx, domain.Account{
		UserID:     user.ID,
		ProviderID: domain.ProviderGithub,
		AccountID:  "github-bob",
	})
	s.Require().NoError(err)

	accounts, err := s.repo.FindByUserID(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Len(accounts, 2)
}

func TestAccountRepoSuite(t *testing.T) {
	suite.Run(t, new(AccountRepoSuite))
}
