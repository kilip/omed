// Package repository provides database access implementations for finance entities using Ent.
package repository

import (
	"context"
	"strconv"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/account"
	"github.com/kilip/omed/api/ent/predicate"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/entx"
)

// AccountRepository defines operations for persisting and querying chart of accounts entities.
type AccountRepository struct {
	cli *ent.Client
}

// NewAccountRepository constructs a new AccountRepository instance backed by the given Ent client.
func NewAccountRepository(cli *ent.Client) AccountRepository {
	return AccountRepository{cli: cli}
}

// GetByID retrieves an active account by its ID within a workspace.
func (r AccountRepository) GetByID(c context.Context, workspaceID, id int64) (domain.Account, error) {
	var dom domain.Account

	a, err := r.cli.Account.Query().
		Where(
			account.ID(id),
			account.WorkspaceID(workspaceID),
			account.DeletedAtIsNil(),
		).
		WithCreator().
		WithUpdater().
		WithDeleter().
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom, base.ErrNotFound
		}
		return dom, err
	}

	return toAccountDomain(a), nil
}

// Search queries accounts matching the filter criteria with cursor pagination.
func (r AccountRepository) Search(c context.Context, workspaceID int64, req model.SearchAccountRequest) ([]domain.Account, string, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	predicates := []predicate.Account{
		account.WorkspaceID(workspaceID),
	}

	if !req.IncludeDeleted {
		predicates = append(predicates, account.DeletedAtIsNil())
	}

	if req.Code != "" {
		predicates = append(predicates, account.CodeContainsFold(req.Code))
	}

	if req.Query != "" {
		predicates = append(predicates, account.Or(
			account.CodeContainsFold(req.Query),
			account.NameContainsFold(req.Query),
		))
	}

	if req.Type != "" {
		predicates = append(predicates, account.TypeEQ(account.Type(req.Type)))
	}

	if req.ParentID != nil {
		predicates = append(predicates, account.ParentIDEQ(*req.ParentID))
	}

	if req.IsActive != nil {
		predicates = append(predicates, account.IsActiveEQ(*req.IsActive))
	}

	if req.IsPostable != nil {
		predicates = append(predicates, account.IsPostableEQ(*req.IsPostable))
	}

	if req.Cursor != "" {
		if cursorID, err := strconv.ParseInt(req.Cursor, 10, 64); err == nil {
			predicates = append(predicates, account.IDGT(cursorID))
		}
	}

	items, err := r.cli.Account.Query().
		Where(predicates...).
		Order(ent.Asc(account.FieldID)).
		Limit(limit + 1).
		WithCreator().
		WithUpdater().
		WithDeleter().
		All(c)
	if err != nil {
		return nil, "", err
	}

	var cursor string
	if len(items) > limit {
		cursor = strconv.FormatInt(items[limit-1].ID, 10)
		items = items[:limit]
	}

	res := make([]domain.Account, len(items))
	for i, it := range items {
		res[i] = toAccountDomain(it)
	}

	return res, cursor, nil
}

// Create inserts a new account within a database transaction.
func (r AccountRepository) Create(c context.Context, workspaceID int64, req model.CreateAccountRequest, normalBalance string) (domain.Account, error) {
	var dom domain.Account

	var userID int64
	if u, ok := base.UserFromContext(c); ok && u.ID != 0 {
		userID = u.ID
	}

	var createdID int64
	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		builder := tx.Account.Create().
			SetWorkspaceID(workspaceID).
			SetCode(req.Code).
			SetName(req.Name).
			SetType(account.Type(req.Type)).
			SetNormalBalance(account.NormalBalance(normalBalance))

		if req.ParentID != nil && *req.ParentID > 0 {
			builder.SetParentID(*req.ParentID)
		}
		if req.Description != "" {
			builder.SetDescription(req.Description)
		}
		if req.IsActive != nil {
			builder.SetIsActive(*req.IsActive)
		} else {
			builder.SetIsActive(true)
		}
		if req.IsPostable != nil {
			builder.SetIsPostable(*req.IsPostable)
		} else {
			builder.SetIsPostable(true)
		}
		if userID != 0 {
			builder.SetCreatedByID(userID).SetUpdatedByID(userID)
		}

		res, err := builder.Save(c)
		if err != nil {
			if ent.IsConstraintError(err) {
				return domain.ErrAccountCodeTaken
			}
			return err
		}
		createdID = res.ID
		return nil
	})

	if err != nil {
		return dom, err
	}

	return r.GetByID(c, workspaceID, createdID)
}

// Update modifies an existing account within a database transaction.
func (r AccountRepository) Update(c context.Context, workspaceID, id int64, req model.UpdateAccountRequest) (domain.Account, error) {
	var dom domain.Account

	var userID int64
	if u, ok := base.UserFromContext(c); ok && u.ID != 0 {
		userID = u.ID
	}

	err := entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		// Verify account exists and is not soft deleted
		exists, err := tx.Account.Query().
			Where(
				account.ID(id),
				account.WorkspaceID(workspaceID),
				account.DeletedAtIsNil(),
			).
			Exist(c)
		if err != nil {
			return err
		}
		if !exists {
			return base.ErrNotFound
		}

		builder := tx.Account.UpdateOneID(id).
			SetName(req.Name)

		if req.Code != "" {
			builder.SetCode(req.Code)
		}
		if req.NormalBalance != "" {
			builder.SetNormalBalance(account.NormalBalance(req.NormalBalance))
		}
		if req.ParentID != nil {
			if *req.ParentID > 0 {
				builder.SetParentID(*req.ParentID)
			} else {
				builder.ClearParent()
			}
		}
		if req.Description != "" {
			builder.SetDescription(req.Description)
		}
		if req.IsActive != nil {
			builder.SetIsActive(*req.IsActive)
		}
		if req.IsPostable != nil {
			builder.SetIsPostable(*req.IsPostable)
		}
		if userID != 0 {
			builder.SetUpdatedByID(userID)
		}

		_, err = builder.Save(c)
		if err != nil {
			if ent.IsConstraintError(err) {
				return domain.ErrAccountCodeTaken
			}
			return err
		}
		return nil
	})

	if err != nil {
		return dom, err
	}

	return r.GetByID(c, workspaceID, id)
}

