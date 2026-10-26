// Package repository provides database access implementations for finance entities using Ent.
package repository

import (
	"context"
	"strconv"
	"time"

	"entgo.io/ent/dialect"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/entry"
	"github.com/kilip/omed/api/ent/period"
	"github.com/kilip/omed/api/ent/posting"
	"github.com/kilip/omed/api/ent/transaction"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/entx"
	"github.com/shopspring/decimal"
)

// forUpdate applies row-level locking (FOR UPDATE) if the dialect supports it (e.g. Postgres).
// In SQLite (used in test memory db), FOR UPDATE is not supported and transactions are serialized.
func forUpdate(q *ent.EntryQuery, tx *ent.Tx) *ent.EntryQuery {
	if tx != nil && tx.Dialect() != dialect.SQLite {
		return q.ForUpdate()
	}
	return q
}

// EntryRepository defines operations for persisting and querying journal entries and postings.
type EntryRepository struct {
	cli *ent.Client
}

// AccountBalanceSummary holds aggregated debit and credit sums for an account.
type AccountBalanceSummary struct {
	Debit      decimal.Decimal
	Credit     decimal.Decimal
	BaseDebit  decimal.Decimal
	BaseCredit decimal.Decimal
	Currencies map[string]*CurrencyBalanceSummary
}

// MonthlyAccountFlow holds aggregated base debit and credit amounts for an account in a specific month.
type MonthlyAccountFlow struct {
	Month      string
	AccountID  int64
	BaseDebit  decimal.Decimal
	BaseCredit decimal.Decimal
}

// CurrencyBalanceSummary holds aggregated amounts for a specific currency on an account.
type CurrencyBalanceSummary struct {
	Currency   string
	Debit      decimal.Decimal
	Credit     decimal.Decimal
	BaseDebit  decimal.Decimal
	BaseCredit decimal.Decimal
}

// NewEntryRepository constructs a new EntryRepository backed by the given Ent client.
func NewEntryRepository(cli *ent.Client) EntryRepository {
	return EntryRepository{cli: cli}
}

// GetByID retrieves an active journal entry by its ID within a workspace, including its postings ordered by position asc.
func (r EntryRepository) GetByID(c context.Context, id int64) (domain.Entry, error) {
	var dom domain.Entry

	e, err := r.cli.Entry.Query().
		Where(
			entry.ID(id),
			entry.DeletedAtIsNil(),
		).
		WithCreator().
		WithUpdater().
		WithDeleter().
		WithPostings(func(pq *ent.PostingQuery) {
			pq.Order(ent.Asc(posting.FieldPosition), ent.Asc(posting.FieldID)).
				WithCreator().
				WithUpdater()
		}).
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom, domain.ErrEntryNotFound
		}
		return dom, err
	}

	return toEntryDomain(e), nil
}

// Create creates a draft entry with JE-... number and inline postings within a single entx.WithTx.
// Retries if a unique constraint conflict occurs on (workspace_id, number).
func (r EntryRepository) Create(c context.Context, txID int64, req model.CreateEntryRequest) (domain.Entry, error) {
	var dom domain.Entry
	var createdID int64

	year := req.EntryDate.Time().Year()
	const maxRetries = 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
			// Get transaction to inherit exchange rate for base amounts
			t, err := tx.Transaction.Query().
				Where(
					transaction.ID(txID),
					transaction.DeletedAtIsNil(),
				).
				Only(c)
			if err != nil {
				if ent.IsNotFound(err) {
					return domain.ErrTransactionNotFound
				}
				return err
			}

			rate := t.ExchangeRate
			if !rate.IsPositive() {
				rate = decimal.NewFromInt(1)
			}

			num, err := GenerateEntryNumber(c, tx, year)
			if err != nil {
				return err
			}

			eBuilder := tx.Entry.Create().
				SetTransactionID(txID).
				SetPeriodID(req.PeriodID).
				SetNumber(num).
				SetEntryDate(req.EntryDate.Time()).
				SetStatus(entry.Status(domain.EntryStatusDraft))

			if req.Description != "" {
				eBuilder.SetDescription(req.Description)
			}

			createdEntry, err := eBuilder.Save(c)
			if err != nil {
				return err
			}
			createdID = createdEntry.ID

			for pos, pReq := range req.Postings {
				baseAmt := pReq.Amount.Mul(rate)
				if pReq.BaseAmount != nil && pReq.BaseAmount.IsPositive() {
					baseAmt = *pReq.BaseAmount
				}
				pBuilder := tx.Posting.Create().
					SetEntryID(createdEntry.ID).
					SetAccountID(pReq.AccountID).
					SetDirection(posting.Direction(pReq.Direction)).
					SetAmount(pReq.Amount).
					SetBaseAmount(baseAmt).
					SetPosition(pos)

				pDesc := pReq.Description
				if pDesc == "" {
					pDesc = req.Description
				}
				if pDesc != "" {
					pBuilder.SetDescription(pDesc)
				}

				if _, err := pBuilder.Save(c); err != nil {
					return err
				}
			}

			return nil
		})

		if err == nil {
			return r.GetByID(c, createdID)
		}

		if ent.IsConstraintError(err) || isLockOrConflictError(err) {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		return dom, err
	}

	return dom, domain.ErrEntryNotFound
}

