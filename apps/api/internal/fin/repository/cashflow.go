// Package repository provides database access implementations for finance entities using Ent.
package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/account"
	"github.com/kilip/omed/api/ent/entry"
	"github.com/kilip/omed/api/ent/posting"
	"github.com/kilip/omed/api/ent/predicate"
	"github.com/kilip/omed/api/ent/transaction"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/entx"
	"github.com/shopspring/decimal"
)

// CashFlowItemParams carries item line information for creating or updating a cash flow document.
type CashFlowItemParams struct {
	AccountID   int64
	Description string
	Quantity    decimal.Decimal
	UnitPrice   decimal.Decimal
	Amount      decimal.Decimal
	BaseAmount  decimal.Decimal
}

// CreateCashFlowParams carries parameters required to record a multi-item cash flow document.
type CreateCashFlowParams struct {
	FlowType         string
	PeriodID         int64
	PaymentAccountID int64
	Items            []CashFlowItemParams
	TotalAmount      decimal.Decimal
	TotalBaseAmount  decimal.Decimal
	Status           string
	TransactionDate  time.Time
	Description      string
	Reference        string
	Currency         string
	ExchangeRate     decimal.Decimal
	PostedBy         int64
}

// CashFlowRepository defines database operations for income and expense quick entry transactions.
type CashFlowRepository struct {
	cli *ent.Client
}

// NewCashFlowRepository constructs a new CashFlowRepository backed by the given Ent client.
func NewCashFlowRepository(cli *ent.Client) CashFlowRepository {
	return CashFlowRepository{cli: cli}
}

// Create creates a transaction, journal entry, and balanced postings for a cash flow document in a single atomic transaction.
// Automatically retries on sequence number collision.
func (r CashFlowRepository) Create(c context.Context, params CreateCashFlowParams) (domain.CashFlowItem, error) {
	var dom domain.CashFlowItem
	var createdID int64

	year := params.TransactionDate.Year()
	const maxRetries = 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
			num, err := GenerateTransactionNumber(c, tx, year)
			if err != nil {
				return err
			}

			curr := "IDR"
			wsID := base.WorkspaceIDFromContext(c)
			if wsID > 0 {
				if ws, err := tx.Workspace.Get(c, wsID); err == nil && ws.BaseCurrency != "" {
					curr = ws.BaseCurrency
				}
			}
			if params.Currency != "" {
				curr = params.Currency
			}

			rate := decimal.NewFromInt(1)
			if params.ExchangeRate.IsPositive() {
				rate = params.ExchangeRate
			}

			tBuilder := tx.Transaction.Create().
				SetNumber(num).
				SetTransactionDate(params.TransactionDate).
				SetCurrency(curr).
				SetExchangeRate(rate).
				SetSource(transaction.Source(domain.TransactionSourceManual))

			if params.Description != "" {
				tBuilder.SetDescription(params.Description)
			}
			if params.Reference != "" {
				tBuilder.SetReference(params.Reference)
			}

			t, err := tBuilder.Save(c)
			if err != nil {
				return err
			}
			createdID = t.ID

			entryNum, err := GenerateEntryNumber(c, tx, year)
			if err != nil {
				return err
			}

			targetStatus := params.Status
			if targetStatus == "" {
				targetStatus = domain.EntryStatusPosted
			}

			eBuilder := tx.Entry.Create().
				SetTransactionID(t.ID).
				SetPeriodID(params.PeriodID).
				SetNumber(entryNum).
				SetEntryDate(params.TransactionDate).
				SetStatus(entry.Status(targetStatus))

			if targetStatus == domain.EntryStatusPosted {
				eBuilder.SetPostedAt(time.Now())
				if params.PostedBy > 0 {
					eBuilder.SetPostedBy(params.PostedBy)
				}
			}

			if params.Description != "" {
				eBuilder.SetDescription(params.Description)
			}

			createdEntry, err := eBuilder.Save(c)
			if err != nil {
				return err
			}

			var paymentDir, itemDir posting.Direction
			if params.FlowType == domain.CashFlowTypeExpense {
				paymentDir = posting.DirectionCREDIT
				itemDir = posting.DirectionDEBIT
			} else {
				paymentDir = posting.DirectionDEBIT
				itemDir = posting.DirectionCREDIT
			}

			p0 := tx.Posting.Create().
				SetEntryID(createdEntry.ID).
				SetAccountID(params.PaymentAccountID).
				SetDirection(paymentDir).
				SetAmount(params.TotalAmount).
				SetBaseAmount(params.TotalBaseAmount).
				SetPosition(0)
			if params.Description != "" {
				p0.SetDescription(params.Description)
			}
			if _, err := p0.Save(c); err != nil {
				return err
			}

			for i, it := range params.Items {
				pi := tx.Posting.Create().
					SetEntryID(createdEntry.ID).
					SetAccountID(it.AccountID).
					SetDirection(itemDir).
					SetAmount(it.Amount).
					SetBaseAmount(it.BaseAmount).
					SetPosition(i + 1)
				if it.Description != "" {
					pi.SetDescription(it.Description)
				} else if params.Description != "" {
					pi.SetDescription(params.Description)
				}
				if !it.Quantity.IsZero() {
					pi.SetQuantity(it.Quantity)
				}
				if !it.UnitPrice.IsNegative() {
					pi.SetUnitPrice(it.UnitPrice)
				}
				if _, err := pi.Save(c); err != nil {
					return err
				}
			}

			return nil
		})

		if err == nil {
			return r.GetByID(c, params.FlowType, createdID)
		}

		if ent.IsConstraintError(err) || isLockOrConflictError(err) {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		return dom, err
	}

	return dom, domain.ErrTransactionNumberTaken
}

