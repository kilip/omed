package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/ent/account"
	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/model"
	seed_coa "github.com/kilip/omed/finance/seed/coa"
)

type AccountRepository struct {
	cl  *ent.Client
	log *slog.Logger
}

func toAccount(e *ent.Account) *model.AccountResponse {
	if e == nil {
		return nil
	}
	return &model.AccountResponse{
		ID:          e.ID,
		WorkspaceID: e.WorkspaceID,
		Code:        e.Code,
		Name:        e.Name,
		Description: e.Description,
		Type:        model.AccountType(e.Type),
		Currency:    e.Currency,
		Status:      model.AccountStatus(e.Status),
		ParentID:    e.ParentId,
		CreatedBy:   e.CreatedBy,
		CreatedAt:   e.CreatedAt,
		UpdatedBy:   e.UpdatedBy,
		UpdatedAt:   e.UpdatedAt,
	}
}

func NewAccountRepository(cl *ent.Client, log *slog.Logger) AccountRepository {
	return AccountRepository{
		cl,
		log,
	}
}

func (r AccountRepository) List(ctx context.Context, req model.ListAccountRequest) ([]model.AccountResponse, error) {
	q := r.cl.Account.Query()

	if req.Type != "" {
		q.Where(account.TypeEQ(account.Type(req.Type)))
	}

	if req.Status != "" {
		q.Where(account.StatusEQ(account.Status(req.Status)))
	}

	rows, err := q.All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]model.AccountResponse, 0, len(rows))

	for _, row := range rows {
		if row != nil {
			result = append(result, *toAccount(row))
		}
	}

	return result, nil
}

func (r AccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.AccountResponse, error) {
	acc, err := r.cl.Account.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(core.ErrItemNotFound, err)
		}
		return nil, err
	}

	return toAccount(acc), nil
}

func (r AccountRepository) Create(ctx context.Context, req model.CreateAccountRequest) (*model.AccountResponse, error) {
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
	return toAccount(created), nil
}

func (r AccountRepository) Update(ctx context.Context, id uuid.UUID, req model.UpdateAccountRequest) (*model.AccountResponse, error) {
	var updated *ent.Account

	err := WithTx(ctx, r.cl, func(tx *ent.Tx) error {
		var err error

		u := tx.Account.UpdateOneID(id).
			SetNillableName(req.Name).
			SetNillableDescription(req.Description).
			SetNillableParentID(req.ParentID)

		if req.Status != nil {
			u.SetStatus(account.Status(*req.Status))
		}

		updated, err = u.Save(ctx)
		return err
	})

	if err != nil {
		if ent.IsNotFound(err) {
			return nil, errors.Join(core.ErrItemNotFound, err)
		}
		return nil, err
	}

	return toAccount(updated), nil
}

func (r AccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.cl.Account.DeleteOneID(id).Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return errors.Join(core.ErrItemNotFound, err)
		}
		return err
	}
	return nil
}

func (r AccountRepository) Count(ctx context.Context) (int, error) {
	return r.cl.Account.Query().Count(ctx)
}

func (r AccountRepository) Seed(ctx context.Context, req model.SeedAccountRequest) ([]model.AccountResponse, error) {
	seedAccounts, err := seed_coa.Load(req.Profile, req.Lang)
	if err != nil {
		return nil, err
	}

	currency := req.Currency
	if currency == "" {
		currency = "IDR"
	}

	createdAccounts := make([]model.AccountResponse, 0, len(seedAccounts))
	codeToIDMap := make(map[string]uuid.UUID)

	err = WithTx(ctx, r.cl, func(tx *ent.Tx) error {
		if req.Force {
			if _, err := tx.Account.Delete().Exec(ctx); err != nil {
				return err
			}
		}

		for _, sa := range seedAccounts {
			builder := tx.Account.Create().
				SetCode(sa.Code).
				SetName(sa.Name).
				SetType(account.Type(sa.Type)).
				SetCurrency(currency)

			if sa.ParentCode != nil && *sa.ParentCode != "" {
				if parentID, exists := codeToIDMap[*sa.ParentCode]; exists {
					builder.SetParentID(parentID)
				}
			}

			acc, err := builder.Save(ctx)
			if err != nil {
				return err
			}

			codeToIDMap[acc.Code] = acc.ID
			createdAccounts = append(createdAccounts, *toAccount(acc))
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdAccounts, nil
}