// Update replaces postings of an entry. Only allowed if status is DRAFT.
// Removes old postings and saves new postings in a single entx.WithTx.
func (r EntryRepository) Update(c context.Context, id int64, req model.UpdateEntryRequest) (domain.Entry, error) {
	var dom domain.Entry

	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		e, err := forUpdate(tx.Entry.Query(), tx).
			Where(
				entry.ID(id),
				entry.DeletedAtIsNil(),
			).
			WithTransaction().
			Only(c)
		if err != nil {
			if ent.IsNotFound(err) {
				return domain.ErrEntryNotFound
			}
			return err
		}

		if string(e.Status) != domain.EntryStatusDraft {
			return domain.ErrEntryNotDraft
		}

		rate := decimal.NewFromInt(1)
		if e.Edges.Transaction != nil && e.Edges.Transaction.ExchangeRate.IsPositive() {
			rate = e.Edges.Transaction.ExchangeRate
		}

		u := tx.Entry.UpdateOneID(id)
		if req.Description != nil {
			u.SetDescription(*req.Description)
		}
		if err := u.Exec(c); err != nil {
			return err
		}

		// Delete existing postings
		if _, err := tx.Posting.Delete().Where(posting.EntryIDEQ(id)).Exec(c); err != nil {
			return err
		}

		// Insert replacement postings
		for pos, pReq := range req.Postings {
			baseAmt := pReq.Amount.Mul(rate)
			if pReq.BaseAmount != nil && pReq.BaseAmount.IsPositive() {
				baseAmt = *pReq.BaseAmount
			}
			pBuilder := tx.Posting.Create().
				SetEntryID(id).
				SetAccountID(pReq.AccountID).
				SetDirection(posting.Direction(pReq.Direction)).
				SetAmount(pReq.Amount).
				SetBaseAmount(baseAmt).
				SetPosition(pos)

			pDesc := pReq.Description
			if pDesc == "" && req.Description != nil {
				pDesc = *req.Description
			}
			if pDesc != "" {
				pBuilder.SetDescription(pDesc)
			}

			if _, err := pBuilder.Save(c); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return dom, err
	}

	return r.GetByID(c, id)
}

// Delete deletes an entry. Only allowed if status is DRAFT.
// Deletes its postings and soft-deletes the entry in a single entx.WithTx.
func (r EntryRepository) Delete(c context.Context, id int64) error {
	return entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		e, err := forUpdate(tx.Entry.Query(), tx).
			Where(
				entry.ID(id),
				entry.DeletedAtIsNil(),
			).
			Only(c)
		if err != nil {
			if ent.IsNotFound(err) {
				return domain.ErrEntryNotFound
			}
			return err
		}

		if string(e.Status) != domain.EntryStatusDraft {
			return domain.ErrEntryNotDraft
		}

		// Delete postings
		if _, err := tx.Posting.Delete().Where(posting.EntryIDEQ(id)).Exec(c); err != nil {
			return err
		}

		now := time.Now()
		return tx.Entry.UpdateOneID(id).
			SetDeletedAt(now).
			Exec(c)
	})
}

