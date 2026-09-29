package shared

import "errors"

var (
	ErrInvalidID               = errors.New("INVALID_ID_FORMAT")
	ErrGenerateID              = errors.New("GENERATE_ID_FAILED")
	ErrItemNotFound            = errors.New("ITEM_NOT_FOUND")
	ErrUnimplemented           = errors.New("NOT_IMPLEMENTED")
	ErrInvalidDateRange        = errors.New("INVALID_DATE_RANGE")
	ErrPeriodOverlap           = errors.New("PERIOD_OVERLAP")
	ErrInvalidStatusTransition = errors.New("INVALID_STATUS_TRANSITION")
	ErrUnbalancedEntry         = errors.New("UNBALANCED_ENTRY")
	ErrMinPostingsCount        = errors.New("MIN_POSTINGS_COUNT")
	ErrPeriodClosed            = errors.New("PERIOD_CLOSED")
	ErrInactiveAccount         = errors.New("INACTIVE_ACCOUNT")
	ErrSameCurrencyNotAllowed  = errors.New("SAME_CURRENCY_NOT_ALLOWED")
	ErrInvalidExchangeRate     = errors.New("INVALID_EXCHANGE_RATE")
	ErrAccountsNotEmpty        = errors.New("ACCOUNTS_NOT_EMPTY")
	ErrEntriesNotEmpty         = errors.New("ENTRIES_NOT_EMPTY")
)

