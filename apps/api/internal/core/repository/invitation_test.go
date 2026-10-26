package repository_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/core/repository"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type InvitationRepoSuite struct {
	suite.Suite
	client *ent.Client
	repo   repository.InvitationRepository
}

func (s *InvitationRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:invitation_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.repo = repository.NewInvitationRepository(s.client)
}

func (s *InvitationRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *InvitationRepoSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(base.WithSkipWorkspaceFilter(context.Background()))
	_, _ = s.client.WorkspaceInvitation.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)
}

func (s *InvitationRepoSuite) userCtx(uid int64) context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:            uid,
		Name:          fmt.Sprintf("User %d", uid),
		EmailVerified: true,
		Roles:         []base.UserRole{base.UserRoleUser},
	})
	return base.WithSkipWorkspaceFilter(ctx)
}

func (s *InvitationRepoSuite) createTestUser(name, email string) *ent.User {
	ctx := base.WithSkipAuditMutator(context.Background())
	u, err := s.client.User.Create().
		SetName(name).
		SetEmail(email).
		Save(ctx)
	s.Require().NoError(err)
	return u
}

func (s *InvitationRepoSuite) createTestWorkspace(name string, ownerID int64) *ent.Workspace {
	ctx := s.userCtx(ownerID)
	w, err := s.client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(ctx)
	s.Require().NoError(err)
	return w
}

func (s *InvitationRepoSuite) Test_Accept_Success() {
	owner := s.createTestUser("Owner", "owner@example.com")
	user := s.createTestUser("User", "user@example.com")
	ws := s.createTestWorkspace("Acme", owner.ID)
	ctx := s.userCtx(owner.ID)

	inv, err := s.repo.Create(ctx, ws.ID, model.CreateInvitationRequest{
		Roles:   []string{"core:member"},
		MaxUses: 2,
	}, "token-1")
	s.Require().NoError(err)

	m, err := s.repo.Accept(s.userCtx(user.ID), inv.ID, user.ID)
	s.Require().NoError(err)
	s.Equal(ws.ID, m.WorkspaceID)
	s.Equal(user.ID, m.User.ID)
	s.Equal([]string{"core:member"}, m.Roles)

	// Check DB state
	dbInv, err := s.repo.FindByID(ctx, ws.ID, inv.ID)
	s.Require().NoError(err)
	s.Equal(1, dbInv.UsesCount)
}

func (s *InvitationRepoSuite) Test_Accept_MaxUsesZero_Unlimited() {
	owner := s.createTestUser("Owner", "owner-zero@example.com")
	u1 := s.createTestUser("User 1", "u1-zero@example.com")
	u2 := s.createTestUser("User 2", "u2-zero@example.com")
	u3 := s.createTestUser("User 3", "u3-zero@example.com")
	ws := s.createTestWorkspace("Acme Zero", owner.ID)
	ctx := s.userCtx(owner.ID)

	inv, err := s.repo.Create(ctx, ws.ID, model.CreateInvitationRequest{
		Roles:   []string{"core:member"},
		MaxUses: 0, // unlimited
	}, "token-zero")
	s.Require().NoError(err)

	// User 1 accepts
	m1, err := s.repo.Accept(s.userCtx(u1.ID), inv.ID, u1.ID)
	s.Require().NoError(err)
	s.Equal(u1.ID, m1.User.ID)

	// User 2 accepts
	m2, err := s.repo.Accept(s.userCtx(u2.ID), inv.ID, u2.ID)
	s.Require().NoError(err)
	s.Equal(u2.ID, m2.User.ID)

	// User 3 accepts
	m3, err := s.repo.Accept(s.userCtx(u3.ID), inv.ID, u3.ID)
	s.Require().NoError(err)
	s.Equal(u3.ID, m3.User.ID)

	dbInv, err := s.repo.FindByID(ctx, ws.ID, inv.ID)
	s.Require().NoError(err)
	s.Equal(3, dbInv.UsesCount)
}

func (s *InvitationRepoSuite) Test_Accept_RollbackOnMemberFailure_AlreadyMember() {
	owner := s.createTestUser("Owner", "owner-rb@example.com")
	user := s.createTestUser("User", "user-rb@example.com")
	ws := s.createTestWorkspace("Acme Rollback", owner.ID)
	ctx := s.userCtx(owner.ID)

	// Pre-create member
	_, err := s.client.WorkspaceMember.Create().
		SetWorkspaceID(ws.ID).
		SetUserID(user.ID).
		SetRoles([]string{"core:viewer"}).
		Save(ctx)
	s.Require().NoError(err)

	inv, err := s.repo.Create(ctx, ws.ID, model.CreateInvitationRequest{
		Roles:   []string{"core:member"},
		MaxUses: 1,
	}, "token-rollback")
	s.Require().NoError(err)

	// Accept should fail because user is already a member
	_, err = s.repo.Accept(s.userCtx(user.ID), inv.ID, user.ID)
	s.Require().Error(err)
	s.True(errors.Is(err, domain.ErrAlreadyMember))

	// Verify that uses_count was rolled back and is still 0
	dbInv, err := s.repo.FindByID(ctx, ws.ID, inv.ID)
	s.Require().NoError(err)
	s.Equal(0, dbInv.UsesCount)
}

