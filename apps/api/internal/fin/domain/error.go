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
	// ErrAccountWorkspaceNotEmpty indicates accounts cannot be seeded because the workspace already has accounts.
	ErrAccountWorkspaceNotEmpty = errors.New("ACCOUNT_WORKSPACE_NOT_EMPTY")
	// ErrAccountTemplateNotFound indicates the requested COA template or language was not found.
	ErrAccountTemplateNotFound = errors.New("ACCOUNT_TEMPLATE_NOT_FOUND")

	// ErrPeriodNotFound indicates the requested accounting period was not found.
	ErrPeriodNotFound = errors.New("PERIOD_NOT_FOUND")
	// ErrPeriodInvalidDates indicates start_date is after end_date.
	ErrPeriodInvalidDates = errors.New("PERIOD_INVALID_DATES")
	// ErrPeriodOverlap indicates the date range overlaps with an existing period in the workspace.
	ErrPeriodOverlap = errors.New("PERIOD_OVERLAP")
	// ErrPeriodClosed indicates the period is closed and cannot be updated or deleted.
	ErrPeriodClosed = errors.New("PERIOD_CLOSED")
	// ErrPeriodAlreadyClosed indicates the period is already in closed state.
	ErrPeriodAlreadyClosed = errors.New("PERIOD_ALREADY_CLOSED")
	// ErrPeriodAlreadyOpen indicates the period is already in open state.
	ErrPeriodAlreadyOpen = errors.New("PERIOD_ALREADY_OPEN")
	// ErrPeriodInvalidFiscalYear indicates an invalid fiscal year.
	ErrPeriodInvalidFiscalYear = errors.New("PERIOD_INVALID_FISCAL_YEAR")
)

func init() {
	base.AddToErrorMap(ErrAccountCodeTaken, fiber.StatusConflict, "account code already in use in this workspace")
	base.AddToErrorMap(ErrAccountHasChildren, fiber.StatusConflict, "cannot delete account with existing sub-accounts")
	base.AddToErrorMap(ErrAccountParentNotFound, fiber.StatusNotFound, "parent account not found")
	base.AddToErrorMap(ErrAccountCyclicParent, fiber.StatusUnprocessableEntity, "account cannot be its own ancestor")
	base.AddToErrorMap(ErrAccountParentWorkspaceMismatch, fiber.StatusBadRequest, "parent account belongs to a different workspace")
	base.AddToErrorMap(ErrAccountInvalidType, fiber.StatusUnprocessableEntity, "invalid account type")
	base.AddToErrorMap(ErrAccountInvalidBalance, fiber.StatusUnprocessableEntity, "invalid normal balance")
	base.AddToErrorMap(ErrAccountWorkspaceNotEmpty, fiber.StatusConflict, "workspace already has accounts, cannot seed template")
	base.AddToErrorMap(ErrAccountTemplateNotFound, fiber.StatusBadRequest, "account template or language not found")

	base.AddToErrorMap(ErrPeriodNotFound, fiber.StatusNotFound, "accounting period not found")
	base.AddToErrorMap(ErrPeriodInvalidDates, fiber.StatusUnprocessableEntity, "start date must be before or equal to end date")
	base.AddToErrorMap(ErrPeriodOverlap, fiber.StatusConflict, "period date range overlaps with an existing period in this workspace")
	base.AddToErrorMap(ErrPeriodClosed, fiber.StatusConflict, "period is closed and cannot be modified or deleted")
	base.AddToErrorMap(ErrPeriodAlreadyClosed, fiber.StatusConflict, "period is already closed")
	base.AddToErrorMap(ErrPeriodAlreadyOpen, fiber.StatusConflict, "period is already open")
	base.AddToErrorMap(ErrPeriodInvalidFiscalYear, fiber.StatusUnprocessableEntity, "fiscal year must be a positive integer")
}
