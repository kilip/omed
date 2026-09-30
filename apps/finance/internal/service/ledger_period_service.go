package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
)

type LedgerPeriodRepository interface {
	List(ctx context.Context, req model.ListLedgerPeriodRequest) ([]model.LedgerPeriod, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.LedgerPeriod, error)
	Create(ctx context.Context, req model.CreateLedgerPeriodRequest) (*model.LedgerPeriod, error)
	Update(ctx context.Context, id uuid.UUID, req model.UpdateLedgerPeriodRequest) (*model.LedgerPeriod, error)
	Delete(ctx context.Context, id uuid.UUID) error
	FindActiveByDate(ctx context.Context, date time.Time) (*model.LedgerPeriod, error)
	HasOverlap(ctx context.Context, startDate, endDate time.Time, excludeID *uuid.UUID) (bool, error)
}

type LedgerPeriodService struct {
	repo LedgerPeriodRepository
	log  *slog.Logger
}

func NewLedgerPeriodService(repo LedgerPeriodRepository, log *slog.Logger) LedgerPeriodService {
	return LedgerPeriodService{
		repo: repo,
		log:  log,
	}
}

func (s LedgerPeriodService) List(ctx context.Context, req model.ListLedgerPeriodRequest) ([]model.LedgerPeriod, error) {
	return s.repo.List(ctx, req)
}

func (s LedgerPeriodService) GetByID(ctx context.Context, id uuid.UUID) (*model.LedgerPeriod, error) {
	return s.repo.GetByID(ctx, id)
}

func (s LedgerPeriodService) Create(ctx context.Context, req model.CreateLedgerPeriodRequest) (*model.LedgerPeriod, error) {
	// 1. Validasi urutan tanggal (end_date > start_date)
	if !req.EndDate.After(req.StartDate) {
		return nil, shared.ErrInvalidDateRange
	}

	// 2. Pencegahan tumpang tindih tanggal antar periode
	hasOverlap, err := s.repo.HasOverlap(ctx, req.StartDate, req.EndDate, nil)
	if err != nil {
		return nil, err
	}
	if hasOverlap {
		return nil, shared.ErrPeriodOverlap
	}

	// 3. Status awal harus open (default jika kosong)
	if req.Status == "" {
		req.Status = model.LedgerPeriodStatusOpen
	}
	if req.Status != model.LedgerPeriodStatusOpen {
		return nil, shared.ErrInvalidStatusTransition
	}

	return s.repo.Create(ctx, req)
}

func (s LedgerPeriodService) Update(ctx context.Context, id uuid.UUID, req model.UpdateLedgerPeriodRequest) (*model.LedgerPeriod, error) {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validasi alur transisi status: open -> closed -> locked
	if current.Status != req.Status {
		isValidTransition := (current.Status == model.LedgerPeriodStatusOpen && req.Status == model.LedgerPeriodStatusClosed) ||
			(current.Status == model.LedgerPeriodStatusClosed && req.Status == model.LedgerPeriodStatusLocked)

		if !isValidTransition {
			return nil, shared.ErrInvalidStatusTransition
		}
	}

	return s.repo.Update(ctx, id, req)
}

func (s LedgerPeriodService) Delete(ctx context.Context, id uuid.UUID) error {
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if current.Status == model.LedgerPeriodStatusLocked {
		return shared.ErrInvalidStatusTransition
	}

	return s.repo.Delete(ctx, id)
}

func (s LedgerPeriodService) FindActiveByDate(ctx context.Context, date time.Time) (*model.LedgerPeriod, error) {
	return s.repo.FindActiveByDate(ctx, date)
}
