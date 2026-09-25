package http

import (
	"log/slog"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/authz"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/service"
)

type AccountController struct {
	service service.AccountService
	logger  *slog.Logger
}

func NewAccountController(app *fiber.App, svc service.AccountService, enforcer *casbin.Enforcer, logger *slog.Logger) AccountController {
	ctl := AccountController{
		service: svc,
	}

	ctl.initRoutes(app, enforcer)

	return ctl
}

func (ctl AccountController) initRoutes(app *fiber.App, enforcer *casbin.Enforcer) {
	app.Post("/accounts", RequirePermission(enforcer, authz.ResourceAccounts, authz.ActionWrite), ctl.create)
	app.Put("/accounts/:id", RequirePermission(enforcer, authz.ResourceAccounts, authz.ActionWrite), ctl.update)
	app.Delete("/accounts/:id", RequirePermission(enforcer, authz.ResourceAccounts, authz.ActionWrite), ctl.delete)
	app.Get("/accounts/:id", RequirePermission(enforcer, authz.ResourceAccounts, authz.ActionRead), ctl.getById)
	app.Get("/accounts", RequirePermission(enforcer, authz.ResourceAccounts, authz.ActionRead), ctl.list)
}

// @Summary		Create Account
// @Description	Create a new account based on authenticated user workspace
// @Tags		accounts
// @Accept		json
// @Produce		json
// @Param		account	body		model.AccountRequest	true	"Account Payload"
// @Success		201		{object}	model.Envelope[model.AccountResponse]
// @Failure		400		{object}	http.ErrorResponse	"Invalid request payload"
// @Failure		401		{object}	http.ErrorResponse	"Missing or invalid authentication"
// @Failure		403		{object}	http.ErrorResponse	"Insufficient permission"
// @Failure		422		{object}	http.ErrorResponse	"Validation failed"
// @Failure		500		{object}	http.ErrorResponse	"Internal server error"
// @Security		BearerAuth
// @Router			/accounts [post]
func (ctl AccountController) create(c fiber.Ctx) error {
	request := new(model.AccountRequest)

	if err := c.Bind().Body(request); err != nil {
		return ErrBadRequest(err.Error())
	}

	created, err := ctl.service.Create(c, *request)
	if err != nil {
		return err
	}
	return Success(c, fiber.StatusCreated, created)
}

// @Summary		Update Account
// @Description	Update Account
// @Tags		accounts
// @Accept		json
// @Produce		json
// @Param		id		path		string					true 	"Account ID"
// @Param		account	body		model.AccountRequest	true	"Account Payload"
// @Success		201		{object}	model.Envelope[model.AccountResponse]
// @Failure		400		{object}	http.ErrorResponse	"Invalid request payload"
// @Failure		401		{object}	http.ErrorResponse	"Missing or invalid authentication"
// @Failure		403		{object}	http.ErrorResponse	"Insufficient permission"
// @Failure		422		{object}	http.ErrorResponse	"Validation failed"
// @Failure		500		{object}	http.ErrorResponse	"Internal server error"
// @Security	BearerAuth
// @Router		/accounts/{id} [put]
func (ctl AccountController) update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return Fail(c, fiber.StatusBadRequest, "INVALID_ACOUNT_ID", err.Error(), nil)
	}

	request := new(model.AccountRequest)
	if err := c.Bind().Body(request); err != nil {
		return Fail(c, fiber.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
	}

	env, err := ctl.service.Update(c, id, *request)
	if err != nil {
		return err
	}
	return Success(c, fiber.StatusOK, env)
}

func (ctl AccountController) delete(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return Fail(c, fiber.StatusBadRequest, "INVALID_ACOUNT_ID", err.Error(), nil)
	}

	err = ctl.service.Delete(c, id)
	if err != nil {
		return Fail(c, fiber.StatusBadRequest, "ACCOUNT_DELETE_ERROR", err.Error(), nil)
	}

	return NoContent(c)
}

func (ctl AccountController) getById(c fiber.Ctx) error {
	id, err := GetIdParam(c)
	if err != nil {
		return err
	}

	env, err := ctl.service.GetByID(c, *id)
	if err != nil {
		return Fail(c, fiber.StatusInternalServerError, "ACCOUNT_GETBYID", err.Error(), nil)
	}

	return Success(c, fiber.StatusOK, env)
}

// @Summary		List available accounts
// @Description	List available accounts
// @Tags		accounts
// @Accept		json
// @Produce		json
// @Param		account	body		model.AccountFilter	false	"Account Filter"
// @Success		200		{object}	model.Envelope[model.Account[]]
// @Failure		400		{object}	http.ErrorResponse	"Invalid request payload"
// @Failure		401		{object}	http.ErrorResponse	"Missing or invalid authentication"
// @Failure		403		{object}	http.ErrorResponse	"Insufficient permission"
// @Failure		422		{object}	http.ErrorResponse	"Validation failed"
// @Failure		500		{object}	http.ErrorResponse	"Internal server error"
// @Security	BearerAuth
// @Router		/accounts [get]
func (ctl AccountController) list(c fiber.Ctx) error {
	request := new(model.AccountFilter)

	if err := c.Bind().Body(request); err != nil {
		return Fail(c, fiber.StatusBadRequest, "Bad Account Filter", err.Error(), nil)
	}

	accounts, err := ctl.service.List(c, *request)
	if err != nil {
		return err
	}
	return Success(c, fiber.StatusOK, accounts)
}
