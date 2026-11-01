package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/loginotp"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/repository"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type OTPRepoSuite struct {
	suite.Suite
	client *ent.Client
	repo   repository.OTPRepository
	ctx    context.Context
}

func (s *OTPRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:otp_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.repo = repository.NewOTPRepository(s.client)
	s.ctx = context.Background()
}

func (s *OTPRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *OTPRepoSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.LoginOTP.Delete().Exec(ctx)
}

func (s *OTPRepoSuite) Test_Issue_SupersedesPreviousUnconsumed() {
	now := time.Now().Truncate(time.Second)
	email := "user@example.com"

	otpA := domain.OTP{
		Email:     email,
		CodeHash:  "hashA",
		ExpiresAt: now.Add(15 * time.Minute),
		IPAddress: "127.0.0.1",
	}
	issuedA, err := s.repo.Issue(s.ctx, otpA)
	s.Require().NoError(err)
	s.Require().NotZero(issuedA.ID)
	s.Nil(issuedA.ConsumedAt)

	otpB := domain.OTP{
		Email:     email,
		CodeHash:  "hashB",
		ExpiresAt: now.Add(15 * time.Minute),
		IPAddress: "127.0.0.1",
	}
	issuedB, err := s.repo.Issue(s.ctx, otpB)
	s.Require().NoError(err)
	s.Require().NotZero(issuedB.ID)
	s.NotEqual(issuedA.ID, issuedB.ID)

	// FindActive should return B
	active, err := s.repo.FindActive(s.ctx, email, now)
	s.Require().NoError(err)
	s.Equal(issuedB.ID, active.ID)
	s.Equal("hashB", active.CodeHash)

	// A should be marked consumed
	rowA, err := s.client.LoginOTP.Get(s.ctx, issuedA.ID)
	s.Require().NoError(err)
	s.NotNil(rowA.ConsumedAt)
}

func (s *OTPRepoSuite) Test_Issue_DoesNotTouchOtherEmails() {
	now := time.Now().Truncate(time.Second)

	otpA := domain.OTP{
		Email:     "alice@example.com",
		CodeHash:  "hashA",
		ExpiresAt: now.Add(15 * time.Minute),
	}
	issuedA, err := s.repo.Issue(s.ctx, otpA)
	s.Require().NoError(err)

	otpB := domain.OTP{
		Email:     "bob@example.com",
		CodeHash:  "hashB",
		ExpiresAt: now.Add(15 * time.Minute),
	}
	issuedB, err := s.repo.Issue(s.ctx, otpB)
	s.Require().NoError(err)

	// Alice's OTP should still be active and unconsumed
	activeA, err := s.repo.FindActive(s.ctx, "alice@example.com", now)
	s.Require().NoError(err)
	s.Equal(issuedA.ID, activeA.ID)
	s.Nil(activeA.ConsumedAt)

	// Bob's OTP should also be active and unconsumed
	activeB, err := s.repo.FindActive(s.ctx, "bob@example.com", now)
	s.Require().NoError(err)
	s.Equal(issuedB.ID, activeB.ID)
	s.Nil(activeB.ConsumedAt)
}

func (s *OTPRepoSuite) Test_FindActive_IgnoresExpiredAndConsumed() {
	now := time.Now().Truncate(time.Second)

	// Case 1: Expired OTP
	expiredOTP := domain.OTP{
		Email:     "expired@example.com",
		CodeHash:  "hashExp",
		ExpiresAt: now.Add(-1 * time.Minute),
	}
	_, err := s.repo.Issue(s.ctx, expiredOTP)
	s.Require().NoError(err)

	_, err = s.repo.FindActive(s.ctx, "expired@example.com", now)
	s.Require().ErrorIs(err, base.ErrNotFound)

	// Case 2: Consumed OTP
	validOTP := domain.OTP{
		Email:     "consumed@example.com",
		CodeHash:  "hashCon",
		ExpiresAt: now.Add(15 * time.Minute),
	}
	issued, err := s.repo.Issue(s.ctx, validOTP)
	s.Require().NoError(err)

	consumed, err := s.repo.Consume(s.ctx, issued.ID, now)
	s.Require().NoError(err)
	s.True(consumed)

	_, err = s.repo.FindActive(s.ctx, "consumed@example.com", now)
	s.Require().ErrorIs(err, base.ErrNotFound)
}

