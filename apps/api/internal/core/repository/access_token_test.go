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

type AccessTokenRepoSuite struct {
	suite.Suite
	client *ent.Client
	repo   repository.AccessTokenRepository
	ctx    context.Context
}

func (s *AccessTokenRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:access_token_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.repo = repository.NewAccessTokenRepository(s.client)
	s.ctx = testutil.NewUserContext()
}

func (s *AccessTokenRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *AccessTokenRepoSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.PersonalAccessToken.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)
}

func (s *AccessTokenRepoSuite) createTestUser(name, email string) *ent.User {
	ctx := context.Background()
	u, err := s.client.User.Create().
		SetName(name).
		SetEmail(email).
		Save(ctx)
	s.Require().NoError(err)
	return u
}

func (s *AccessTokenRepoSuite) Test_CreateThenFindByHash() {
	user := s.createTestUser("Alice", "alice@example.com")
	expiresAt := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Second)

	tok := domain.AccessToken{
		UserID:    user.ID,
		Name:      "cli-token",
		TokenHash: "hash-alice-12345",
		Prefix:    "omed_pat_abc123",
		ExpiresAt: expiresAt,
	}

	created, err := s.repo.Create(s.ctx, tok)
	s.Require().NoError(err)
	s.NotZero(created.ID)
	s.Equal(user.ID, created.UserID)
	s.Equal("cli-token", created.Name)
	s.Equal("hash-alice-12345", created.TokenHash)
	s.Equal("omed_pat_abc123", created.Prefix)
	s.True(created.ExpiresAt.Equal(expiresAt))
	s.Nil(created.RevokedAt)
	s.False(created.CreatedAt.IsZero())

	found, err := s.repo.FindByHash(s.ctx, "hash-alice-12345")
	s.Require().NoError(err)
	s.Equal(created.ID, found.ID)
	s.Equal(user.ID, found.UserID)
	s.Equal(domain.AccessTokenActive, found.Status(time.Now()))
}

func (s *AccessTokenRepoSuite) Test_FindByHash_Unknown_ReturnsNotFound() {
	_, err := s.repo.FindByHash(s.ctx, "nonexistent-hash")
	s.Require().ErrorIs(err, base.ErrNotFound)
}

