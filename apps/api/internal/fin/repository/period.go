// Package repository provides database access implementations for finance entities using Ent.
package repository

import (
	"context"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/entry"
	"github.com/kilip/omed/api/ent/period"
	"github.com/kilip/omed/api/ent/transaction"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/entx"
)

// PeriodRepository defines operations for persisting and querying accounting period entities.
type PeriodRepository struct {
	cli *ent.Client
}

// NewPeriodRepository constructs a new PeriodRepository instance backed by the given Ent client.
func NewPeriodRepository(cli *ent.Client) PeriodRepository {
	return PeriodRepository{cli: cli}
}

// GetByID retrieves an active period by its ID within a workspace.
func (r PeriodRepository) GetByID(c context.Context, id int64) (domain.Period, error) {
	var dom domain.Period

	p, err := r.cli.Period.Query().
		Where(
			period.ID(id),
			period.DeletedAtIsNil(),
		).
		WithCreator().
		WithUpdater().
		WithDeleter().
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom, base.ErrNotFound
		}
		return dom, err
	}

	return toPeriodDomain(p), nil
}

// Search returns all active periods in the workspace matching the filter criteria.
func (r PeriodRepository) Search(c context.Context, req model.SearchPeriodRequest) ([]domain.Period, string, error) {
	q := r.cli.Period.Query().
		Where(
			period.DeletedAtIsNil(),
		)

	if req.FiscalYear != nil {
		q = q.Where(period.FiscalYearEQ(*req.FiscalYear))
	}
	if req.Status != nil && *req.Status != "" {
		q = q.Where(period.StatusEQ(period.Status(*req.Status)))
	}

	items, err := q.
		Order(ent.Asc(period.FieldStartDate), ent.Asc(period.FieldID)).
		WithCreator().
		WithUpdater().
		WithDeleter().
		All(c)
	if err != nil {
		return nil, "", err
	}

	res := make([]domain.Period, len(items))
	for i, it := range items {
		res[i] = toPeriodDomain(it)
	}

	return res, "", nil
}

// Create inserts a new accounting period within a database transaction.
func (r PeriodRepository) Create(c context.Context, req model.CreatePeriodRequest) (domain.Period, error) {
	var dom domain.Period

	var createdID int64
	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		builder := tx.Period.Create().
			SetName(req.Name).
			SetStartDate(req.StartDate.Time()).
			SetEndDate(req.EndDate.Time()).
			SetFiscalYear(req.FiscalYear).
			SetStatus(period.Status(domain.PeriodStatusOpen))

		if req.Description != "" {
			builder.SetDescription(req.Description)
		}

		res, err := builder.Save(c)
		if err != nil {
			return err
		}
		createdID = res.ID
		return nil
	})

	if err != nil {
		return dom, err
	}

	return r.GetByID(c, createdID)
}

// Update modifies an existing period within a database transaction.
func (r PeriodRepository) Update(c context.Context, id int64, req model.UpdatePeriodRequest) (domain.Period, error) {
	var dom domain.Period

	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		exists, err := tx.Period.Query().
			Where(
				period.ID(id),
				period.DeletedAtIsNil(),
			).
			Exist(c)
		if err != nil {
			return err
		}
		if !exists {
			return base.ErrNotFound
		}

		builder := tx.Period.UpdateOneID(id).
			SetName(req.Name).
			SetStartDate(req.StartDate.Time()).
			SetEndDate(req.EndDate.Time()).
			SetFiscalYear(req.FiscalYear)

		if req.Description != "" {
			builder.SetDescription(req.Description)
		} else {
			builder.ClearDescription()
		}

		return builder.Exec(c)
	})

	if err != nil {
		return dom, err
	}

	return r.GetByID(c, id)
}

// UpdateStatus changes the status of a period.
func (r PeriodRepository) UpdateStatus(c context.Context, id int64, status string) (domain.Period, error) {
	var dom domain.Period

	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		exists, err := tx.Period.Query().
			Where(
				period.ID(id),
				period.DeletedAtIsNil(),
			).
			Exist(c)
		if err != nil {
			return err
		}
		if !exists {
			return base.ErrNotFound
		}

		return tx.Period.UpdateOneID(id).
			SetStatus(period.Status(status)).
			Exec(c)
	})

	if err != nil {
		return dom, err
	}

	return r.GetByID(c, id)
}

// Delete marks an existing period as soft-deleted.
func (r PeriodRepository) Delete(c context.Context, id int64) error {
	return entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		exists, err := tx.Period.Query().
			Where(
				period.ID(id),
				period.DeletedAtIsNil(),
			).
			Exist(c)
		if err != nil {
			return err
		}
		if !exists {
			return base.ErrNotFound
		}

		now := time.Now()
		return tx.Period.UpdateOneID(id).
			SetDeletedAt(now).
			Exec(c)
	})
}

// HasOverlapping checks if any active period in the workspace overlaps with [startDate, endDate], excluding excludeID.
func (r PeriodRepository) HasOverlapping(c context.Context, startDate, endDate time.Time, excludeID int64) (bool, error) {
	q := r.cli.Period.Query().
		Where(
			period.DeletedAtIsNil(),
			period.StartDateLTE(endDate),
			period.EndDateGTE(startDate),
		)
	if excludeID > 0 {
		q = q.Where(period.IDNEQ(excludeID))
	}
	return q.Exist(c)
}

// HasDraftEntries checks if there are any active draft entries in the specified period.
func (r PeriodRepository) HasDraftEntries(c context.Context, periodID int64) (bool, error) {
	return r.cli.Entry.Query().
		Where(
			entry.PeriodIDEQ(periodID),
			entry.StatusEQ(entry.Status(domain.EntryStatusDraft)),
			entry.DeletedAtIsNil(),
		).
		Exist(c)
}

// HasRevaluationEntry checks if there is already an active posted revaluation entry for the period.
func (r PeriodRepository) HasRevaluationEntry(c context.Context, periodID int64) (bool, error) {
	return r.cli.Entry.Query().
		Where(
			entry.PeriodIDEQ(periodID),
			entry.DeletedAtIsNil(),
			entry.HasTransactionWith(
				transaction.ReferenceEQ(domain.PeriodRevaluationReference(periodID)),
				transaction.DeletedAtIsNil(),
			),
		).
		Exist(c)
}


func toPeriodDomain(p *ent.Period) domain.Period {
	dom := domain.Period{
		ID:          p.ID,
		WorkspaceID: p.WorkspaceID,
		Name:        p.Name,
		StartDate:   p.StartDate,
		EndDate:     p.EndDate,
		FiscalYear:  p.FiscalYear,
		Status:      string(p.Status),
		Description: p.Description,
		AuditMeta: base.AuditMeta{
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
			DeletedAt: p.DeletedAt,
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
	if p.Edges.Deleter != nil {
		dom.AuditMeta.DeletedBy = &base.ByUser{
			Name:   p.Edges.Deleter.Name,
			Avatar: p.Edges.Deleter.Avatar,
		}
	}

	return dom
}
