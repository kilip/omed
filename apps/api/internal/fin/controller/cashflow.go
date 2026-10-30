// Package controller provides HTTP handlers and endpoint bindings for finance operations.
package controller

import (
	"context"

	"github.com/gofiber/fiber/v3"
	coremodel "github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
)

// CashFlowServiceContract defines the operations required by CashFlowController.
type CashFlowServiceContract interface {
	CreateExpense(c context.Context, req model.CreateExpenseRequest) (domain.CashFlowItem, error)
	CreateIncome(c context.Context, req model.CreateIncomeRequest) (domain.CashFlowItem, error)
	SearchExpenses(c context.Context, req model.SearchCashFlowRequest) ([]domain.CashFlowItem, string, error)
	SearchIncomes(c context.Context, req model.SearchCashFlowRequest) ([]domain.CashFlowItem, string, error)
}

// CashFlowController handles HTTP requests for quick-entry cash flows (expenses & incomes).
type CashFlowController struct {
	svc CashFlowServiceContract
}

// NewCashFlowController constructs a new CashFlowController backed by CashFlowServiceContract.
func NewCashFlowController(svc CashFlowServiceContract) CashFlowController {
	return CashFlowController{svc: svc}
}

// CreateExpense creates a new posted expense cash flow entry.
//
//	@Summary		Create expense
//	@Description	Create a new posted expense cash flow with payment and expense category accounts
//	@Tags			Finance Expenses
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.CreateExpenseRequest	true	"Expense creation payload"
//	@Success		201		{object}	httpx.WebResponse[model.CashFlowItemResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/expense [post]
func (ctl CashFlowController) CreateExpense(c fiber.Ctx) error {
	var req model.CreateExpenseRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	item, err := ctl.svc.CreateExpense(c, req)
	if err != nil {
		return err
	}
	return httpx.Created(c, toCashFlowItemResponse(item))
}

// SearchExpenses searches expense cash flows within a workspace based on query parameters.
//
//	@Summary		Search expenses
//	@Description	Search expense cash flows in the workspace with filters
//	@Tags			Finance Expenses
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int								true	"Workspace ID"
//	@Param			request	body		model.SearchCashFlowRequest		true	"Search criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.CashFlowItemResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/expense/search [post]
func (ctl CashFlowController) SearchExpenses(c fiber.Ctx) error {
	var req model.SearchCashFlowRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	items, cursor, err := ctl.svc.SearchExpenses(c, req)
	if err != nil {
		return err
	}
	out := make([]model.CashFlowItemResponse, len(items))
	for i, it := range items {
		out[i] = toCashFlowItemResponse(it)
	}
	return httpx.WithCursor(c, out, cursor)
}

// CreateIncome creates a new posted income cash flow entry.
//
//	@Summary		Create income
//	@Description	Create a new posted income cash flow with payment and revenue category accounts
//	@Tags			Finance Incomes
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int							true	"Workspace ID"
//	@Param			request	body		model.CreateIncomeRequest	true	"Income creation payload"
//	@Success		201		{object}	httpx.WebResponse[model.CashFlowItemResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		404		{object}	httpx.ErrorResponse
//	@Failure		409		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/income [post]
func (ctl CashFlowController) CreateIncome(c fiber.Ctx) error {
	var req model.CreateIncomeRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	item, err := ctl.svc.CreateIncome(c, req)
	if err != nil {
		return err
	}
	return httpx.Created(c, toCashFlowItemResponse(item))
}

// SearchIncomes searches income cash flows within a workspace based on query parameters.
//
//	@Summary		Search incomes
//	@Description	Search income cash flows in the workspace with filters
//	@Tags			Finance Incomes
//	@Accept			json
//	@Produce		json
//	@Param			wid		path		int								true	"Workspace ID"
//	@Param			request	body		model.SearchCashFlowRequest		true	"Search criteria"
//	@Success		200		{object}	httpx.WebResponse[[]model.CashFlowItemResponse]
//	@Failure		400		{object}	httpx.ErrorResponse
//	@Failure		401		{object}	httpx.ErrorResponse
//	@Failure		403		{object}	httpx.ErrorResponse
//	@Failure		422		{object}	httpx.ErrorResponse
//	@Failure		500		{object}	httpx.ErrorResponse
//	@Security		BearerAuth
//	@Router			/workspaces/{wid}/fin/income/search [post]
func (ctl CashFlowController) SearchIncomes(c fiber.Ctx) error {
	var req model.SearchCashFlowRequest
	if err := c.Bind().Body(&req); err != nil {
		return err
	}
	items, cursor, err := ctl.svc.SearchIncomes(c, req)
	if err != nil {
		return err
	}
	out := make([]model.CashFlowItemResponse, len(items))
	for i, it := range items {
		out[i] = toCashFlowItemResponse(it)
	}
	return httpx.WithCursor(c, out, cursor)
}

func toCashFlowItemResponse(it domain.CashFlowItem) model.CashFlowItemResponse {
	var attachments []coremodel.FileResponse
	if it.Attachments != nil {
		attachments = make([]coremodel.FileResponse, len(it.Attachments))
		for i, a := range it.Attachments {
			attachments[i] = coremodel.FileResponse{
				ID:          a.ID,
				WorkspaceID: a.WorkspaceID,
				Name:        a.Name,
				Bucket:      a.Bucket,
				Key:         a.Key,
				MimeType:    a.MimeType,
				Size:        a.Size,
				EntityType:  a.EntityType,
				EntityID:    a.EntityID,
				CreatedBy:   a.CreatedBy,
				CreatedAt:   a.CreatedAt,
				UpdatedBy:   a.UpdatedBy,
				UpdatedAt:   a.UpdatedAt,
			}
		}
	}
	return model.CashFlowItemResponse{
		ID:              it.ID,
		WorkspaceID:     it.WorkspaceID,
		Number:          it.Number,
		TransactionDate: model.Date(it.TransactionDate),
		Type:            it.Type,
		PaymentAccount: model.CashFlowAccountSummaryResponse{
			ID:   it.PaymentAccount.ID,
			Code: it.PaymentAccount.Code,
			Name: it.PaymentAccount.Name,
			Type: it.PaymentAccount.Type,
		},
		CategoryAccount: model.CashFlowAccountSummaryResponse{
			ID:   it.CategoryAccount.ID,
			Code: it.CategoryAccount.Code,
			Name: it.CategoryAccount.Name,
			Type: it.CategoryAccount.Type,
		},
		Amount:       it.Amount,
		BaseAmount:   it.BaseAmount,
		Currency:     it.Currency,
		ExchangeRate: it.ExchangeRate,
		Description:  it.Description,
		Reference:    it.Reference,
		EntryID:      it.EntryID,
		EntryNumber:  it.EntryNumber,
		EntryStatus:  it.EntryStatus,
		Attachments:  attachments,
		AuditMeta:    it.AuditMeta,
	}
}
