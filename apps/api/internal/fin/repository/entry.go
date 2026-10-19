// Package repository provides database access implementations for finance entities using Ent.
package repository

import (
	"context"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/entry"
	"github.com/kilip/omed/api/ent/posting"
	"github.com/kilip/omed/api/ent/transaction"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/entx"
)

// EntryRepository defines operations for persisting and querying journal entries and postings.
type EntryRepository struct {
	cli *ent.Client
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
			if rate <= 0 {
				rate = 1.0
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
				baseAmt := pReq.Amount * rate
				if pReq.BaseAmount != nil && *pReq.BaseAmount > 0 {
					baseAmt = *pReq.BaseAmount
				}
				pBuilder := tx.Posting.Create().
					SetEntryID(createdEntry.ID).
					SetAccountID(pReq.AccountID).
					SetDirection(posting.Direction(pReq.Direction)).
					SetAmount(pReq.Amount).
					SetBaseAmount(baseAmt).
					SetPosition(pos)

				if pReq.Description != "" {
					pBuilder.SetDescription(pReq.Description)
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
		e, err := tx.Entry.Query().
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

		rate := 1.0
		if e.Edges.Transaction != nil && e.Edges.Transaction.ExchangeRate > 0 {
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
			baseAmt := pReq.Amount * rate
			if pReq.BaseAmount != nil && *pReq.BaseAmount > 0 {
				baseAmt = *pReq.BaseAmount
			}
			pBuilder := tx.Posting.Create().
				SetEntryID(id).
				SetAccountID(pReq.AccountID).
				SetDirection(posting.Direction(pReq.Direction)).
				SetAmount(pReq.Amount).
				SetBaseAmount(baseAmt).
				SetPosition(pos)

			if pReq.Description != "" {
				pBuilder.SetDescription(pReq.Description)
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
		e, err := tx.Entry.Query().
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
	if req.DateFrom != nil && !req.DateFrom.IsZero() {
		q = q.Where(entry.EntryDateGTE(req.DateFrom.Time()))
	}
	if req.DateTo != nil && !req.DateTo.IsZero() {
		q = q.Where(entry.EntryDateLTE(req.DateTo.Time()))
	}

	items, err := q.
		Order(ent.Desc(entry.FieldEntryDate), ent.Desc(entry.FieldID)).
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

	res := make([]domain.Entry, len(items))
	for i, it := range items {
		res[i] = toEntryDomain(it)
	}

	return res, "", nil
}

// Post transitions entry status from DRAFT to POSTED, setting posted_at and posted_by.
func (r EntryRepository) Post(c context.Context, id int64, postedBy int64) (domain.Entry, error) {
	var dom domain.Entry

	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		e, err := tx.Entry.Query().
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

		now := time.Now()
		return tx.Entry.UpdateOneID(id).
			SetStatus(entry.Status(domain.EntryStatusPosted)).
			SetPostedAt(now).
			SetPostedBy(postedBy).
			Exec(c)
	})

	if err != nil {
		return dom, err
	}

	return r.GetByID(c, id)
}

// Reverse creates a reversing entry with POSTED status in a single entx.WithTx.
// Sets reversal_of_id to id, and swaps posting directions (DEBIT -> CREDIT, CREDIT -> DEBIT).
func (r EntryRepository) Reverse(c context.Context, id int64, postedBy int64, entryDate time.Time, description string) (domain.Entry, error) {
	var dom domain.Entry
	var createdID int64

	year := entryDate.Year()
	const maxRetries = 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
			orig, err := tx.Entry.Query().
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
				SetPeriodID(orig.PeriodID).
				SetNumber(num).
				SetEntryDate(entryDate).
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

		if ent.IsConstraintError(err) {
			continue
		}

		return dom, err
	}

	return dom, domain.ErrEntryNotFound
}

// Void marks an entry status as VOID.
func (r EntryRepository) Void(c context.Context, id int64) (domain.Entry, error) {
	var dom domain.Entry

	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		exists, err := tx.Entry.Query().
			Where(
				entry.ID(id),
				entry.DeletedAtIsNil(),
			).
			Exist(c)
		if err != nil {
			return err
		}
		if !exists {
			return domain.ErrEntryNotFound
		}

		return tx.Entry.UpdateOneID(id).
			SetStatus(entry.Status(domain.EntryStatusVoid)).
			Exec(c)
	})

	if err != nil {
		return dom, err
	}

	return r.GetByID(c, id)
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

// GetAccountBalanceBefore calculates cumulative debit and credit sums for an account prior to a given posting ID.
func (r EntryRepository) GetAccountBalanceBefore(c context.Context, accountID int64, beforePostingID int64) (float64, float64, error) {
	if beforePostingID <= 0 {
		return 0, 0, nil
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
		return 0, 0, err
	}

	var debits, credits float64
	for _, it := range items {
		if it.Direction == posting.DirectionDEBIT {
			debits += it.Amount
		} else {
			credits += it.Amount
		}
	}
	return debits, credits, nil
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
