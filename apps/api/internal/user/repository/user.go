package repository

import (
	"context"
	"strconv"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/predicate"
	"github.com/kilip/omed/api/ent/user"
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/platform/entx"
	"github.com/kilip/omed/api/internal/user/model"
)

type UserRepository struct {
	core.ServiceRepository[int64, model.UserResponse, model.CreateUserRequest, model.UpdateUserRequest, model.SearchRequest]
	cli *ent.Client
}

func NewUserRepository(cli *ent.Client) UserRepository {
	return UserRepository{
		cli: cli,
	}
}

func (r UserRepository) GetByID(c context.Context, id int64) (model.UserResponse, error) {
	var res model.UserResponse

	user, err := r.cli.User.Get(c, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return res, core.ErrNotFound
		}
		return res, err
	}

	return toUserResponse(user), nil
}

func (r UserRepository) Search(c context.Context, req model.SearchRequest) ([]model.UserResponse, string, error) {
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

	res := make([]model.UserResponse, len(users))
	for i, u := range users {
		res[i] = toUserResponse(u)
	}

	return res, cursor, nil
}

func (r UserRepository) Create(c context.Context, req model.CreateUserRequest) (model.UserResponse, error) {
	var user *ent.User
	var res model.UserResponse

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
		return res, err
	}

	return toUserResponse(user), nil
}

func (r UserRepository) Update(c context.Context, id int64, req model.UpdateUserRequest) (model.UserResponse, error) {
	var res model.UserResponse
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
		return res, err
	}

	return toUserResponse(usr), nil
}

func (r UserRepository) Delete(c context.Context, id int64) error {
	return entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		return tx.User.DeleteOneID(id).Exec(c)
	})
}

func toUserResponse(user *ent.User) model.UserResponse {
	return model.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Avatar:    user.Avatar,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
