package service_test

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
	"github.com/kilip/omed/api/internal/core/service"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type InvitationServiceSuite struct {
	suite.Suite
	client      *ent.Client
	invRepo     repository.InvitationRepository
	wsRepo      repository.WorkspaceRepository
	memberRepo  repository.MemberRepository
	userRepo    repository.UserRepository
	userSvc     service.UserService
	wsSvc       service.WorkspaceService
	svc         service.InvitationService
}

func (s *InvitationServiceSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:invitation_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.invRepo = repository.NewInvitationRepository(s.client)
	s.wsRepo = repository.NewWorkspaceRepository(s.client)
	s.memberRepo = repository.NewMemberRepository(s.client)
	s.userRepo = repository.NewUserRepository(s.client)
	s.userSvc = service.NewUserService(s.userRepo)
	s.wsSvc = service.NewWorkspaceService(s.wsRepo, nil)
	s.svc = service.NewInvitationService(s.invRepo, s.wsSvc, s.memberRepo, s.userSvc, nil, "http://localhost:3000")
}

func (s *InvitationServiceSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *InvitationServiceSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(base.WithSkipWorkspaceFilter(context.Background()))
	_, _ = s.client.WorkspaceInvitation.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)
}

func (s *InvitationServiceSuite) userCtx(uid int64) context.Context {
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:            uid,
		Name:          fmt.Sprintf("User %d", uid),
		EmailVerified: true,
		Roles:         []base.UserRole{base.UserRoleUser},
	})
	return base.WithSkipWorkspaceFilter(ctx)
}

func (s *InvitationServiceSuite) createTestUser(name, email string) *ent.User {
	ctx := base.WithSkipAuditMutator(context.Background())
	u, err := s.client.User.Create().
		SetName(name).
		SetEmail(email).
		Save(ctx)
	s.Require().NoError(err)
	return u
}

func (s *InvitationServiceSuite) createTestWorkspace(name string, ownerID int64, personal bool) *ent.Workspace {
	ctx := s.userCtx(ownerID)
	w, err := s.client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		SetPersonal(personal).
		Save(ctx)
	s.Require().NoError(err)
	return w
}

func (s *InvitationServiceSuite) Test_Accept_Success() {
	owner := s.createTestUser("Owner", "owner-svc@example.com")
	user := s.createTestUser("User", "user-svc@example.com")
	ws := s.createTestWorkspace("Acme Svc", owner.ID, false)
	ctx := s.userCtx(owner.ID)

	inv, err := s.svc.Create(ctx, ws.ID, model.CreateInvitationRequest{
		Roles:   []string{"core:member"},
		MaxUses: 2,
	})
	s.Require().NoError(err)

	m, err := s.svc.Accept(s.userCtx(user.ID), inv.Token, user.ID)
	s.Require().NoError(err)
	s.Equal(ws.ID, m.WorkspaceID)
	s.Equal(user.ID, m.User.ID)
	s.Equal([]string{"core:member"}, m.Roles)

	// Verify uses_count in DB
	dbInv, err := s.invRepo.FindByID(ctx, ws.ID, inv.ID)
	s.Require().NoError(err)
	s.Equal(1, dbInv.UsesCount)
}

func (s *InvitationServiceSuite) Test_Accept_PersonalWorkspace_Forbidden() {
	owner := s.createTestUser("Owner", "owner-pw@example.com")
	user := s.createTestUser("User", "user-pw@example.com")
	pw := s.createTestWorkspace("Personal WS", owner.ID, true)
	ctx := s.userCtx(owner.ID)

	// Bypass service Create which forbids personal workspace
	inv, err := s.invRepo.Create(ctx, pw.ID, model.CreateInvitationRequest{
		Roles: []string{"core:member"},
	}, "personal-token")
	s.Require().NoError(err)

	_, err = s.svc.Accept(s.userCtx(user.ID), inv.Token, user.ID)
	s.Require().ErrorIs(err, domain.ErrPersonalWorkspaceMember)
}

func (s *InvitationServiceSuite) Test_Accept_OwnerForbidden() {
	owner := s.createTestUser("Owner", "owner-self@example.com")
	ws := s.createTestWorkspace("Acme Owner", owner.ID, false)
	ctx := s.userCtx(owner.ID)

	inv, err := s.svc.Create(ctx, ws.ID, model.CreateInvitationRequest{
		Roles: []string{"core:member"},
	})
	s.Require().NoError(err)

	_, err = s.svc.Accept(s.userCtx(owner.ID), inv.Token, owner.ID)
	s.Require().ErrorIs(err, domain.ErrAlreadyMember)
}

func (s *InvitationServiceSuite) Test_Accept_AlreadyMember() {
	owner := s.createTestUser("Owner", "owner-alr@example.com")
	user := s.createTestUser("User", "user-alr@example.com")
	ws := s.createTestWorkspace("Acme Member", owner.ID, false)
	ctx := s.userCtx(owner.ID)

	// Add member first
	_, err := s.memberRepo.Create(ctx, ws.ID, user.ID, []string{"core:member"})
	s.Require().NoError(err)

	inv, err := s.svc.Create(ctx, ws.ID, model.CreateInvitationRequest{
		Roles:   []string{"core:member"},
		MaxUses: 1,
	})
	s.Require().NoError(err)

	_, err = s.svc.Accept(s.userCtx(user.ID), inv.Token, user.ID)
	s.Require().ErrorIs(err, domain.ErrAlreadyMember)

	// Check uses_count is not incremented
	dbInv, err := s.invRepo.FindByID(ctx, ws.ID, inv.ID)
	s.Require().NoError(err)
	s.Equal(0, dbInv.UsesCount)
}

