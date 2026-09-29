package repository

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/ent/ledgerperiod"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
)

type LedgerPeriodRepository struct {
	cl  *ent.Client
	log *slog.Logger
}

func toLedgerPeriod(e *ent.LedgerPeriod) *model.LedgerPeriod {
	if e == nil {
		return nil
	}
	return &model.LedgerPeriod{
		ID:            e.ID,
		WorkspaceID:   e.WorkspaceID,
		StartDate:     e.StartDate,
		EndDate:       e.EndDate,
		Status:        model.LedgerPeriodStatus(e.Status),
		CreatedBy:     e.CreatedBy,
		CreatedByName: e.CreatedByName,
		CreatedAt:     e.CreatedAt,
		UpdatedBy:     e.UpdatedBy,
		UpdatedByName: e.UpdatedByName,
		UpdatedAt:     e.UpdatedAt,
	}
}

func NewLedgerPeriodRepository(cl *ent.Client, log *slog.Logger) LedgerPeriodRepository {
	return LedgerPeriodRepository{
		cl:  cl,
		log: log,
	}
}

func (r LedgerPeriodRepository) List(ctx context.Context, req model.ListLedgerPeriodRequest) ([]model.LedgerPeriod, error) {
	q := r.cl.LedgerPeriod.Query()

	if req.Status != "" {
		q.Where(ledgerperiod.StatusEQ(ledgerperiod.Status(req.Status)))
	}

	if req.Date != "" {
		for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02"} {
			if t, err := time.Parse(layout, req.Date); err == nil {
				q.Where(ledgerperiod.StartDateLTE(t), ledgerperiod.EndDateGTE(t))
				break
			}
		}
	}

	q.Order(ledgerperiod.ByStartDate(sql.OrderAsc()))

	rows, err := q.All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]model.LedgerPeriod, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			result = append(result, *toLedgerPeriod(row))
		}
	}

	return result, nil
}

func (r LedgerPeriodRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.LedgerPeriod, error) {
	lp, err := r.cl.LedgerPeriod.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(shared.ErrItemNotFound, err)
		}
		return nil, err
	}
	return toLedgerPeriod(lp), nil
}

func (r LedgerPeriodRepository) Create(ctx context.Context, req model.CreateLedgerPeriodRequest) (*model.LedgerPeriod, error) {
	var created *ent.LedgerPeriod

	status := ledgerperiod.StatusOpen
	if req.Status != "" {
		status = ledgerperiod.Status(req.Status)
	}

	err := WithTx(ctx, r.cl, func(tx *ent.Tx) error {
		var err error
		created, err = tx.LedgerPeriod.Create().
			SetStartDate(req.StartDate).
			SetEndDate(req.EndDate).
			SetStatus(status).
			Save(ctx)
		return err
	})

	if err != nil {
		return nil, err
	}
	return toLedgerPeriod(created), nil
}

func (r LedgerPeriodRepository) Update(ctx context.Context, id uuid.UUID, req model.UpdateLedgerPeriodRequest) (*model.LedgerPeriod, error) {
	var updated *ent.LedgerPeriod

	err := WithTx(ctx, r.cl, func(tx *ent.Tx) error {
		var err error
		updated, err = tx.LedgerPeriod.UpdateOneID(id).
			SetStatus(ledgerperiod.Status(req.Status)).
			Save(ctx)
		return err
	})

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(shared.ErrItemNotFound, err)
		}
		return nil, err
	}

	return toLedgerPeriod(updated), nil
}

func (r LedgerPeriodRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.cl.LedgerPeriod.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return errors.Join(shared.ErrItemNotFound, err)
		}
		return err
	}
	return nil
}

func (r LedgerPeriodRepository) FindActiveByDate(ctx context.Context, date time.Time) (*model.LedgerPeriod, error) {
	lp, err := r.cl.LedgerPeriod.Query().
		Where(
			ledgerperiod.StatusEQ(ledgerperiod.StatusOpen),
			ledgerperiod.StartDateLTE(date),
			ledgerperiod.EndDateGTE(date),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(shared.ErrItemNotFound, err)
		}
		return nil, err
	}
	return toLedgerPeriod(lp), nil
}

func (r LedgerPeriodRepository) HasOverlap(ctx context.Context, startDate, endDate time.Time, excludeID *uuid.UUID) (bool, error) {
	q := r.cl.LedgerPeriod.Query().
		Where(
			ledgerperiod.StartDateLTE(endDate),
			ledgerperiod.EndDateGTE(startDate),
		)
	if excludeID != nil {
		q.Where(ledgerperiod.IDNEQ(*excludeID))
	}
	return q.Exist(ctx)
}
