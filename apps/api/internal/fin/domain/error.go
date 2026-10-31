// Package domain defines the domain models and business errors for the finance module.
package domain

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
)

var (
	// ErrAccountNotFound indicates the requested account was not found.
	ErrAccountNotFound = errors.New("ACCOUNT_NOT_FOUND")
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

	// ErrTransactionNotFound indicates the requested transaction was not found.
	ErrTransactionNotFound = errors.New("TRANSACTION_NOT_FOUND")
	// ErrTransactionNumberTaken indicates the transaction number is already in use.
	ErrTransactionNumberTaken = errors.New("TRANSACTION_NUMBER_TAKEN")
	// ErrTransactionHasPostedEntries indicates the transaction cannot be modified or deleted because it has posted entries.
	ErrTransactionHasPostedEntries = errors.New("TRANSACTION_HAS_POSTED_ENTRIES")
	// ErrAttachmentNotFound indicates the requested transaction attachment was not found.
	ErrAttachmentNotFound = errors.New("ATTACHMENT_NOT_FOUND")
	// ErrAttachmentTooLarge indicates the uploaded attachment exceeds the maximum allowed size.
	ErrAttachmentTooLarge = errors.New("ATTACHMENT_TOO_LARGE")
	// ErrAttachmentInvalidType indicates the uploaded attachment has an unsupported file type.
	ErrAttachmentInvalidType = errors.New("ATTACHMENT_INVALID_TYPE")

	// ErrEntryNotFound indicates the requested journal entry was not found.
	ErrEntryNotFound = errors.New("ENTRY_NOT_FOUND")
	// ErrEntryNotDraft indicates the entry is not in draft status.
	ErrEntryNotDraft = errors.New("ENTRY_NOT_DRAFT")
	// ErrEntryNotBalanced indicates total debits and credits do not balance in transaction currency.
	ErrEntryNotBalanced = errors.New("ENTRY_NOT_BALANCED")
	// ErrEntryBaseNotBalanced indicates total debits and credits do not balance in base currency.
	ErrEntryBaseNotBalanced = errors.New("ENTRY_BASE_NOT_BALANCED")
	// ErrEntryMinPostings indicates an entry must have at least two postings.
	ErrEntryMinPostings = errors.New("ENTRY_MIN_POSTINGS")
	// ErrEntryDateOutsidePeriod indicates the entry date does not fall within the accounting period.
	ErrEntryDateOutsidePeriod = errors.New("ENTRY_DATE_OUTSIDE_PERIOD")
	// ErrEntryAlreadyReversed indicates the entry has already been reversed.
	ErrEntryAlreadyReversed = errors.New("ENTRY_ALREADY_REVERSED")
	// ErrEntryNotPosted indicates the entry is not in posted status and cannot be reversed.
	ErrEntryNotPosted = errors.New("ENTRY_NOT_POSTED")
	// ErrEntryRetryExhausted indicates entry operation exceeded retry limit due to concurrent updates or conflicts.
	ErrEntryRetryExhausted = errors.New("ENTRY_RETRY_EXHAUSTED")

	// ErrPostingAccountNotPostable indicates an account cannot accept direct postings.
	ErrPostingAccountNotPostable = errors.New("POSTING_ACCOUNT_NOT_POSTABLE")
	// ErrPostingAccountInactive indicates an account is inactive and cannot accept postings.
	ErrPostingAccountInactive = errors.New("POSTING_ACCOUNT_INACTIVE")
	// ErrPostingCurrencyMismatch indicates the posting currency does not match the account's restricted currency.
	ErrPostingCurrencyMismatch = errors.New("POSTING_CURRENCY_MISMATCH")
	// ErrPostingInvalidAmount indicates the posting amount is non-positive or invalid.
	ErrPostingInvalidAmount = errors.New("POSTING_INVALID_AMOUNT")
	// ErrPostingInvalidQuantity indicates the posting quantity is non-positive.
	ErrPostingInvalidQuantity = errors.New("POSTING_INVALID_QUANTITY")
	// ErrPostingInvalidUnitPrice indicates the posting unit price is negative.
	ErrPostingInvalidUnitPrice = errors.New("POSTING_INVALID_UNIT_PRICE")

	// ErrPeriodHasDraftEntries indicates a period cannot be closed while it still has draft entries.
	ErrPeriodHasDraftEntries = errors.New("PERIOD_HAS_DRAFT_ENTRIES")
	// ErrPeriodAlreadyRevalued indicates foreign currency balances have already been revalued for this period.
	ErrPeriodAlreadyRevalued = errors.New("PERIOD_ALREADY_REVALUED")

	// ErrCurrencyNotFound indicates the requested currency was not found.
	ErrCurrencyNotFound = errors.New("CURRENCY_NOT_FOUND")
	// ErrExchangeRateNotFound indicates no exchange rate was found for the currency pair on the given date.
	ErrExchangeRateNotFound = errors.New("EXCHANGE_RATE_NOT_FOUND")
	// ErrExchangeRateDuplicate indicates an exchange rate already exists for the given workspace, currency pair, and date.
	ErrExchangeRateDuplicate = errors.New("EXCHANGE_RATE_DUPLICATE")
	// ErrExchangeRateInvalid indicates an exchange rate is non-positive or invalid.
	ErrExchangeRateInvalid = errors.New("EXCHANGE_RATE_INVALID")
	// ErrBaseCurrencyLocked indicates the workspace base currency cannot be changed once posted entries exist.
	ErrBaseCurrencyLocked = errors.New("BASE_CURRENCY_LOCKED")

	// ErrFinSettingNotFound indicates the workspace financial settings were not found.
	ErrFinSettingNotFound = errors.New("FIN_SETTING_NOT_FOUND")
	// ErrFXAccountsNotConfigured indicates foreign exchange accounts (gain, loss, or rounding) are not configured.
	ErrFXAccountsNotConfigured = errors.New("FX_ACCOUNTS_NOT_CONFIGURED")
	// ErrFXGainAccountNotFound indicates the configured FX gain account was not found.
	ErrFXGainAccountNotFound = errors.New("FX_GAIN_ACCOUNT_NOT_FOUND")
	// ErrFXLossAccountNotFound indicates the configured FX loss account was not found.
	ErrFXLossAccountNotFound = errors.New("FX_LOSS_ACCOUNT_NOT_FOUND")
	// ErrFXRoundingAccountNotFound indicates the configured FX rounding account was not found.
	ErrFXRoundingAccountNotFound = errors.New("FX_ROUNDING_ACCOUNT_NOT_FOUND")
	// ErrFXAccountInvalid indicates the configured FX account is inactive or not postable.
	ErrFXAccountInvalid = errors.New("FX_ACCOUNT_INVALID")

	// ErrInvalidArgument indicates an invalid input parameter.
	ErrInvalidArgument = errors.New("INVALID_ARGUMENT")
)

