package controller

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/core/service"
)

// OTPController handles passwordless one-time password requests and verification.
type OTPController struct {
	otp     service.OTPService
	cookies sessionCookies
}

// NewOTPController creates a new OTPController.
func NewOTPController(otp service.OTPService, cookieName string, cookieSecure bool) OTPController {
	if cookieName == "" {
		cookieName = "omed_session"
	}
	return OTPController{
		otp: otp,
		cookies: sessionCookies{
			name:   cookieName,
			secure: cookieSecure,
		},
	}
}

// Request initiates a passwordless OTP login request.
//
//	@Summary		Request OTP code
//	@Description	Send a one-time password to the specified email address
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.OTPRequest	true	"OTP request payload"
//	@Success		202		{object}	model.OTPRequestResponse
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		503		{object}	httpx.ErrorResponse
//	@Router			/auth/otp/request [post]
func (o OTPController) Request(c fiber.Ctx) error {
	var req model.OTPRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		return fiber.NewError(fiber.StatusBadRequest, "invalid email")
	}
	if req.Locale == "" {
		req.Locale = c.Get(fiber.HeaderAcceptLanguage)
	}

	ip := c.IP()
	res, err := o.otp.Request(c.Context(), req, ip)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusAccepted).JSON(res)
}

// Verify verifies an OTP code and establishes an authenticated session.
//
//	@Summary		Verify OTP code
//	@Description	Verify one-time password code and return authenticated session
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.OTPVerifyRequest	true	"OTP verify payload"
//	@Success		200		{object}	model.SessionResponse
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		503		{object}	httpx.ErrorResponse
//	@Router			/auth/otp/verify [post]
func (o OTPController) Verify(c fiber.Ctx) error {
	var req model.OTPVerifyRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Email == "" || !strings.Contains(req.Email, "@") || req.Code == "" {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request")
	}

	ip := c.IP()
	userAgent := c.Get(fiber.HeaderUserAgent)

	sess, usr, err := o.otp.Verify(c.Context(), req, ip, userAgent)
	if err != nil {
		return err
	}

	o.cookies.set(c, sess)

	return c.JSON(toSessionResponse(sess, usr))
}