func (s *InvitationRepoSuite) Test_Accept_Concurrent_MaxUsesOne() {
	owner := s.createTestUser("Owner", "owner-concurrent@example.com")
	ws := s.createTestWorkspace("Acme Concurrent", owner.ID)
	ctx := s.userCtx(owner.ID)

	inv, err := s.repo.Create(ctx, ws.ID, model.CreateInvitationRequest{
		Roles:   []string{"core:member"},
		MaxUses: 1,
	}, "token-concurrent")
	s.Require().NoError(err)

	const numGoroutines = 10
	users := make([]*ent.User, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		users[i] = s.createTestUser(fmt.Sprintf("Concurrent User %d", i), fmt.Sprintf("concurrent-%d@example.com", i))
	}

	var wg sync.WaitGroup
	var successCount int64
	var maxUsesCount int64
	var otherErrors []error
	var mu sync.Mutex

	startBarrier := make(chan struct{})

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(u *ent.User) {
			defer wg.Done()
			<-startBarrier

			_, acceptErr := s.repo.Accept(s.userCtx(u.ID), inv.ID, u.ID)
			if acceptErr == nil {
				atomic.AddInt64(&successCount, 1)
			} else if errors.Is(acceptErr, domain.ErrInvitationMaxUsesReached) {
				atomic.AddInt64(&maxUsesCount, 1)
			} else {
				mu.Lock()
				otherErrors = append(otherErrors, acceptErr)
				mu.Unlock()
			}
		}(users[i])
	}

	// Release all goroutines simultaneously
	close(startBarrier)
	wg.Wait()

	s.Empty(otherErrors, "Expected no unexpected errors, got: %v", otherErrors)
	s.Equal(int64(1), successCount, "Exactly 1 concurrent accept must succeed")
	s.Equal(int64(numGoroutines-1), maxUsesCount, "All other concurrent accepts must fail with ErrInvitationMaxUsesReached")

	// Verify database state: uses_count must be exactly 1
	dbInv, err := s.repo.FindByID(ctx, ws.ID, inv.ID)
	s.Require().NoError(err)
	s.Equal(1, dbInv.UsesCount)

	// Verify only 1 member was added in DB
	memberCount, err := s.client.WorkspaceMember.Query().
		Where(
			// Skip owner, count invited members
		).
		Count(ctx)
	s.Require().NoError(err)
	s.Equal(1, memberCount)
}

func (s *InvitationRepoSuite) Test_Accept_Revoked_Fails() {
	owner := s.createTestUser("Owner", "owner-rev@example.com")
	user := s.createTestUser("User", "user-rev@example.com")
	ws := s.createTestWorkspace("Acme Revoked", owner.ID)
	ctx := s.userCtx(owner.ID)

	inv, err := s.repo.Create(ctx, ws.ID, model.CreateInvitationRequest{
		Roles:   []string{"core:member"},
		MaxUses: 5,
	}, "token-rev")
	s.Require().NoError(err)

	err = s.repo.Revoke(ctx, ws.ID, inv.ID)
	s.Require().NoError(err)

	_, err = s.repo.Accept(s.userCtx(user.ID), inv.ID, user.ID)
	s.Require().ErrorIs(err, domain.ErrInvitationRevoked)
}

func (s *InvitationRepoSuite) Test_Accept_Expired_Fails() {
	owner := s.createTestUser("Owner", "owner-exp@example.com")
	user := s.createTestUser("User", "user-exp@example.com")
	ws := s.createTestWorkspace("Acme Expired", owner.ID)
	ctx := s.userCtx(owner.ID)

	past := time.Now().Add(-1 * time.Hour)
	inv, err := s.repo.Create(ctx, ws.ID, model.CreateInvitationRequest{
		Roles:     []string{"core:member"},
		MaxUses:   5,
		ExpiresAt: &past,
	}, "token-exp")
	s.Require().NoError(err)

	_, err = s.repo.Accept(s.userCtx(user.ID), inv.ID, user.ID)
	s.Require().ErrorIs(err, domain.ErrInvitationExpired)
}

func (s *InvitationRepoSuite) Test_IncrementUses_Conditional() {
	owner := s.createTestUser("Owner", "owner-inc@example.com")
	ws := s.createTestWorkspace("Acme Inc", owner.ID)
	ctx := s.userCtx(owner.ID)

	inv, err := s.repo.Create(ctx, ws.ID, model.CreateInvitationRequest{
		Roles:   []string{"core:member"},
		MaxUses: 1,
	}, "token-inc")
	s.Require().NoError(err)

	// First increment succeeds
	err = s.repo.IncrementUses(ctx, inv.ID)
	s.Require().NoError(err)

	// Second increment fails because MaxUses=1 reached
	err = s.repo.IncrementUses(ctx, inv.ID)
	s.Require().ErrorIs(err, domain.ErrInvitationMaxUsesReached)
}

func TestInvitationRepoSuite(t *testing.T) {
	suite.Run(t, new(InvitationRepoSuite))
}