// Update replaces the postings and updates header fields of a DRAFT cash flow document.
func (r CashFlowRepository) Update(c context.Context, id int64, params CreateCashFlowParams) (domain.CashFlowItem, error) {
	var dom domain.CashFlowItem

	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		t, err := tx.Transaction.Query().
			Where(
				transaction.ID(id),
				transaction.DeletedAtIsNil(),
			).
			WithEntries(func(eq *ent.EntryQuery) {
				eq.Where(entry.DeletedAtIsNil()).
					Order(ent.Asc(entry.FieldID))
			}).
			Only(c)
		if err != nil {
				if ent.IsNotFound(err) {
					return domain.ErrTransactionNotFound
				}
				return err
			}

		if len(t.Edges.Entries) == 0 {
			return domain.ErrTransactionNotFound
		}
		e := t.Edges.Entries[0]
		if string(e.Status) != domain.EntryStatusDraft {
			return domain.ErrEntryNotDraft
		}

		curr := t.Currency
		if params.Currency != "" {
			curr = params.Currency
		}
		rate := t.ExchangeRate
		if params.ExchangeRate.IsPositive() {
			rate = params.ExchangeRate
		}

		tUpdate := tx.Transaction.UpdateOneID(t.ID).
			SetTransactionDate(params.TransactionDate).
			SetCurrency(curr).
			SetExchangeRate(rate)
		if params.Description != "" {
			tUpdate.SetDescription(params.Description)
		} else {
			tUpdate.ClearDescription()
		}
		if params.Reference != "" {
			tUpdate.SetReference(params.Reference)
		} else {
			tUpdate.ClearReference()
		}
		if _, err := tUpdate.Save(c); err != nil {
			return err
		}

		targetStatus := params.Status
		if targetStatus == "" {
			targetStatus = domain.EntryStatusDraft
		}

		eUpdate := tx.Entry.UpdateOneID(e.ID).
			SetPeriodID(params.PeriodID).
			SetEntryDate(params.TransactionDate).
			SetStatus(entry.Status(targetStatus))
		if params.Description != "" {
			eUpdate.SetDescription(params.Description)
		} else {
			eUpdate.ClearDescription()
		}
		if targetStatus == domain.EntryStatusPosted {
			eUpdate.SetPostedAt(time.Now())
			if params.PostedBy > 0 {
				eUpdate.SetPostedBy(params.PostedBy)
			}
		}
		if _, err := eUpdate.Save(c); err != nil {
			return err
		}

		if _, err := tx.Posting.Delete().Where(posting.EntryIDEQ(e.ID)).Exec(c); err != nil {
			return err
		}

		var paymentDir, itemDir posting.Direction
		if params.FlowType == domain.CashFlowTypeExpense {
			paymentDir = posting.DirectionCREDIT
			itemDir = posting.DirectionDEBIT
		} else {
			paymentDir = posting.DirectionDEBIT
			itemDir = posting.DirectionCREDIT
		}

		p0 := tx.Posting.Create().
			SetEntryID(e.ID).
			SetAccountID(params.PaymentAccountID).
			SetDirection(paymentDir).
			SetAmount(params.TotalAmount).
			SetBaseAmount(params.TotalBaseAmount).
			SetPosition(0)
		if params.Description != "" {
			p0.SetDescription(params.Description)
		}
		if _, err := p0.Save(c); err != nil {
			return err
		}

		for i, it := range params.Items {
			pi := tx.Posting.Create().
				SetEntryID(e.ID).
				SetAccountID(it.AccountID).
				SetDirection(itemDir).
				SetAmount(it.Amount).
				SetBaseAmount(it.BaseAmount).
				SetPosition(i + 1)
			if it.Description != "" {
				pi.SetDescription(it.Description)
			} else if params.Description != "" {
				pi.SetDescription(params.Description)
			}
			if !it.Quantity.IsZero() {
				pi.SetQuantity(it.Quantity)
			}
			if !it.UnitPrice.IsNegative() {
				pi.SetUnitPrice(it.UnitPrice)
			}
			if _, err := pi.Save(c); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return dom, err
	}

	return r.GetByID(c, params.FlowType, id)
}

