// Package repository provides database access implementations for user entities using Ent.
package repository

import (
	"context"
	"strconv"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/file"
	"github.com/kilip/omed/api/ent/predicate"
	"github.com/kilip/omed/api/ent/user"
	"github.com/kilip/omed/api/ent/workspace"
	"github.com/kilip/omed/api/ent/workspacemember"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/entx"
	"github.com/kilip/omed/api/internal/platform/s3x"
)

// UserRepository handles user persistence operations against PostgreSQL via Ent ORM.
type UserRepository struct {
	base.ServiceRepository[int64, domain.User, model.CreateUserRequest, model.UpdateUserRequest, model.SearchRequest]
	cli     *ent.Client
	storage s3x.Storage
}

// NewUserRepository constructs a new UserRepository instance backed by the given Ent client and optional S3 storage.
func NewUserRepository(cli *ent.Client, storage ...s3x.Storage) UserRepository {
	var st s3x.Storage
	if len(storage) > 0 {
		st = storage[0]
	}
	return UserRepository{
		cli:     cli,
		storage: st,
	}
}

// GetByID retrieves a user domain entity by ID. Returns base.ErrNotFound if the user does not exist.
func (r UserRepository) GetByID(c context.Context, id int64) (domain.User, error) {
	var usr domain.User

	user, err := r.cli.User.Get(c, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return usr, base.ErrNotFound
		}
		return usr, err
	}

	return toDomain(user), nil
}

// FindByEmail retrieves a user domain entity by email. Returns base.ErrNotFound if not found.
func (r UserRepository) FindByEmail(c context.Context, email string) (domain.User, error) {
	var usr domain.User
	u, err := r.cli.User.Query().Where(user.EmailEQ(email)).Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return usr, base.ErrNotFound
		}
		return usr, err
	}
	return toDomain(u), nil
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
			return nil, "", base.ErrInvalidCursor
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

// Create persists a new user and creates their personal workspace in a database transaction.
func (r UserRepository) Create(c context.Context, req model.CreateUserRequest) (domain.User, error) {
	var usr *ent.User
	var personalWS *ent.Workspace
	var dom domain.User

	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		var err error
		locale := req.Locale
		if locale == "" {
			locale = "en"
		}
		currency := req.Currency
		if currency == "" {
			if locale == "id" {
				currency = "IDR"
			} else {
				currency = "USD"
			}
		}

		uCreate := tx.User.Create().
			SetName(req.Name).
			SetEmail(req.Email).
			SetLocale(locale).
			SetCurrency(currency).
			SetNillableAvatar(&req.Avatar).
			SetEmailVerified(req.EmailVerified)

		usr, err = uCreate.Save(c)
		if err != nil {
			if ent.IsConstraintError(err) {
				return domain.ErrEmailTaken
			}
			return err
		}

		// Auto-create personal workspace
		wsName := req.Name + "'s Workspace"
		wsCtx := context.WithValue(c, base.USER_CONTEXT_KEY, base.UserContext{ID: usr.ID, Name: usr.Name})
		personalWS, err = tx.Workspace.Create().
			SetName(wsName).
			SetOwnerID(usr.ID).
			SetPersonal(true).
			SetBaseCurrency(usr.Currency).
			Save(wsCtx)
		return err
	})

	if err != nil {
		return dom, err
	}

	if r.storage != nil && personalWS != nil && personalWS.ID > 0 {
		_, _ = r.storage.EnsureWorkspaceBucket(c, personalWS.ID)
	}

	return toDomain(usr), nil
}

// Update modifies an existing user by ID within a transaction.
func (r UserRepository) Update(c context.Context, id int64, req model.UpdateUserRequest) (domain.User, error) {
	var dom domain.User
	var usr *ent.User
	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		var err error
		uUpdate := tx.User.UpdateOneID(id)
		if req.Name != "" {
			uUpdate.SetName(req.Name)
		}
		if req.Avatar != "" {
			uUpdate.SetAvatar(req.Avatar)
		}
		if req.Locale != "" {
			uUpdate.SetLocale(req.Locale)
		}
		if req.Currency != "" {
			uUpdate.SetCurrency(req.Currency)
		}

		usr, err = uUpdate.Save(c)
		return err
	})

	if err != nil {
		return dom, err
	}

	return toDomain(usr), nil
}

// Delete removes a user by ID within a transaction.
func (r UserRepository) Delete(c context.Context, id int64) error {
	var ownedWSIDs []int64
	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		unfilteredCtx := base.WithSkipWorkspaceFilter(c)
		ownedWSIDs, _ = tx.Workspace.Query().Where(workspace.OwnerIDEQ(id)).IDs(unfilteredCtx)
		if len(ownedWSIDs) > 0 {
			_, _ = tx.File.Delete().Where(file.WorkspaceIDIn(ownedWSIDs...)).Exec(unfilteredCtx)
		}
		// Clean up memberships and owned workspaces
		_, _ = tx.WorkspaceMember.Delete().Where(workspacemember.UserIDEQ(id)).Exec(unfilteredCtx)
		_, _ = tx.Workspace.Delete().Where(workspace.OwnerIDEQ(id)).Exec(unfilteredCtx)

		err := tx.User.DeleteOneID(id).Exec(c)
		if err != nil {
			if ent.IsNotFound(err) {
				return base.ErrNotFound
			}
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	if r.storage != nil {
		for _, wsID := range ownedWSIDs {
			_ = r.storage.DeleteWorkspaceBucket(c, wsID)
		}
	}
	return nil
}

func toDomain(u *ent.User) domain.User {
	return domain.User{
		ID:        u.ID,
		Name:      u.Name,
		Email:     u.Email,
		Avatar:    u.Avatar,
		Locale:        u.Locale,
		Currency:      u.Currency,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}