// Delete marks an existing account as soft-deleted.
func (r AccountRepository) Delete(c context.Context, workspaceID, id int64) error {
	var userID *int64
	if u, ok := base.UserFromContext(c); ok && u.ID != 0 {
		userID = &u.ID
	}

	return entx.WithTx(c, r.cli, func(tx *ent.Tx) error {
		exists, err := tx.Account.Query().
			Where(
				account.ID(id),
				account.WorkspaceID(workspaceID),
				account.DeletedAtIsNil(),
			).
			Exist(c)
		if err != nil {
			return err
		}
		if !exists {
			return base.ErrNotFound
		}

		now := time.Now()
		builder := tx.Account.UpdateOneID(id).
			SetDeletedAt(now)

		if userID != nil {
			builder.SetDeletedByID(*userID)
		}

		return builder.Exec(c)
	})
}

// HasActiveChildren reports whether any active child accounts reference this account as parent.
func (r AccountRepository) HasActiveChildren(c context.Context, workspaceID, id int64) (bool, error) {
	return r.cli.Account.Query().
		Where(
			account.WorkspaceID(workspaceID),
			account.ParentIDEQ(id),
			account.DeletedAtIsNil(),
		).
		Exist(c)
}

// FindByCode finds an active account by workspace and code.
func (r AccountRepository) FindByCode(c context.Context, workspaceID int64, code string) (domain.Account, error) {
	var dom domain.Account
	a, err := r.cli.Account.Query().
		Where(
			account.WorkspaceID(workspaceID),
			account.CodeEQ(code),
			account.DeletedAtIsNil(),
		).
		WithCreator().
		WithUpdater().
		WithDeleter().
		Only(c)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom, base.ErrNotFound
		}
		return dom, err
	}
	return toAccountDomain(a), nil
}

type treeBuilderNode struct {
	account  domain.Account
	children []*treeBuilderNode
}

func buildAccountTree(nodes []*treeBuilderNode) []domain.AccountTreeNode {
	res := make([]domain.AccountTreeNode, len(nodes))
	for i, n := range nodes {
		res[i] = domain.AccountTreeNode{
			Account:  n.account,
			Children: buildAccountTree(n.children),
		}
	}
	return res
}

// GetTree returns all active accounts in the workspace assembled into a hierarchy.
func (r AccountRepository) GetTree(c context.Context, workspaceID int64) ([]domain.AccountTreeNode, error) {
	items, err := r.cli.Account.Query().
		Where(
			account.WorkspaceID(workspaceID),
			account.DeletedAtIsNil(),
		).
		Order(ent.Asc(account.FieldCode)).
		WithCreator().
		WithUpdater().
		WithDeleter().
		All(c)
	if err != nil {
		return nil, err
	}

	nodesMap := make(map[int64]*treeBuilderNode, len(items))
	for _, it := range items {
		dom := toAccountDomain(it)
		nodesMap[dom.ID] = &treeBuilderNode{
			account:  dom,
			children: []*treeBuilderNode{},
		}
	}

	var rootPointers []*treeBuilderNode
	for _, it := range items {
		node := nodesMap[it.ID]
		if it.ParentID != nil && *it.ParentID > 0 {
			if parentNode, exists := nodesMap[*it.ParentID]; exists {
				parentNode.children = append(parentNode.children, node)
				continue
			}
		}
		rootPointers = append(rootPointers, node)
	}

	return buildAccountTree(rootPointers), nil
}

func toAccountDomain(a *ent.Account) domain.Account {
	dom := domain.Account{
		ID:            a.ID,
		WorkspaceID:   a.WorkspaceID,
		Code:          a.Code,
		Name:          a.Name,
		Type:          string(a.Type),
		NormalBalance: string(a.NormalBalance),
		ParentID:      a.ParentID,
		Description:   a.Description,
		IsActive:      a.IsActive,
		IsPostable:    a.IsPostable,
		DeletedAt:     a.DeletedAt,
		AuditMeta: base.AuditMeta{
			CreatedAt: a.CreatedAt,
			UpdatedAt: a.UpdatedAt,
		},
	}

	if a.Edges.Creator != nil {
		dom.AuditMeta.CreatedBy = base.ByUser{
			Name:   a.Edges.Creator.Name,
			Avatar: a.Edges.Creator.Avatar,
		}
	}
	if a.Edges.Updater != nil {
		dom.AuditMeta.UpdatedBy = base.ByUser{
			Name:   a.Edges.Updater.Name,
			Avatar: a.Edges.Updater.Avatar,
		}
	}
	if a.Edges.Deleter != nil {
		dom.DeletedBy = &base.ByUser{
			Name:   a.Edges.Deleter.Name,
			Avatar: a.Edges.Deleter.Avatar,
		}
	}

	return dom
}