func (s *OTPRepoSuite) Test_IncrementAttempts_StopsAtMax() {
	now := time.Now().Truncate(time.Second)
	otp := domain.OTP{
		Email:     "max@example.com",
		CodeHash:  "hashMax",
		ExpiresAt: now.Add(15 * time.Minute),
	}
	issued, err := s.repo.Issue(s.ctx, otp)
	s.Require().NoError(err)

	// max = 3
	c1, err := s.repo.IncrementAttempts(s.ctx, issued.ID, 3)
	s.Require().NoError(err)
	s.Equal(1, c1)

	c2, err := s.repo.IncrementAttempts(s.ctx, issued.ID, 3)
	s.Require().NoError(err)
	s.Equal(2, c2)

	c3, err := s.repo.IncrementAttempts(s.ctx, issued.ID, 3)
	s.Require().NoError(err)
	s.Equal(3, c3)

	// 4th attempt should exceed max and return base.ErrNotFound
	_, err = s.repo.IncrementAttempts(s.ctx, issued.ID, 3)
	s.Require().ErrorIs(err, base.ErrNotFound)
}

func (s *OTPRepoSuite) Test_IncrementAttempts_IgnoresConsumed() {
	now := time.Now().Truncate(time.Second)
	otp := domain.OTP{
		Email:     "consumed_inc@example.com",
		CodeHash:  "hashConsumedInc",
		ExpiresAt: now.Add(15 * time.Minute),
	}
	issued, err := s.repo.Issue(s.ctx, otp)
	s.Require().NoError(err)

	ok, err := s.repo.Consume(s.ctx, issued.ID, now)
	s.Require().NoError(err)
	s.True(ok)

	_, err = s.repo.IncrementAttempts(s.ctx, issued.ID, 5)
	s.Require().ErrorIs(err, base.ErrNotFound)
}

func (s *OTPRepoSuite) Test_Consume_OnlyOnce() {
	now := time.Now().Truncate(time.Second)
	otp := domain.OTP{
		Email:     "once@example.com",
		CodeHash:  "hashOnce",
		ExpiresAt: now.Add(15 * time.Minute),
	}
	issued, err := s.repo.Issue(s.ctx, otp)
	s.Require().NoError(err)

	// First call should succeed (true)
	first, err := s.repo.Consume(s.ctx, issued.ID, now)
	s.Require().NoError(err)
	s.True(first)

	// Second call should return false
	second, err := s.repo.Consume(s.ctx, issued.ID, now)
	s.Require().NoError(err)
	s.False(second)
}

