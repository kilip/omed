package service_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/loginotp"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/core/repository"
	"github.com/kilip/omed/api/internal/core/service"
	"github.com/kilip/omed/api/internal/platform/mailx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type OTPServiceSuite struct {
	suite.Suite
	client      *ent.Client
	otpRepo     repository.OTPRepository
	userRepo    repository.UserRepository
	userSvc     service.UserService
	sessionRepo repository.SessionRepository
	sessionSvc  service.SessionService
	mailer      mailx.Mailer
	authSecret  string
	defaultCfg  service.OTPConfig
	ctx         context.Context
}

func TestOTPServiceSuite(t *testing.T) {
	suite.Run(t, new(OTPServiceSuite))
}

func (s *OTPServiceSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:otp_svc_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.otpRepo = repository.NewOTPRepository(s.client)
	s.userRepo = repository.NewUserRepository(s.client)
	s.userSvc = service.NewUserService(s.userRepo)
	s.sessionRepo = repository.NewSessionRepository(s.client)
	s.sessionSvc = service.NewSessionService(s.sessionRepo, s.userRepo, 24*time.Hour)

	var err error
	s.mailer, err = mailx.New(mailx.Config{Driver: "memory"})
	s.Require().NoError(err)

	s.authSecret = "test-auth-secret-for-otp-testing-32b"
	s.defaultCfg = service.OTPConfig{
		Enabled:        true,
		Length:         6,
		TTL:            10 * time.Minute,
		MaxAttempts:    3,
		ResendCooldown: 1 * time.Millisecond,
		HourlyLimit:    5,
	}
	s.ctx = context.Background()
}

func (s *OTPServiceSuite) TearDownSuite() {
	if s.mailer != nil {
		_ = s.mailer.Close()
	}
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *OTPServiceSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.LoginOTP.Delete().Exec(ctx)
	_, _ = s.client.Session.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)

	if memDriver, ok := s.mailer.Driver().(*mailx.MemoryDriver); ok {
		memDriver.Reset()
	}
}

func (s *OTPServiceSuite) createService(cfg service.OTPConfig, nowFn func() time.Time, mailer mailx.Mailer) service.OTPService {
	m := mailer
	if m == nil {
		m = s.mailer
	}
	return service.NewOTPService(service.OTPServiceParams{
		Repo:       s.otpRepo,
		Users:      s.userSvc,
		Sessions:   s.sessionSvc,
		Mailer:     m,
		AuthSecret: s.authSecret,
		Config:     cfg,
		Now:        nowFn,
	})
}

func (s *OTPServiceSuite) extractCodeFromMail(expectedCount int) string {
	memDriver := s.mailer.Driver().(*mailx.MemoryDriver)
	var messages []mailx.Message
	s.Eventually(func() bool {
		messages = memDriver.SentMessages()
		return len(messages) >= expectedCount
	}, 2*time.Second, 10*time.Millisecond, "expected %d sent email(s)", expectedCount)

	s.Require().NotEmpty(messages)
	lastMsg := messages[len(messages)-1]

	codeRe := regexp.MustCompile(`\b\d{6}\b`)
	match := codeRe.FindString(lastMsg.TextBody)
	s.Require().NotEmpty(match, "could not find 6-digit code in mail TextBody: %s", lastMsg.TextBody)
	return match
}

// Request tests

func (s *OTPServiceSuite) Test_Request_SendsCodeAndStoresOnlyHash() {
	fixedTime := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	svc := s.createService(s.defaultCfg, func() time.Time { return fixedTime }, nil)

	email := "user@example.com"
	res, err := svc.Request(s.ctx, model.OTPRequest{
		Email:  email,
		Locale: "en",
	}, "127.0.0.1")

	s.Require().NoError(err)
	s.True(res.Success)
	s.Equal(6, res.CodeLength)
	s.Equal(int(10*time.Minute.Seconds()), res.TTLSeconds)
	s.Equal(int(1*time.Millisecond.Seconds()), res.ResendAfterSeconds)

	code := s.extractCodeFromMail(1)
	s.Len(code, 6)

	otpRecord, err := s.client.LoginOTP.Query().
		Where(loginotp.EmailEQ(email)).
		Only(s.ctx)
	s.Require().NoError(err)

	s.Len(otpRecord.CodeHash, 64)
	s.NotEqual(code, otpRecord.CodeHash)
	s.Equal(0, otpRecord.Attempts)
	s.Nil(otpRecord.ConsumedAt)
	s.Equal(fixedTime.Add(10*time.Minute), otpRecord.ExpiresAt)
}

