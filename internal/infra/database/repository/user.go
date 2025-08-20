package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilip/omed/internal/entity"
	"github.com/kilip/omed/internal/infra/database/dal"
)

type UserRepository struct {
	q *dal.Query
}

func NewUserRepository(dal *dal.Query) *UserRepository {
	return &UserRepository{
		q: dal,
	}
}

func (r UserRepository) Create(ctx context.Context, user *entity.User) error {
	return r.q.Transaction(func(tx *dal.Query) error {
		q := tx.User
		if err := q.WithContext(ctx).Create(user); err != nil {
			return err
		}

		return nil
	})
}

func (r UserRepository) Update(ctx context.Context, user *entity.User) error {
	return r.q.Transaction(func(tx *dal.Query) error {
		q := tx.User
		if err := q.WithContext(ctx).Save(user); err != nil {
			return err
		}
		return nil
	})
}

func (r UserRepository) Delete(ctx context.Context, id string) error {
	return r.q.Transaction(func(tx *dal.Query) error {
		q := tx.User
		id, err := uuid.Parse(id)
		if err != nil {
			return err
		}

		if _, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Delete(); err != nil {
			return err
		}

		return nil
	})
}
