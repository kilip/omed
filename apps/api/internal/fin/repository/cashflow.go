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

// CreateCashFlowParams carries parameters required to record a posted income or expense event.
type CreateCashFlowParams struct {
	FlowType          string
	PeriodID          int64
	PaymentAccountID  int64
	CategoryAccountID int64
	Amount            decimal.Decimal
	BaseAmount        decimal.Decimal
	TransactionDate   time.Time
	Description       string
	Reference         string
	Currency          string
	ExchangeRate      decimal.Decimal
	PostedBy          int64
}

// CashFlowRepository defines database operations for income and expense quick entry transactions.
type CashFlowRepository struct {
	cli *ent.Client
}

// NewCashFlowRepository constructs a new CashFlowRepository backed by the given Ent client.
func NewCashFlowRepository(cli *ent.Client) CashFlowRepository {
	return CashFlowRepository{cli: cli}
}

// CreatePosted creates a transaction, posted journal entry, and balanced postings for an income or expense event in a single atomic transaction.
// Automatically retries on sequence number collision.
func (r CashFlowRepository) CreatePosted(c context.Context, params CreateCashFlowParams) (domain.CashFlowItem, error) {
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

			baseAmt := params.Amount.Mul(rate)
			if params.BaseAmount.IsPositive() {
				baseAmt = params.BaseAmount
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

			now := time.Now()
			eBuilder := tx.Entry.Create().
				SetTransactionID(t.ID).
				SetPeriodID(params.PeriodID).
				SetNumber(entryNum).
				SetEntryDate(params.TransactionDate).
				SetStatus(entry.Status(domain.EntryStatusPosted)).
				SetPostedAt(now).
				SetPostedBy(params.PostedBy)

			if params.Description != "" {
				eBuilder.SetDescription(params.Description)
			}

			createdEntry, err := eBuilder.Save(c)
			if err != nil {
				return err
			}

			var debitAccID, creditAccID int64
			if params.FlowType == domain.CashFlowTypeExpense {
				// EXPENSE: debit Category (Expense), credit Payment (Asset)
				debitAccID = params.CategoryAccountID
				creditAccID = params.PaymentAccountID
			} else {
				// INCOME: debit Payment (Asset), credit Category (Revenue)
				debitAccID = params.PaymentAccountID
				creditAccID = params.CategoryAccountID
			}

			p0 := tx.Posting.Create().
				SetEntryID(createdEntry.ID).
				SetAccountID(debitAccID).
				SetDirection(posting.Direction(domain.PostingDirectionDebit)).
				SetAmount(params.Amount).
				SetBaseAmount(baseAmt).
				SetPosition(0)
			if params.Description != "" {
				p0.SetDescription(params.Description)
			}
			if _, err := p0.Save(c); err != nil {
				return err
			}

			p1 := tx.Posting.Create().
				SetEntryID(createdEntry.ID).
				SetAccountID(creditAccID).
				SetDirection(posting.Direction(domain.PostingDirectionCredit)).
				SetAmount(params.Amount).
				SetBaseAmount(baseAmt).
				SetPosition(1)
			if params.Description != "" {
				p1.SetDescription(params.Description)
			}
			if _, err := p1.Save(c); err != nil {
				return err
			}

			return nil
		})

		if err == nil {
			return r.getByID(c, params.FlowType, createdID)
		}

		if ent.IsConstraintError(err) || isLockOrConflictError(err) {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		return dom, err
	}

	return dom, domain.ErrTransactionNumberTaken
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

	entryPreds := []predicate.Entry{
		entry.DeletedAtIsNil(),
		entry.StatusEQ(entry.Status(domain.EntryStatusPosted)),
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

func (r CashFlowRepository) getByID(c context.Context, flowType string, id int64) (domain.CashFlowItem, error) {
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

	return toCashFlowDomain(t, flowType), nil
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
				if dom.Amount.IsZero() {
					dom.Amount = p.Amount
					dom.BaseAmount = p.BaseAmount
				}
			} else if string(p.Direction) == domain.PostingDirectionDebit {
				dom.CategoryAccount = summary
				if dom.Amount.IsZero() {
					dom.Amount = p.Amount
					dom.BaseAmount = p.BaseAmount
				}
			}
		} else {
			if string(p.Direction) == domain.PostingDirectionDebit {
				dom.PaymentAccount = summary
				if dom.Amount.IsZero() {
					dom.Amount = p.Amount
					dom.BaseAmount = p.BaseAmount
				}
			} else if string(p.Direction) == domain.PostingDirectionCredit {
				dom.CategoryAccount = summary
				if dom.Amount.IsZero() {
					dom.Amount = p.Amount
					dom.BaseAmount = p.BaseAmount
				}
			}
		}
	}

	return dom
}