func (s *OTPRepoSuite) Test_LastIssuedAt_And_CountSince() {
	now := time.Now().Truncate(time.Second)
	email := "rate@example.com"

	// Unknown email: ok=false, count=0
	lastTime, ok, err := s.repo.LastIssuedAt(s.ctx, "unknown@example.com")
	s.Require().NoError(err)
	s.False(ok)
	s.True(lastTime.IsZero())

	count, err := s.repo.CountSince(s.ctx, "unknown@example.com", now.Add(-1*time.Hour))
	s.Require().NoError(err)
	s.Equal(0, count)

	// Issue 1 at t - 10 min
	t1 := now.Add(-10 * time.Minute)
	_, err = s.repo.Issue(s.ctx, domain.OTP{
		Email:     email,
		CodeHash:  "hash1",
		ExpiresAt: t1.Add(15 * time.Minute),
		CreatedAt: t1,
	})
	s.Require().NoError(err)

	// Issue 2 at t - 5 min
	t2 := now.Add(-5 * time.Minute)
	_, err = s.repo.Issue(s.ctx, domain.OTP{
		Email:     email,
		CodeHash:  "hash2",
		ExpiresAt: t2.Add(15 * time.Minute),
		CreatedAt: t2,
	})
	s.Require().NoError(err)

	// Issue 3 at t
	t3 := now
	_, err = s.repo.Issue(s.ctx, domain.OTP{
		Email:     email,
		CodeHash:  "hash3",
		ExpiresAt: t3.Add(15 * time.Minute),
		CreatedAt: t3,
	})
	s.Require().NoError(err)

	// LastIssuedAt should return approximately t3 with ok=true
	lastTime, ok, err = s.repo.LastIssuedAt(s.ctx, email)
	s.Require().NoError(err)
	s.True(ok)
	s.WithinDuration(t3, lastTime, time.Second)

	// CountSince with since = t - 7 min should count t2 and t3 (2 rows)
	count, err = s.repo.CountSince(s.ctx, email, now.Add(-7*time.Minute))
	s.Require().NoError(err)
	s.Equal(2, count)

	// CountSince with since = t - 15 min should count all 3
	count, err = s.repo.CountSince(s.ctx, email, now.Add(-15*time.Minute))
	s.Require().NoError(err)
	s.Equal(3, count)

	// CountSince with since = t + 1 min should count 0
	count, err = s.repo.CountSince(s.ctx, email, now.Add(1*time.Minute))
	s.Require().NoError(err)
	s.Equal(0, count)
}

func (s *OTPRepoSuite) Test_DeleteOlderThan_ScopedToEmail() {
	now := time.Now().Truncate(time.Second)
	emailA := "target@example.com"
	emailB := "other@example.com"

	// Old row for emailA (t - 2h)
	_, err := s.repo.Issue(s.ctx, domain.OTP{
		Email:     emailA,
		CodeHash:  "hashAOld",
		ExpiresAt: now.Add(-1 * time.Hour),
		CreatedAt: now.Add(-2 * time.Hour),
	})
	s.Require().NoError(err)

	// New row for emailA (t)
	_, err = s.repo.Issue(s.ctx, domain.OTP{
		Email:     emailA,
		CodeHash:  "hashANew",
		ExpiresAt: now.Add(15 * time.Minute),
		CreatedAt: now,
	})
	s.Require().NoError(err)

	// Old row for emailB (t - 2h)
	_, err = s.repo.Issue(s.ctx, domain.OTP{
		Email:     emailB,
		CodeHash:  "hashBOld",
		ExpiresAt: now.Add(-1 * time.Hour),
		CreatedAt: now.Add(-2 * time.Hour),
	})
	s.Require().NoError(err)

	// Delete older than 1 hour ago for emailA
	err = s.repo.DeleteOlderThan(s.ctx, emailA, now.Add(-1*time.Hour))
	s.Require().NoError(err)

	// EmailA should have 1 row left (the new one)
	countA, err := s.client.LoginOTP.Query().
		Where(loginotp.EmailEQ(emailA)).
		Count(s.ctx)
	s.Require().NoError(err)
	s.Equal(1, countA)

	// EmailB's old row should still exist (unaffected)
	countB, err := s.client.LoginOTP.Query().
		Where(loginotp.EmailEQ(emailB)).
		Count(s.ctx)
	s.Require().NoError(err)
	s.Equal(1, countB)
}

func (s *OTPRepoSuite) Test_ErrorMapping() {
	itemInvalid := base.MapError(domain.ErrOTPInvalid)
	s.Equal(400, itemInvalid.Status)
	s.Equal("AUTH_OTP_INVALID", itemInvalid.Code)
	s.Equal("invalid or expired code", itemInvalid.Message)

	itemUnavailable := base.MapError(domain.ErrOTPUnavailable)
	s.Equal(503, itemUnavailable.Status)
	s.Equal("AUTH_OTP_UNAVAILABLE", itemUnavailable.Code)
	s.Equal("email sign-in is not available", itemUnavailable.Message)
}

func TestOTPRepoSuite(t *testing.T) {
	suite.Run(t, new(OTPRepoSuite))
}