func (s *OTPServiceSuite) Test_Request_ResponseIdenticalForKnownAndUnknownEmail() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, err := s.client.User.Create().
		SetName("Known User").
		SetEmail("known@example.com").
		SetEmailVerified(true).
		SetLocale("en").
		SetCurrency("USD").
		Save(ctx)
	s.Require().NoError(err)

	svc := s.createService(s.defaultCfg, nil, nil)

	resKnown, errKnown := svc.Request(s.ctx, model.OTPRequest{
		Email:  "known@example.com",
		Locale: "en",
	}, "127.0.0.1")
	s.Require().NoError(errKnown)

	resUnknown, errUnknown := svc.Request(s.ctx, model.OTPRequest{
		Email:  "unknown@example.com",
		Locale: "en",
	}, "127.0.0.1")
	s.Require().NoError(errUnknown)

	s.Equal(resKnown, resUnknown)
	s.True(resKnown.Success)
	s.Equal(6, resKnown.CodeLength)
	s.Equal(600, resKnown.TTLSeconds)
}

func (s *OTPServiceSuite) Test_Request_CooldownSkipsSend() {
	cfg := s.defaultCfg
	cfg.ResendCooldown = 60 * time.Second

	now := time.Now()
	svc := s.createService(cfg, func() time.Time { return now }, nil)

	email := "cooldown@example.com"
	res1, err := svc.Request(s.ctx, model.OTPRequest{Email: email}, "127.0.0.1")
	s.Require().NoError(err)

	_ = s.extractCodeFromMail(1)
	memDriver := s.mailer.Driver().(*mailx.MemoryDriver)
	s.Len(memDriver.SentMessages(), 1)

	// Second request within cooldown (time did not advance)
	res2, err := svc.Request(s.ctx, model.OTPRequest{Email: email}, "127.0.0.1")
	s.Require().NoError(err)
	s.Equal(res1, res2)

	// Ensure no second mail was sent
	time.Sleep(50 * time.Millisecond)
	s.Len(memDriver.SentMessages(), 1)
}

func (s *OTPServiceSuite) Test_Request_HourlyLimitSkipsSend() {
	cfg := s.defaultCfg
	cfg.HourlyLimit = 2
	cfg.ResendCooldown = 1 * time.Millisecond

	currentTime := time.Now()
	svc := s.createService(cfg, func() time.Time { return currentTime }, nil)
	memDriver := s.mailer.Driver().(*mailx.MemoryDriver)

	email := "limit@example.com"

	// Call 1: success, email sent
	res1, err := svc.Request(s.ctx, model.OTPRequest{Email: email}, "127.0.0.1")
	s.Require().NoError(err)
	s.True(res1.Success)
	_ = s.extractCodeFromMail(1)

	// Advance time past cooldown
	currentTime = currentTime.Add(10 * time.Millisecond)

	// Call 2: success, email sent
	res2, err := svc.Request(s.ctx, model.OTPRequest{Email: email}, "127.0.0.1")
	s.Require().NoError(err)
	s.True(res2.Success)
	_ = s.extractCodeFromMail(2)

	// Advance time past cooldown but still within 1 hour
	currentTime = currentTime.Add(10 * time.Millisecond)

	// Call 3: hourly limit reached, send skipped, response still success
	res3, err := svc.Request(s.ctx, model.OTPRequest{Email: email}, "127.0.0.1")
	s.Require().NoError(err)
	s.True(res3.Success)
	s.Equal(res1, res3)

	time.Sleep(50 * time.Millisecond)
	s.Len(memDriver.SentMessages(), 2)
}

func (s *OTPServiceSuite) Test_Request_Unavailable() {
	cfg := s.defaultCfg
	cfg.Enabled = false
	svc := s.createService(cfg, nil, nil)

	_, err := svc.Request(s.ctx, model.OTPRequest{Email: "user@example.com"}, "127.0.0.1")
	s.ErrorIs(err, domain.ErrOTPUnavailable)
}

type failingAsyncMailer struct {
	mailx.Mailer
}

func (f *failingAsyncMailer) SendTemplateAsync(ctx context.Context, tmplName string, data any, msg mailx.Message) error {
	return errors.New("simulated mailer failure")
}