// Search searches active journal entries matching the filter criteria.
func (r EntryRepository) Search(c context.Context, req model.SearchEntryRequest) ([]domain.Entry, string, error) {
	limit := 20
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
		if limit > 100 {
			limit = 100
		}
	}

	q := r.cli.Entry.Query().
		Where(
			entry.DeletedAtIsNil(),
		)

	if req.PeriodID != nil && *req.PeriodID > 0 {
		q = q.Where(entry.PeriodIDEQ(*req.PeriodID))
	}
	if req.Status != nil && *req.Status != "" {
		q = q.Where(entry.StatusEQ(entry.Status(*req.Status)))
	}
	if req.AccountID != nil && *req.AccountID > 0 {
		q = q.Where(entry.HasPostingsWith(posting.AccountIDEQ(*req.AccountID)))
	}
	if req.TransactionID != nil && *req.TransactionID > 0 {
		q = q.Where(entry.TransactionIDEQ(*req.TransactionID))
	}
	if req.DateFrom != nil && !req.DateFrom.IsZero() {
		q = q.Where(entry.EntryDateGTE(req.DateFrom.Time()))
	}
	if req.DateTo != nil && !req.DateTo.IsZero() {
		q = q.Where(entry.EntryDateLTE(req.DateTo.Time()))
	}
	if req.Cursor != nil && *req.Cursor != "" {
		cursorID, err := strconv.ParseInt(*req.Cursor, 10, 64)
		if err != nil || cursorID < 0 {
			return nil, "", base.ErrInvalidCursor
		}
		q = q.Where(entry.IDLT(cursorID))
	}

	items, err := q.
		Order(ent.Desc(entry.FieldID)).
		Limit(limit + 1).
		WithCreator().
		WithUpdater().
		WithDeleter().
		WithPostings(func(pq *ent.PostingQuery) {
			pq.Order(ent.Asc(posting.FieldPosition), ent.Asc(posting.FieldID)).
				WithCreator().
				WithUpdater()
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

	res := make([]domain.Entry, len(items))
	for i, it := range items {
		res[i] = toEntryDomain(it)
	}

	return res, cursor, nil
}

// Post transitions entry status from DRAFT to POSTED, setting posted_at and posted_by.
// If balancedPostings are provided and non-empty, replaces existing postings with them inside the same transaction.
func (r EntryRepository) Post(c context.Context, id int64, postedBy int64, balancedPostings ...[]model.CreatePostingRequest) (domain.Entry, error) {
	var dom domain.Entry

	const maxRetries = 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
			e, err := forUpdate(tx.Entry.Query(), tx).
				Where(
					entry.ID(id),
					entry.DeletedAtIsNil(),
				).
				WithTransaction().
				WithPostings(func(pq *ent.PostingQuery) {
					pq.Order(ent.Asc(posting.FieldPosition))
				}).
				Only(c)
			if err != nil {
				if ent.IsNotFound(err) {
					return domain.ErrEntryNotFound
				}
				return err
			}

			if string(e.Status) != domain.EntryStatusDraft {
				return domain.ErrEntryNotDraft
			}

			if len(balancedPostings) > 0 && len(balancedPostings[0]) > 0 {
				rate := decimal.NewFromInt(1)
				if e.Edges.Transaction != nil && e.Edges.Transaction.ExchangeRate.IsPositive() {
					rate = e.Edges.Transaction.ExchangeRate
				}

				// Delete existing postings
				if _, err := tx.Posting.Delete().Where(posting.EntryIDEQ(id)).Exec(c); err != nil {
					return err
				}

				// Insert replacement postings
				for pos, pReq := range balancedPostings[0] {
					baseAmt := pReq.Amount.Mul(rate)
					if pReq.BaseAmount != nil {
						baseAmt = *pReq.BaseAmount
					}
					pBuilder := tx.Posting.Create().
						SetEntryID(id).
						SetAccountID(pReq.AccountID).
						SetDirection(posting.Direction(pReq.Direction)).
						SetAmount(pReq.Amount).
						SetBaseAmount(baseAmt).
						SetPosition(pos)

					pDesc := pReq.Description
					if pDesc == "" && e.Description != "" {
						pDesc = e.Description
					}
					if pDesc != "" {
						pBuilder.SetDescription(pDesc)
					}

					if _, err := pBuilder.Save(c); err != nil {
						return err
					}
				}
			}

			now := time.Now()
			return tx.Entry.UpdateOneID(id).
				SetStatus(entry.Status(domain.EntryStatusPosted)).
				SetPostedAt(now).
				SetPostedBy(postedBy).
				Exec(c)
		})

		if err == nil {
			return r.GetByID(c, id)
		}

		if ent.IsConstraintError(err) || isLockOrConflictError(err) {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		return dom, err
	}

	return dom, domain.ErrEntryRetryExhausted
}

