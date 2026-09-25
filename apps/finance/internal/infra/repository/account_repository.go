package repository

import (
	"context"

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

func toAccountResponse(e *ent.Account) *model.AccountResponse {
	return &model.AccountResponse{
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

func (r AccountRepository) Create(ctx context.Context, request model.AccountRequest) (*model.AccountResponse, error) {
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

	return toAccountResponse(created), nil
}

func (r AccountRepository) Update(ctx context.Context, id uuid.UUID, request model.AccountRequest) (*model.AccountResponse, error) {
	updated, err := r.client.Account.UpdateOneID(id).
		SetCode(request.Code).
		SetName(request.Name).
		SetCurrency(request.Currency).
		SetDetailType(account.DetailType(request.DetailType)).
		SetIsActive(request.Active).
		SetIsPlaceholder(request.Placeholder).
		SetIsSystem(request.System).
		Save(ctx)

	return toAccountResponse(updated), err
}

func (r AccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.client.Account.DeleteOneID(id).Exec(ctx)
}

func (r AccountRepository) List(ctx context.Context, filter model.AccountFilter) ([]*model.AccountResponse, error) {
	q := r.client.Account.Query()

	if filter.ParentID != nil {
		q = q.Where(account.ParentIdEQ(*filter.ParentID))
	}
	if filter.Type != nil {
		q = q.Where(account.TypeEQ(account.Type(*filter.Type)))
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*model.AccountResponse, 0, len(rows))
	for _, e := range rows {
		result = append(result, toAccountResponse(e))
	}

	return result, nil
}

func (r AccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.AccountResponse, error) {
	acc, err := r.client.Account.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, err
		}
		return nil, err
	}

	return toAccountResponse(acc), nil
}
