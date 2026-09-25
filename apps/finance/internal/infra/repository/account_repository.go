package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/ent/account"
	"github.com/kilip/omed/finance/internal/model"
)

type AccountRepository struct {
	client *ent.Client
}

func NewAccountRepository(db *ent.Client) *AccountRepository {
	return &AccountRepository{client: db}
}

func toAccountModel(e *ent.Account) *model.Account {
	return &model.Account{
		ID:            e.ID,
		ParentID:      e.ParentId,
		Code:          e.Code,
		Name:          e.Name,
		Type:          model.AccountType(e.Type),
		DetailType:    model.AccountDetailType(e.DetailType),
		NormalBalance: model.AccountNormalBalance(e.NormalBalance),
		Currency:      e.Currency,
		CreatedBy:     e.CreatedBy,
		CreatedAt:     e.CreatedAt,
		UpdatedBy:     e.UpdatedBy,
		UpdatedAt:     e.UpdatedAt,
	}
}

func (r AccountRepository) Create(ctx context.Context, request *model.Account) (*model.Account, error) {
	q := r.client.Account.Create().
		SetCode(request.Code).
		SetName(request.Name).
		SetType(account.Type(request.Type)).
		SetDetailType(account.DetailType(request.DetailType)).
		SetNormalBalance(account.NormalBalance(request.NormalBalance)).
		SetCurrency(request.Currency)

	if request.ParentID != nil {
		q.SetParentId(*request.ParentID)
	}

	created, err := q.Save(ctx)
	if err != nil {
		return nil, err
	}

	return toAccountModel(created), nil
}

func (r AccountRepository) Update(ctx context.Context, request *model.Account) (*model.Account, error) {
	return nil, errors.New("Not implemented")
}

func (r AccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return errors.New("Not Implemented")
}

func (r AccountRepository) List(ctx context.Context, filter model.AccountFilter) ([]*model.Account, error) {
	return nil, errors.New("Not Implemented")
}

func (r AccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Account, error) {
	return nil, errors.New("not implemented")
}
