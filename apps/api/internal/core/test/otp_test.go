package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/session"
	"github.com/kilip/omed/api/ent/user"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/mailx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type OTPSuite struct {
	testutil.ApiTestSuite[any]
}

func (s *OTPSuite) SetupTest() {
	s.ApiTestSuite.SetupTest()
	mailer := testutil.GetMailer()
	if memDriver, ok := mailer.Driver().(*mailx.MemoryDriver); ok {
		memDriver.Reset()
	}
}

func (s *OTPSuite) waitForEmail(memDriver *mailx.MemoryDriver, toEmail string) mailx.Message {
	var found *mailx.Message
	s.Eventually(func() bool {
		for _, m := range memDriver.SentMessages() {
			for _, to := range m.To {
				if to.Address == toEmail {
					found = &m
					return true
				}
			}
		}
		return false
	}, 2*time.Second, 50*time.Millisecond)
	s.Require().NotNil(found, "expected email to %s", toEmail)
	return *found
}

func (s *OTPSuite) Test_RequestThenVerify_NewUser_CookieAndBearer() {
	mailer := testutil.GetMailer()
	s.Require().NotNil(mailer)
	memDriver, ok := mailer.Driver().(*mailx.MemoryDriver)
	s.Require().True(ok)
	memDriver.Reset()

	email := "newuser@example.com"
	s.RequestWithToken("/auth/otp/request", fiber.MethodPost, model.OTPRequest{Email: email}, "")
	s.AssertStatus(fiber.StatusAccepted)

	var reqRes model.OTPRequestResponse
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&reqRes))
	_ = s.HttpResponse.Body.Close()

	s.True(reqRes.Success)
	s.Equal(6, reqRes.CodeLength)
	s.Equal(600, reqRes.TTLSeconds)
	s.Equal(60, reqRes.ResendAfterSeconds)

	msg := s.waitForEmail(memDriver, email)
	s.Require().Len(msg.To, 1)
	s.Equal(email, msg.To[0].Address)

	codeRe := regexp.MustCompile(`\b\d{6}\b`)
	code := codeRe.FindString(msg.TextBody)
	s.Require().NotEmpty(code, "6-digit OTP code must be present in email body: %s", msg.TextBody)

	s.RequestWithToken("/auth/otp/verify", fiber.MethodPost, model.OTPVerifyRequest{
		Email: email,
		Code:  code,
	}, "")
	s.OK()

	var sessionCookie *http.Cookie
	for _, c := range s.HttpResponse.Cookies() {
		if c.Name == "omed_session" {
			sessionCookie = c
			break
		}
	}
	s.Require().NotNil(sessionCookie, "omed_session cookie must be set")
	s.NotEmpty(sessionCookie.Value)

	var verifyRes model.SessionResponse
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&verifyRes))
	_ = s.HttpResponse.Body.Close()
	s.Require().NotNil(verifyRes.User)
	s.Equal(email, verifyRes.User.Email)
	s.True(verifyRes.User.EmailVerified)
	s.Require().NotNil(verifyRes.Session)

	// Check session: GET /auth/get-session with cookie
	s.ClearCookies()
	s.SetCookie(sessionCookie)
	s.RequestWithToken("/auth/get-session", fiber.MethodGet, nil, "")
	s.OK()

	var sessResCookie model.SessionResponse
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&sessResCookie))
	_ = s.HttpResponse.Body.Close()
	s.Require().NotNil(sessResCookie.User)
	s.Equal(email, sessResCookie.User.Email)
	s.True(sessResCookie.User.EmailVerified)
	s.Require().NotNil(sessResCookie.Session)
	s.Equal(verifyRes.Session.ID, sessResCookie.Session.ID)

	// Check session: GET /auth/get-session with Authorization: Bearer <cookie-value>
	s.ClearCookies()
	s.RequestWithToken("/auth/get-session", fiber.MethodGet, nil, sessionCookie.Value)
	s.OK()

	var sessResBearer model.SessionResponse
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&sessResBearer))
	_ = s.HttpResponse.Body.Close()
	s.Require().NotNil(sessResBearer.User)
	s.Equal(email, sessResBearer.User.Email)
	s.True(sessResBearer.User.EmailVerified)
	s.Require().NotNil(sessResBearer.Session)
	s.Equal(verifyRes.Session.ID, sessResBearer.Session.ID)

	// Check user in DB: user has email_verified: true
	dbUser, err := testutil.GetEntCli().User.Query().Where(user.Email(email)).Only(context.Background())
	s.Require().NoError(err)
	s.True(dbUser.EmailVerified)
}

