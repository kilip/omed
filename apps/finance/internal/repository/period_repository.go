package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/ent/period"
	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/model"
)

type PeriodRepository struct {
	cl  *ent.Client
	log *slog.Logger
}

func toPeriod(e *ent.Period) *model.Period {
	if e == nil {
		return nil
	}
	res := &model.Period{
		ID:          e.ID,
		WorkspaceID: e.WorkspaceID,
		StartDate:   e.StartDate,
		EndDate:     e.EndDate,
		Status:      model.PeriodStatus(e.Status),
		CreatedBy:   e.CreatedBy,
		CreatedAt:   e.CreatedAt,
		UpdatedBy:   e.UpdatedBy,
		UpdatedAt:   e.UpdatedAt,
	}
	if e.Edges.Creator != nil {
		res.CreatedByName = e.Edges.Creator.Name
	}
	if e.Edges.Updater != nil {
		res.UpdatedByName = e.Edges.Updater.Name
	}
	return res
}

func NewPeriodRepository(cl *ent.Client, log *slog.Logger) PeriodRepository {
	return PeriodRepository{
		cl:  cl,
		log: log,
	}
}

func parseDateQuery(dateStr string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", dateStr); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("invalid date format: %s", dateStr)
}

func (r PeriodRepository) List(ctx context.Context, req model.ListPeriodRequest) ([]model.Period, error) {
	q := r.cl.Period.Query().
		WithCreator().
		WithUpdater()

	if req.Status != "" {
		q.Where(period.StatusEQ(period.Status(req.Status)))
	}

	if req.Date != "" {
		targetDate, err := parseDateQuery(req.Date)
		if err != nil {
			return nil, err
		}
		q.Where(
			period.StartDateLTE(targetDate),
			period.EndDateGTE(targetDate),
		)
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]model.Period, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			result = append(result, *toPeriod(row))
		}
	}

	return result, nil
}

func (r PeriodRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Period, error) {
	row, err := r.cl.Period.Query().
		Where(period.IDEQ(id)).
		WithCreator().
		WithUpdater().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(core.ErrItemNotFound, err)
		}
		return nil, err
	}

	return toPeriod(row), nil
}

func (r PeriodRepository) Create(ctx context.Context, req model.CreatePeriodRequest) (*model.Period, error) {
	var created *ent.Period

	status := req.Status
	if status == "" {
		status = model.PeriodStatusOpen
	}

	err := WithTx(ctx, r.cl, func(tx *ent.Tx) error {
		var err error
		created, err = tx.Period.Create().
			SetStartDate(req.StartDate).
			SetEndDate(req.EndDate).
			SetStatus(period.Status(status)).
			Save(ctx)
		return err
	})

	if err != nil {
		return nil, err
	}

	// Reload with creator/updater
	return r.GetByID(ctx, created.ID)
}

func (r PeriodRepository) Update(ctx context.Context, id uuid.UUID, req model.UpdatePeriodRequest) (*model.Period, error) {
	err := WithTx(ctx, r.cl, func(tx *ent.Tx) error {
		return tx.Period.UpdateOneID(id).
			SetStatus(period.Status(req.Status)).
			Exec(ctx)
	})

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(core.ErrItemNotFound, err)
		}
		return nil, err
	}

	return r.GetByID(ctx, id)
}

func (r PeriodRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.cl.Period.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return errors.Join(core.ErrItemNotFound, err)
		}
		return err
	}
	return nil
}

func (r PeriodRepository) FindOverlapping(ctx context.Context, startDate, endDate time.Time, excludeID *uuid.UUID) ([]*ent.Period, error) {
	q := r.cl.Period.Query().
		Where(
			period.StartDateLTE(endDate),
			period.EndDateGTE(startDate),
		)

	if excludeID != nil {
		q.Where(period.IDNEQ(*excludeID))
	}

	return q.All(ctx)
}

func (r PeriodRepository) CountEntries(ctx context.Context, periodID uuid.UUID) (int, error) {
	return r.cl.Period.Query().
		Where(period.IDEQ(periodID)).
		QueryEntries().
		Count(ctx)
}