func (s *OTPServiceSuite) Test_Request_MailerFailureStillSucceeds() {
	failing := &failingAsyncMailer{Mailer: s.mailer}
	svc := s.createService(s.defaultCfg, nil, failing)

	email := "mailer-fail@example.com"
	res, err := svc.Request(s.ctx, model.OTPRequest{Email: email}, "127.0.0.1")
	s.Require().NoError(err)
	s.True(res.Success)

	// OTP record was still created in the database
	exists, err := s.client.LoginOTP.Query().
		Where(loginotp.EmailEQ(email)).
		Exist(s.ctx)
	s.Require().NoError(err)
	s.True(exists)
}

func (s *OTPServiceSuite) Test_Request_UnsupportedLocaleFallsBackToEnglish() {
	svc := s.createService(s.defaultCfg, nil, nil)

	_, err := svc.Request(s.ctx, model.OTPRequest{
		Email:  "french@example.com",
		Locale: "fr-FR",
	}, "127.0.0.1")
	s.Require().NoError(err)

	_ = s.extractCodeFromMail(1)
	memDriver := s.mailer.Driver().(*mailx.MemoryDriver)
	lastMsg := memDriver.LastMessage()
	s.Require().NotNil(lastMsg)
	s.Equal("Your Omed sign-in code", lastMsg.Subject)
}

func (s *OTPServiceSuite) Test_Request_IndonesianLocale() {
	svc := s.createService(s.defaultCfg, nil, nil)

	_, err := svc.Request(s.ctx, model.OTPRequest{
		Email:  "indo@example.com",
		Locale: "id-ID",
	}, "127.0.0.1")
	s.Require().NoError(err)

	code := s.extractCodeFromMail(1)
	memDriver := s.mailer.Driver().(*mailx.MemoryDriver)
	lastMsg := memDriver.LastMessage()
	s.Require().NotNil(lastMsg)
	s.Equal("Kode masuk Omed Anda", lastMsg.Subject)
	s.Contains(lastMsg.TextBody, "menit")
	s.Contains(lastMsg.TextBody, code)
}

// Verify tests

func (s *OTPServiceSuite) Test_Verify_NewUser_ProvisionsVerifiedUserAndSession() {
	svc := s.createService(s.defaultCfg, nil, nil)
	email := "newuser@example.com"

	_, err := svc.Request(s.ctx, model.OTPRequest{Email: email}, "192.168.1.100")
	s.Require().NoError(err)

	code := s.extractCodeFromMail(1)

	sess, user, err := svc.Verify(s.ctx, model.OTPVerifyRequest{
		Email: email,
		Code:  code,
	}, "192.168.1.100", "Mozilla/5.0")

	s.Require().NoError(err)
	s.Equal(email, user.Email)
	s.True(user.EmailVerified)
	s.Equal(user.ID, sess.UserID)
	s.NotEmpty(sess.Token)

	// Validate session using SessionService
	valSess, valUser, err := s.sessionSvc.ValidateSession(s.ctx, sess.Token)
	s.Require().NoError(err)
	s.Equal(user.ID, valUser.ID)
	s.Equal(sess.ID, valSess.ID)
}

func (s *OTPServiceSuite) Test_Verify_ExistingUser_ReusesUser() {
	ctx := base.WithSkipAuditMutator(context.Background())
	existingUser, err := s.client.User.Create().
		SetName("Existing Person").
		SetEmail("existing@example.com").
		SetEmailVerified(true).
		SetLocale("en").
		SetCurrency("USD").
		Save(ctx)
	s.Require().NoError(err)

	svc := s.createService(s.defaultCfg, nil, nil)

	_, err = svc.Request(s.ctx, model.OTPRequest{Email: "existing@example.com"}, "127.0.0.1")
	s.Require().NoError(err)

	code := s.extractCodeFromMail(1)

	sess, user, err := svc.Verify(s.ctx, model.OTPVerifyRequest{
		Email: "existing@example.com",
		Code:  code,
	}, "127.0.0.1", "Agent")

	s.Require().NoError(err)
	s.Equal(existingUser.ID, user.ID)
	s.Equal("Existing Person", user.Name)
	s.Equal(existingUser.ID, sess.UserID)

	count, err := s.client.User.Query().Count(s.ctx)
	s.Require().NoError(err)
	s.Equal(1, count)
}