// Reverse creates a reversing entry with POSTED status in a single entx.WithTx.
// Sets reversal_of_id to id, and swaps posting directions (DEBIT -> CREDIT, CREDIT -> DEBIT).
func (r EntryRepository) Reverse(c context.Context, id int64, postedBy int64, entryDate time.Time, description string) (domain.Entry, error) {
	var dom domain.Entry
	var createdID int64

	const maxRetries = 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
			orig, err := forUpdate(tx.Entry.Query(), tx).
				Where(
					entry.ID(id),
					entry.DeletedAtIsNil(),
				).
				WithPostings(func(pq *ent.PostingQuery) {
					pq.Order(ent.Asc(posting.FieldPosition))
				}).
				Only(c)
			if err != nil {
				if ent.IsNotFound(err) {
					return domain.ErrEntryNotFound
				}
				return err
			}

			if string(orig.Status) != domain.EntryStatusPosted {
				return domain.ErrEntryNotPosted
			}

			// Check if already reversed
			alreadyReversed, err := tx.Entry.Query().
				Where(
					entry.ReversalOfIDEQ(id),
					entry.DeletedAtIsNil(),
				).
				Exist(c)
			if err != nil {
				return err
			}
			if alreadyReversed {
				return domain.ErrEntryAlreadyReversed
			}

			effectiveDate := entryDate
			if effectiveDate.IsZero() {
				effectiveDate = orig.EntryDate
			}

			d := time.Date(effectiveDate.Year(), effectiveDate.Month(), effectiveDate.Day(), 0, 0, 0, 0, time.UTC)
			dEnd := time.Date(effectiveDate.Year(), effectiveDate.Month(), effectiveDate.Day(), 23, 59, 59, 999999999, time.UTC)

			p, err := tx.Period.Query().
				Where(
					period.DeletedAtIsNil(),
					period.StartDateLTE(dEnd),
					period.EndDateGTE(d),
				).
				First(c)
			if err != nil {
				if ent.IsNotFound(err) {
					return domain.ErrPeriodNotFound
				}
				return err
			}

			start := time.Date(p.StartDate.Year(), p.StartDate.Month(), p.StartDate.Day(), 0, 0, 0, 0, time.UTC)
			end := time.Date(p.EndDate.Year(), p.EndDate.Month(), p.EndDate.Day(), 0, 0, 0, 0, time.UTC)
			if d.Before(start) || d.After(end) {
				return domain.ErrPeriodNotFound
			}

			if string(p.Status) == domain.PeriodStatusClosed {
				return domain.ErrPeriodClosed
			}

			year := effectiveDate.Year()
			num, err := GenerateEntryNumber(c, tx, year)
			if err != nil {
				return err
			}

			now := time.Now()
			desc := description
			if desc == "" {
				desc = "Reversal of " + orig.Number
			}

			revBuilder := tx.Entry.Create().
				SetTransactionID(orig.TransactionID).
				SetPeriodID(p.ID).
				SetNumber(num).
				SetEntryDate(effectiveDate).
				SetDescription(desc).
				SetStatus(entry.Status(domain.EntryStatusPosted)).
				SetPostedAt(now).
				SetPostedBy(postedBy).
				SetReversalOfID(orig.ID)

			revEntry, err := revBuilder.Save(c)
			if err != nil {
				return err
			}
			createdID = revEntry.ID

			for pos, p := range orig.Edges.Postings {
				newDirection := posting.DirectionCREDIT
				if p.Direction == posting.DirectionCREDIT {
					newDirection = posting.DirectionDEBIT
				}

				pBuilder := tx.Posting.Create().
					SetEntryID(revEntry.ID).
					SetAccountID(p.AccountID).
					SetDirection(newDirection).
					SetAmount(p.Amount).
					SetBaseAmount(p.BaseAmount).
					SetPosition(pos)

				if p.Description != "" {
					pBuilder.SetDescription("Reversal: " + p.Description)
				}

				if _, err := pBuilder.Save(c); err != nil {
					return err
				}
			}

			return nil
		})

		if err == nil {
			return r.GetByID(c, createdID)
		}

		if ent.IsConstraintError(err) || isLockOrConflictError(err) {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		return dom, err
	}

	return dom, domain.ErrEntryRetryExhausted
}

