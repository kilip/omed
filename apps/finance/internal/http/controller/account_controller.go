package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/http"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared/authz"
)

type AccountService interface {
	List(ctx context.Context, req model.ListAccountRequest) ([]model.AccountResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.AccountResponse, error)
	Create(ctx context.Context, req model.CreateAccountRequest) (*model.AccountResponse, error)
	Update(ctx context.Context, id uuid.UUID, req model.UpdateAccountRequest) (*model.AccountResponse, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Seed(ctx context.Context, req model.SeedAccountRequest) ([]model.AccountResponse, error)
}

type AccountController struct {
	accounts AccountService
}

func NewAccountController(accounts AccountService) AccountController {
	return AccountController{
		accounts,
	}
}

func (ac AccountController) InitRoutes(r fiber.Router) {
	g := r.Group("/accounts")
	read := http.RequirePermission(authz.ResourceAccounts, authz.ActionRead)
	write := http.RequirePermission(authz.ResourceAccounts, authz.ActionWrite)

	g.Get("/", read, ac.List)
	g.Post("/seed", write, ac.Seed)
	g.Get("/:id", read, ac.GetById)
	g.Post("/", write, ac.Create)
	g.Put("/:id", write, ac.Update)
	g.Delete("/:id", write, ac.Delete)
}

// List godoc
//
//	@Summary		List accounts
//	@Description	Return all accounts in the active workspace (flat list, tree is built client-side).
//	@Tags			accounts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			type	query		string	false	"Filter by account type"	Enums(asset, liability, equity, revenue, expense)
//	@Param			status	query		string	false	"Filter by status"			Enums(active, archived)
//	@Success		200		{object}	model.WebResponse[[]model.AccountResponse]
//	@Failure		400		{object}	model.ErrorResponse
//	@Failure		401		{object}	model.ErrorResponse
//	@Failure		403		{object}	model.ErrorResponse
//	@Failure		422		{object}	model.ErrorResponse
//	@Failure		500		{object}	model.ErrorResponse
//	@Router			/accounts [get]
func (ac *AccountController) List(c fiber.Ctx) error {
	var req model.ListAccountRequest
	if err := c.Bind().Query(&req); err != nil {
		return err
	}

	accounts, err := ac.accounts.List(c, req)
	if err != nil {
		return err
	}
	return http.OK(c, accounts)
}

// Get godoc
//
//	@Summary		Get account
//	@Description	Get a single account by ID.
//	@Tags			accounts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Account ID"	format(uuid)
//	@Success		200	{object}	model.WebResponse[model.AccountResponse]
//	@Failure		400	{object}	model.ErrorResponse
//	@Failure		401	{object}	model.ErrorResponse
//	@Failure		403	{object}	model.ErrorResponse
//	@Failure		404	{object}	model.ErrorResponse
//	@Failure		500	{object}	model.ErrorResponse
//	@Router			/accounts/{id} [get]
func (ac AccountController) GetById(c fiber.Ctx) error {
	id, err := http.GetID(c)
	if err != nil {
		return err
	}

	account, err := ac.accounts.GetByID(c, *id)
	if err != nil {
		return err
	}
	return http.OK(c, account)
}

// Create godoc
//
//	@Summary		Create account
//	@Description	Create a new account in the active workspace.
//	@Tags			accounts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		model.CreateAccountRequest	true	"Account payload"
//	@Success		201		{object}	model.WebResponse[model.AccountResponse]
//	@Failure		400		{object}	model.ErrorResponse
//	@Failure		401		{object}	model.ErrorResponse
//	@Failure		403		{object}	model.ErrorResponse
//	@Failure		422		{object}	model.ErrorResponse
//	@Failure		500		{object}	model.ErrorResponse
//	@Router			/accounts [post]
func (ac AccountController) Create(c fiber.Ctx) error {
	var req model.CreateAccountRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}

	account, err := ac.accounts.Create(c, req)
	if err != nil {
		return err
	}
	return http.Created(c, account)
}

// Update godoc
//
//	@Summary		Update account
//	@Description	Update mutable fields of an account. Code, type and currency cannot be changed.
//	@Tags			accounts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		string						true	"Account ID"	format(uuid)
//	@Param			request	body		model.UpdateAccountRequest	true	"Account payload"
//	@Success		200		{object}	model.WebResponse[model.AccountResponse]
//	@Failure		400		{object}	model.ErrorResponse
//	@Failure		401		{object}	model.ErrorResponse
//	@Failure		403		{object}	model.ErrorResponse
//	@Failure		404		{object}	model.ErrorResponse
//	@Failure		422		{object}	model.ErrorResponse
//	@Failure		500		{object}	model.ErrorResponse
//	@Router			/accounts/{id} [put]
func (ac *AccountController) Update(c fiber.Ctx) error {
	id, err := http.GetID(c)
	if err != nil {
		return err
	}

	var req model.UpdateAccountRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}

	account, err := ac.accounts.Update(c, *id, req)
	if err != nil {
		return err
	}
	return http.OK(c, account)
}

// Delete godoc
//
//	@Summary		Delete account
//	@Description	Delete an account by ID.
//	@Tags			accounts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Account ID"	format(uuid)
//	@Success		204
//	@Failure		400	{object}	model.ErrorResponse
//	@Failure		401	{object}	model.ErrorResponse
//	@Failure		403	{object}	model.ErrorResponse
//	@Failure		404	{object}	model.ErrorResponse
//	@Failure		500	{object}	model.ErrorResponse
//	@Router			/accounts/{id} [delete]
func (ac *AccountController) Delete(c fiber.Ctx) error {
	id, err := http.GetID(c)
	if err != nil {
		return err
	}

	if err := ac.accounts.Delete(c, *id); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// Seed godoc
//
//	@Summary		Seed accounts
//	@Description	Seed chart of accounts using preset profile and language.
//	@Tags			accounts
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		model.SeedAccountRequest	true	"Seed request payload"
//	@Success		201		{object}	model.WebResponse[[]model.AccountResponse]
//	@Failure		400		{object}	model.ErrorResponse
//	@Failure		401		{object}	model.ErrorResponse
//	@Failure		403		{object}	model.ErrorResponse
//	@Failure		422		{object}	model.ErrorResponse
//	@Failure		500		{object}	model.ErrorResponse
//	@Router			/accounts/seed [post]
func (ac *AccountController) Seed(c fiber.Ctx) error {
	var req model.SeedAccountRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}

	accounts, err := ac.accounts.Seed(c, req)
	if err != nil {
		return err
	}
	return http.Created(c, accounts)
}
