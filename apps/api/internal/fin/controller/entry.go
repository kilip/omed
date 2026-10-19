// Package controller provides HTTP handlers and endpoint bindings for finance operations.
package controller

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// EntryServiceContract defines the operations required by EntryController.
type EntryServiceContract interface {
	GetByID(c context.Context, id int64) (domain.Entry, error)
	Create(c context.Context, txID int64, req model.CreateEntryRequest) (domain.Entry, error)
	Update(c context.Context, id int64, req model.UpdateEntryRequest) (domain.Entry, error)
	Delete(c context.Context, id int64) error
	Search(c context.Context, req model.SearchEntryRequest) ([]domain.Entry, string, error)
	Post(c context.Context, id int64, postedBy int64) (domain.Entry, error)
	Reverse(c context.Context, id int64, postedBy int64, entryDate time.Time, description string) (domain.Entry, error)
	Void(c context.Context, id int64, postedBy int64) (domain.Entry, error)
	GetAccountLedger(c context.Context, accountID int64, page int, limit int) ([]domain.Posting, int64, error)
}

// EntryController handles HTTP requests for journal entries.
type EntryController struct {
	entries EntryServiceContract
}

// NewEntryController constructs a new EntryController backed by EntryServiceContract.
func NewEntryController(entries EntryServiceContract) EntryController {
	return EntryController{entries: entries}
}

// Search searches journal entries within a workspace based on query parameters.
//
//	@Summary		Search entries
//	@Description	Search journal entries in the workspace with filters
//	@Tags			Finance Entries
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.SearchEntryRequest	true	"Search criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.EntryResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/entries/search [post]
func (ctl EntryController) Search(c fiber.Ctx) error {
	var req model.SearchEntryRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	items, cursor, err := ctl.entries.Search(c, req)
	if err != nil {
		return err
	}
	out := make([]model.EntryResponse, len(items))
	for i, it := range items {
		out[i] = toEntryResponse(it)
	}
	return httpx.WithCursor(c, out, cursor)
}

// Create creates a new draft journal entry within a transaction.
//
//	@Summary		Create entry
//	@Description	Add a draft journal entry to an existing transaction
//	@Tags			Finance Entries
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.CreateEntryRequest	true	"Entry creation payload"
//	@Success		201		{object}	httpx.WebResponse[model.EntryResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/entries [post]
func (ctl EntryController) Create(c fiber.Ctx) error {
	var req model.CreateEntryRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	entry, err := ctl.entries.Create(c, req.TransactionID, req)
	if err != nil {
		return err
	}
	return httpx.Created(c, toEntryResponse(entry))
}

// GetByID retrieves a single journal entry by ID with its postings.
//
//	@Summary		Get entry by ID
//	@Description	Retrieve journal entry detail by ID with associated postings
//	@Tags			Finance Entries
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Entry ID"
//	@Success		200	{object}	httpx.WebResponse[model.EntryResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/entries/{id} [get]
func (ctl EntryController) GetByID(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	entry, err := ctl.entries.GetByID(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, toEntryResponse(entry))
}

// Update updates a draft journal entry's description and postings.
//
//	@Summary		Update entry
//	@Description	Replace postings for a draft journal entry (DRAFT only)
//	@Tags			Finance Entries
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			id		path		int							true	"Entry ID"
//	@Param			request	body		model.UpdateEntryRequest	true	"Entry update payload"
//	@Success		200		{object}	httpx.WebResponse[model.EntryResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/entries/{id} [put]
func (ctl EntryController) Update(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	var req model.UpdateEntryRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	entry, err := ctl.entries.Update(c, id, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, toEntryResponse(entry))
}

// Delete soft-deletes a draft journal entry.
//
//	@Summary		Delete entry
//	@Description	Delete a journal entry and cascade delete postings (DRAFT only)
//	@Tags			Finance Entries
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Entry ID"
//	@Success		200	{object}	httpx.WebResponse[httpx.NoContent]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/entries/{id} [delete]
func (ctl EntryController) Delete(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	if err := ctl.entries.Delete(c, id); err != nil {
		return err
	}
	return httpx.OK(c, httpx.NoContent{})
}

