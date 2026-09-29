package shared

import "errors"

var (
	ErrInvalidID    = errors.New("INVALID_ID_FORMAT")
	ErrGenerateID   = errors.New("GENERATE_ID_FILED")
	ErrItemNotFound = errors.New("ITEM_NOT_FOUND")
)