func (s *OTPServiceSuite) Test_Verify_WrongCodeCountsAttemptsThenLocks() {
	svc := s.createService(s.defaultCfg, nil, nil)
	email := "wrongcode@example.com"

	_, err := svc.Request(s.ctx, model.OTPRequest{Email: email}, "127.0.0.1")
	s.Require().NoError(err)

	realCode := s.extractCodeFromMail(1)
	wrongCode := "000000"
	if realCode == wrongCode {
		wrongCode = "111111"
	}

	// 3 wrong attempts (MaxAttempts = 3)
	for i := 1; i <= 3; i++ {
		_, _, err := svc.Verify(s.ctx, model.OTPVerifyRequest{
			Email: email,
			Code:  wrongCode,
		}, "127.0.0.1", "Agent")
		s.ErrorIs(err, domain.ErrOTPInvalid)

		rec, getErr := s.client.LoginOTP.Query().Where(loginotp.EmailEQ(email)).Only(s.ctx)
		s.Require().NoError(getErr)
		s.Equal(i, rec.Attempts)
	}

	// 4th attempt with CORRECT code should still fail because attempts >= max
	_, _, err = svc.Verify(s.ctx, model.OTPVerifyRequest{
		Email: email,
		Code:  realCode,
	}, "127.0.0.1", "Agent")
	s.ErrorIs(err, domain.ErrOTPInvalid)

	// Attempts count stays at 3
	rec, err := s.client.LoginOTP.Query().Where(loginotp.EmailEQ(email)).Only(s.ctx)
	s.Require().NoError(err)
	s.Equal(3, rec.Attempts)
}

func (s *OTPServiceSuite) Test_Verify_ReplayRejected() {
	svc := s.createService(s.defaultCfg, nil, nil)
	email := "replay@example.com"

	_, err := svc.Request(s.ctx, model.OTPRequest{Email: email}, "127.0.0.1")
	s.Require().NoError(err)

	code := s.extractCodeFromMail(1)

	// First verify succeeds
	_, _, err = svc.Verify(s.ctx, model.OTPVerifyRequest{
		Email: email,
		Code:  code,
	}, "127.0.0.1", "Agent")
	s.Require().NoError(err)

	// Replay attempt fails
	_, _, err = svc.Verify(s.ctx, model.OTPVerifyRequest{
		Email: email,
		Code:  code,
	}, "127.0.0.1", "Agent")
	s.ErrorIs(err, domain.ErrOTPInvalid)
}

func (s *OTPServiceSuite) Test_Verify_ExpiredRejected() {
	currentTime := time.Now()
	svc := s.createService(s.defaultCfg, func() time.Time { return currentTime }, nil)

	email := "expired@example.com"
	_, err := svc.Request(s.ctx, model.OTPRequest{Email: email}, "127.0.0.1")
	s.Require().NoError(err)

	code := s.extractCodeFromMail(1)

	// Advance time past TTL (10m)
	currentTime = currentTime.Add(11 * time.Minute)

	_, _, err = svc.Verify(s.ctx, model.OTPVerifyRequest{
		Email: email,
		Code:  code,
	}, "127.0.0.1", "Agent")
	s.ErrorIs(err, domain.ErrOTPInvalid)
}

func (s *OTPServiceSuite) Test_Verify_SupersededCodeRejected() {
	currentTime := time.Now()
	svc := s.createService(s.defaultCfg, func() time.Time { return currentTime }, nil)

	email := "superseded@example.com"

	// Request 1
	_, err := svc.Request(s.ctx, model.OTPRequest{Email: email}, "127.0.0.1")
	s.Require().NoError(err)
	code1 := s.extractCodeFromMail(1)

	// Advance past cooldown
	currentTime = currentTime.Add(10 * time.Millisecond)

	// Request 2
	_, err = svc.Request(s.ctx, model.OTPRequest{Email: email}, "127.0.0.1")
	s.Require().NoError(err)
	code2 := s.extractCodeFromMail(2)
	s.NotEqual(code1, code2)

	// Verify with code 1 fails (superseded)
	_, _, err = svc.Verify(s.ctx, model.OTPVerifyRequest{
		Email: email,
		Code:  code1,
	}, "127.0.0.1", "Agent")
	s.ErrorIs(err, domain.ErrOTPInvalid)

	// Verify with code 2 succeeds
	_, _, err = svc.Verify(s.ctx, model.OTPVerifyRequest{
		Email: email,
		Code:  code2,
	}, "127.0.0.1", "Agent")
	s.Require().NoError(err)
}

func (s *OTPServiceSuite) Test_Verify_NoActiveOTP() {
	svc := s.createService(s.defaultCfg, nil, nil)

	_, _, err := svc.Verify(s.ctx, model.OTPVerifyRequest{
		Email: "no-otp@example.com",
		Code:  "123456",
	}, "127.0.0.1", "Agent")
	s.ErrorIs(err, domain.ErrOTPInvalid)
}

