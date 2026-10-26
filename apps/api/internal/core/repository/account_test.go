package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/useraccount"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/repository"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

const testAuthSecret = "test-secret-key-at-least-32-bytes-long"

type AccountRepoSuite struct {
	suite.Suite
	client *ent.Client
	repo   repository.AccountRepository
	ctx    context.Context
}

func (s *AccountRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:account_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.repo = repository.NewAccountRepository(s.client, testAuthSecret)
	s.ctx = testutil.NewUserContext()
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
		IDToken:              "id-token-v1",
		AccessTokenExpiresAt: &now,
		Scope:                "openid profile email",
	}

	created, err := s.repo.Upsert(s.ctx, acc)
	s.Require().NoError(err)
	s.Equal(user.ID, created.UserID)
	s.Equal("google-user-123", created.AccountID)
	s.Equal("token-v1", created.AccessToken)
	s.Equal("refresh-v1", created.RefreshToken)
	s.Equal("id-token-v1", created.IDToken)

	// Verify that stored tokens in the database are encrypted and cannot be read as plaintext
	rawAccount, err := s.client.UserAccount.Query().
		Where(
			useraccount.ProviderIDEQ(domain.ProviderGoogle),
			useraccount.AccountIDEQ("google-user-123"),
		).
		Only(s.ctx)
	s.Require().NoError(err)
	s.NotEmpty(rawAccount.AccessToken)
	s.NotEqual("token-v1", rawAccount.AccessToken)
	s.NotEmpty(rawAccount.RefreshToken)
	s.NotEqual("refresh-v1", rawAccount.RefreshToken)
	s.NotEmpty(rawAccount.IDToken)
	s.NotEqual("id-token-v1", rawAccount.IDToken)

	// Test find returns decrypted plaintext tokens
	found, err := s.repo.FindByProviderAndAccountID(s.ctx, domain.ProviderGoogle, "google-user-123")
	s.Require().NoError(err)
	s.Equal(created.ID, found.ID)
	s.Equal("token-v1", found.AccessToken)
	s.Equal("refresh-v1", found.RefreshToken)
	s.Equal("id-token-v1", found.IDToken)

	// Test update via Upsert
	acc.AccessToken = "token-v2"
	acc.RefreshToken = "refresh-v2"
	updated, err := s.repo.Upsert(s.ctx, acc)
	s.Require().NoError(err)
	s.Equal("token-v2", updated.AccessToken)
	s.Equal("refresh-v2", updated.RefreshToken)

	// Verify updated tokens are also encrypted in DB
	rawUpdated, err := s.client.UserAccount.Query().
		Where(
			useraccount.ProviderIDEQ(domain.ProviderGoogle),
			useraccount.AccountIDEQ("google-user-123"),
		).
		Only(s.ctx)
	s.Require().NoError(err)
	s.NotEqual("token-v2", rawUpdated.AccessToken)
	s.NotEqual("refresh-v2", rawUpdated.RefreshToken)

	foundUpdated, err := s.repo.FindByProviderAndAccountID(s.ctx, domain.ProviderGoogle, "google-user-123")
	s.Require().NoError(err)
	s.Equal("token-v2", foundUpdated.AccessToken)
	s.Equal("refresh-v2", foundUpdated.RefreshToken)
}

func (s *AccountRepoSuite) Test_FindByUserID() {
	user := s.createTestUser("Bob", "bob@example.com")

	_, err := s.repo.Upsert(s.ctx, domain.Account{
		UserID:       user.ID,
		ProviderID:   domain.ProviderGoogle,
		AccountID:    "google-bob",
		AccessToken:  "token-google",
		RefreshToken: "refresh-google",
	})
	s.Require().NoError(err)

	_, err = s.repo.Upsert(s.ctx, domain.Account{
		UserID:       user.ID,
		ProviderID:   domain.ProviderGithub,
		AccountID:    "github-bob",
		AccessToken:  "token-github",
		RefreshToken: "refresh-github",
	})
	s.Require().NoError(err)

	accounts, err := s.repo.FindByUserID(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Len(accounts, 2)

	for _, a := range accounts {
		switch a.ProviderID {
		case domain.ProviderGoogle:
			s.Equal("token-google", a.AccessToken)
			s.Equal("refresh-google", a.RefreshToken)
		case domain.ProviderGithub:
			s.Equal("token-github", a.AccessToken)
			s.Equal("refresh-github", a.RefreshToken)
		default:
			s.Fail("unexpected provider: " + a.ProviderID)
		}
	}
}

func (s *AccountRepoSuite) Test_DecryptionWithWrongSecret_Fails() {
	user := s.createTestUser("Charlie", "charlie@example.com")
	_, err := s.repo.Upsert(s.ctx, domain.Account{
		UserID:      user.ID,
		ProviderID:  domain.ProviderGoogle,
		AccountID:   "google-charlie",
		AccessToken: "secret-token",
	})
	s.Require().NoError(err)

	wrongRepo := repository.NewAccountRepository(s.client, "wrong-secret-key-different-hash")
	_, err = wrongRepo.FindByProviderAndAccountID(s.ctx, domain.ProviderGoogle, "google-charlie")
	s.Require().Error(err)
}

func TestAccountRepoSuite(t *testing.T) {
	suite.Run(t, new(AccountRepoSuite))
}