// GetByID retrieves a single cash flow transaction by ID and verifies the requested flowType.
func (r CashFlowRepository) GetByID(c context.Context, flowType string, id int64) (domain.CashFlowItem, error) {
	var dom domain.CashFlowItem

	t, err := r.cli.Transaction.Query().
		Where(
			transaction.ID(id),
			transaction.DeletedAtIsNil(),
		).
		WithCreator().
		WithUpdater().
		WithDeleter().
		WithEntries(func(eq *ent.EntryQuery) {
			eq.Where(entry.DeletedAtIsNil()).
				Order(ent.Asc(entry.FieldID)).
				WithPostings(func(pq *ent.PostingQuery) {
					pq.Order(ent.Asc(posting.FieldPosition), ent.Asc(posting.FieldID)).
						WithAccount()
				})
		}).
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom, domain.ErrTransactionNotFound
		}
		return dom, err
	}

	res := toCashFlowDomain(t, "")
	if flowType != "" && res.Type != flowType {
		return dom, domain.ErrTransactionNotFound
	}
	return res, nil
}

// Search queries cash flow transactions matching the given filter criteria and flow type (INCOME or EXPENSE).
func (r CashFlowRepository) Search(c context.Context, flowType string, req model.SearchCashFlowRequest) ([]domain.CashFlowItem, string, error) {
	limit := 20
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
		if limit > 100 {
			limit = 100
		}
	}

	var paymentDir, categoryDir posting.Direction
	var categoryAccType account.Type
	if flowType == domain.CashFlowTypeExpense {
		paymentDir = posting.DirectionCREDIT
		categoryDir = posting.DirectionDEBIT
		categoryAccType = account.TypeEXPENSE
	} else {
		paymentDir = posting.DirectionDEBIT
		categoryDir = posting.DirectionCREDIT
		categoryAccType = account.TypeREVENUE
	}

	paymentPostPreds := []predicate.Posting{
		posting.DirectionEQ(paymentDir),
		posting.HasAccountWith(account.TypeEQ(account.TypeASSET)),
	}
	if req.PaymentAccountID != nil && *req.PaymentAccountID > 0 {
		paymentPostPreds = append(paymentPostPreds, posting.AccountIDEQ(*req.PaymentAccountID))
	}

	categoryPostPreds := []predicate.Posting{
		posting.DirectionEQ(categoryDir),
		posting.HasAccountWith(account.TypeEQ(categoryAccType)),
	}
	if req.CategoryAccountID != nil && *req.CategoryAccountID > 0 {
		categoryPostPreds = append(categoryPostPreds, posting.AccountIDEQ(*req.CategoryAccountID))
	}

	var allowedStatuses []entry.Status
	if req.Status != nil && *req.Status != "" {
		allowedStatuses = []entry.Status{entry.Status(*req.Status)}
	} else {
		allowedStatuses = []entry.Status{
			entry.Status(domain.EntryStatusDraft),
			entry.Status(domain.EntryStatusPosted),
		}
	}

	entryPreds := []predicate.Entry{
		entry.DeletedAtIsNil(),
		entry.StatusIn(allowedStatuses...),
		entry.HasPostingsWith(paymentPostPreds...),
		entry.HasPostingsWith(categoryPostPreds...),
	}

	q := r.cli.Transaction.Query().
		Where(
			transaction.DeletedAtIsNil(),
			transaction.HasEntriesWith(entryPreds...),
		)

	if req.DateFrom != nil && !req.DateFrom.IsZero() {
		q = q.Where(transaction.TransactionDateGTE(req.DateFrom.Time()))
	}
	if req.DateTo != nil && !req.DateTo.IsZero() {
		q = q.Where(transaction.TransactionDateLTE(req.DateTo.Time()))
	}
	if req.Query != nil && *req.Query != "" {
		q = q.Where(transaction.Or(
			transaction.NumberContainsFold(*req.Query),
			transaction.DescriptionContainsFold(*req.Query),
			transaction.ReferenceContainsFold(*req.Query),
		))
	}
	if req.Cursor != nil && *req.Cursor != "" {
		cursorID, err := strconv.ParseInt(*req.Cursor, 10, 64)
		if err != nil || cursorID < 0 {
			return nil, "", base.ErrInvalidCursor
		}
		q = q.Where(transaction.IDLT(cursorID))
	}

	items, err := q.
		Order(ent.Desc(transaction.FieldID)).
		Limit(limit + 1).
		WithCreator().
		WithUpdater().
		WithDeleter().
		WithEntries(func(eq *ent.EntryQuery) {
			eq.Where(entryPreds...).
				Order(ent.Desc(entry.FieldID)).
				WithPostings(func(pq *ent.PostingQuery) {
					pq.Order(ent.Asc(posting.FieldPosition), ent.Asc(posting.FieldID)).
						WithAccount()
				})
		}).
		All(c)
	if err != nil {
		return nil, "", err
	}

	var cursor string
	if len(items) > limit {
		cursor = strconv.FormatInt(items[limit-1].ID, 10)
		items = items[:limit]
	}

	res := make([]domain.CashFlowItem, len(items))
	for i, it := range items {
		res[i] = toCashFlowDomain(it, flowType)
	}

	return res, cursor, nil
}

