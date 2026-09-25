package http

import (
	"log/slog"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v3"
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
	app.Get("/accounts", RequirePermission(enforcer, authz.ResourceAccounts, authz.ActionRead), ctl.list)
}

// @Summary		Create Account
// @Description	Create a new account based on authenticated user workspace
// @Tags			accounts
// @Accept			json
// @Produce		json
// @Param			account	body		model.Account	true	"Account Payload"
// @Success		201		{object}	model.Envelope[model.Account]
// @Failure		400		{object}	http.ErrorResponse	"Invalid request payload"
// @Failure		401		{object}	http.ErrorResponse	"Missing or invalid authentication"
// @Failure		403		{object}	http.ErrorResponse	"Insufficient permission"
// @Failure		422		{object}	http.ErrorResponse	"Validation failed"
// @Failure		500		{object}	http.ErrorResponse	"Internal server error"
// @Security		BearerAuth
// @Router			/accounts [post]
func (ctl AccountController) create(c fiber.Ctx) error {
	request := new(model.Account)

	if err := c.Bind().Body(request); err != nil {
		return ErrBadRequest(err.Error())
	}

	created, err := ctl.service.Create(c, *request)
	if err != nil {
		return err
	}
	return Success(c, fiber.StatusCreated, created)
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
	return Success(c, fiber.StatusCreated, accounts)
}
