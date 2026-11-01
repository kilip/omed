// Package service provides business logic and orchestration for the core module.
package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/mailx"
)

// OTPRepository defines the persistence operations needed by OTPService.
type OTPRepository interface {
	Issue(ctx context.Context, o domain.OTP) (domain.OTP, error)
	LastIssuedAt(ctx context.Context, email string) (time.Time, bool, error)
	CountSince(ctx context.Context, email string, since time.Time) (int, error)
	FindActive(ctx context.Context, email string, now time.Time) (domain.OTP, error)
	IncrementAttempts(ctx context.Context, id int64, max int) (int, error)
	Consume(ctx context.Context, id int64, now time.Time) (bool, error)
	DeleteOlderThan(ctx context.Context, email string, before time.Time) error
}

// OTPConfig holds configuration parameters for the OTP service.
type OTPConfig struct {
	Enabled        bool
	Length         int
	TTL            time.Duration
	MaxAttempts    int
	ResendCooldown time.Duration
	HourlyLimit    int
}

// OTPServiceParams contains the dependencies required to create an OTPService.
type OTPServiceParams struct {
	Repo       OTPRepository
	Users      UserProvisioner
	Sessions   SessionService
	Mailer     mailx.Mailer
	AuthSecret string
	Config     OTPConfig
	Logger     *slog.Logger
	Now        func() time.Time
}

// OTPService defines passwordless one-time password operations for authentication.
type OTPService interface {
	Request(ctx context.Context, req model.OTPRequest, ipAddress string) (model.OTPRequestResponse, error)
	Verify(ctx context.Context, req model.OTPVerifyRequest, ipAddress, userAgent string) (domain.Session, domain.User, error)
}

type otpService struct {
	repo       OTPRepository
	users      UserProvisioner
	sessions   SessionService
	mailer     mailx.Mailer
	authSecret string
	cfg        OTPConfig
	logger     *slog.Logger
	now        func() time.Time
}

type otpTemplateData struct {
	Code       string
	TTLMinutes int
	Locale     string
}

// NewOTPService constructs a new OTPService instance.
func NewOTPService(p OTPServiceParams) OTPService {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	logger := p.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &otpService{
		repo:       p.Repo,
		users:      p.Users,
		sessions:   p.Sessions,
		mailer:     p.Mailer,
		authSecret: p.AuthSecret,
		cfg:        p.Config,
		logger:     logger,
		now:        now,
	}
}

func computeOTPHash(authSecret, email, code string) string {
	mac := hmac.New(sha256.New, []byte(authSecret))
	mac.Write([]byte("omed:otp:v1:" + email + ":" + code))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateOTPCode(length int) (string, error) {
	digits := make([]byte, length)
	max := big.NewInt(10)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		digits[i] = byte('0' + n.Int64())
	}
	return string(digits), nil
}

// Request processes an OTP login request by cleaning up expired records, checking rate limits,
// generating a one-time password, persisting its HMAC hash, and sending it via email asynchronously.
func (s *otpService) Request(ctx context.Context, req model.OTPRequest, ipAddress string) (model.OTPRequestResponse, error) {
	if !s.cfg.Enabled {
		return model.OTPRequestResponse{}, domain.ErrOTPUnavailable
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	locale, _ := NormalizeLocaleAndCurrency(req.Locale)

	now := s.now()

	if err := s.repo.DeleteOlderThan(ctx, email, now.Add(-24*time.Hour)); err != nil {
		return model.OTPRequestResponse{}, err
	}

	skipSend := false
	lastIssuedAt, ok, err := s.repo.LastIssuedAt(ctx, email)
	if err != nil {
		return model.OTPRequestResponse{}, err
	}
	if ok && now.Sub(lastIssuedAt) < s.cfg.ResendCooldown {
		skipSend = true
	}

	if !skipSend {
		count, err := s.repo.CountSince(ctx, email, now.Add(-1*time.Hour))
		if err != nil {
			return model.OTPRequestResponse{}, err
		}
		if count >= s.cfg.HourlyLimit {
			skipSend = true
		}
	}

	if !skipSend {
		code, err := generateOTPCode(s.cfg.Length)
		if err != nil {
			return model.OTPRequestResponse{}, err
		}

		codeHash := computeOTPHash(s.authSecret, email, code)
		otp := domain.OTP{
			Email:     email,
			CodeHash:  codeHash,
			ExpiresAt: now.Add(s.cfg.TTL),
			IPAddress: ipAddress,
			CreatedAt: now,
		}
		if _, err := s.repo.Issue(ctx, otp); err != nil {
			return model.OTPRequestResponse{}, err
		}

		var subject string
		if locale == "id" {
			subject = "Kode masuk Omed Anda"
		} else {
			subject = "Your Omed sign-in code"
		}

		ttlMinutes := int(s.cfg.TTL / time.Minute)
		if ttlMinutes < 1 {
			ttlMinutes = 1
		}

		data := otpTemplateData{
			Code:       code,
			TTLMinutes: ttlMinutes,
			Locale:     locale,
		}

		msg := mailx.NewMessage()
		msg.AddTo(email, "")
		msg.Subject = subject

		if err := s.mailer.SendTemplateAsync(ctx, "otp", data, msg); err != nil {
			s.logger.Error("failed to send otp email", "error", err)
		}
	}

	return model.OTPRequestResponse{
		Success:            true,
		CodeLength:         s.cfg.Length,
		TTLSeconds:         int(s.cfg.TTL.Seconds()),
		ResendAfterSeconds: int(s.cfg.ResendCooldown.Seconds()),
	}, nil
}

// Verify validates the supplied OTP code, marks it consumed, provisions or retrieves the user,
// and issues an active login session.
func (s *otpService) Verify(ctx context.Context, req model.OTPVerifyRequest, ipAddress, userAgent string) (domain.Session, domain.User, error) {
	if !s.cfg.Enabled {
		return domain.Session{}, domain.User{}, domain.ErrOTPUnavailable
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	code := strings.TrimSpace(req.Code)
	if len(code) != s.cfg.Length {
		return domain.Session{}, domain.User{}, domain.ErrOTPInvalid
	}
	for i := 0; i < len(code); i++ {
		if code[i] < '0' || code[i] > '9' {
			return domain.Session{}, domain.User{}, domain.ErrOTPInvalid
		}
	}

	now := s.now()

	otp, err := s.repo.FindActive(ctx, email, now)
	if err != nil {
		return domain.Session{}, domain.User{}, domain.ErrOTPInvalid
	}

	if _, err := s.repo.IncrementAttempts(ctx, otp.ID, s.cfg.MaxAttempts); err != nil {
		return domain.Session{}, domain.User{}, domain.ErrOTPInvalid
	}

	expectedHash := computeOTPHash(s.authSecret, email, code)
	if !hmac.Equal([]byte(expectedHash), []byte(otp.CodeHash)) {
		return domain.Session{}, domain.User{}, domain.ErrOTPInvalid
	}

	consumed, err := s.repo.Consume(ctx, otp.ID, now)
	if err != nil || !consumed {
		return domain.Session{}, domain.User{}, domain.ErrOTPInvalid
	}

	user, err := s.users.FindOrCreateVerified(ctx, ProvisionInput{Email: email})
	if err != nil {
		return domain.Session{}, domain.User{}, err
	}

	sess, err := s.sessions.CreateSession(ctx, user.ID, ipAddress, userAgent)
	if err != nil {
		return domain.Session{}, domain.User{}, err
	}

	return sess, user, nil
}