// Void voids a journal entry.
// If the entry is POSTED, creates a reversing entry and marks the original VOID atomically in the same transaction.
// If the entry is DRAFT, marks it VOID.
// If the entry is already VOID, returns the current entry without changes.
func (r EntryRepository) Void(c context.Context, id int64, postedBy ...int64) (domain.Entry, error) {
	var dom domain.Entry
	var pBy int64
	if len(postedBy) > 0 {
		pBy = postedBy[0]
	}

	const maxRetries = 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
			orig, err := forUpdate(tx.Entry.Query(), tx).
				Where(
					entry.ID(id),
					entry.DeletedAtIsNil(),
				).
				WithPostings(func(pq *ent.PostingQuery) {
					pq.Order(ent.Asc(posting.FieldPosition))
				}).
				Only(c)
			if err != nil {
				if ent.IsNotFound(err) {
					return domain.ErrEntryNotFound
				}
				return err
			}

			// If already VOID, nothing to do
			if string(orig.Status) == domain.EntryStatusVoid {
				return nil
			}

			// If DRAFT, mark as VOID directly
			if string(orig.Status) == domain.EntryStatusDraft {
				return tx.Entry.UpdateOneID(id).
					SetStatus(entry.Status(domain.EntryStatusVoid)).
					Exec(c)
			}

			if string(orig.Status) != domain.EntryStatusPosted {
				return domain.ErrEntryNotPosted
			}

			// Entry is POSTED: check if already reversed
			alreadyReversed, err := tx.Entry.Query().
				Where(
					entry.ReversalOfIDEQ(id),
					entry.DeletedAtIsNil(),
				).
				Exist(c)
			if err != nil {
				return err
			}

			if !alreadyReversed {
				effectiveDate := orig.EntryDate
				d := time.Date(effectiveDate.Year(), effectiveDate.Month(), effectiveDate.Day(), 0, 0, 0, 0, time.UTC)
				dEnd := time.Date(effectiveDate.Year(), effectiveDate.Month(), effectiveDate.Day(), 23, 59, 59, 999999999, time.UTC)

				p, err := tx.Period.Query().
					Where(
						period.DeletedAtIsNil(),
						period.StartDateLTE(dEnd),
						period.EndDateGTE(d),
					).
					First(c)
				if err != nil {
					if ent.IsNotFound(err) {
						return domain.ErrPeriodNotFound
					}
					return err
				}

				start := time.Date(p.StartDate.Year(), p.StartDate.Month(), p.StartDate.Day(), 0, 0, 0, 0, time.UTC)
				end := time.Date(p.EndDate.Year(), p.EndDate.Month(), p.EndDate.Day(), 0, 0, 0, 0, time.UTC)
				if d.Before(start) || d.After(end) {
					return domain.ErrPeriodNotFound
				}

				if string(p.Status) == domain.PeriodStatusClosed {
					return domain.ErrPeriodClosed
				}

				year := effectiveDate.Year()
				num, err := GenerateEntryNumber(c, tx, year)
				if err != nil {
					return err
				}

				now := time.Now()
				desc := "Void reversal of " + orig.Number

				revBuilder := tx.Entry.Create().
					SetTransactionID(orig.TransactionID).
					SetPeriodID(p.ID).
					SetNumber(num).
					SetEntryDate(effectiveDate).
					SetDescription(desc).
					SetStatus(entry.Status(domain.EntryStatusPosted)).
					SetPostedAt(now).
					SetReversalOfID(orig.ID)

				if pBy > 0 {
					revBuilder.SetPostedBy(pBy)
				}

				revEntry, err := revBuilder.Save(c)
				if err != nil {
					return err
				}

				for pos, p := range orig.Edges.Postings {
					newDirection := posting.DirectionCREDIT
					if p.Direction == posting.DirectionCREDIT {
						newDirection = posting.DirectionDEBIT
					}

					pBuilder := tx.Posting.Create().
						SetEntryID(revEntry.ID).
						SetAccountID(p.AccountID).
						SetDirection(newDirection).
						SetAmount(p.Amount).
						SetBaseAmount(p.BaseAmount).
						SetPosition(pos)

					pDesc := p.Description
					if pDesc == "" {
						pDesc = desc
					}
					pBuilder.SetDescription(pDesc)

					if _, err := pBuilder.Save(c); err != nil {
						return err
					}
				}
			}

			// Mark original entry as VOID
			return tx.Entry.UpdateOneID(id).
				SetStatus(entry.Status(domain.EntryStatusVoid)).
				Exec(c)
		})

		if err == nil {
			return r.GetByID(c, id)
		}

		if ent.IsConstraintError(err) || isLockOrConflictError(err) {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		return dom, err
	}

	return dom, domain.ErrEntryRetryExhausted
}

