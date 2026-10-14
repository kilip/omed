// Package domain defines the domain models and business errors for the finance module.
package domain

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
)

var (
	// ErrAccountCodeTaken indicates the account code is already used in the workspace.
	ErrAccountCodeTaken = errors.New("ACCOUNT_CODE_TAKEN")
	// ErrAccountHasChildren indicates an account cannot be deleted because it still has active sub-accounts.
	ErrAccountHasChildren = errors.New("ACCOUNT_HAS_CHILDREN")
	// ErrAccountParentNotFound indicates the specified parent account does not exist.
	ErrAccountParentNotFound = errors.New("ACCOUNT_PARENT_NOT_FOUND")
	// ErrAccountCyclicParent indicates an account cannot be configured with itself or a descendant as its parent.
	ErrAccountCyclicParent = errors.New("ACCOUNT_CYCLIC_PARENT")
	// ErrAccountParentWorkspaceMismatch indicates the specified parent account belongs to a different workspace.
	ErrAccountParentWorkspaceMismatch = errors.New("ACCOUNT_PARENT_WORKSPACE_MISMATCH")
	// ErrAccountInvalidType indicates an invalid account type was provided.
	ErrAccountInvalidType = errors.New("ACCOUNT_INVALID_TYPE")
	// ErrAccountInvalidBalance indicates an invalid normal balance was provided.
	ErrAccountInvalidBalance = errors.New("ACCOUNT_INVALID_BALANCE")
)

func init() {
	base.AddToErrorMap(ErrAccountCodeTaken, fiber.StatusConflict, "account code already in use in this workspace")
	base.AddToErrorMap(ErrAccountHasChildren, fiber.StatusConflict, "cannot delete account with existing sub-accounts")
	base.AddToErrorMap(ErrAccountParentNotFound, fiber.StatusNotFound, "parent account not found")
	base.AddToErrorMap(ErrAccountCyclicParent, fiber.StatusUnprocessableEntity, "account cannot be its own ancestor")
	base.AddToErrorMap(ErrAccountParentWorkspaceMismatch, fiber.StatusBadRequest, "parent account belongs to a different workspace")
	base.AddToErrorMap(ErrAccountInvalidType, fiber.StatusUnprocessableEntity, "invalid account type")
	base.AddToErrorMap(ErrAccountInvalidBalance, fiber.StatusUnprocessableEntity, "invalid normal balance")
}
