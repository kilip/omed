// Package controller provides HTTP handlers and endpoint bindings for finance operations.
package controller

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// AccountServiceContract defines the operations required by AccountController.
type AccountServiceContract interface {
	GetByID(c context.Context, id int64) (domain.Account, error)
	Search(c context.Context, req model.SearchAccountRequest) ([]domain.Account, string, error)
	Create(c context.Context, req model.CreateAccountRequest) (domain.Account, error)
	Update(c context.Context, id int64, req model.UpdateAccountRequest) (domain.Account, error)
	Delete(c context.Context, id int64) error
	Seed(c context.Context, req model.SeedAccountsRequest) (model.SeedAccountsResponse, error)
	PreviewTemplate(c context.Context, template, lang string) (model.AccountTemplatePreviewResponse, error)
}

// AccountController handles HTTP requests for Chart of Accounts.
type AccountController struct {
	accounts AccountServiceContract
	entries  EntryServiceContract
}

// NewAccountController constructs a new AccountController backed by AccountServiceContract and EntryServiceContract.
func NewAccountController(accounts AccountServiceContract, entries EntryServiceContract) AccountController {
	return AccountController{accounts: accounts, entries: entries}
}

// Search searches accounts within a workspace based on query parameters.
//
//	@Summary		Search accounts
//	@Description	Search Chart of Accounts (returns all accounts in the workspace)
//	@Tags			Finance Accounts
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.SearchAccountRequest	true	"Search criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.AccountResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/accounts/search [post]
func (ctl AccountController) Search(c fiber.Ctx) error {
	var req model.SearchAccountRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	items, cursor, err := ctl.accounts.Search(c, req)
	if err != nil {
		return err
	}
	out := make([]model.AccountResponse, len(items))
	for i, it := range items {
		out[i] = toAccountResponse(it)
	}
	return httpx.WithCursor(c, out, cursor)
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
	var req model.CreateAccountRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	acc, err := ctl.accounts.Create(c, req)
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
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	acc, err := ctl.accounts.GetByID(c, id)
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
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	var req model.UpdateAccountRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	acc, err := ctl.accounts.Update(c, id, req)
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
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	if err := ctl.accounts.Delete(c, id); err != nil {
		return err
	}
	return httpx.OK(c, httpx.NoContent{})
}

// Seed populates the workspace with a predefined Chart of Accounts template.
//
//	@Summary		Seed Chart of Accounts
//	@Description	Populate workspace with a standard Chart of Accounts template
//	@Tags			Finance Accounts
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.SeedAccountsRequest	true	"Seed template request"
//	@Success		201		{object}	httpx.WebResponse[model.SeedAccountsResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/accounts/seed [post]
func (ctl AccountController) Seed(c fiber.Ctx) error {
	var req model.SeedAccountsRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	res, err := ctl.accounts.Seed(c, req)
	if err != nil {
		return err
	}
	return httpx.Created(c, res)
}

// PreviewTemplate retrieves account structure for a chart of accounts template.
//
//	@Summary		Preview Chart of Accounts template
//	@Description	Retrieve all account definitions within a COA template for preview before seeding
//	@Tags			Finance Accounts
//	@Produce		json
//	@Param			wid			path		int		true	"Workspace ID"
//	@Param			template	path		string	true	"Template name (e.g. freelancer)"
//	@Param			lang		query		string	false	"Language code (id or en, default id)"
//	@Success		200			{object}	httpx.WebResponse[model.AccountTemplatePreviewResponse]
//	@Failure		400			{object}	httpx.ErrorResponse
//	@Failure		403			{object}	httpx.ErrorResponse
//	@Failure		404			{object}	httpx.ErrorResponse
//	@Failure		500			{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/accounts/templates/{template} [get]
func (ctl AccountController) PreviewTemplate(c fiber.Ctx) error {
	template := c.Params("template")
	lang := c.Query("lang")
	if lang == "" {
		lang = "id"
	}
	res, err := ctl.accounts.PreviewTemplate(c, template, lang)
	if err != nil {
		return err
	}
	return httpx.OK(c, res)
}

// Ledger retrieves postings for an account with running balance (POSTED only, paginated).
//
//	@Summary		Get account ledger
//	@Description	Retrieve posted entries and postings for an account with running balance
//	@Tags			Finance Accounts
//	@Produce		json
//	@Param			wid		path		int	true	"Workspace ID"
//	@Param			id		path		int	true	"Account ID"
//	@Param			page	query		int	false	"Page number (default 1)"
//	@Param			limit	query		int	false	"Page size (default 20, max 100)"
//	@Success		200		{object}	httpx.WebResponse[model.AccountLedgerResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/accounts/{id}/ledger [get]
func (ctl AccountController) Ledger(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	page := 1
	if pStr := c.Query("page"); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}
	limit := 20
	if lStr := c.Query("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	acc, err := ctl.accounts.GetByID(c, id)
	if err != nil {
		return err
	}

	postings, total, err := ctl.entries.GetAccountLedger(c, id, page, limit)
	if err != nil {
		return err
	}

	out := make([]model.PostingResponse, len(postings))
	var lastBalance, lastBaseBalance float64
	for i, p := range postings {
		out[i] = toPostingResponse(p)
	}
	if len(postings) > 0 {
		lastBalance = postings[len(postings)-1].Balance
		lastBaseBalance = postings[len(postings)-1].BaseBalance
	}

	return httpx.OK(c, model.AccountLedgerResponse{
		AccountID:     acc.ID,
		NormalBalance: acc.NormalBalance,
		Balance:       lastBalance,
		BaseBalance:   lastBaseBalance,
		Total:         total,
		Page:          page,
		Limit:         limit,
		Postings:      out,
	})
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
		AuditMeta:     a.AuditMeta,
	}
}
