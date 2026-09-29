package controller

import (
	"context"
	"uuid"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/http"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
	"github.com/kilip/omed/finance/internal/shared/authz"
)

type AccountService interface {
	List(ctx context.Context, req model.ListAccountRequest) ([]model.Account, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Account, error)
	Create(ctx context.Context, id uuid.UUID, req model.CreateAccountRequest) (*model.Account, error)
	Update(ctx context.Context, id uuid.UUID, req model.UpdateAccountRequest) (*model.Account, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type AccountController struct {
	svc AccountService
}

func (h *AccountController) Register(r fiber.Router) {
	g := r.Group("/accounts")
	read := http.RequirePermission(authz.ResourceAccounts, authz.ActionRead)
	write := http.RequirePermission(authz.ResourceAccounts, authz.ActionWrite)

	g.Get("/", read, h.List)
	g.Get("/:id", read, h.Get)
	g.Post("/", write, h.Create)
	g.Put("/:id", write, h.Update)
	g.Delete("/:id", write, h.Delete)
}

func (h *AccountController) List(c fiber.Ctx) error {
	return shared.ErrUnimplemented
}

func (h *AccountController) Get(c fiber.Ctx) error {
	return shared.ErrUnimplemented
}

func (h *AccountController) Create(c fiber.Ctx) error {
	return shared.ErrUnimplemented
}

func (h *AccountController) Update(c fiber.Ctx) error {
	return shared.ErrUnimplemented
}

func (h *AccountController) Delete(c fiber.Ctx) error {
	return shared.ErrUnimplemented
}