// Post transitions a journal entry from DRAFT to POSTED after validation.
//
//	@Summary		Post entry
//	@Description	Transition journal entry from DRAFT to POSTED status
//	@Tags			Finance Entries
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Entry ID"
//	@Success		200	{object}	httpx.WebResponse[model.EntryResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse
//	@Failure		422	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/entries/{id}/post [post]
func (ctl EntryController) Post(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	user := base.UserFromContext(c)
	if user.ID == 0 {
		return base.ErrUnauthorized
	}
	entry, err := ctl.entries.Post(c, id, user.ID)
	if err != nil {
		return err
	}
	return httpx.OK(c, toEntryResponse(entry))
}

// Reverse creates a reversing journal entry with POSTED status, swapping DEBIT and CREDIT.
//
//	@Summary		Reverse entry
//	@Description	Create a POSTED reversing entry for a posted journal entry
//	@Tags			Finance Entries
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			id		path		int							true	"Entry ID"
//	@Param			request	body		model.ReverseEntryRequest	false	"Reversal options"
//	@Success		201		{object}	httpx.WebResponse[model.EntryResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/entries/{id}/reverse [post]
func (ctl EntryController) Reverse(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	user := base.UserFromContext(c)
	if user.ID == 0 {
		return base.ErrUnauthorized
	}
	var req model.ReverseEntryRequest
	_ = c.Bind().Body(&req)

	var entryDate time.Time
	if req.EntryDate != nil && !req.EntryDate.IsZero() {
		entryDate = req.EntryDate.Time()
	}
	entry, err := ctl.entries.Reverse(c, id, user.ID, entryDate, req.Description)
	if err != nil {
		return err
	}
	return httpx.Created(c, toEntryResponse(entry))
}

// Void voids a journal entry (if POSTED, reverses it first, then marks VOID).
//
//	@Summary		Void entry
//	@Description	Void a journal entry (reverses if POSTED, then marks VOID)
//	@Tags			Finance Entries
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Entry ID"
//	@Success		200	{object}	httpx.WebResponse[model.EntryResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		401	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse
//	@Failure		422	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/entries/{id}/void [post]
func (ctl EntryController) Void(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	user := base.UserFromContext(c)
	if user.ID == 0 {
		return base.ErrUnauthorized
	}
	entry, err := ctl.entries.Void(c, id, user.ID)
	if err != nil {
		return err
	}
	return httpx.OK(c, toEntryResponse(entry))
}

func toPostingResponse(p domain.Posting) model.PostingResponse {
	return model.PostingResponse{
		ID:             p.ID,
		WorkspaceID:    p.WorkspaceID,
		EntryID:        p.EntryID,
		AccountID:      p.AccountID,
		Direction:      p.Direction,
		Amount:         p.Amount,
		BaseAmount:     p.BaseAmount,
		Position:       p.Position,
		Description:    p.Description,
		RunningBalance: p.RunningBalance,
		AuditMeta:      p.AuditMeta,
	}
}

func toEntryResponse(e domain.Entry) model.EntryResponse {
	var postings []model.PostingResponse
	if e.Postings != nil {
		postings = make([]model.PostingResponse, len(e.Postings))
		for i, p := range e.Postings {
			postings[i] = toPostingResponse(p)
		}
	}
	return model.EntryResponse{
		ID:            e.ID,
		WorkspaceID:   e.WorkspaceID,
		TransactionID: e.TransactionID,
		PeriodID:      e.PeriodID,
		Number:        e.Number,
		EntryDate:     model.Date(e.EntryDate),
		Description:   e.Description,
		Status:        e.Status,
		PostedAt:      e.PostedAt,
		PostedBy:      e.PostedBy,
		ReversalOfID:  e.ReversalOfID,
		Postings:      postings,
		AuditMeta:     e.AuditMeta,
	}
}
