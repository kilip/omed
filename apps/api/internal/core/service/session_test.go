package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/repository"
	"github.com/kilip/omed/api/internal/core/service"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type SessionServiceSuite struct {
	suite.Suite
	client      *ent.Client
	sessionRepo repository.SessionRepository
	userRepo    repository.UserRepository
	svc         service.SessionService
	ctx         context.Context
}

func (s *SessionServiceSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:session_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.sessionRepo = repository.NewSessionRepository(s.client)
	s.userRepo = repository.NewUserRepository(s.client)
	s.svc = service.NewSessionService(s.sessionRepo, s.userRepo, 24*time.Hour)
	s.ctx = testutil.NewAdminContext()
}

func (s *SessionServiceSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *SessionServiceSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.Session.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)
}

func (s *SessionServiceSuite) createTestUser(name, email string) *ent.User {
	ctx := context.Background()
	u, err := s.client.User.Create().
		SetName(name).
		SetEmail(email).
		Save(ctx)
	s.Require().NoError(err)
	return u
}

func (s *SessionServiceSuite) Test_CreateAndValidateSession() {
	user := s.createTestUser("Alice", "alice@example.com")

	sess, err := s.svc.CreateSession(s.ctx, user.ID, "192.168.1.1", "Chrome")
	s.Require().NoError(err)
	s.NotEmpty(sess.Token)
	s.Equal(user.ID, sess.UserID)
	s.Equal("192.168.1.1", sess.IPAddress)
	s.Equal("Chrome", sess.UserAgent)

	validSess, validUser, err := s.svc.ValidateSession(s.ctx, sess.Token)
	s.Require().NoError(err)
	s.Equal(sess.ID, validSess.ID)
	s.Equal(user.ID, validUser.ID)
	s.Equal(user.Email, validUser.Email)
}

func (s *SessionServiceSuite) Test_ValidateSession_Expired() {
	user := s.createTestUser("Bob", "bob@example.com")

	// Create an expired session directly in repository
	expiredSess, err := s.sessionRepo.Create(s.ctx, domain.Session{
		UserID:    user.ID,
		Token:     "expired-token",
		ExpiresAt: time.Now().Add(-1 * time.Hour),
	})
	s.Require().NoError(err)

	_, _, err = s.svc.ValidateSession(s.ctx, expiredSess.Token)
	s.Require().ErrorIs(err, domain.ErrSessionExpired)
}

func (s *SessionServiceSuite) Test_ValidateSession_NotFound() {
	_, _, err := s.svc.ValidateSession(s.ctx, "non-existent-token")
	s.Require().ErrorIs(err, domain.ErrSessionNotFound)
}

func (s *SessionServiceSuite) Test_RevokeSession() {
	user := s.createTestUser("Charlie", "charlie@example.com")

	sess, err := s.svc.CreateSession(s.ctx, user.ID, "127.0.0.1", "Firefox")
	s.Require().NoError(err)

	err = s.svc.RevokeSession(s.ctx, sess.Token)
	s.Require().NoError(err)

	_, _, err = s.svc.ValidateSession(s.ctx, sess.Token)
	s.Require().ErrorIs(err, domain.ErrSessionNotFound)
}

func (s *SessionServiceSuite) Test_RevokeAllUserSessions() {
	user := s.createTestUser("Dave", "dave@example.com")

	sess1, err := s.svc.CreateSession(s.ctx, user.ID, "127.0.0.1", "Web")
	s.Require().NoError(err)

	sess2, err := s.svc.CreateSession(s.ctx, user.ID, "127.0.0.1", "Native Mobile")
	s.Require().NoError(err)

	err = s.svc.RevokeAllUserSessions(s.ctx, user.ID)
	s.Require().NoError(err)

	_, _, err = s.svc.ValidateSession(s.ctx, sess1.Token)
	s.Require().ErrorIs(err, domain.ErrSessionNotFound)

	_, _, err = s.svc.ValidateSession(s.ctx, sess2.Token)
	s.Require().ErrorIs(err, domain.ErrSessionNotFound)
}

func TestSessionServiceSuite(t *testing.T) {
	suite.Run(t, new(SessionServiceSuite))
}
