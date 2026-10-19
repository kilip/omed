// Package service provides business logic and orchestration for finance entities.
package service

import (
	"context"
	"time"

	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
)

// PeriodRepository defines the persistence operations required by PeriodService.
type PeriodRepository interface {
	GetByID(c context.Context, id int64) (domain.Period, error)
	Search(c context.Context, req model.SearchPeriodRequest) ([]domain.Period, string, error)
	Create(c context.Context, req model.CreatePeriodRequest) (domain.Period, error)
	Update(c context.Context, id int64, req model.UpdatePeriodRequest) (domain.Period, error)
	UpdateStatus(c context.Context, id int64, status string) (domain.Period, error)
	Delete(c context.Context, id int64) error
	HasOverlapping(c context.Context, startDate, endDate time.Time, excludeID int64) (bool, error)
	HasDraftEntries(c context.Context, periodID int64) (bool, error)
}

// PeriodService manages business logic and validation for Accounting Periods.
type PeriodService struct {
	repo PeriodRepository
}

// NewPeriodService constructs a new PeriodService backed by the given repository.
func NewPeriodService(repo PeriodRepository) PeriodService {
	return PeriodService{repo: repo}
}

// GetByID retrieves an active period by its ID within a workspace.
func (s PeriodService) GetByID(c context.Context, id int64) (domain.Period, error) {
	return s.repo.GetByID(c, id)
}

// Search searches active periods within a workspace based on filter criteria.
func (s PeriodService) Search(c context.Context, req model.SearchPeriodRequest) ([]domain.Period, string, error) {
	return s.repo.Search(c, req)
}

// Create validates and creates a new accounting period.
func (s PeriodService) Create(c context.Context, req model.CreatePeriodRequest) (domain.Period, error) {
	if req.FiscalYear <= 0 {
		return domain.Period{}, domain.ErrPeriodInvalidFiscalYear
	}
	if req.StartDate.Time().After(req.EndDate.Time()) {
		return domain.Period{}, domain.ErrPeriodInvalidDates
	}

	overlap, err := s.repo.HasOverlapping(c, req.StartDate.Time(), req.EndDate.Time(), 0)
	if err != nil {
		return domain.Period{}, err
	}
	if overlap {
		return domain.Period{}, domain.ErrPeriodOverlap
	}

	return s.repo.Create(c, req)
}

// Update validates and updates an existing accounting period.
func (s PeriodService) Update(c context.Context, id int64, req model.UpdatePeriodRequest) (domain.Period, error) {
	curr, err := s.repo.GetByID(c, id)
	if err != nil {
		return domain.Period{}, err
	}
	if curr.Status == domain.PeriodStatusClosed {
		return domain.Period{}, domain.ErrPeriodClosed
	}

	if req.FiscalYear <= 0 {
		return domain.Period{}, domain.ErrPeriodInvalidFiscalYear
	}
	if req.StartDate.Time().After(req.EndDate.Time()) {
		return domain.Period{}, domain.ErrPeriodInvalidDates
	}

	overlap, err := s.repo.HasOverlapping(c, req.StartDate.Time(), req.EndDate.Time(), id)
	if err != nil {
		return domain.Period{}, err
	}
	if overlap {
		return domain.Period{}, domain.ErrPeriodOverlap
	}

	return s.repo.Update(c, id, req)
}

// Delete validates and soft deletes an existing accounting period.
func (s PeriodService) Delete(c context.Context, id int64) error {
	curr, err := s.repo.GetByID(c, id)
	if err != nil {
		return err
	}
	if curr.Status == domain.PeriodStatusClosed {
		return domain.ErrPeriodClosed
	}

	return s.repo.Delete(c, id)
}

// Close locks an open accounting period.
func (s PeriodService) Close(c context.Context, id int64) (domain.Period, error) {
	curr, err := s.repo.GetByID(c, id)
	if err != nil {
		return domain.Period{}, err
	}
	if curr.Status == domain.PeriodStatusClosed {
		return domain.Period{}, domain.ErrPeriodAlreadyClosed
	}

	hasDraft, err := s.repo.HasDraftEntries(c, id)
	if err != nil {
		return domain.Period{}, err
	}
	if hasDraft {
		return domain.Period{}, domain.ErrPeriodHasDraftEntries
	}

	return s.repo.UpdateStatus(c, id, domain.PeriodStatusClosed)
}

// Reopen unlocks a closed accounting period.
func (s PeriodService) Reopen(c context.Context, id int64) (domain.Period, error) {
	curr, err := s.repo.GetByID(c, id)
	if err != nil {
		return domain.Period{}, err
	}
	if curr.Status == domain.PeriodStatusOpen {
		return domain.Period{}, domain.ErrPeriodAlreadyOpen
	}

	return s.repo.UpdateStatus(c, id, domain.PeriodStatusOpen)
}