// GetAccountLedger queries postings for the given account from entries with POSTED status, paginated.
// Returns postings, total count, and error.
func (r EntryRepository) GetAccountLedger(c context.Context, accountID int64, page int, limit int) ([]domain.Posting, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	q := r.cli.Posting.Query().
		Where(
			posting.AccountIDEQ(accountID),
			posting.HasEntryWith(
				entry.StatusEQ(entry.Status(domain.EntryStatusPosted)),
				entry.DeletedAtIsNil(),
			),
		)

	total, err := q.Count(c)
	if err != nil {
		return nil, 0, err
	}

	items, err := q.
		Order(ent.Asc(posting.FieldID)).
		Limit(limit).
		Offset(offset).
		WithCreator().
		WithUpdater().
		All(c)
	if err != nil {
		return nil, 0, err
	}

	postings := make([]domain.Posting, len(items))
	for i, item := range items {
		postings[i] = toPostingDomain(item)
	}

	return postings, int64(total), nil
}

// GetAccountBalanceBefore calculates cumulative debit and credit sums (amount and baseAmount) for an account prior to a given posting ID.
func (r EntryRepository) GetAccountBalanceBefore(c context.Context, accountID int64, beforePostingID int64) (decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal, error) {
	if beforePostingID <= 0 {
		return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, nil
	}

	items, err := r.cli.Posting.Query().
		Where(
			posting.AccountIDEQ(accountID),
			posting.IDLT(beforePostingID),
			posting.HasEntryWith(
				entry.StatusEQ(entry.Status(domain.EntryStatusPosted)),
				entry.DeletedAtIsNil(),
			),
		).
		All(c)
	if err != nil {
		return decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero, err
	}

	var debits, credits, baseDebits, baseCredits decimal.Decimal
	for _, it := range items {
		if it.Direction == posting.DirectionDEBIT {
			debits = debits.Add(it.Amount)
			baseDebits = baseDebits.Add(it.BaseAmount)
		} else {
			credits = credits.Add(it.Amount)
			baseCredits = baseCredits.Add(it.BaseAmount)
		}
	}
	return debits, credits, baseDebits, baseCredits, nil
}

func toEntryDomain(e *ent.Entry) domain.Entry {
	dom := domain.Entry{
		ID:            e.ID,
		WorkspaceID:   e.WorkspaceID,
		TransactionID: e.TransactionID,
		PeriodID:      e.PeriodID,
		Number:        e.Number,
		EntryDate:     e.EntryDate,
		Description:   e.Description,
		Status:        string(e.Status),
		PostedAt:      e.PostedAt,
		PostedBy:      e.PostedBy,
		ReversalOfID:  e.ReversalOfID,
		AuditMeta: base.AuditMeta{
			CreatedAt: e.CreatedAt,
			UpdatedAt: e.UpdatedAt,
			DeletedAt: e.DeletedAt,
		},
	}

	if e.Edges.Creator != nil {
		dom.AuditMeta.CreatedBy = base.ByUser{
			Name:   e.Edges.Creator.Name,
			Avatar: e.Edges.Creator.Avatar,
		}
	}
	if e.Edges.Updater != nil {
		dom.AuditMeta.UpdatedBy = base.ByUser{
			Name:   e.Edges.Updater.Name,
			Avatar: e.Edges.Updater.Avatar,
		}
	}
	if e.Edges.Deleter != nil {
		dom.AuditMeta.DeletedBy = &base.ByUser{
			Name:   e.Edges.Deleter.Name,
			Avatar: e.Edges.Deleter.Avatar,
		}
	}

	if len(e.Edges.Postings) > 0 {
		dom.Postings = make([]domain.Posting, len(e.Edges.Postings))
		for i, p := range e.Edges.Postings {
			dom.Postings[i] = toPostingDomain(p)
		}
	}

	return dom
}

