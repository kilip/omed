// Package repository provides database access implementations for finance entities using Ent.
package repository

import (
	"context"
	"strconv"
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

// TransactionRepository defines operations for persisting and querying financial transactions.
type TransactionRepository struct {
	cli *ent.Client
}

// NewTransactionRepository constructs a new TransactionRepository backed by the given Ent client.
func NewTransactionRepository(cli *ent.Client) TransactionRepository {
	return TransactionRepository{cli: cli}
}

// GetByID retrieves an active transaction by its ID within a workspace, including its entries and postings.
func (r TransactionRepository) GetByID(c context.Context, id int64) (domain.Transaction, error) {
	var dom domain.Transaction

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
				WithCreator().
				WithUpdater().
				WithDeleter().
				WithPostings(func(pq *ent.PostingQuery) {
					pq.Order(ent.Asc(posting.FieldPosition), ent.Asc(posting.FieldID)).
						WithCreator().
						WithUpdater()
				})
		}).
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom, domain.ErrTransactionNotFound
		}
		return dom, err
	}

	return toTransactionDomain(t), nil
}

// Create inserts a new transaction with automatic numbering inside a database transaction.
// If req has inline Entry & Postings, creates entry & postings in the same entx.WithTx.
// Retries with a new sequence number if a unique constraint conflict occurs on (workspace_id, number).
func (r TransactionRepository) Create(c context.Context, req model.CreateTransactionRequest) (domain.Transaction, error) {
	var dom domain.Transaction
	var createdID int64

	year := req.TransactionDate.Time().Year()
	const maxRetries = 5

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
			num, err := GenerateTransactionNumber(c, tx, year)
			if err != nil {
				return err
			}

			curr := "IDR"
			if req.Currency != "" {
				curr = req.Currency
			}

			rate := 1.0
			if req.ExchangeRate != nil && *req.ExchangeRate > 0 {
				rate = *req.ExchangeRate
			}

			tBuilder := tx.Transaction.Create().
				SetNumber(num).
				SetTransactionDate(req.TransactionDate.Time()).
				SetCurrency(curr).
				SetExchangeRate(rate).
				SetSource(transaction.Source(domain.TransactionSourceManual))

			if req.Description != "" {
				tBuilder.SetDescription(req.Description)
			}
			if req.Reference != "" {
				tBuilder.SetReference(req.Reference)
			}

			t, err := tBuilder.Save(c)
			if err != nil {
				return err
			}
			createdID = t.ID

			if req.Entry != nil {
				entryYear := req.Entry.EntryDate.Time().Year()
				entryNum, err := GenerateEntryNumber(c, tx, entryYear)
				if err != nil {
					return err
				}

				eBuilder := tx.Entry.Create().
					SetTransactionID(t.ID).
					SetPeriodID(req.Entry.PeriodID).
					SetNumber(entryNum).
					SetEntryDate(req.Entry.EntryDate.Time()).
					SetStatus(entry.Status(domain.EntryStatusDraft))

				if req.Entry.Description != "" {
					eBuilder.SetDescription(req.Entry.Description)
				}

				createdEntry, err := eBuilder.Save(c)
				if err != nil {
					return err
				}

				for pos, pReq := range req.Entry.Postings {
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

	return dom, domain.ErrTransactionNumberTaken
}

// Update updates the transaction's header fields (description, reference).
// If any entry has status POSTED, rejects with domain.ErrTransactionHasPostedEntries.
func (r TransactionRepository) Update(c context.Context, id int64, req model.UpdateTransactionRequest) (domain.Transaction, error) {
	var dom domain.Transaction

	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		t, err := tx.Transaction.Query().
			Where(
				transaction.ID(id),
				transaction.DeletedAtIsNil(),
			).
			Only(c)
		if err != nil {
			if ent.IsNotFound(err) {
				return domain.ErrTransactionNotFound
			}
			return err
		}

		hasPosted, err := tx.Entry.Query().
			Where(
				entry.TransactionIDEQ(t.ID),
				entry.StatusEQ(entry.Status(domain.EntryStatusPosted)),
				entry.DeletedAtIsNil(),
			).
			Exist(c)
		if err != nil {
			return err
		}
		if hasPosted {
			return domain.ErrTransactionHasPostedEntries
		}

		u := tx.Transaction.UpdateOneID(id)
		if req.Description != nil {
			u.SetDescription(*req.Description)
		}
		if req.Reference != nil {
			u.SetReference(*req.Reference)
		}

		return u.Exec(c)
	})

	if err != nil {
		return dom, err
	}

	return r.GetByID(c, id)
}

