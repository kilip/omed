// Package controller provides HTTP handlers and endpoint bindings for finance operations.
package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// AccountServiceContract defines the operations required by AccountController.
type AccountServiceContract interface {
	GetByID(c context.Context, workspaceID, id int64) (domain.Account, error)
	Search(c context.Context, workspaceID int64, req model.SearchAccountRequest) ([]domain.Account, string, error)
	GetTree(c context.Context, workspaceID int64) ([]domain.AccountTreeNode, error)
	Create(c context.Context, workspaceID int64, req model.CreateAccountRequest) (domain.Account, error)
	Update(c context.Context, workspaceID, id int64, req model.UpdateAccountRequest) (domain.Account, error)
	Delete(c context.Context, workspaceID, id int64) error
}

// AccountController handles HTTP requests for Chart of Accounts.
type AccountController struct {
	accounts AccountServiceContract
}

// NewAccountController constructs a new AccountController backed by AccountServiceContract.
func NewAccountController(accounts AccountServiceContract) AccountController {
	return AccountController{accounts: accounts}
}

// Search searches accounts within a workspace based on query parameters.
//
//	@Summary		Search accounts
//	@Description	Search Chart of Accounts with filters and cursor pagination
//	@Tags			Finance Accounts
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.SearchAccountRequest	true	"Search filter criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.AccountResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/accounts/search [post]
func (ctl AccountController) Search(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	var req model.SearchAccountRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	items, cursor, err := ctl.accounts.Search(c, wid, req)
	if err != nil {
		return err
	}
	out := make([]model.AccountResponse, len(items))
	for i, it := range items {
		out[i] = toAccountResponse(it)
	}
	return httpx.WithCursor(c, out, cursor)
}

// GetTree retrieves the hierarchical tree of accounts in a workspace.
//
//	@Summary		Get accounts tree
//	@Description	Retrieve hierarchical tree structure of active accounts in the workspace
//	@Tags			Finance Accounts
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Success		200	{object}	httpx.WebResponse[[]model.AccountTreeNodeResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/accounts/tree [get]
func (ctl AccountController) GetTree(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	tree, err := ctl.accounts.GetTree(c, wid)
	if err != nil {
		return err
	}
	return httpx.OK(c, toAccountTreeResponses(tree))
}

// Create creates a new account in the workspace.
//
//	@Summary		Create account
//	@Description	Create a new account entry in Chart of Accounts
//	@Tags			Finance Accounts
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.CreateAccountRequest	true	"Account creation payload"
//	@Success		201		{object}	httpx.WebResponse[model.AccountResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/accounts [post]
func (ctl AccountController) Create(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	var req model.CreateAccountRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	acc, err := ctl.accounts.Create(c, wid, req)
	if err != nil {
		return err
	}
	return httpx.Created(c, toAccountResponse(acc))
}

// GetByID retrieves a single account by ID in the workspace.
//
//	@Summary		Get account by ID
//	@Description	Retrieve account detail by ID within the workspace
//	@Tags			Finance Accounts
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Account ID"
//	@Success		200	{object}	httpx.WebResponse[model.AccountResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/accounts/{id} [get]
func (ctl AccountController) GetByID(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	acc, err := ctl.accounts.GetByID(c, wid, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, toAccountResponse(acc))
}

// Update updates an existing account by ID in the workspace.
//
//	@Summary		Update account
//	@Description	Update an existing account by ID within the workspace
//	@Tags			Finance Accounts
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			id		path		int							true	"Account ID"
//	@Param			request	body		model.UpdateAccountRequest	true	"Account update payload"
//	@Success		200		{object}	httpx.WebResponse[model.AccountResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/accounts/{id} [put]
func (ctl AccountController) Update(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	var req model.UpdateAccountRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	acc, err := ctl.accounts.Update(c, wid, id, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, toAccountResponse(acc))
}

// Delete marks an existing account as soft-deleted.
//
//	@Summary		Delete account
//	@Description	Soft delete an account by ID within the workspace
//	@Tags			Finance Accounts
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Account ID"
//	@Success		200	{object}	httpx.WebResponse[httpx.NoContent]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/accounts/{id} [delete]
func (ctl AccountController) Delete(c fiber.Ctx) error {
	wid, err := httpx.GetParamID[int64](c, "wid")
	if err != nil {
		return err
	}
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	if err := ctl.accounts.Delete(c, wid, id); err != nil {
		return err
	}
	return httpx.OK(c, httpx.NoContent{})
}

func toAccountResponse(a domain.Account) model.AccountResponse {
	return model.AccountResponse{
		ID:            a.ID,
		WorkspaceID:   a.WorkspaceID,
		Code:          a.Code,
		Name:          a.Name,
		Type:          a.Type,
		NormalBalance: a.NormalBalance,
		ParentID:      a.ParentID,
		Description:   a.Description,
		IsActive:      a.IsActive,
		IsPostable:    a.IsPostable,
		DeletedAt:     a.DeletedAt,
		DeletedBy:     a.DeletedBy,
		CreatedBy:     a.AuditMeta.CreatedBy,
		CreatedAt:     a.AuditMeta.CreatedAt,
		UpdatedBy:     a.AuditMeta.UpdatedBy,
		UpdatedAt:     a.AuditMeta.UpdatedAt,
	}
}

func toAccountTreeResponses(nodes []domain.AccountTreeNode) []model.AccountTreeNodeResponse {
	out := make([]model.AccountTreeNodeResponse, len(nodes))
	for i, n := range nodes {
		out[i] = model.AccountTreeNodeResponse{
			ID:            n.ID,
			WorkspaceID:   n.WorkspaceID,
			Code:          n.Code,
			Name:          n.Name,
			Type:          n.Type,
			NormalBalance: n.NormalBalance,
			ParentID:      n.ParentID,
			Description:   n.Description,
			IsActive:      n.IsActive,
			IsPostable:    n.IsPostable,
			Children:      toAccountTreeResponses(n.Children),
			CreatedBy:     n.AuditMeta.CreatedBy,
			CreatedAt:     n.AuditMeta.CreatedAt,
			UpdatedBy:     n.AuditMeta.UpdatedBy,
			UpdatedAt:     n.AuditMeta.UpdatedAt,
		}
	}
	return out
}
