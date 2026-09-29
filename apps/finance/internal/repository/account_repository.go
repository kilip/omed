package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/ent/account"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
)

type AccountRepository struct {
	cl  *ent.Client
	log *slog.Logger
}

func NewAccountRepository(cl *ent.Client, log *slog.Logger) AccountRepository {
	return AccountRepository{
		cl,
		log,
	}
}

func (r AccountRepository) List(ctx context.Context, req model.ListAccountRequest) ([]model.Account, error) {
	q := r.cl.Account.Query()

	if req.Type != "" {
		q.Where(account.TypeEQ(account.Type(req.Type)))
	}

	if req.Status != "" {
		q.Where(account.StatusEQ(account.Status(req.Status)))
	}

	rows := q.AllX(ctx)
	result := make([]model.Account, 0, len(rows))

	for _, row := range rows {
		if row != nil {
			var account model.Account
			shared.ToValue(row, &account)
			result = append(result, account)
		}
	}

	return result, nil
}

func (r AccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	acc, err := r.cl.Account.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(shared.ErrItemNotFound, err)
		}
	}

	var m model.Account
	shared.ToValue(acc, &m)
	return &m, nil
}

func (r AccountRepository) Create(ctx context.Context, req model.CreateAccountRequest) (*model.Account, error) {
	var created *ent.Account

	err := WithTx(ctx, r.cl, func(tx *ent.Tx) error {
		var err error

		created, err = tx.Account.Create().
			SetCode(req.Code).
			SetName(req.Name).
			SetNillableDescription(req.Description).
			SetCurrency(req.Currency).
			SetNillableParentID(req.ParentID).
			SetType(account.Type(req.Type)).
			Save(ctx)

		return err
	})

	if err != nil {
		return nil, err
	}
	var account model.Account
	shared.ToValue(created, &account)
	return &account, nil
}

func (r AccountRepository) Update(ctx context.Context, id uuid.UUID, req model.UpdateAccountRequest) (*model.Account, error) {
	var updated *ent.Account

	err := WithTx(ctx, r.cl, func(tx *ent.Tx) error {
		var err error

		updated, err = tx.Account.UpdateOneID(id).
			SetName(*req.Name).
			SetDescription(*req.Description).
			SetParentID(*req.ParentID).
			SetStatus(account.Status(*req.Status)).
			Save(ctx)

		return err
	})

	if err != nil {
		return nil, err
	}

	var account model.Account
	shared.ToValue(updated, &account)

	return &account, nil
}

func (r AccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.cl.Account.DeleteOneID(id).Exec(ctx)
}
