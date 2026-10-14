package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// InvitationServiceContract defines operations required by InvitationController.
type InvitationServiceContract interface {
	Create(ctx context.Context, wid int64, req model.CreateInvitationRequest) (domain.Invitation, error)
	GetPublicInfo(ctx context.Context, token string) (model.PublicInvitationResponse, error)
	Accept(ctx context.Context, token string, userID int64) (domain.Member, error)
	Search(ctx context.Context, wid int64, req model.SearchInvitationRequest) ([]domain.Invitation, string, error)
	Revoke(ctx context.Context, wid, id int64) error
}

// InvitationController handles workspace invitation management and join endpoints.
type InvitationController struct {
	invitations InvitationServiceContract
}

// NewInvitationController constructs an InvitationController.
func NewInvitationController(invitations InvitationServiceContract) InvitationController {
	return InvitationController{invitations: invitations}
}

// Create generates a new workspace invitation link.
//
//	@Summary		Create workspace invitation
//	@Description	Generate a new shareable workspace invitation link with roles, max uses, and expiry
//	@Tags			Workspace Invitations
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int								true	"Workspace ID"
//	@Param			request	body		model.CreateInvitationRequest	true	"Create invitation payload"
//	@Success		201		{object}	httpx.WebResponse[model.InvitationResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/invitations [post]
func (ctl InvitationController) Create(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	var req model.CreateInvitationRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	inv, err := ctl.invitations.Create(c, wid, req)
	if err != nil {
		return err
	}
	return httpx.Created(c, toInvitationResponse(inv))
}

// Search searches invitations for a given workspace.
//
//	@Summary		Search workspace invitations
//	@Description	List active and historical invitation links for a workspace with cursor pagination
//	@Tags			Workspace Invitations
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int								true	"Workspace ID"
//	@Param			request	body		model.SearchInvitationRequest	true	"Search criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.InvitationResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/invitations/search [post]
func (ctl InvitationController) Search(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	var req model.SearchInvitationRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	invitations, cursor, err := ctl.invitations.Search(c, wid, req)
	if err != nil {
		return err
	}
	out := make([]model.InvitationResponse, len(invitations))
	for i, inv := range invitations {
		out[i] = toInvitationResponse(inv)
	}
	return httpx.WithCursor(c, out, cursor)
}

// Revoke cancels a workspace invitation link.
//
//	@Summary		Revoke workspace invitation
//	@Description	Revoke an active invitation link so it can no longer be used
//	@Tags			Workspace Invitations
//	@Produce		json
//	@Param			wid	path	int	true	"Workspace ID"
//	@Param			id	path	int	true	"Invitation ID"
//	@Success		200
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/invitations/{id} [delete]
func (ctl InvitationController) Revoke(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	if err := ctl.invitations.Revoke(c, wid, id); err != nil {
		return err
	}
	return httpx.OK(c, fiber.Map{})
}

// GetPublic retrieves preview information for an invitation token without authentication.
//
//	@Summary		Get public invitation preview
//	@Description	Returns workspace details, inviter name, and token validity for previewing an invitation
//	@Tags			Workspace Invitations
//	@Produce		json
//	@Param			token	path		string	true	"Invitation Token"
//	@Success		200		{object}	httpx.WebResponse[model.PublicInvitationResponse]
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Router			/invitations/{token} [get]
func (ctl InvitationController) GetPublic(c fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return base.ErrNotFound
	}
	preview, err := ctl.invitations.GetPublicInfo(c, token)
	if err != nil {
		return err
	}
	return httpx.OK(c, preview)
}

// Accept allows an authenticated user to accept an invitation and join the workspace.
//
//	@Summary		Accept workspace invitation
//	@Description	Join a workspace using an active invitation token
//	@Tags			Workspace Invitations
//	@Produce		json
//	@Param			token	path		string	true	"Invitation Token"
//	@Success		201		{object}	httpx.WebResponse[model.MemberResponse]
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/invitations/{token}/accept [post]
func (ctl InvitationController) Accept(c fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return base.ErrNotFound
	}
	user := base.UserFromContext(c)
	if user.ID == 0 {
		return base.ErrUnauthorized
	}
	member, err := ctl.invitations.Accept(c, token, user.ID)
	if err != nil {
		return err
	}
	return httpx.Created(c, toMemberResponse(member))
}

func toInvitationResponse(inv domain.Invitation) model.InvitationResponse {
	return model.InvitationResponse{
		ID:          inv.ID,
		WorkspaceID: inv.WorkspaceID,
		Token:       inv.Token,
		Roles:       inv.Roles,
		MaxUses:     inv.MaxUses,
		UsesCount:   inv.UsesCount,
		ExpiresAt:   inv.ExpiresAt,
		IsRevoked:   inv.IsRevoked,
		CreatedAt:   inv.CreatedAt,
	}
}

