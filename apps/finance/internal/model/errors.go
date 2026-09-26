package model

import "errors"

var (
	ErrItemNotFound = errors.New("item not found")
	ErrUpdateFailed = errors.New("update failed")
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrItemNotFound)
}
