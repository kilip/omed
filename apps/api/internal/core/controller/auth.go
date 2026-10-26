// Package controller provides HTTP handlers and endpoint bindings for authentication and OAuth.
package controller

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/core/service"
)

// OAuthNonceCookieName is the default cookie name used to bind OAuth state to the browser session.
const OAuthNonceCookieName = "omed_oauth_nonce"

// AuthController handles social sign-in, OAuth callback, session retrieval, and logout.
type AuthController struct {
	oauth           service.OAuthService
	sessions        service.SessionService
	cookieName      string
	nonceCookieName string
	cookieSecure    bool
}

// AuthOption configures an AuthController.
type AuthOption func(*AuthController)

// WithCookieName configures the session cookie name.
func WithCookieName(name string) AuthOption {
	return func(a *AuthController) {
		if name != "" {
			a.cookieName = name
		}
	}
}

// WithOAuthNonceCookieName configures the OAuth nonce cookie name.
func WithOAuthNonceCookieName(name string) AuthOption {
	return func(a *AuthController) {
		if name != "" {
			a.nonceCookieName = name
		}
	}
}

// WithCookieSecure configures whether the session cookie has the Secure attribute.
func WithCookieSecure(secure bool) AuthOption {
	return func(a *AuthController) {
		a.cookieSecure = secure
	}
}

// NewAuthController creates a new AuthController.
func NewAuthController(oauth service.OAuthService, sessions service.SessionService, opts ...AuthOption) AuthController {
	a := AuthController{
		oauth:           oauth,
		sessions:        sessions,
		cookieName:      "omed_session",
		nonceCookieName: OAuthNonceCookieName,
		cookieSecure:    false,
	}
	for _, opt := range opts {
		opt(&a)
	}
	return a
}

// SignInSocial initiates a social login flow with an OAuth2 provider.
//
//	@Summary		Sign in with social provider
//	@Description	Initiate OAuth2 social login (Google, GitHub) and get authorization URL
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.SignInSocialRequest	true	"Sign-in payload"
//	@Success		200		{object}	model.SignInSocialResponse
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/auth/sign-in/social [post]
func (a AuthController) SignInSocial(c fiber.Ctx) error {
	var req model.SignInSocialRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if req.Provider == "" {
		return fiber.NewError(fiber.StatusBadRequest, "provider is required")
	}
	if req.Locale == "" {
		req.Locale = c.Get(fiber.HeaderAcceptLanguage)
	}

	res, err := a.oauth.GetAuthURL(req)
	if err != nil {
		return err
	}

	c.Cookie(&fiber.Cookie{
		Name:     a.nonceCookieName,
		Value:    res.Nonce,
		Path:     "/",
		Expires:  time.Now().Add(10 * time.Minute),
		HTTPOnly: true,
		Secure:   a.cookieSecure,
		SameSite: "Lax",
	})

	return c.JSON(res)
}

// Callback handles the authorization code redirect from OAuth2 provider.
//
//	@Summary		OAuth2 callback
//	@Description	Handle authorization code callback from OAuth provider and establish session
//	@Tags			Authentication
//	@Produce		json
//	@Param			id		path		string	true	"OAuth Provider (e.g. google, github)"
//	@Param			code	query		string	false	"Authorization code"
//	@Param			state	query		string	false	"CSRF state token"
//	@Param			error	query		string	false	"Error reason from provider"
//	@Success		302		{string}	string	"Redirects to frontend callback URL"
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/auth/callback/{id} [get]
func (a AuthController) Callback(c fiber.Ctx) error {
	provider := c.Params("id")
	code := c.Query("code")
	state := c.Query("state")
	errQuery := c.Query("error")

	// Also check body parameters if sent via POST
	if c.Method() == fiber.MethodPost {
		var body struct {
			Code  string `json:"code"`
			State string `json:"state"`
			Error string `json:"error"`
		}
		_ = c.Bind().Body(&body)
		if body.Code != "" {
			code = body.Code
		}
		if body.State != "" {
			state = body.State
		}
		if body.Error != "" {
			errQuery = body.Error
		}
	}

	if errQuery != "" {
		return fiber.NewError(fiber.StatusBadRequest, "provider error: "+errQuery)
	}
	if code == "" || state == "" {
		return fiber.NewError(fiber.StatusBadRequest, "code and state are required")
	}

	nonce := c.Cookies(a.nonceCookieName)

	// Clear the oauth nonce cookie
	c.Cookie(&fiber.Cookie{
		Name:     a.nonceCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   a.cookieSecure,
		SameSite: "Lax",
	})

	ip := c.IP()
	userAgent := c.Get(fiber.HeaderUserAgent)

	sess, usr, redirectURL, err := a.oauth.HandleCallback(c.Context(), provider, code, state, nonce, ip, userAgent)
	if err != nil {
		return err
	}

	// Set HttpOnly session cookie
	c.Cookie(&fiber.Cookie{
		Name:     a.cookieName,
		Value:    sess.Token,
		Path:     "/",
		Expires:  sess.ExpiresAt,
		HTTPOnly: true,
		Secure:   a.cookieSecure,
		SameSite: "Lax",
	})

	// If request came from API client or explicit POST, return JSON
	if strings.Contains(c.Get(fiber.HeaderAccept), "application/json") && c.Method() == fiber.MethodPost {
		return c.JSON(toSessionResponse(sess, usr))
	}

	return c.Redirect().Status(fiber.StatusFound).To(redirectURL)
}

// GetSession retrieves the current active session and user profile.
//
//	@Summary		Get current session
//	@Description	Retrieve authenticated session and user details
//	@Tags			Authentication
//	@Produce		json
//	@Success		200	{object}	model.SessionResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Router			/auth/get-session [get]
func (a AuthController) GetSession(c fiber.Ctx) error {
	token := a.extractToken(c)
	if token == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "missing session token")
	}

	sess, usr, err := a.sessions.ValidateSession(c.Context(), token)
	if err != nil {
		return err
	}

	return c.JSON(toSessionResponse(sess, usr))
}

// SignOut terminates the current session and clears cookies.
//
//	@Summary		Sign out
//	@Description	Revoke current session from database and clear session cookie
//	@Tags			Authentication
//	@Produce		json
//	@Success		200	{object}	map[string]bool
//	@Router			/auth/sign-out [post]
func (a AuthController) SignOut(c fiber.Ctx) error {
	token := a.extractToken(c)
	if token != "" {
		_ = a.sessions.RevokeSession(c.Context(), token)
	}

	c.Cookie(&fiber.Cookie{
		Name:     a.cookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   a.cookieSecure,
		SameSite: "Lax",
	})

	return c.JSON(fiber.Map{"success": true})
}

func (a AuthController) extractToken(c fiber.Ctx) string {
	authHeader := c.Get(fiber.HeaderAuthorization)
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	if cookie := c.Cookies(a.cookieName); cookie != "" {
		return cookie
	}
	return ""
}

func toSessionResponse(s domain.Session, u domain.User) model.SessionResponse {
	return model.SessionResponse{
		Session: &model.SessionData{
			ID:        s.ID,
			UserID:    s.UserID,
			ExpiresAt: s.ExpiresAt,
			IPAddress: s.IPAddress,
			UserAgent: s.UserAgent,
			CreatedAt: s.CreatedAt,
			UpdatedAt: s.UpdatedAt,
		},
		User: &model.UserData{
			ID:            u.ID,
			Email:         u.Email,
			Name:          u.Name,
			Avatar:        u.Avatar,
			Locale:        u.Locale,
			Currency:      u.Currency,
			EmailVerified: u.EmailVerified,
		},
	}
}
