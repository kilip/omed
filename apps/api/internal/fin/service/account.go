// Package service provides business logic and orchestration for finance entities.
package service

import (
	"context"
	"errors"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/seed"
)

// AccountRepository defines the persistence operations required by AccountService.
type AccountRepository interface {
	GetByID(c context.Context, id int64) (domain.Account, error)
	Search(c context.Context, req model.SearchAccountRequest) ([]domain.Account, string, error)
	Create(c context.Context, req model.CreateAccountRequest, normalBalance string) (domain.Account, error)
	Update(c context.Context, id int64, req model.UpdateAccountRequest) (domain.Account, error)
	Delete(c context.Context, id int64) error
	HasActiveChildren(c context.Context, id int64) (bool, error)
	FindByCode(c context.Context, code string) (domain.Account, error)
	Count(c context.Context) (int, error)
	Seed(c context.Context, template, lang string) (int, error)
	LoadTemplate(template, lang string) ([]seed.AccountItem, error)
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
func (s AccountService) GetByID(c context.Context, id int64) (domain.Account, error) {
	return s.repo.GetByID(c, id)
}

// Search searches accounts within a workspace based on filter criteria.
func (s AccountService) Search(c context.Context, req model.SearchAccountRequest) ([]domain.Account, string, error) {
	return s.repo.Search(c, req)
}

// Create validates and creates a new account in the workspace.
func (s AccountService) Create(c context.Context, req model.CreateAccountRequest) (domain.Account, error) {
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
	if req.ParentID > 0 {
		_, err := s.repo.GetByID(c, req.ParentID)
		if err != nil {
			if errors.Is(err, base.ErrNotFound) {
				return domain.Account{}, domain.ErrAccountParentNotFound
			}
			return domain.Account{}, err
		}
	}

	// Verify code is not already in use in this workspace
	existing, err := s.repo.FindByCode(c, req.Code)
	if err == nil && existing.ID != 0 {
		return domain.Account{}, domain.ErrAccountCodeTaken
	} else if err != nil && !errors.Is(err, base.ErrNotFound) {
		return domain.Account{}, err
	}

	return s.repo.Create(c, req, normalBalance)
}

// Update validates and updates an existing account in the workspace.
func (s AccountService) Update(c context.Context, id int64, req model.UpdateAccountRequest) (domain.Account, error) {
	existing, err := s.repo.GetByID(c, id)
	if err != nil {
		return domain.Account{}, err
	}

	// If code is being changed, ensure it's not already in use
	if req.Code != "" && req.Code != existing.Code {
		other, err := s.repo.FindByCode(c, req.Code)
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
	if req.ParentID > 0 {
		targetParentID := req.ParentID
		if targetParentID == id {
			return domain.Account{}, domain.ErrAccountCyclicParent
		}

		parent, err := s.repo.GetByID(c, targetParentID)
		if err != nil {
			if errors.Is(err, base.ErrNotFound) {
				return domain.Account{}, domain.ErrAccountParentNotFound
			}
			return domain.Account{}, err
		}

		// Traverse ancestors of target parent to make sure `id` is not among them
		curr := parent.ParentID
		for curr > 0 {
			if curr == id {
				return domain.Account{}, domain.ErrAccountCyclicParent
			}
			ancestor, err := s.repo.GetByID(c, curr)
			if err != nil {
				break
			}
			curr = ancestor.ParentID
		}
	}

	return s.repo.Update(c, id, req)
}

// Delete validates and performs soft deletion on an account.
func (s AccountService) Delete(c context.Context, id int64) error {
	_, err := s.repo.GetByID(c, id)
	if err != nil {
		return err
	}

	hasChildren, err := s.repo.HasActiveChildren(c, id)
	if err != nil {
		return err
	}
	if hasChildren {
		return domain.ErrAccountHasChildren
	}

	return s.repo.Delete(c, id)
}

// Seed populates the workspace with a predefined Chart of Accounts template.
func (s AccountService) Seed(c context.Context, req model.SeedAccountsRequest) (model.SeedAccountsResponse, error) {
	if req.Template == "" || req.Lang == "" {
		return model.SeedAccountsResponse{}, domain.ErrAccountTemplateNotFound
	}

	count, err := s.repo.Seed(c, req.Template, req.Lang)
	if err != nil {
		return model.SeedAccountsResponse{}, err
	}

	return model.SeedAccountsResponse{Count: count}, nil
}

// PreviewTemplate loads and formats a COA template for preview.
func (s AccountService) PreviewTemplate(c context.Context, template, lang string) (model.AccountTemplatePreviewResponse, error) {
	if template == "" || lang == "" {
		return model.AccountTemplatePreviewResponse{}, domain.ErrAccountTemplateNotFound
	}

	items, err := s.repo.LoadTemplate(template, lang)
	if err != nil {
		return model.AccountTemplatePreviewResponse{}, err
	}

	accounts := make([]model.AccountTemplateItemResponse, len(items))
	for i, item := range items {
		var parentCode *string
		if item.ParentCode != "" {
			p := item.ParentCode
			parentCode = &p
		}
		accounts[i] = model.AccountTemplateItemResponse{
			Code:          item.Code,
			Name:          item.Name,
			Type:          item.Type,
			NormalBalance: item.NormalBalance,
			ParentCode:    parentCode,
			Description:   item.Description,
			IsPostable:    item.IsPostable,
		}
	}

	return model.AccountTemplatePreviewResponse{
		Template: template,
		Lang:     lang,
		Accounts: accounts,
	}, nil
}
