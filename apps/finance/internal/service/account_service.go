package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/model"
	seed_coa "github.com/kilip/omed/finance/seed/coa"
)

type AccountRepository interface {
	List(ctx context.Context, req model.ListAccountRequest) ([]model.AccountResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.AccountResponse, error)
	Create(ctx context.Context, req model.CreateAccountRequest) (*model.AccountResponse, error)
	Update(ctx context.Context, id uuid.UUID, req model.UpdateAccountRequest) (*model.AccountResponse, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context) (int, error)
	Seed(ctx context.Context, req model.SeedAccountRequest) ([]model.AccountResponse, error)
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

func (s AccountService) List(ctx context.Context, req model.ListAccountRequest) ([]model.AccountResponse, error) {
	return s.accounts.List(ctx, req)
}

func (s AccountService) GetByID(ctx context.Context, id uuid.UUID) (*model.AccountResponse, error) {
	return s.accounts.GetByID(ctx, id)
}
func (s AccountService) Create(ctx context.Context, req model.CreateAccountRequest) (*model.AccountResponse, error) {
	return s.accounts.Create(ctx, req)
}

func (s AccountService) Update(ctx context.Context, id uuid.UUID, req model.UpdateAccountRequest) (*model.AccountResponse, error) {
	return s.accounts.Update(ctx, id, req)
}

func (s AccountService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.accounts.Delete(ctx, id)
}

func (s AccountService) Seed(ctx context.Context, req model.SeedAccountRequest) ([]model.AccountResponse, error) {
	// 1. Business Rule: Tidak bisa seed jika terdapat entri jurnal
	entryCount, err := s.entries.Count(ctx)
	if err != nil {
		return nil, err
	}
	if entryCount > 0 {
		return nil, core.ErrEntriesNotEmpty
	}

	// 2. Business Rule: Tidak bisa seed jika akun sudah ada, kecuali force=true
	accCount, err := s.accounts.Count(ctx)
	if err != nil {
		return nil, err
	}
	if accCount > 0 && !req.Force {
		return nil, core.ErrAccountsNotEmpty
	}

	return s.accounts.Seed(ctx, req)
}

func (s AccountService) GetSeedTemplates(ctx context.Context) ([]model.SeedTemplateResponse, error) {
	templates := seed_coa.ListTemplates()
	res := make([]model.SeedTemplateResponse, len(templates))
	for i, t := range templates {
		res[i] = model.SeedTemplateResponse{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Languages:   t.Languages,
		}
	}
	return res, nil
}

func (s AccountService) GetSeedPreview(ctx context.Context, profile, lang string) (*model.SeedPreviewResponse, error) {
	accounts, err := seed_coa.Load(profile, lang)
	if err != nil {
		return nil, err
	}

	accCount, err := s.accounts.Count(ctx)
	if err != nil {
		return nil, err
	}

	entryCount, err := s.entries.Count(ctx)
	if err != nil {
		return nil, err
	}

	previewAccounts := make([]model.SeedPreviewAccount, len(accounts))
	for i, a := range accounts {
		previewAccounts[i] = model.SeedPreviewAccount{
			Code:       a.Code,
			Name:       a.Name,
			Type:       a.Type,
			ParentCode: a.ParentCode,
		}
	}

	return &model.SeedPreviewResponse{
		Profile:      profile,
		Lang:         lang,
		AccountCount: accCount,
		EntryCount:   entryCount,
		Accounts:     previewAccounts,
	}, nil
}
