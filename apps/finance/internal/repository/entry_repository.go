package repository

import (
	"context"
	"log/slog"

	"github.com/kilip/omed/finance/ent"
)

type EntryRepository struct {
	cl     *ent.Client
	logger *slog.Logger
}

func NewEntryRepository(cl *ent.Client, logger *slog.Logger) EntryRepository {
	return EntryRepository{cl, logger}
}

func (r EntryRepository) Count(ctx context.Context) (int, error) {
	return r.cl.Entry.Query().Count(ctx)
}