// HasPostedEntries checks whether the specified workspace has any journal entries with POSTED status.
func (r EntryRepository) HasPostedEntries(c context.Context, workspaceID int64) (bool, error) {
	return r.cli.Entry.Query().
		Where(
			entry.WorkspaceIDEQ(workspaceID),
			entry.StatusEQ(entry.StatusPOSTED),
			entry.DeletedAtIsNil(),
		).
		Exist(c)
}

func toPostingDomain(p *ent.Posting) domain.Posting {
	dom := domain.Posting{
		ID:          p.ID,
		WorkspaceID: p.WorkspaceID,
		EntryID:     p.EntryID,
		AccountID:   p.AccountID,
		Direction:   string(p.Direction),
		Amount:      p.Amount,
		BaseAmount:  p.BaseAmount,
		Position:    p.Position,
		Description: p.Description,
		AuditMeta: base.AuditMeta{
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		},
	}

	if p.Edges.Creator != nil {
		dom.AuditMeta.CreatedBy = base.ByUser{
			Name:   p.Edges.Creator.Name,
			Avatar: p.Edges.Creator.Avatar,
		}
	}
	if p.Edges.Updater != nil {
		dom.AuditMeta.UpdatedBy = base.ByUser{
			Name:   p.Edges.Updater.Name,
			Avatar: p.Edges.Updater.Avatar,
		}
	}

	return dom
}

// GetAccountBalancesAsOf calculates cumulative debit and credit sums (amount and baseAmount)
// for all accounts up to and including asOfDate for entries with POSTED status.
func (r EntryRepository) GetAccountBalancesAsOf(c context.Context, asOfDate time.Time) (map[int64]AccountBalanceSummary, error) {
	// Truncate to end of the day
	endOfDay := time.Date(asOfDate.Year(), asOfDate.Month(), asOfDate.Day(), 23, 59, 59, 999999999, asOfDate.Location())

	items, err := r.cli.Posting.Query().
		Where(
			posting.HasEntryWith(
				entry.StatusEQ(entry.Status(domain.EntryStatusPosted)),
				entry.EntryDateLTE(endOfDay),
				entry.DeletedAtIsNil(),
			),
		).
		WithEntry(func(eq *ent.EntryQuery) {
			eq.WithTransaction()
		}).
		All(c)
	if err != nil {
		return nil, err
	}

	result := make(map[int64]AccountBalanceSummary)
	for _, it := range items {
		bal := result[it.AccountID]
		if bal.Currencies == nil {
			bal.Currencies = make(map[string]*CurrencyBalanceSummary)
		}

		curr := "IDR"
		if it.Edges.Entry != nil && it.Edges.Entry.Edges.Transaction != nil && it.Edges.Entry.Edges.Transaction.Currency != "" {
			curr = it.Edges.Entry.Edges.Transaction.Currency
		}

		cbal := bal.Currencies[curr]
		if cbal == nil {
			cbal = &CurrencyBalanceSummary{Currency: curr}
			bal.Currencies[curr] = cbal
		}

		if it.Direction == posting.DirectionDEBIT {
			bal.Debit = bal.Debit.Add(it.Amount)
			bal.BaseDebit = bal.BaseDebit.Add(it.BaseAmount)
			cbal.Debit = cbal.Debit.Add(it.Amount)
			cbal.BaseDebit = cbal.BaseDebit.Add(it.BaseAmount)
		} else {
			bal.Credit = bal.Credit.Add(it.Amount)
			bal.BaseCredit = bal.BaseCredit.Add(it.BaseAmount)
			cbal.Credit = cbal.Credit.Add(it.Amount)
			cbal.BaseCredit = cbal.BaseCredit.Add(it.BaseAmount)
		}
		result[it.AccountID] = bal
	}

	return result, nil
}