func (s *InvitationServiceSuite) Test_Accept_Revoked() {
	owner := s.createTestUser("Owner", "owner-revsvc@example.com")
	user := s.createTestUser("User", "user-revsvc@example.com")
	ws := s.createTestWorkspace("Acme Rev", owner.ID, false)
	ctx := s.userCtx(owner.ID)

	inv, err := s.svc.Create(ctx, ws.ID, model.CreateInvitationRequest{
		Roles: []string{"core:member"},
	})
	s.Require().NoError(err)

	err = s.svc.Revoke(ctx, ws.ID, inv.ID)
	s.Require().NoError(err)

	_, err = s.svc.Accept(s.userCtx(user.ID), inv.Token, user.ID)
	s.Require().ErrorIs(err, domain.ErrInvitationRevoked)
}

func (s *InvitationServiceSuite) Test_Accept_Expired() {
	owner := s.createTestUser("Owner", "owner-expsvc@example.com")
	user := s.createTestUser("User", "user-expsvc@example.com")
	ws := s.createTestWorkspace("Acme Exp", owner.ID, false)
	ctx := s.userCtx(owner.ID)

	past := time.Now().Add(-2 * time.Hour)
	inv, err := s.invRepo.Create(ctx, ws.ID, model.CreateInvitationRequest{
		Roles:     []string{"core:member"},
		ExpiresAt: &past,
	}, "token-expired-svc")
	s.Require().NoError(err)

	_, err = s.svc.Accept(s.userCtx(user.ID), inv.Token, user.ID)
	s.Require().ErrorIs(err, domain.ErrInvitationExpired)
}

func (s *InvitationServiceSuite) Test_Accept_MaxUsesReached() {
	owner := s.createTestUser("Owner", "owner-max@example.com")
	u1 := s.createTestUser("User 1", "u1-max@example.com")
	u2 := s.createTestUser("User 2", "u2-max@example.com")
	ws := s.createTestWorkspace("Acme Max", owner.ID, false)
	ctx := s.userCtx(owner.ID)

	inv, err := s.svc.Create(ctx, ws.ID, model.CreateInvitationRequest{
		Roles:   []string{"core:member"},
		MaxUses: 1,
	})
	s.Require().NoError(err)

	// First user succeeds
	_, err = s.svc.Accept(s.userCtx(u1.ID), inv.Token, u1.ID)
	s.Require().NoError(err)

	// Second user fails
	_, err = s.svc.Accept(s.userCtx(u2.ID), inv.Token, u2.ID)
	s.Require().ErrorIs(err, domain.ErrInvitationMaxUsesReached)
}

func (s *InvitationServiceSuite) Test_Accept_Concurrent_MaxUsesOne() {
	owner := s.createTestUser("Owner", "owner-svcconcurrent@example.com")
	ws := s.createTestWorkspace("Acme Svc Concurrent", owner.ID, false)
	ctx := s.userCtx(owner.ID)

	inv, err := s.svc.Create(ctx, ws.ID, model.CreateInvitationRequest{
		Roles:   []string{"core:member"},
		MaxUses: 1,
	})
	s.Require().NoError(err)

	const numGoroutines = 10
	users := make([]*ent.User, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		users[i] = s.createTestUser(fmt.Sprintf("Svc Concurrent User %d", i), fmt.Sprintf("svc-concurrent-%d@example.com", i))
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

			_, acceptErr := s.svc.Accept(s.userCtx(u.ID), inv.Token, u.ID)
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

	close(startBarrier)
	wg.Wait()

	s.Empty(otherErrors, "Expected no unexpected errors, got: %v", otherErrors)
	s.Equal(int64(1), successCount, "Exactly 1 concurrent accept must succeed")
	s.Equal(int64(numGoroutines-1), maxUsesCount, "All other concurrent accepts must fail with ErrInvitationMaxUsesReached")

	dbInv, err := s.invRepo.FindByID(ctx, ws.ID, inv.ID)
	s.Require().NoError(err)
	s.Equal(1, dbInv.UsesCount)
}

func (s *InvitationServiceSuite) Test_Accept_MaxUsesZero_Unlimited() {
	owner := s.createTestUser("Owner", "owner-svczero@example.com")
	u1 := s.createTestUser("User 1", "u1-svczero@example.com")
	u2 := s.createTestUser("User 2", "u2-svczero@example.com")
	ws := s.createTestWorkspace("Acme Svc Zero", owner.ID, false)
	ctx := s.userCtx(owner.ID)

	inv, err := s.svc.Create(ctx, ws.ID, model.CreateInvitationRequest{
		Roles:   []string{"core:member"},
		MaxUses: 0,
	})
	s.Require().NoError(err)

	m1, err := s.svc.Accept(s.userCtx(u1.ID), inv.Token, u1.ID)
	s.Require().NoError(err)
	s.Equal(u1.ID, m1.User.ID)

	m2, err := s.svc.Accept(s.userCtx(u2.ID), inv.Token, u2.ID)
	s.Require().NoError(err)
	s.Equal(u2.ID, m2.User.ID)

	dbInv, err := s.invRepo.FindByID(ctx, ws.ID, inv.ID)
	s.Require().NoError(err)
	s.Equal(2, dbInv.UsesCount)
}

func TestInvitationServiceSuite(t *testing.T) {
	suite.Run(t, new(InvitationServiceSuite))
}
