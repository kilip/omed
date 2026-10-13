// Package repository provides database access implementations for user entities using Ent.
package repository

import (
	"context"
	"strconv"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/predicate"
	"github.com/kilip/omed/api/ent/user"
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/platform/entx"
	"github.com/kilip/omed/api/internal/user/domain"
	"github.com/kilip/omed/api/internal/user/model"
)

// UserRepository handles user persistence operations against PostgreSQL via Ent ORM.
type UserRepository struct {
	core.ServiceRepository[int64, domain.User, model.CreateUserRequest, model.UpdateUserRequest, model.SearchRequest]
	cli *ent.Client
}

// NewUserRepository constructs a new UserRepository instance backed by the given Ent client.
func NewUserRepository(cli *ent.Client) UserRepository {
	return UserRepository{
		cli: cli,
	}
}

// GetByID retrieves a user domain entity by ID. Returns core.ErrNotFound if the user does not exist.
func (r UserRepository) GetByID(c context.Context, id int64) (domain.User, error) {
	var usr domain.User

	user, err := r.cli.User.Get(c, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return usr, core.ErrNotFound
		}
		return usr, err
	}

	return toDomain(user), nil
}

// Search searches users matching the request filter with cursor pagination.
func (r UserRepository) Search(c context.Context, req model.SearchRequest) ([]domain.User, string, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	var predicates []predicate.User
	if req.Name != "" {
		predicates = append(predicates, user.NameContainsFold(req.Name))
	}

	if req.Cursor != "" {
		cursorID, err := strconv.ParseInt(req.Cursor, 10, 64)
		if err != nil || cursorID < 0 {
			return nil, "", core.ErrInvalidCursor
		}
		predicates = append(predicates, user.IDLT(cursorID))
	}

	users, err := r.cli.User.Query().
		Where(predicates...).
		Order(ent.Desc(user.FieldID)).
		Limit(limit + 1).
		All(c)
	if err != nil {
		return nil, "", err
	}

	var cursor string
	if len(users) > limit {
		cursor = strconv.FormatInt(users[limit-1].ID, 10)
		users = users[:limit]
	}

	res := make([]domain.User, len(users))
	for i, u := range users {
		res[i] = toDomain(u)
	}

	return res, cursor, nil
}

// Create persists a new user in a database transaction.
func (r UserRepository) Create(c context.Context, req model.CreateUserRequest) (domain.User, error) {
	var user *ent.User
	var dom domain.User

	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		var err error
		user, err = tx.User.Create().
			SetName(req.Name).
			SetEmail(req.Email).
			SetNillableAvatar(&req.Avatar).
			Save(c)

		return err
	})

	if err != nil {
		return dom, err
	}

	return toDomain(user), nil
}

// Update modifies an existing user by ID within a transaction.
func (r UserRepository) Update(c context.Context, id int64, req model.UpdateUserRequest) (domain.User, error) {
	var dom domain.User
	var usr *ent.User
	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		var err error
		usr, err = tx.User.UpdateOneID(id).
			SetName(req.Name).
			SetNillableAvatar(&req.Avatar).
			Save(c)

		return err
	})

	if err != nil {
		return dom, err
	}

	return toDomain(usr), nil
}

// Delete removes a user by ID within a transaction.
func (r UserRepository) Delete(c context.Context, id int64) error {
	return entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		return tx.User.DeleteOneID(id).Exec(c)
	})
}

func toDomain(u *ent.User) domain.User {
	return domain.User{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Avatar:    u.Avatar,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