// GetMonthlyAccountFlows retrieves aggregated debit and credit base amounts per month and account for posted entries between startDate and endDate.
func (r EntryRepository) GetMonthlyAccountFlows(c context.Context, startDate time.Time, endDate time.Time) ([]MonthlyAccountFlow, error) {
	items, err := r.cli.Posting.Query().
		Where(
			posting.HasEntryWith(
				entry.StatusEQ(entry.Status(domain.EntryStatusPosted)),
				entry.EntryDateGTE(startDate),
				entry.EntryDateLTE(endDate),
				entry.DeletedAtIsNil(),
			),
		).
		WithEntry().
		All(c)
	if err != nil {
		return nil, err
	}

	type key struct {
		month     string
		accountID int64
	}
	grouped := make(map[key]*MonthlyAccountFlow)

	for _, it := range items {
		if it.Edges.Entry == nil {
			continue
		}
		m := it.Edges.Entry.EntryDate.Format("2006-01")
		k := key{month: m, accountID: it.AccountID}
		flow, ok := grouped[k]
		if !ok {
			flow = &MonthlyAccountFlow{
				Month:     m,
				AccountID: it.AccountID,
			}
			grouped[k] = flow
		}
		if it.Direction == posting.DirectionDEBIT {
			flow.BaseDebit = flow.BaseDebit.Add(it.BaseAmount)
		} else {
			flow.BaseCredit = flow.BaseCredit.Add(it.BaseAmount)
		}
	}

	result := make([]MonthlyAccountFlow, 0, len(grouped))
	for _, flow := range grouped {
		result = append(result, *flow)
	}
	return result, nil
}

// CreateSystemPostedEntry creates a Transaction with SYSTEM source and an Entry with POSTED status in a single transaction.
func (r EntryRepository) CreateSystemPostedEntry(
	c context.Context,
	periodID int64,
	entryDate time.Time,
	trxDescription string,
	entryDescription string,
	reference string,
	postings []model.CreatePostingRequest,
	postedBy int64,
) (domain.Entry, error) {
	var dom domain.Entry
	var createdEntryID int64

	year := entryDate.Year()
	const maxRetries = 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
			trxNum, err := GenerateTransactionNumber(c, tx, year)
			if err != nil {
				return err
			}

			tBuilder := tx.Transaction.Create().
				SetNumber(trxNum).
				SetTransactionDate(entryDate).
				SetCurrency("IDR").
				SetExchangeRate(decimal.NewFromInt(1)).
				SetSource(transaction.Source(domain.TransactionSourceSystem))

			if trxDescription != "" {
				tBuilder.SetDescription(trxDescription)
			}
			if reference != "" {
				tBuilder.SetReference(reference)
			}

			t, err := tBuilder.Save(c)
			if err != nil {
				return err
			}

			entryNum, err := GenerateEntryNumber(c, tx, year)
			if err != nil {
				return err
			}

			now := time.Now()
			eBuilder := tx.Entry.Create().
				SetTransactionID(t.ID).
				SetPeriodID(periodID).
				SetNumber(entryNum).
				SetEntryDate(entryDate).
				SetStatus(entry.Status(domain.EntryStatusPosted)).
				SetPostedAt(now).
				SetPostedBy(postedBy)

			if entryDescription != "" {
				eBuilder.SetDescription(entryDescription)
			}

			e, err := eBuilder.Save(c)
			if err != nil {
				return err
			}
			createdEntryID = e.ID

			for pos, pReq := range postings {
				baseAmt := pReq.Amount
				if pReq.BaseAmount != nil {
					baseAmt = *pReq.BaseAmount
				}

				pBuilder := tx.Posting.Create().
					SetEntryID(e.ID).
					SetAccountID(pReq.AccountID).
					SetDirection(posting.Direction(pReq.Direction)).
					SetAmount(pReq.Amount).
					SetBaseAmount(baseAmt).
					SetPosition(pos)

				pDesc := pReq.Description
				if pDesc == "" {
					pDesc = entryDescription
				}
				if pDesc != "" {
					pBuilder.SetDescription(pDesc)
				}

				if _, err := pBuilder.Save(c); err != nil {
					return err
				}
			}

			return nil
		})

		if err == nil {
			return r.GetByID(c, createdEntryID)
		}

		if ent.IsConstraintError(err) || isLockOrConflictError(err) {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		return dom, err
	}

	return dom, domain.ErrEntryNotFound
}