// Delete marks an existing transaction as soft-deleted.
// If any entry has status POSTED, rejects with domain.ErrTransactionHasPostedEntries.
func (r TransactionRepository) Delete(c context.Context, id int64) error {
	return entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		t, err := tx.Transaction.Query().
			Where(
				transaction.ID(id),
				transaction.DeletedAtIsNil(),
			).
			Only(c)
		if err != nil {
			if ent.IsNotFound(err) {
				return domain.ErrTransactionNotFound
			}
			return err
		}

		hasPosted, err := tx.Entry.Query().
			Where(
				entry.TransactionIDEQ(t.ID),
				entry.StatusEQ(entry.Status(domain.EntryStatusPosted)),
				entry.DeletedAtIsNil(),
			).
			Exist(c)
		if err != nil {
			return err
		}
		if hasPosted {
			return domain.ErrTransactionHasPostedEntries
		}

		now := time.Now()
		return tx.Transaction.UpdateOneID(id).
			SetDeletedAt(now).
			Exec(c)
	})
}

// Search searches active transactions matching the filter criteria.
func (r TransactionRepository) Search(c context.Context, req model.SearchTransactionRequest) ([]domain.Transaction, string, error) {
	limit := 20
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
		if limit > 100 {
			limit = 100
		}
	}

	q := r.cli.Transaction.Query().
		Where(
			transaction.DeletedAtIsNil(),
		)

	if req.DateFrom != nil && !req.DateFrom.IsZero() {
		q = q.Where(transaction.TransactionDateGTE(req.DateFrom.Time()))
	}
	if req.DateTo != nil && !req.DateTo.IsZero() {
		q = q.Where(transaction.TransactionDateLTE(req.DateTo.Time()))
	}
	if req.Reference != nil && *req.Reference != "" {
		q = q.Where(transaction.ReferenceEQ(*req.Reference))
	}
	if req.Source != nil && *req.Source != "" {
		q = q.Where(transaction.SourceEQ(transaction.Source(*req.Source)))
	}
	if req.Query != nil && *req.Query != "" {
		q = q.Where(transaction.Or(
			transaction.NumberContainsFold(*req.Query),
			transaction.DescriptionContainsFold(*req.Query),
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
			eq.Where(entry.DeletedAtIsNil()).
				Order(ent.Asc(entry.FieldID)).
				WithCreator().
				WithUpdater().
				WithDeleter().
				WithPostings(func(pq *ent.PostingQuery) {
					pq.Order(ent.Asc(posting.FieldPosition), ent.Asc(posting.FieldID)).
						WithCreator().
						WithUpdater()
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

	res := make([]domain.Transaction, len(items))
	for i, it := range items {
		res[i] = toTransactionDomain(it)
	}

	return res, cursor, nil
}

func toTransactionDomain(t *ent.Transaction) domain.Transaction {
	dom := domain.Transaction{
		ID:              t.ID,
		WorkspaceID:     t.WorkspaceID,
		Number:          t.Number,
		TransactionDate: t.TransactionDate,
		Description:     t.Description,
		Reference:       t.Reference,
		Currency:        t.Currency,
		ExchangeRate:    t.ExchangeRate,
		Source:          string(t.Source),
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

	if len(t.Edges.Entries) > 0 {
		dom.Entries = make([]domain.Entry, len(t.Edges.Entries))
		for i, e := range t.Edges.Entries {
			dom.Entries[i] = toEntryDomain(e)
		}
	}

	return dom
}

