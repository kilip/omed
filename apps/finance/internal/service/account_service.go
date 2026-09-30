package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
)

type AccountRepository interface {
	List(ctx context.Context, req model.ListAccountRequest) ([]model.Account, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Account, error)
	Create(ctx context.Context, req model.CreateAccountRequest) (*model.Account, error)
	Update(ctx context.Context, id uuid.UUID, req model.UpdateAccountRequest) (*model.Account, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int, error)
	Seed(ctx context.Context, req model.SeedAccountRequest) ([]model.Account, error)
}

type AccountEntryRepository interface {
	Count(ctx context.Context) (int, error)
}

type AccountService struct {
	accounts AccountRepository
	entries  AccountEntryRepository
	log      *slog.Logger
}

func NewAccountService(accounts AccountRepository, entries AccountEntryRepository, log *slog.Logger) AccountService {
	return AccountService{
		accounts: accounts,
		entries:  entries,
		log:      log,
	}
}

func (s AccountService) List(ctx context.Context, req model.ListAccountRequest) ([]model.Account, error) {
	return s.accounts.List(ctx, req)
}

func (s AccountService) GetByID(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	return s.accounts.GetByID(ctx, id)
}
func (s AccountService) Create(ctx context.Context, req model.CreateAccountRequest) (*model.Account, error) {
	return s.accounts.Create(ctx, req)
}

func (s AccountService) Update(ctx context.Context, id uuid.UUID, req model.UpdateAccountRequest) (*model.Account, error) {
	return s.accounts.Update(ctx, id, req)
}

func (s AccountService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.accounts.Delete(ctx, id)
}

func (s AccountService) Seed(ctx context.Context, req model.SeedAccountRequest) ([]model.Account, error) {
	// 1. Business Rule: Tidak bisa seed jika terdapat entri jurnal
	entryCount, err := s.entries.Count(ctx)
	if err != nil {
		return nil, err
	}
	if entryCount > 0 {
		return nil, shared.ErrEntriesNotEmpty
	}

	// 2. Business Rule: Tidak bisa seed jika akun sudah ada, kecuali force=true
	accCount, err := s.accounts.Count(ctx)
	if err != nil {
		return nil, err
	}
	if accCount > 0 && !req.Force {
		return nil, shared.ErrAccountsNotEmpty
	}

	return s.accounts.Seed(ctx, req)
}