func toCashFlowDomain(t *ent.Transaction, flowType string) domain.CashFlowItem {
	dom := domain.CashFlowItem{
		ID:              t.ID,
		WorkspaceID:     t.WorkspaceID,
		Number:          t.Number,
		TransactionDate: t.TransactionDate,
		Type:            flowType,
		Currency:        t.Currency,
		ExchangeRate:    t.ExchangeRate,
		Description:     t.Description,
		Reference:       t.Reference,
		Items:           []domain.CashFlowLine{},
		AuditMeta: base.AuditMeta{
			CreatedAt: t.CreatedAt,
			UpdatedAt: t.UpdatedAt,
			DeletedAt: t.DeletedAt,
		},
	}

	if t.Edges.Creator != nil {
		dom.AuditMeta.CreatedBy = base.ByUser{
			Name:   t.Edges.Creator.Name,
			Avatar: t.Edges.Creator.Avatar,
		}
	}
	if t.Edges.Updater != nil {
		dom.AuditMeta.UpdatedBy = base.ByUser{
			Name:   t.Edges.Updater.Name,
			Avatar: t.Edges.Updater.Avatar,
		}
	}
	if t.Edges.Deleter != nil {
		dom.AuditMeta.DeletedBy = &base.ByUser{
			Name:   t.Edges.Deleter.Name,
			Avatar: t.Edges.Deleter.Avatar,
		}
	}

	if len(t.Edges.Entries) == 0 {
		return dom
	}

	var e *ent.Entry
	for _, entryCandidate := range t.Edges.Entries {
		if string(entryCandidate.Status) == domain.EntryStatusPosted {
			e = entryCandidate
			break
		}
	}
	if e == nil {
		for _, entryCandidate := range t.Edges.Entries {
			if string(entryCandidate.Status) == domain.EntryStatusDraft {
				e = entryCandidate
				break
			}
		}
	}
	if e == nil {
		e = t.Edges.Entries[0]
	}

	dom.EntryID = e.ID
	dom.EntryNumber = e.Number
	dom.EntryStatus = string(e.Status)

	detectedType := flowType
	if detectedType == "" {
		for _, p := range e.Edges.Postings {
			if p.Edges.Account != nil {
				if string(p.Edges.Account.Type) == domain.AccountTypeExpense {
					detectedType = domain.CashFlowTypeExpense
					break
				}
				if string(p.Edges.Account.Type) == domain.AccountTypeRevenue {
					detectedType = domain.CashFlowTypeIncome
					break
				}
			}
		}
	}
	if detectedType == "" {
		detectedType = domain.CashFlowTypeExpense
	}
	dom.Type = detectedType

	for _, p := range e.Edges.Postings {
		if p.Edges.Account == nil {
			continue
		}
		summary := domain.CashFlowAccountSummary{
			ID:   p.Edges.Account.ID,
			Code: p.Edges.Account.Code,
			Name: p.Edges.Account.Name,
			Type: string(p.Edges.Account.Type),
		}

		if detectedType == domain.CashFlowTypeExpense {
			if string(p.Direction) == domain.PostingDirectionCredit {
				dom.PaymentAccount = summary
				dom.Amount = p.Amount
				dom.BaseAmount = p.BaseAmount
			} else if string(p.Direction) == domain.PostingDirectionDebit {
				qty := decimal.NewFromInt(1)
				if p.Quantity != nil {
					qty = *p.Quantity
				}
				uPrice := p.Amount
				if p.UnitPrice != nil {
					uPrice = *p.UnitPrice
				}
				dom.Items = append(dom.Items, domain.CashFlowLine{
					ID:          p.ID,
					Account:     summary,
					Description: p.Description,
					Quantity:    qty,
					UnitPrice:   uPrice,
					Amount:      p.Amount,
					BaseAmount:  p.BaseAmount,
				})
			}
		} else {
			if string(p.Direction) == domain.PostingDirectionDebit {
				dom.PaymentAccount = summary
				dom.Amount = p.Amount
				dom.BaseAmount = p.BaseAmount
			} else if string(p.Direction) == domain.PostingDirectionCredit {
				qty := decimal.NewFromInt(1)
				if p.Quantity != nil {
					qty = *p.Quantity
				}
				uPrice := p.Amount
				if p.UnitPrice != nil {
					uPrice = *p.UnitPrice
				}
				dom.Items = append(dom.Items, domain.CashFlowLine{
					ID:          p.ID,
					Account:     summary,
					Description: p.Description,
					Quantity:    qty,
					UnitPrice:   uPrice,
					Amount:      p.Amount,
					BaseAmount:  p.BaseAmount,
				})
			}
		}
	}

	return dom
}