func (s *AccessTokenRepoSuite) Test_ListByUser_OnlyOwn_NewestFirst() {
	bob := s.createTestUser("Bob", "bob@example.com")
	charlie := s.createTestUser("Charlie", "charlie@example.com")

	t1, err := s.repo.Create(s.ctx, domain.AccessToken{
		UserID:    bob.ID,
		Name:      "token-1",
		TokenHash: "hash-bob-1",
		Prefix:    "omed_pat_1",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	s.Require().NoError(err)

	time.Sleep(10 * time.Millisecond)

	t2, err := s.repo.Create(s.ctx, domain.AccessToken{
		UserID:    bob.ID,
		Name:      "token-2",
		TokenHash: "hash-bob-2",
		Prefix:    "omed_pat_2",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	s.Require().NoError(err)

	_, err = s.repo.Create(s.ctx, domain.AccessToken{
		UserID:    charlie.ID,
		Name:      "token-charlie",
		TokenHash: "hash-charlie-1",
		Prefix:    "omed_pat_c",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	s.Require().NoError(err)

	list, err := s.repo.ListByUser(s.ctx, bob.ID)
	s.Require().NoError(err)
	s.Require().Len(list, 2)
	// Newest first: t2 then t1
	s.Equal(t2.ID, list[0].ID)
	s.Equal(t1.ID, list[1].ID)

	// User with no tokens returns non-nil empty slice
	emptyList, err := s.repo.ListByUser(s.ctx, 999999)
	s.Require().NoError(err)
	s.NotNil(emptyList)
	s.Empty(emptyList)
}

func (s *AccessTokenRepoSuite) Test_CountActiveByUser_ExcludesExpiredAndRevoked() {
	user := s.createTestUser("Dave", "dave@example.com")
	now := time.Now().Truncate(time.Second)
	revokedAt := now.Add(-10 * time.Minute)

	// 1. Active token
	_, err := s.repo.Create(s.ctx, domain.AccessToken{
		UserID:    user.ID,
		Name:      "active-token",
		TokenHash: "hash-active",
		Prefix:    "omed_pat_act",
		ExpiresAt: now.Add(2 * time.Hour),
	})
	s.Require().NoError(err)

	// 2. Expired token
	_, err = s.repo.Create(s.ctx, domain.AccessToken{
		UserID:    user.ID,
		Name:      "expired-token",
		TokenHash: "hash-expired",
		Prefix:    "omed_pat_exp",
		ExpiresAt: now.Add(-1 * time.Hour),
	})
	s.Require().NoError(err)

	// 3. Revoked token
	_, err = s.repo.Create(s.ctx, domain.AccessToken{
		UserID:    user.ID,
		Name:      "revoked-token",
		TokenHash: "hash-revoked",
		Prefix:    "omed_pat_rev",
		ExpiresAt: now.Add(2 * time.Hour),
		RevokedAt: &revokedAt,
	})
	s.Require().NoError(err)

	// 4. Both expired and revoked
	_, err = s.repo.Create(s.ctx, domain.AccessToken{
		UserID:    user.ID,
		Name:      "both-token",
		TokenHash: "hash-both",
		Prefix:    "omed_pat_both",
		ExpiresAt: now.Add(-2 * time.Hour),
		RevokedAt: &revokedAt,
	})
	s.Require().NoError(err)

	count, err := s.repo.CountActiveByUser(s.ctx, user.ID, now)
	s.Require().NoError(err)
	s.Equal(1, count)
}

func (s *AccessTokenRepoSuite) Test_Revoke_Idempotent_KeepsFirstRevokedAt() {
	user := s.createTestUser("Eve", "eve@example.com")
	token, err := s.repo.Create(s.ctx, domain.AccessToken{
		UserID:    user.ID,
		Name:      "token-to-revoke",
		TokenHash: "hash-eve-1",
		Prefix:    "omed_pat_eve",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	s.Require().NoError(err)

	t1 := time.Now().Add(-1 * time.Hour).Truncate(time.Second)
	err = s.repo.Revoke(s.ctx, user.ID, token.ID, t1)
	s.Require().NoError(err)

	found1, err := s.repo.FindByHash(s.ctx, "hash-eve-1")
	s.Require().NoError(err)
	s.Require().NotNil(found1.RevokedAt)
	s.True(found1.RevokedAt.Equal(t1))
	s.Equal(domain.AccessTokenRevoked, found1.Status(time.Now()))

	// Second revoke at t2 should be idempotent and keep t1
	t2 := time.Now().Truncate(time.Second)
	err = s.repo.Revoke(s.ctx, user.ID, token.ID, t2)
	s.Require().NoError(err)

	found2, err := s.repo.FindByHash(s.ctx, "hash-eve-1")
	s.Require().NoError(err)
	s.Require().NotNil(found2.RevokedAt)
	s.True(found2.RevokedAt.Equal(t1), "revoked_at should remain t1, got %v", found2.RevokedAt)
}

func (s *AccessTokenRepoSuite) Test_Revoke_OtherUsersToken_ReturnsNotFound() {
	frank := s.createTestUser("Frank", "frank@example.com")
	grace := s.createTestUser("Grace", "grace@example.com")

	token, err := s.repo.Create(s.ctx, domain.AccessToken{
		UserID:    frank.ID,
		Name:      "frank-token",
		TokenHash: "hash-frank-1",
		Prefix:    "omed_pat_frk",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	s.Require().NoError(err)

	// Grace tries to revoke Frank's token
	err = s.repo.Revoke(s.ctx, grace.ID, token.ID, time.Now())
	s.Require().ErrorIs(err, base.ErrNotFound)

	// Non-existent token ID
	err = s.repo.Revoke(s.ctx, frank.ID, 999999, time.Now())
	s.Require().ErrorIs(err, base.ErrNotFound)
}

func (s *AccessTokenRepoSuite) Test_DeleteUser_CascadesTokens() {
	heidi := s.createTestUser("Heidi", "heidi@example.com")
	token, err := s.repo.Create(s.ctx, domain.AccessToken{
		UserID:    heidi.ID,
		Name:      "heidi-token",
		TokenHash: "hash-heidi-1",
		Prefix:    "omed_pat_hdi",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	s.Require().NoError(err)

	found, err := s.repo.FindByHash(s.ctx, token.TokenHash)
	s.Require().NoError(err)
	s.Equal(token.ID, found.ID)

	// Delete user Heidi
	ctx := base.WithSkipAuditMutator(context.Background())
	err = s.client.User.DeleteOneID(heidi.ID).Exec(ctx)
	s.Require().NoError(err)

	// Token should be cascade-deleted
	_, err = s.repo.FindByHash(s.ctx, token.TokenHash)
	s.Require().ErrorIs(err, base.ErrNotFound)
}

func (s *AccessTokenRepoSuite) Test_AccessToken_Status_RevokedWinsOverExpired() {
	now := time.Now().Truncate(time.Second)
	revokedAt := now.Add(-2 * time.Hour)

	// Expired and revoked: revoked must win
	tok := domain.AccessToken{
		ExpiresAt: now.Add(-1 * time.Hour),
		RevokedAt: &revokedAt,
	}
	s.Equal(domain.AccessTokenRevoked, tok.Status(now))

	// Just expired
	tokExpired := domain.AccessToken{
		ExpiresAt: now.Add(-1 * time.Hour),
	}
	s.Equal(domain.AccessTokenExpired, tokExpired.Status(now))

	// Active
	tokActive := domain.AccessToken{
		ExpiresAt: now.Add(1 * time.Hour),
	}
	s.Equal(domain.AccessTokenActive, tokActive.Status(now))
}

func TestAccessTokenRepoSuite(t *testing.T) {
	suite.Run(t, new(AccessTokenRepoSuite))
}
