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
)
