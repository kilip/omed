package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilip/omed/internal/domain/user"
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

func (r UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error){
	user := r.q.User

	return user.WithContext(ctx).Where(user.ID.Eq(id)).First()
}

func (r UserRepository) FindByEmail(ctx context.Context, email string)  (*user.User, error){
	user := r.q.User
	return user.WithContext(ctx).Where(user.Email.Eq(email)).First()
}

func (r UserRepository) Create(ctx context.Context, user *user.User) error {
	return r.q.Transaction(func(tx *dal.Query) error {
		q := tx.User
		if err := q.WithContext(ctx).Create(user); err != nil {
			return err
		}

		return nil
	})
}

func (r UserRepository) Update(ctx context.Context, user *user.User) error {
	return r.q.Transaction(func(tx *dal.Query) error {
		q := tx.User
		if err := q.WithContext(ctx).Save(user); err != nil {
			return err
		}
		return nil
	})
}

func (r UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.Transaction(func(tx *dal.Query) error {
		q := tx.User

		if _, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Delete(); err != nil {
			return err
		}

		return nil
	})
}
