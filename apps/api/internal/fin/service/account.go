// Package service provides business logic and orchestration for finance entities.
package service

import (
	"context"
	"errors"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
)

// AccountRepository defines the persistence operations required by AccountService.
type AccountRepository interface {
	GetByID(c context.Context, workspaceID, id int64) (domain.Account, error)
	Search(c context.Context, workspaceID int64, req model.SearchAccountRequest) ([]domain.Account, string, error)
	Create(c context.Context, workspaceID int64, req model.CreateAccountRequest, normalBalance string) (domain.Account, error)
	Update(c context.Context, workspaceID, id int64, req model.UpdateAccountRequest) (domain.Account, error)
	Delete(c context.Context, workspaceID, id int64) error
	HasActiveChildren(c context.Context, workspaceID, id int64) (bool, error)
	FindByCode(c context.Context, workspaceID int64, code string) (domain.Account, error)
	GetTree(c context.Context, workspaceID int64) ([]domain.AccountTreeNode, error)
}

// AccountService manages business logic and validation for Chart of Accounts.
type AccountService struct {
	repo AccountRepository
}

// NewAccountService constructs a new AccountService backed by the given repository.
func NewAccountService(repo AccountRepository) AccountService {
	return AccountService{repo: repo}
}

// GetByID retrieves an active account by its ID within a workspace.
func (s AccountService) GetByID(c context.Context, workspaceID, id int64) (domain.Account, error) {
	return s.repo.GetByID(c, workspaceID, id)
}

// Search searches accounts within a workspace based on filter criteria.
func (s AccountService) Search(c context.Context, workspaceID int64, req model.SearchAccountRequest) ([]domain.Account, string, error) {
	return s.repo.Search(c, workspaceID, req)
}

// GetTree returns all accounts assembled into a hierarchical tree for the workspace.
func (s AccountService) GetTree(c context.Context, workspaceID int64) ([]domain.AccountTreeNode, error) {
	return s.repo.GetTree(c, workspaceID)
}

// Create validates and creates a new account in the workspace.
func (s AccountService) Create(c context.Context, workspaceID int64, req model.CreateAccountRequest) (domain.Account, error) {
	if !domain.ValidAccountTypes[req.Type] {
		return domain.Account{}, domain.ErrAccountInvalidType
	}

	normalBalance := req.NormalBalance
	if normalBalance != "" {
		if !domain.ValidNormalBalances[normalBalance] {
			return domain.Account{}, domain.ErrAccountInvalidBalance
		}
	} else {
		normalBalance = domain.DefaultNormalBalance(req.Type)
	}

	// Check if parent account exists and belongs to the workspace
	if req.ParentID != nil && *req.ParentID > 0 {
		_, err := s.repo.GetByID(c, workspaceID, *req.ParentID)
		if err != nil {
			if errors.Is(err, base.ErrNotFound) {
				return domain.Account{}, domain.ErrAccountParentNotFound
			}
			return domain.Account{}, err
		}
	}

	// Verify code is not already in use in this workspace
	existing, err := s.repo.FindByCode(c, workspaceID, req.Code)
	if err == nil && existing.ID != 0 {
		return domain.Account{}, domain.ErrAccountCodeTaken
	} else if err != nil && !errors.Is(err, base.ErrNotFound) {
		return domain.Account{}, err
	}

	return s.repo.Create(c, workspaceID, req, normalBalance)
}

// Update validates and updates an existing account in the workspace.
func (s AccountService) Update(c context.Context, workspaceID, id int64, req model.UpdateAccountRequest) (domain.Account, error) {
	existing, err := s.repo.GetByID(c, workspaceID, id)
	if err != nil {
		return domain.Account{}, err
	}

	// If code is being changed, ensure it's not already in use
	if req.Code != "" && req.Code != existing.Code {
		other, err := s.repo.FindByCode(c, workspaceID, req.Code)
		if err == nil && other.ID != id {
			return domain.Account{}, domain.ErrAccountCodeTaken
		} else if err != nil && !errors.Is(err, base.ErrNotFound) {
			return domain.Account{}, err
		}
	}

	if req.NormalBalance != "" {
		if !domain.ValidNormalBalances[req.NormalBalance] {
			return domain.Account{}, domain.ErrAccountInvalidBalance
		}
	}

	// Cycle detection for parent account
	if req.ParentID != nil && *req.ParentID > 0 {
		targetParentID := *req.ParentID
		if targetParentID == id {
			return domain.Account{}, domain.ErrAccountCyclicParent
		}

		parent, err := s.repo.GetByID(c, workspaceID, targetParentID)
		if err != nil {
			if errors.Is(err, base.ErrNotFound) {
				return domain.Account{}, domain.ErrAccountParentNotFound
			}
			return domain.Account{}, err
		}

		// Traverse ancestors of target parent to make sure `id` is not among them
		curr := parent.ParentID
		for curr != nil && *curr > 0 {
			if *curr == id {
				return domain.Account{}, domain.ErrAccountCyclicParent
			}
			ancestor, err := s.repo.GetByID(c, workspaceID, *curr)
			if err != nil {
				break
			}
			curr = ancestor.ParentID
		}
	}

	return s.repo.Update(c, workspaceID, id, req)
}

// Delete validates and performs soft deletion on an account.
func (s AccountService) Delete(c context.Context, workspaceID, id int64) error {
	_, err := s.repo.GetByID(c, workspaceID, id)
	if err != nil {
		return err
	}

	hasChildren, err := s.repo.HasActiveChildren(c, workspaceID, id)
	if err != nil {
		return err
	}
	if hasChildren {
		return domain.ErrAccountHasChildren
	}

	return s.repo.Delete(c, workspaceID, id)
}