func (s *OTPServiceSuite) Test_Verify_ConcurrentExactlyOneSuccess() {
	svc := s.createService(s.defaultCfg, nil, nil)
	email := "concurrent@example.com"

	_, err := svc.Request(s.ctx, model.OTPRequest{Email: email}, "127.0.0.1")
	s.Require().NoError(err)

	code := s.extractCodeFromMail(1)

	const goroutines = 20
	var wg sync.WaitGroup
	var successCount int32
	var invalidCount int32

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			_, _, verifyErr := svc.Verify(s.ctx, model.OTPVerifyRequest{
				Email: email,
				Code:  code,
			}, fmt.Sprintf("192.168.1.%d", idx), "Agent")
			if verifyErr == nil {
				atomic.AddInt32(&successCount, 1)
			} else if errors.Is(verifyErr, domain.ErrOTPInvalid) {
				atomic.AddInt32(&invalidCount, 1)
			}
		}(i)
	}
	wg.Wait()

	s.Equal(int32(1), successCount, "exactly one concurrent verify must succeed")
	s.Equal(int32(goroutines-1), invalidCount, "other concurrent verifies must receive ErrOTPInvalid")

	// Verify exactly 1 session row was created
	sessionCount, err := s.client.Session.Query().Count(s.ctx)
	s.Require().NoError(err)
	s.Equal(1, sessionCount)
}

func (s *OTPServiceSuite) Test_Request_ThenVerify_WithDifferentlyCasedEmail() {
	svc := s.createService(s.defaultCfg, nil, nil)

	_, err := svc.Request(s.ctx, model.OTPRequest{
		Email: "MixEd.Case@Example.COM",
	}, "127.0.0.1")
	s.Require().NoError(err)

	code := s.extractCodeFromMail(1)

	_, user, err := svc.Verify(s.ctx, model.OTPVerifyRequest{
		Email: "mixed.case@EXAMPLE.com",
		Code:  code,
	}, "127.0.0.1", "Agent")
	s.Require().NoError(err)
	s.Equal("mixed.case@example.com", user.Email)
}

func (s *OTPServiceSuite) Test_Verify_TrimsWhitespaceAroundCode() {
	svc := s.createService(s.defaultCfg, nil, nil)
	email := "whitespace@example.com"

	_, err := svc.Request(s.ctx, model.OTPRequest{Email: email}, "127.0.0.1")
	s.Require().NoError(err)

	code := s.extractCodeFromMail(1)

	_, _, err = svc.Verify(s.ctx, model.OTPVerifyRequest{
		Email: email,
		Code:  "  " + code + "\n\t",
	}, "127.0.0.1", "Agent")
	s.Require().NoError(err)
}

func (s *OTPServiceSuite) Test_Verify_RejectsNonASCIIDigitsAndLongCodes() {
	svc := s.createService(s.defaultCfg, nil, nil)
	email := "nonascii@example.com"

	_, err := svc.Request(s.ctx, model.OTPRequest{Email: email}, "127.0.0.1")
	s.Require().NoError(err)

	_ = s.extractCodeFromMail(1)

	// Non-ASCII Arabic numerals
	_, _, err = svc.Verify(s.ctx, model.OTPVerifyRequest{
		Email: email,
		Code:  "١٢٣٤٥٦",
	}, "127.0.0.1", "Agent")
	s.ErrorIs(err, domain.ErrOTPInvalid)

	// 10,000 character string
	longCode := strings.Repeat("1", 10000)
	_, _, err = svc.Verify(s.ctx, model.OTPVerifyRequest{
		Email: email,
		Code:  longCode,
	}, "127.0.0.1", "Agent")
	s.ErrorIs(err, domain.ErrOTPInvalid)

	// Database row: attempts must remain 0
	rec, err := s.client.LoginOTP.Query().
		Where(loginotp.EmailEQ(email)).
		Only(s.ctx)
	s.Require().NoError(err)
	s.Equal(0, rec.Attempts)
}

func (s *OTPServiceSuite) Test_Verify_Unavailable() {
	cfg := s.defaultCfg
	cfg.Enabled = false
	svc := s.createService(cfg, nil, nil)

	_, _, err := svc.Verify(s.ctx, model.OTPVerifyRequest{
		Email: "user@example.com",
		Code:  "123456",
	}, "127.0.0.1", "Agent")
	s.ErrorIs(err, domain.ErrOTPUnavailable)
}