func (s *OTPSuite) Test_Verify_WrongCode_400_AUTH_OTP_INVALID() {
	mailer := testutil.GetMailer()
	memDriver, ok := mailer.Driver().(*mailx.MemoryDriver)
	s.Require().True(ok)
	memDriver.Reset()

	email := "wrongcode@example.com"
	s.RequestWithToken("/auth/otp/request", fiber.MethodPost, model.OTPRequest{Email: email}, "")
	s.AssertStatus(fiber.StatusAccepted)
	_ = s.HttpResponse.Body.Close()

	msg := s.waitForEmail(memDriver, email)
	codeRe := regexp.MustCompile(`\b\d{6}\b`)
	sentCode := codeRe.FindString(msg.TextBody)
	wrongCode := "000000"
	if sentCode == "000000" {
		wrongCode = "111111"
	}

	s.RequestWithToken("/auth/otp/verify", fiber.MethodPost, model.OTPVerifyRequest{
		Email: email,
		Code:  wrongCode,
	}, "")
	s.AssertStatus(fiber.StatusBadRequest)

	errResp := s.GetErrorResponse()
	s.Equal("AUTH_OTP_INVALID", errResp.Error.Code)
}

func (s *OTPSuite) Test_Request_InvalidEmail_400() {
	s.RequestWithToken("/auth/otp/request", fiber.MethodPost, model.OTPRequest{Email: "invalid-email"}, "")
	s.AssertStatus(fiber.StatusBadRequest)
	_ = s.HttpResponse.Body.Close()

	s.RequestWithToken("/auth/otp/request", fiber.MethodPost, model.OTPRequest{Email: ""}, "")
	s.AssertStatus(fiber.StatusBadRequest)
	_ = s.HttpResponse.Body.Close()
}

func (s *OTPSuite) Test_Request_SameResponseForKnownAndUnknownEmail() {
	knownUser := s.EnsureUser(ent.User{
		Name:  "Known User",
		Email: "known.user@example.com",
	})
	s.Require().NotNil(knownUser)

	s.RequestWithToken("/auth/otp/request", fiber.MethodPost, model.OTPRequest{Email: "known.user@example.com"}, "")
	s.AssertStatus(fiber.StatusAccepted)
	var knownRes model.OTPRequestResponse
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&knownRes))
	_ = s.HttpResponse.Body.Close()

	s.RequestWithToken("/auth/otp/request", fiber.MethodPost, model.OTPRequest{Email: "unknown.user@example.com"}, "")
	s.AssertStatus(fiber.StatusAccepted)
	var unknownRes model.OTPRequestResponse
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&unknownRes))
	_ = s.HttpResponse.Body.Close()

	s.Equal(knownRes.Success, unknownRes.Success)
	s.Equal(knownRes.CodeLength, unknownRes.CodeLength)
	s.Equal(knownRes.TTLSeconds, unknownRes.TTLSeconds)
	s.Equal(knownRes.ResendAfterSeconds, unknownRes.ResendAfterSeconds)

	s.True(knownRes.Success)
	s.Equal(6, knownRes.CodeLength)
	s.Equal(600, knownRes.TTLSeconds)
	s.Equal(60, knownRes.ResendAfterSeconds)
}

func (s *OTPSuite) Test_Verify_ConcurrentExactlyOneSession() {
	mailer := testutil.GetMailer()
	memDriver, ok := mailer.Driver().(*mailx.MemoryDriver)
	s.Require().True(ok)
	memDriver.Reset()

	email := "concurrent@example.com"
	s.RequestWithToken("/auth/otp/request", fiber.MethodPost, model.OTPRequest{Email: email}, "")
	s.AssertStatus(fiber.StatusAccepted)
	_ = s.HttpResponse.Body.Close()

	msg := s.waitForEmail(memDriver, email)
	codeRe := regexp.MustCompile(`\b\d{6}\b`)
	code := codeRe.FindString(msg.TextBody)
	s.Require().NotEmpty(code)

	reqBody, err := json.Marshal(model.OTPVerifyRequest{
		Email: email,
		Code:  code,
	})
	s.Require().NoError(err)

	app := testutil.GetState().FiberApp
	var wg sync.WaitGroup
	statusCodes := make([]int, 10)
	start := make(chan struct{})

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			req := httptest.NewRequest(fiber.MethodPost, "/auth/otp/verify", bytes.NewReader(reqBody))
			req.Header.Set("Content-Type", "application/json")
			resp, err := app.Test(req)
			if err != nil {
				return
			}
			defer func() { _ = resp.Body.Close() }()
			statusCodes[idx] = resp.StatusCode
		}(i)
	}
	close(start)
	wg.Wait()

	var count200, count400 int
	for _, status := range statusCodes {
		switch status {
		case fiber.StatusOK:
			count200++
		case fiber.StatusBadRequest:
			count400++
		}
	}
	s.Equal(1, count200, "expected exactly one 200 OK")
	s.Equal(9, count400, "expected exactly nine 400 Bad Request")

	usr, err := testutil.GetEntCli().User.Query().Where(user.Email(email)).Only(context.Background())
	s.Require().NoError(err)
	sessionCount, err := testutil.GetEntCli().Session.Query().Where(session.UserID(usr.ID)).Count(context.Background())
	s.Require().NoError(err)
	s.Equal(1, sessionCount)
}

func TestOTPSuite(t *testing.T) {
	suite.Run(t, new(OTPSuite))
}
