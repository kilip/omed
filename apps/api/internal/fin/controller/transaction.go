// Package controller provides HTTP handlers and endpoint bindings for finance operations.
package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// TransactionServiceContract defines the operations required by TransactionController.
type TransactionServiceContract interface {
	GetByID(c context.Context, id int64) (domain.Transaction, error)
	Create(c context.Context, req model.CreateTransactionRequest) (domain.Transaction, error)
	Update(c context.Context, id int64, req model.UpdateTransactionRequest) (domain.Transaction, error)
	Delete(c context.Context, id int64) error
	Search(c context.Context, req model.SearchTransactionRequest) ([]domain.Transaction, string, error)
}

// TransactionController handles HTTP requests for financial transactions.
type TransactionController struct {
	txs TransactionServiceContract
}

// NewTransactionController constructs a new TransactionController backed by TransactionServiceContract.
func NewTransactionController(txs TransactionServiceContract) TransactionController {
	return TransactionController{txs: txs}
}

// Search searches transactions within a workspace based on query parameters.
//
//	@Summary		Search transactions
//	@Description	Search transactions in the workspace with filters
//	@Tags			Finance Transactions
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int								true	"Workspace ID"
//	@Param			request	body		model.SearchTransactionRequest	true	"Search criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.TransactionResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/transactions/search [post]
func (ctl TransactionController) Search(c fiber.Ctx) error {
	var req model.SearchTransactionRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	items, cursor, err := ctl.txs.Search(c, req)
	if err != nil {
		return err
	}
	out := make([]model.TransactionResponse, len(items))
	for i, it := range items {
		out[i] = toTransactionResponse(it)
	}
	return httpx.WithCursor(c, out, cursor)
}

// Create creates a new transaction with optional inline entry and postings.
//
//	@Summary		Create transaction
//	@Description	Create a new transaction with optional inline entry and postings
//	@Tags			Finance Transactions
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int								true	"Workspace ID"
//	@Param			request	body		model.CreateTransactionRequest	true	"Transaction creation payload"
//	@Success		201		{object}	httpx.WebResponse[model.TransactionResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/transactions [post]
func (ctl TransactionController) Create(c fiber.Ctx) error {
	var req model.CreateTransactionRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	tx, err := ctl.txs.Create(c, req)
	if err != nil {
		return err
	}
	return httpx.Created(c, toTransactionResponse(tx))
}

// GetByID retrieves a single transaction by ID with its entries.
//
//	@Summary		Get transaction by ID
//	@Description	Retrieve transaction detail by ID with associated entries
//	@Tags			Finance Transactions
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Transaction ID"
//	@Success		200	{object}	httpx.WebResponse[model.TransactionResponse]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/transactions/{id} [get]
func (ctl TransactionController) GetByID(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	tx, err := ctl.txs.GetByID(c, id)
	if err != nil {
		return err
	}
	return httpx.OK(c, toTransactionResponse(tx))
}

// Update updates an existing transaction's header fields (description, reference).
//
//	@Summary		Update transaction
//	@Description	Update transaction header fields (blocked if any entry is posted)
//	@Tags			Finance Transactions
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int								true	"Workspace ID"
//	@Param			id		path		int								true	"Transaction ID"
//	@Param			request	body		model.UpdateTransactionRequest	true	"Transaction update payload"
//	@Success		200		{object}	httpx.WebResponse[model.TransactionResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/transactions/{id} [put]
func (ctl TransactionController) Update(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	var req model.UpdateTransactionRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	tx, err := ctl.txs.Update(c, id, req)
	if err != nil {
		return err
	}
	return httpx.OK(c, toTransactionResponse(tx))
}

// Delete marks an existing transaction as soft-deleted.
//
//	@Summary		Delete transaction
//	@Description	Soft delete a transaction by ID (blocked if any entry is posted)
//	@Tags			Finance Transactions
//	@Produce		json
//	@Param			wid	path		int	true	"Workspace ID"
//	@Param			id	path		int	true	"Transaction ID"
//	@Success		200	{object}	httpx.WebResponse[httpx.NoContent]
//	@Failure		400	{object}	httpx.ErrorResponse
//	@Failure		403	{object}	httpx.ErrorResponse
//	@Failure		404	{object}	httpx.ErrorResponse
//	@Failure		409	{object}	httpx.ErrorResponse
//	@Failure		500	{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/transactions/{id} [delete]
func (ctl TransactionController) Delete(c fiber.Ctx) error {
	id, err := httpx.GetParamID[int64](c, "id")
	if err != nil {
		return err
	}
	if err := ctl.txs.Delete(c, id); err != nil {
		return err
	}
	return httpx.OK(c, httpx.NoContent{})
}

func toTransactionResponse(t domain.Transaction) model.TransactionResponse {
	var entries []model.EntryResponse
	if t.Entries != nil {
		entries = make([]model.EntryResponse, len(t.Entries))
		for i, e := range t.Entries {
			entries[i] = toEntryResponse(e)
		}
	}
	return model.TransactionResponse{
		ID:              t.ID,
		WorkspaceID:     t.WorkspaceID,
		Number:          t.Number,
		TransactionDate: model.Date(t.TransactionDate),
		Description:     t.Description,
		Reference:       t.Reference,
		Currency:        t.Currency,
		ExchangeRate:    t.ExchangeRate,
		Source:          t.Source,
		Entries:         entries,
		AuditMeta:       t.AuditMeta,
	}
}
