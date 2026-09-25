package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/model"
)

type AccountRepository interface {
	Create(ctx context.Context, r model.AccountRequest) (*model.AccountResponse, error)
	Update(ctx context.Context, id uuid.UUID, r model.AccountRequest) (*model.AccountResponse, error)
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter model.AccountFilter) ([]*model.AccountResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.AccountResponse, error)
}

type AccountService struct {
	db         *ent.Client
	logger     *slog.Logger
	repository AccountRepository
}

func NewAccountService(db *ent.Client, logger *slog.Logger, repository AccountRepository) AccountService {
	return AccountService{
		db,
		logger,
		repository,
	}
}

func (s AccountService) Create(ctx context.Context, request model.AccountRequest) (*model.AccountResponse, error) {
	created, err := s.repository.Create(ctx, request)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s AccountService) Update(ctx context.Context, id uuid.UUID, request model.AccountRequest) (*model.AccountResponse, error) {
	updated, err := s.repository.Update(ctx, id, request)
	return updated, err
}

func (s AccountService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repository.Delete(ctx, id)
}

func (s AccountService) GetByID(ctx context.Context, id uuid.UUID) (*model.AccountResponse, error) {
	return s.repository.GetByID(ctx, id)
}

func (s AccountService) List(ctx context.Context, filter model.AccountFilter) ([]*model.AccountResponse, error) {
	accounts, err := s.repository.List(ctx, filter)
	return accounts, err
}
