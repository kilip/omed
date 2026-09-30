package repository

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/ent/entry"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
)

type EntryRepository struct {
	cl  *ent.Client
	log *slog.Logger
}

func NewEntryRepository(cl *ent.Client, log *slog.Logger) EntryRepository {
	return EntryRepository{
		cl:  cl,
		log: log,
	}
}

func toPosting(p *ent.Posting) *model.Posting {
	if p == nil {
		return nil
	}
	res := &model.Posting{
		ID:               p.ID,
		WorkspaceID:      p.WorkspaceID,
		EntryID:          p.EntryId,
		AccountID:        p.AccountId,
		Currency:         p.Currency,
		DebitAmount:      p.DebitAmount,
		CreditAmount:     p.CreditAmount,
		BaseCurrency:     p.BaseCurrency,
		BaseDebitAmount:  p.BaseDebitAmount,
		BaseCreditAmount: p.BaseCreditAmount,
		ExchangeRate:     p.ExchangeRate,
		Memo:             p.Memo,
	}
	if p.Edges.Account != nil {
		res.AccountCode = p.Edges.Account.Code
		res.AccountName = p.Edges.Account.Name
	}
	return res
}

func toEntry(e *ent.Entry) *model.Entry {
	if e == nil {
		return nil
	}
	res := &model.Entry{
		ID:             e.ID,
		WorkspaceID:    e.WorkspaceID,
		EntryDate:      e.EntryDate,
		EntryType:      model.EntryType(e.EntryType),
		Description:    e.Description,
		Reference:      e.Reference,
		LedgerPeriodID: e.LedgerPeriodID,
		CreatedBy:      e.CreatedBy,
		CreatedByName:  e.CreatedByName,
		CreatedAt:      e.CreatedAt,
		UpdatedBy:      e.UpdatedBy,
		UpdatedByName:  e.UpdatedByName,
		UpdatedAt:      e.UpdatedAt,
	}
	if e.Edges.Postings != nil {
		postings := make([]model.Posting, 0, len(e.Edges.Postings))
		for _, p := range e.Edges.Postings {
			if p != nil {
				postings = append(postings, *toPosting(p))
			}
		}
		res.Postings = postings
	}
	return res
}

func (r EntryRepository) List(ctx context.Context, req model.ListEntryRequest) ([]model.Entry, string, error) {
	q := r.cl.Entry.Query()

	if req.EntryType != "" {
		q.Where(entry.EntryTypeEQ(entry.EntryType(req.EntryType)))
	}

	if req.StartDate != "" {
		for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02"} {
			if t, err := time.Parse(layout, req.StartDate); err == nil {
				q.Where(entry.EntryDateGTE(t))
				break
			}
		}
	}

	if req.EndDate != "" {
		for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02"} {
			if t, err := time.Parse(layout, req.EndDate); err == nil {
				q.Where(entry.EntryDateLTE(t))
				break
			}
		}
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset, err := decodeCursor(req.Cursor)
	if err != nil {
		return nil, "", err
	}

	q.WithPostings(func(pq *ent.PostingQuery) {
		pq.WithAccount()
	})
	q.Order(entry.ByEntryDate(sql.OrderDesc()), entry.ByCreatedAt(sql.OrderDesc()))
	q.Offset(offset).Limit(limit + 1)

	rows, err := q.All(ctx)
	if err != nil {
		return nil, "", err
	}

	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	result := make([]model.Entry, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			result = append(result, *toEntry(row))
		}
	}

	var nextCursor string
	if hasMore {
		nextCursor = encodeCursor(offset + limit)
	}

	return result, nextCursor, nil
}

func (r EntryRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Entry, error) {
	e, err := r.cl.Entry.Query().
		Where(entry.IDEQ(id)).
		WithPostings(func(q *ent.PostingQuery) {
			q.WithAccount()
		}).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(shared.ErrItemNotFound, err)
		}
		return nil, err
	}
	return toEntry(e), nil
}

func (r EntryRepository) Create(ctx context.Context, periodID uuid.UUID, req model.CreateEntryRequest) (*model.Entry, error) {
	var createdEntry *ent.Entry

	entryType := entry.EntryTypeNormal
	if req.EntryType != "" {
		entryType = entry.EntryType(req.EntryType)
	}

	err := WithTx(ctx, r.cl, func(tx *ent.Tx) error {
		var err error
		eBuilder := tx.Entry.Create().
			SetEntryDate(req.EntryDate).
			SetEntryType(entryType).
			SetDescription(req.Description).
			SetNillableReference(req.Reference).
			SetLedgerPeriodID(periodID)

		createdEntry, err = eBuilder.Save(ctx)
		if err != nil {
			return err
		}

		for _, pReq := range req.Postings {
			pBuilder := tx.Posting.Create().
				SetEntryId(createdEntry.ID).
				SetAccountId(pReq.AccountID).
				SetCurrency(pReq.Currency).
				SetDebitAmount(pReq.DebitAmount).
				SetCreditAmount(pReq.CreditAmount).
				SetBaseCurrency(pReq.Currency).
				SetBaseDebitAmount(pReq.DebitAmount).
				SetBaseCreditAmount(pReq.CreditAmount).
				SetExchangeRate(decimal.NewFromInt(1)).
				SetNillableMemo(pReq.Memo)

			_, err = pBuilder.Save(ctx)
			if err != nil {
				return err
			}
		}

		createdEntry, err = tx.Entry.Query().
			Where(entry.IDEQ(createdEntry.ID)).
			WithPostings(func(q *ent.PostingQuery) {
				q.WithAccount()
			}).
			Only(ctx)
		return err
	})

	if err != nil {
		return nil, err
	}
	return toEntry(createdEntry), nil
}

func (r EntryRepository) Count(ctx context.Context) (int, error) {
	return r.cl.Entry.Query().Count(ctx)
}
