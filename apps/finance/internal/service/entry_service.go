package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
)

type EntryRepository interface {
	List(ctx context.Context, req model.ListEntryRequest) ([]model.Entry, string, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Entry, error)
	Create(ctx context.Context, periodID uuid.UUID, req model.CreateEntryRequest) (*model.Entry, error)
}

type EntryLedgerPeriodRepository interface {
	FindActiveByDate(ctx context.Context, date time.Time) (*model.LedgerPeriod, error)
}

type EntryAccountRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.Account, error)
}

type EntryService struct {
	repo    EntryRepository
	lpRepo  EntryLedgerPeriodRepository
	accRepo EntryAccountRepository
	log     *slog.Logger
}

func NewEntryService(repo EntryRepository, lpRepo EntryLedgerPeriodRepository, accRepo EntryAccountRepository, log *slog.Logger) EntryService {
	return EntryService{
		repo:    repo,
		lpRepo:  lpRepo,
		accRepo: accRepo,
		log:     log,
	}
}

func (s EntryService) List(ctx context.Context, req model.ListEntryRequest) ([]model.Entry, string, error) {
	return s.repo.List(ctx, req)
}

func (s EntryService) GetByID(ctx context.Context, id uuid.UUID) (*model.Entry, error) {
	return s.repo.GetByID(ctx, id)
}

func (s EntryService) Create(ctx context.Context, req model.CreateEntryRequest) (*model.Entry, error) {
	// 1. Validasi minimal 2 posting line per entry
	if len(req.Postings) < 2 {
		return nil, shared.ErrMinPostingsCount
	}

	// 2. Aturan keseimbangan: sum(debit) == sum(credit)
	sumDebit := decimal.Zero
	sumCredit := decimal.Zero

	for _, p := range req.Postings {
		if p.DebitAmount.IsNegative() || p.CreditAmount.IsNegative() {
			return nil, shared.ErrUnbalancedEntry
		}
		sumDebit = sumDebit.Add(p.DebitAmount)
		sumCredit = sumCredit.Add(p.CreditAmount)
	}

	if !sumDebit.Equal(sumCredit) {
		return nil, shared.ErrUnbalancedEntry
	}

	// 3. Validasi tanggal jurnal berada dalam LedgerPeriod yang berstatus open
	period, err := s.lpRepo.FindActiveByDate(ctx, req.EntryDate)
	if err != nil {
		return nil, shared.ErrPeriodClosed
	}
	if period.Status != model.LedgerPeriodStatusOpen {
		return nil, shared.ErrPeriodClosed
	}

	// 4. Validasi akun tujuan aktif (status == active), menolak posting ke akun archived
	for _, p := range req.Postings {
		acc, err := s.accRepo.GetByID(ctx, p.AccountID)
		if err != nil {
			return nil, err
		}
		if acc.Status != model.AccountStatusActive {
			return nil, shared.ErrInactiveAccount
		}
	}

	return s.repo.Create(ctx, period.ID, req)
}
