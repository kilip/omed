package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/config"
	"github.com/kilip/omed/api/internal/core/controller"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/stretchr/testify/suite"
)

type mockOTPService struct {
	requestFunc func(ctx context.Context, req model.OTPRequest, ipAddress string) (model.OTPRequestResponse, error)
	verifyFunc  func(ctx context.Context, req model.OTPVerifyRequest, ipAddress, userAgent string) (domain.Session, domain.User, error)
}

func (m *mockOTPService) Request(ctx context.Context, req model.OTPRequest, ipAddress string) (model.OTPRequestResponse, error) {
	if m.requestFunc != nil {
		return m.requestFunc(ctx, req, ipAddress)
	}
	return model.OTPRequestResponse{
		Success:            true,
		CodeLength:         6,
		TTLSeconds:         600,
		ResendAfterSeconds: 60,
	}, nil
}

func (m *mockOTPService) Verify(ctx context.Context, req model.OTPVerifyRequest, ipAddress, userAgent string) (domain.Session, domain.User, error) {
	if m.verifyFunc != nil {
		return m.verifyFunc(ctx, req, ipAddress, userAgent)
	}
	return domain.Session{
		ID:        1,
		Token:     "mock-otp-session-token",
		UserID:    10,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}, domain.User{
		ID:            10,
		Email:         req.Email,
		Name:          "Test User",
		EmailVerified: true,
	}, nil
}

type OTPControllerSuite struct {
	suite.Suite
	app    *fiber.App
	otpSvc *mockOTPService
}

func (s *OTPControllerSuite) SetupTest() {
	s.otpSvc = &mockOTPService{}
	s.app = config.GetFiber(config.Config{Env: "test"}, slog.Default())

	ctl := controller.NewOTPController(s.otpSvc, "omed_session", false)
	auth := s.app.Group("/auth")
	auth.Post("/otp/request", ctl.Request)
	auth.Post("/otp/verify", ctl.Verify)
}

func (s *OTPControllerSuite) Test_Request_Returns202WithBody() {
	reqBody := model.OTPRequest{
		Email: "user@example.com",
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(fiber.MethodPost, "/auth/otp/request", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	s.Equal(fiber.StatusAccepted, resp.StatusCode)

	var res model.OTPRequestResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	s.Require().NoError(err)
	s.True(res.Success)
	s.Equal(6, res.CodeLength)
	s.Equal(600, res.TTLSeconds)
	s.Equal(60, res.ResendAfterSeconds)
}

func (s *OTPControllerSuite) Test_Request_InvalidEmail400() {
	reqBody := model.OTPRequest{
		Email: "not-an-email",
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(fiber.MethodPost, "/auth/otp/request", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	s.Equal(fiber.StatusBadRequest, resp.StatusCode)
}

func (s *OTPControllerSuite) Test_Request_DefaultsLocaleFromAcceptLanguage() {
	var capturedLocale string
	s.otpSvc.requestFunc = func(ctx context.Context, req model.OTPRequest, ipAddress string) (model.OTPRequestResponse, error) {
		capturedLocale = req.Locale
		return model.OTPRequestResponse{
			Success:            true,
			CodeLength:         6,
			TTLSeconds:         600,
			ResendAfterSeconds: 60,
		}, nil
	}

	reqBody := model.OTPRequest{
		Email: "user@example.com",
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(fiber.MethodPost, "/auth/otp/request", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.8")

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	s.Equal(fiber.StatusAccepted, resp.StatusCode)
	s.Equal("fr-FR,fr;q=0.9,en;q=0.8", capturedLocale)
}

func (s *OTPControllerSuite) Test_Request_Unavailable503() {
	s.otpSvc.requestFunc = func(ctx context.Context, req model.OTPRequest, ipAddress string) (model.OTPRequestResponse, error) {
		return model.OTPRequestResponse{}, domain.ErrOTPUnavailable
	}

	reqBody := model.OTPRequest{
		Email: "user@example.com",
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(fiber.MethodPost, "/auth/otp/request", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	s.Equal(fiber.StatusServiceUnavailable, resp.StatusCode)

	var errResp httpx.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	s.Require().NoError(err)
	s.Equal("AUTH_OTP_UNAVAILABLE", errResp.Error.Code)
}

func (s *OTPControllerSuite) Test_Verify_SetsSessionCookieAndReturnsSession() {
	reqBody := model.OTPVerifyRequest{
		Email: "user@example.com",
		Code:  "123456",
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(fiber.MethodPost, "/auth/otp/verify", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	s.Equal(fiber.StatusOK, resp.StatusCode)

	// Verify session cookie
	var foundCookie bool
	for _, c := range resp.Cookies() {
		if c.Name == "omed_session" {
			foundCookie = true
			s.Equal("mock-otp-session-token", c.Value)
			s.True(c.HttpOnly)
			s.Equal("/", c.Path)
			s.Equal(http.SameSiteLaxMode, c.SameSite)
		}
	}
	s.True(foundCookie, "omed_session cookie must be set")

	var res model.SessionResponse
	err = json.NewDecoder(resp.Body).Decode(&res)
	s.Require().NoError(err)
	s.Require().NotNil(res.User)
	s.Equal("user@example.com", res.User.Email)
}

func (s *OTPControllerSuite) Test_Verify_InvalidCode400() {
	s.otpSvc.verifyFunc = func(ctx context.Context, req model.OTPVerifyRequest, ipAddress, userAgent string) (domain.Session, domain.User, error) {
		return domain.Session{}, domain.User{}, domain.ErrOTPInvalid
	}

	reqBody := model.OTPVerifyRequest{
		Email: "user@example.com",
		Code:  "000000",
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(fiber.MethodPost, "/auth/otp/verify", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	s.Equal(fiber.StatusBadRequest, resp.StatusCode)

	var errResp httpx.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	s.Require().NoError(err)
	s.Equal("AUTH_OTP_INVALID", errResp.Error.Code)
}

func (s *OTPControllerSuite) Test_Verify_PassesIPAndUserAgent() {
	var capturedIP, capturedUA string
	s.otpSvc.verifyFunc = func(ctx context.Context, req model.OTPVerifyRequest, ipAddress, userAgent string) (domain.Session, domain.User, error) {
		capturedIP = ipAddress
		capturedUA = userAgent
		return domain.Session{
			ID:        1,
			Token:     "test-token",
			UserID:    1,
			ExpiresAt: time.Now().Add(time.Hour),
		}, domain.User{
			ID:    1,
			Email: req.Email,
		}, nil
	}

	reqBody := model.OTPVerifyRequest{
		Email: "user@example.com",
		Code:  "123456",
	}
	b, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(fiber.MethodPost, "/auth/otp/verify", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestBrowser/1.0")

	resp, err := s.app.Test(req)
	s.Require().NoError(err)
	defer func() { _ = resp.Body.Close() }()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.NotEmpty(capturedIP)
	s.Equal("TestBrowser/1.0", capturedUA)
}

func TestOTPControllerSuite(t *testing.T) {
	suite.Run(t, new(OTPControllerSuite))
}
