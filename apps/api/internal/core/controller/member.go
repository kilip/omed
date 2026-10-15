package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// MemberServiceContract defines the member operations required by MemberController.
type MemberServiceContract interface {
	Search(ctx context.Context, wid int64, req model.SearchMemberRequest) ([]domain.Member, string, error)
	Invite(ctx context.Context, wid int64, req model.InviteMemberRequest) (domain.Member, error)
	UpdateRoles(ctx context.Context, wid, uid int64, req model.UpdateMemberRequest) (domain.Member, error)
	Remove(ctx context.Context, wid, uid int64) error
}

// MemberController handles workspace member management and permission queries.
type MemberController struct {
	members MemberServiceContract
	az      *authz.Authorizer
}

// NewMemberController constructs a MemberController.
func NewMemberController(members MemberServiceContract, az *authz.Authorizer) MemberController {
	return MemberController{members: members, az: az}
}

// Search searches members of a workspace.
//
//	@Summary		Search workspace members
//	@Description	List members of the workspace with pagination
//	@Tags			Workspace Members
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.SearchMemberRequest	true	"Search criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.MemberResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/members/search [post]
func (ctl MemberController) Search(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	var req model.SearchMemberRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	members, cursor, err := ctl.members.Search(c, wid, req)
	if err != nil {
		return err
	}
	out := make([]model.MemberResponse, len(members))
	for i, m := range members {
		out[i] = toMemberResponse(m)
	}
	return httpx.WithCursor(c, out, cursor)
}

// Invite adds or invites a user to the workspace.
//
//	@Summary		Invite workspace member
//	@Description	Add a user to the workspace by email (auto-creating the user if not registered)
//	@Tags			Workspace Members
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.InviteMemberRequest	true	"Invite payload"
//	@Success		201		{object}	httpx.WebResponse[model.MemberResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/members [post]
func (ctl MemberController) Invite(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	var req model.InviteMemberRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	m, err := ctl.members.Invite(c, wid, req)
	if err != nil {
		return err
	}
	return httpx.Created(c, toMemberResponse(m))
}

// Update updates the assigned roles of a workspace member.
//
//	@Summary		Update member roles
//	@Description	Replace roles assigned to a workspace member
//	@Tags			Workspace Members
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			id		path		int							true	"User ID of the member"
//	@Param			request	body		model.UpdateMemberRequest	true	"Update payload"
//	@Success		200		{object}	httpx.WebResponse[model.MemberResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/members/{id} [put]
func (ctl MemberController) Update(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	uid, err := httpx.GetID[int64](c)
	if err != nil {
		return err
	}
	var req model.UpdateMemberRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	m, err := ctl.members.UpdateRoles(c, wid, uid, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, toMemberResponse(m))
}

// Delete removes a user from the workspace.
//
//	@Summary		Remove member from workspace
//	@Description	Remove a user from the workspace membership
//	@Tags			Workspace Members
//	@Produce		json
//	@Param			wid	path	int	true	"Workspace ID"
//	@Param			id	path	int	true	"User ID of the member"
//	@Success		200
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/members/{id} [delete]
func (ctl MemberController) Delete(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	uid, err := httpx.GetID[int64](c)
	if err != nil {
		return err
	}
	if err := ctl.members.Remove(c, wid, uid); err != nil {
		return err
	}
	return httpx.OK(c, fiber.Map{})
}

// Permissions returns the effective roles and permissions of the current user in this workspace.
//
//	@Summary		Get current user's workspace permissions
//	@Description	Returns effective permissions and roles in the active workspace
//	@Tags			Workspace Members
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Success		200	{object}	httpx.WebResponse[model.PermissionsResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/permissions [get]
func (ctl MemberController) Permissions(c fiber.Ctx) error {
	m, ok := authz.MembershipFromContext(c)
	if !ok {
		return base.ErrNotFound
	}
	rawPerms := ctl.az.Permissions(c)
	items := make([]model.PermissionItem, len(rawPerms))
	for i, p := range rawPerms {
		items[i] = model.PermissionItem{Object: p.Object, Action: p.Action}
	}
	return httpx.OK(c, model.PermissionsResponse{
		WorkspaceID: m.WorkspaceID,
		Owner:       m.IsOwner,
		Roles:       m.Roles,
		Permissions: items,
	})
}

// Roles lists the roles that can be assigned to members and invitations.
//
//	@Summary		List assignable roles
//	@Description	Returns every role registered by the modules, with display names, for role pickers
//	@Tags			Workspace Members
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Success		200	{object}	httpx.WebResponse[[]model.RoleResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/roles [get]
func (ctl MemberController) Roles(c fiber.Ctx) error {
	roles := ctl.az.Roles()
	out := make([]model.RoleResponse, len(roles))
	for i, r := range roles {
		out[i] = model.RoleResponse{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Module:      r.Module(),
		}
	}
	return httpx.OK(c, out)
}

func toMemberResponse(m domain.Member) model.MemberResponse {
	return model.MemberResponse{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		Roles:       m.Roles,
		User: model.UserResponse{
			ID:        m.User.ID,
			Name:      m.User.Name,
			Email:     m.User.Email,
			Avatar:    m.User.Avatar,
			CreatedAt: m.User.CreatedAt,
			UpdatedAt: m.User.UpdatedAt,
		},
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
