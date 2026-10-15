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

type SessionRepoSuite struct {
	suite.Suite
	client *ent.Client
	repo   repository.SessionRepository
	ctx    context.Context
}

func (s *SessionRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:session_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.repo = repository.NewSessionRepository(s.client)
	s.ctx = testutil.NewUserContext()
}

func (s *SessionRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *SessionRepoSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.Session.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)
}

func (s *SessionRepoSuite) createTestUser(name, email string) *ent.User {
	ctx := context.Background()
	u, err := s.client.User.Create().
		SetName(name).
		SetEmail(email).
		Save(ctx)
	s.Require().NoError(err)
	return u
}

func (s *SessionRepoSuite) Test_CreateAndFindByToken() {
	user := s.createTestUser("Alice", "alice@example.com")

	sess := domain.Session{
		UserID:    user.ID,
		Token:     "test-token-12345",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "127.0.0.1",
		UserAgent: "Mozilla/5.0",
	}

	created, err := s.repo.Create(s.ctx, sess)
	s.Require().NoError(err)
	s.Equal(user.ID, created.UserID)
	s.Equal("test-token-12345", created.Token)
	s.Equal("127.0.0.1", created.IPAddress)
	s.Equal("Mozilla/5.0", created.UserAgent)

	found, err := s.repo.FindByToken(s.ctx, "test-token-12345")
	s.Require().NoError(err)
	s.Equal(created.ID, found.ID)
	s.Equal(user.ID, found.UserID)
}

func (s *SessionRepoSuite) Test_FindByUserID() {
	user := s.createTestUser("Bob", "bob@example.com")

	_, err := s.repo.Create(s.ctx, domain.Session{
		UserID:    user.ID,
		Token:     "token-web",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		UserAgent: "Chrome Web",
	})
	s.Require().NoError(err)

	_, err = s.repo.Create(s.ctx, domain.Session{
		UserID:    user.ID,
		Token:     "token-native",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		UserAgent: "iOS Native",
	})
	s.Require().NoError(err)

	sessions, err := s.repo.FindByUserID(s.ctx, user.ID)
	s.Require().NoError(err)
	s.Len(sessions, 2)
}

func (s *SessionRepoSuite) Test_DeleteByToken() {
	user := s.createTestUser("Charlie", "charlie@example.com")

	_, err := s.repo.Create(s.ctx, domain.Session{
		UserID:    user.ID,
		Token:     "token-to-delete",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	})
	s.Require().NoError(err)

	err = s.repo.DeleteByToken(s.ctx, "token-to-delete")
	s.Require().NoError(err)

	_, err = s.repo.FindByToken(s.ctx, "token-to-delete")
	s.Require().ErrorIs(err, base.ErrNotFound)
}

func (s *SessionRepoSuite) Test_DeleteExpired() {
	user := s.createTestUser("Dave", "dave@example.com")

	// Expired session
	_, err := s.repo.Create(s.ctx, domain.Session{
		UserID:    user.ID,
		Token:     "token-expired",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})
	s.Require().NoError(err)

	// Valid session
	_, err = s.repo.Create(s.ctx, domain.Session{
		UserID:    user.ID,
		Token:     "token-valid",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	})
	s.Require().NoError(err)

	deletedCount, err := s.repo.DeleteExpired(s.ctx)
	s.Require().NoError(err)
	s.Equal(1, deletedCount)

	_, err = s.repo.FindByToken(s.ctx, "token-expired")
	s.Require().ErrorIs(err, base.ErrNotFound)

	_, err = s.repo.FindByToken(s.ctx, "token-valid")
	s.Require().NoError(err)
}

func TestSessionRepoSuite(t *testing.T) {
	suite.Run(t, new(SessionRepoSuite))
}
