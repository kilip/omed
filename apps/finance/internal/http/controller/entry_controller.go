package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/http"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared/authz"
)

type EntryService interface {
	List(ctx context.Context, req model.ListEntryRequest) ([]model.Entry, string, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Entry, error)
	Create(ctx context.Context, req model.CreateEntryRequest) (*model.Entry, error)
}

type EntryController struct {
	svc EntryService
}

func NewEntryController(svc EntryService) *EntryController {
	return &EntryController{svc: svc}
}

func (h *EntryController) Register(r fiber.Router) {
	g := r.Group("/entries")
	read := http.RequirePermission(authz.ResourceEntries, authz.ActionRead)
	write := http.RequirePermission(authz.ResourceEntries, authz.ActionWrite)

	g.Get("/", read, h.List)
	g.Get("/:id", read, h.Get)
	g.Post("/", write, h.Create)
}

// List godoc
//
//	@Summary		List journal entries
//	@Description	Return journal entries with pagination and filters.
//	@Tags			entries
//	@Produce		json
//	@Security		BearerAuth
//	@Param			cursor		query		string	false	"Cursor for pagination"
//	@Param			limit		query		int		false	"Number of items per page"
//	@Param			start_date	query		string	false	"Filter by start date"
//	@Param			end_date	query		string	false	"Filter by end date"
//	@Param			entry_type	query		string	false	"Filter by entry type"	Enums(normal, opening_balance, adjustment, closing, fx_adjustment)
//	@Success		200			{object}	model.WebResponse[[]model.Entry]
//	@Failure		400			{object}	model.ErrorResponse
//	@Failure		401			{object}	model.ErrorResponse
//	@Failure		403			{object}	model.ErrorResponse
//	@Failure		422			{object}	model.ErrorResponse
//	@Failure		500			{object}	model.ErrorResponse
//	@Router			/entries [get]
func (h *EntryController) List(c fiber.Ctx) error {
	var req model.ListEntryRequest
	if err := c.Bind().Query(&req); err != nil {
		return err
	}

	entries, cursor, err := h.svc.List(c, req)
	if err != nil {
		return err
	}
	return http.WithCursor(c, entries, cursor)
}

// Get godoc
//
//	@Summary		Get journal entry
//	@Description	Get a single journal entry by ID.
//	@Tags			entries
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		string	true	"Entry ID"	format(uuid)
//	@Success		200	{object}	model.WebResponse[model.Entry]
//	@Failure		400	{object}	model.ErrorResponse
//	@Failure		401	{object}	model.ErrorResponse
//	@Failure		403	{object}	model.ErrorResponse
//	@Failure		404	{object}	model.ErrorResponse
//	@Failure		500	{object}	model.ErrorResponse
//	@Router			/entries/{id} [get]
func (h *EntryController) Get(c fiber.Ctx) error {
	id, err := http.GetID(c)
	if err != nil {
		return err
	}

	entry, err := h.svc.GetByID(c, *id)
	if err != nil {
		return err
	}
	return http.OK(c, entry)
}

// Create godoc
//
//	@Summary		Create journal entry
//	@Description	Create a new double-entry journal entry with postings.
//	@Tags			entries
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		model.CreateEntryRequest	true	"Journal entry payload"
//	@Success		201		{object}	model.WebResponse[model.Entry]
//	@Failure		400		{object}	model.ErrorResponse
//	@Failure		401		{object}	model.ErrorResponse
//	@Failure		403		{object}	model.ErrorResponse
//	@Failure		422		{object}	model.ErrorResponse
//	@Failure		500		{object}	model.ErrorResponse
//	@Router			/entries [post]
func (h *EntryController) Create(c fiber.Ctx) error {
	var req model.CreateEntryRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}

	entry, err := h.svc.Create(c, req)
	if err != nil {
		return err
	}
	return http.Created(c, entry)
}