// ArgumentError represents a validation error on a specific argument.
type ArgumentError struct {
	Field  string
	Expect string
}

func (e ArgumentError) Error() string {
	return e.ErrorDetail()
}

func (e ArgumentError) Is(target error) bool {
	return target == ErrInvalidArgument
}

func (e ArgumentError) ErrorDetail() string {
	if e.Field == "" || e.Expect == "" {
		return e.Expect
	}
	if len(e.Expect) >= len(e.Field) && e.Expect[:len(e.Field)] == e.Field {
		return e.Expect
	}
	return e.Field + " " + e.Expect
}

func init() {
	base.AddToErrorMap(ErrAccountNotFound, fiber.StatusNotFound, "account not found")
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

	base.AddToErrorMap(ErrTransactionNotFound, fiber.StatusNotFound, "transaction not found")
	base.AddToErrorMap(ErrTransactionNumberTaken, fiber.StatusConflict, "transaction number already in use")
	base.AddToErrorMap(ErrTransactionHasPostedEntries, fiber.StatusConflict, "transaction has posted entries")
	base.AddToErrorMap(ErrAttachmentNotFound, fiber.StatusNotFound, "attachment not found")
	base.AddToErrorMap(ErrAttachmentTooLarge, fiber.StatusUnprocessableEntity, "attachment exceeds maximum allowed file size")
	base.AddToErrorMap(ErrAttachmentInvalidType, fiber.StatusUnprocessableEntity, "unsupported attachment file type")

	base.AddToErrorMap(ErrEntryNotFound, fiber.StatusNotFound, "entry not found")
	base.AddToErrorMap(ErrEntryNotDraft, fiber.StatusConflict, "entry is not in draft status")
	base.AddToErrorMap(ErrEntryNotBalanced, fiber.StatusUnprocessableEntity, "entry postings debit and credit are not balanced")
	base.AddToErrorMap(ErrEntryBaseNotBalanced, fiber.StatusUnprocessableEntity, "entry base amounts debit and credit are not balanced")
	base.AddToErrorMap(ErrEntryMinPostings, fiber.StatusUnprocessableEntity, "entry must have at least two postings")
	base.AddToErrorMap(ErrEntryDateOutsidePeriod, fiber.StatusUnprocessableEntity, "entry date is outside the accounting period")
	base.AddToErrorMap(ErrEntryAlreadyReversed, fiber.StatusConflict, "entry has already been reversed")
	base.AddToErrorMap(ErrEntryNotPosted, fiber.StatusConflict, "entry is not in posted status")
	base.AddToErrorMap(ErrEntryRetryExhausted, fiber.StatusConflict, "entry operation retry limit exceeded")

	base.AddToErrorMap(ErrPostingAccountNotPostable, fiber.StatusUnprocessableEntity, "account is not postable")
	base.AddToErrorMap(ErrPostingAccountInactive, fiber.StatusUnprocessableEntity, "account is inactive")
	base.AddToErrorMap(ErrPostingCurrencyMismatch, fiber.StatusUnprocessableEntity, "posting currency does not match account currency")
	base.AddToErrorMap(ErrPostingInvalidAmount, fiber.StatusUnprocessableEntity, "posting amount must be positive")
	base.AddToErrorMap(ErrPostingInvalidQuantity, fiber.StatusUnprocessableEntity, "posting quantity must be positive")
	base.AddToErrorMap(ErrPostingInvalidUnitPrice, fiber.StatusUnprocessableEntity, "posting unit price cannot be negative")

	base.AddToErrorMap(ErrPeriodHasDraftEntries, fiber.StatusConflict, "period has draft entries")
	base.AddToErrorMap(ErrPeriodAlreadyRevalued, fiber.StatusConflict, "period has already been revalued")

	base.AddToErrorMap(ErrCurrencyNotFound, fiber.StatusNotFound, "currency not found")
	base.AddToErrorMap(ErrExchangeRateNotFound, fiber.StatusNotFound, "exchange rate not found")
	base.AddToErrorMap(ErrExchangeRateDuplicate, fiber.StatusConflict, "exchange rate already exists for this date and currency pair")
	base.AddToErrorMap(ErrExchangeRateInvalid, fiber.StatusUnprocessableEntity, "exchange rate must be positive")
	base.AddToErrorMap(ErrBaseCurrencyLocked, fiber.StatusConflict, "base currency is locked once posted entries exist")
	base.AddToErrorMap(ErrFinSettingNotFound, fiber.StatusNotFound, "financial settings not found for workspace")
	base.AddToErrorMap(ErrFXAccountsNotConfigured, fiber.StatusUnprocessableEntity, "foreign exchange accounts are not configured for this workspace")
	base.AddToErrorMap(ErrFXGainAccountNotFound, fiber.StatusUnprocessableEntity, "foreign exchange gain account not found or not configured")
	base.AddToErrorMap(ErrFXLossAccountNotFound, fiber.StatusUnprocessableEntity, "foreign exchange loss account not found or not configured")
	base.AddToErrorMap(ErrFXRoundingAccountNotFound, fiber.StatusUnprocessableEntity, "foreign exchange rounding account not found or not configured")
	base.AddToErrorMap(ErrFXAccountInvalid, fiber.StatusUnprocessableEntity, "configured foreign exchange account is invalid, inactive, or not postable")
	base.AddToErrorMap(ErrInvalidArgument, fiber.StatusUnprocessableEntity, "invalid argument")
}
