package service

import (
	"context"
	"log/slog"

	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/model"
)

type AccountService struct {
	db         *ent.Client
	logger     *slog.Logger
	repository model.AccountRepository
}

func NewAccountService(db *ent.Client, logger *slog.Logger, repository model.AccountRepository) AccountService {
	return AccountService{
		db,
		logger,
		repository,
	}
}

func (s AccountService) Create(ctx context.Context, payload model.Account) (*model.Account, error) {
	created, err := s.repository.Create(ctx, &payload)
	if err != nil {
		return nil, err
	}

	return created, nil
}
