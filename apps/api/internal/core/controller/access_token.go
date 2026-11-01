// Package controller provides HTTP handlers and endpoint bindings for user and workspace management.
package controller

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// AccessTokenService defines the business logic operations required by AccessTokenController.
type AccessTokenService interface {
	Create(ctx context.Context, userID int64, req model.CreateAccessTokenRequest) (domain.AccessToken, string, error)
	List(ctx context.Context, userID int64) ([]domain.AccessToken, error)
	Revoke(ctx context.Context, userID, id int64) error
}

// AccessTokenController handles HTTP requests for personal access token management.
type AccessTokenController struct {
	tokens AccessTokenService
}

// NewAccessTokenController constructs a new AccessTokenController backed by the specified AccessTokenService.
func NewAccessTokenController(tokens AccessTokenService) AccessTokenController {
	return AccessTokenController{tokens: tokens}
}

// Create generates a new personal access token for the authenticated user.
//
//	@Summary		Create access token
//	@Description	Create a new personal access token for API/MCP authentication
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			request	body		model.CreateAccessTokenRequest	true	"Create access token payload"
//	@Success		201		{object}	httpx.WebResponse[model.CreatedAccessTokenResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/me/access-tokens [post]
func (ctl AccessTokenController) Create(c fiber.Ctx) error {
	user := base.UserFromContext(c)
	if user.ID == 0 {
		return base.ErrUnauthorized
	}

	var req model.CreateAccessTokenRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	tok, plaintext, err := ctl.tokens.Create(c.Context(), user.ID, req)
	if err != nil {
		return err
	}

	res := model.CreatedAccessTokenResponse{
		ID:        tok.ID,
		Name:      tok.Name,
		Prefix:    tok.Prefix,
		Status:    string(tok.Status(time.Now())),
		Token:     plaintext,
		ExpiresAt: tok.ExpiresAt,
		CreatedAt: tok.CreatedAt,
	}

	return httpx.Created(c, res)
}

// List retrieves all personal access tokens for the authenticated user.
//
//	@Summary		List access tokens
//	@Description	List all personal access tokens owned by the authenticated user
//	@Tags			Users
//	@Produce		json
//	@Success		200	{object}	httpx.WebResponse[[]model.AccessTokenResponse]
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/me/access-tokens [get]
func (ctl AccessTokenController) List(c fiber.Ctx) error {
	user := base.UserFromContext(c)
	if user.ID == 0 {
		return base.ErrUnauthorized
	}

	tokens, err := ctl.tokens.List(c.Context(), user.ID)
	if err != nil {
		return err
	}

	now := time.Now()
	out := make([]model.AccessTokenResponse, len(tokens))
	for i, tok := range tokens {
		out[i] = model.AccessTokenResponse{
			ID:        tok.ID,
			Name:      tok.Name,
			Prefix:    tok.Prefix,
			Status:    string(tok.Status(now)),
			ExpiresAt: tok.ExpiresAt,
			CreatedAt: tok.CreatedAt,
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": out,
		"meta": httpx.NewMeta(c, ""),
	})
}

// Revoke revokes a personal access token by ID.
//
//	@Summary		Revoke access token
//	@Description	Revoke an active personal access token owned by the authenticated user
//	@Tags			Users
//	@Produce		json
//	@Param			id	path	int	true	"Access Token ID"
//	@Success		200
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/users/me/access-tokens/{id} [delete]
func (ctl AccessTokenController) Revoke(c fiber.Ctx) error {
	user := base.UserFromContext(c)
	if user.ID == 0 {
		return base.ErrUnauthorized
	}

	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}

	if err := ctl.tokens.Revoke(c.Context(), user.ID, id); err != nil {
		return err
	}

	return httpx.OK(c, fiber.Map{})
}
