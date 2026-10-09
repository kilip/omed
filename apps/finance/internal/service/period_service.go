package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/model"
)

type PeriodRepository interface {
	List(ctx context.Context, req model.ListPeriodRequest) ([]model.Period, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Period, error)
	Create(ctx context.Context, req model.CreatePeriodRequest) (*model.Period, error)
	Update(ctx context.Context, id uuid.UUID, req model.UpdatePeriodRequest) (*model.Period, error)
	Delete(ctx context.Context, id uuid.UUID) error
	FindOverlapping(ctx context.Context, startDate, endDate time.Time, excludeID *uuid.UUID) ([]*ent.Period, error)
	CountEntries(ctx context.Context, periodID uuid.UUID) (int, error)
}

type PeriodService struct {
	repo PeriodRepository
	log  *slog.Logger
}

func NewPeriodService(repo PeriodRepository, log *slog.Logger) PeriodService {
	return PeriodService{
		repo: repo,
		log:  log,
	}
}

func (s PeriodService) List(ctx context.Context, req model.ListPeriodRequest) ([]model.Period, error) {
	return s.repo.List(ctx, req)
}

func (s PeriodService) GetByID(ctx context.Context, id uuid.UUID) (*model.Period, error) {
	return s.repo.GetByID(ctx, id)
}

func (s PeriodService) Create(ctx context.Context, req model.CreatePeriodRequest) (*model.Period, error) {
	if req.StartDate.After(req.EndDate) {
		return nil, core.ErrInvalidDateRange
	}

	overlaps, err := s.repo.FindOverlapping(ctx, req.StartDate, req.EndDate, nil)
	if err != nil {
		return nil, err
	}
	if len(overlaps) > 0 {
		return nil, core.ErrPeriodOverlap
	}

	return s.repo.Create(ctx, req)
}

func (s PeriodService) Update(ctx context.Context, id uuid.UUID, req model.UpdatePeriodRequest) (*model.Period, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Status transitions validation
	switch existing.Status {
	case model.PeriodStatusLocked:
		return nil, core.ErrInvalidStatusTransition
	case model.PeriodStatusOpen:
		if req.Status != model.PeriodStatusOpen && req.Status != model.PeriodStatusClosed {
			return nil, core.ErrInvalidStatusTransition
		}
	case model.PeriodStatusClosed:
		if req.Status != model.PeriodStatusClosed && req.Status != model.PeriodStatusOpen && req.Status != model.PeriodStatusLocked {
			return nil, core.ErrInvalidStatusTransition
		}
	default:
		return nil, core.ErrInvalidStatusTransition
	}

	return s.repo.Update(ctx, id, req)
}

func (s PeriodService) Delete(ctx context.Context, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if existing.Status == model.PeriodStatusClosed {
		return core.ErrPeriodClosed
	}
	if existing.Status == model.PeriodStatusLocked {
		return core.ErrPeriodLocked
	}
	if existing.Status != model.PeriodStatusOpen {
		return core.ErrPeriodClosed
	}

	entryCount, err := s.repo.CountEntries(ctx, id)
	if err != nil {
		return err
	}
	if entryCount > 0 {
		return core.ErrEntriesNotEmpty
	}

	return s.repo.Delete(ctx, id)
}

