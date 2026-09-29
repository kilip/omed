package shared

import (
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/google/uuid"
)

func GenerateID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		nr := errors.Join(ErrGenerateID, err)
		log.Fatal(nr.Error())
	}

	return id
}

func ToValue[T any](source any, target *T) error {

	jsonData, err := json.Marshal(source)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, target)
}

func Expired(constraint time.Time, target time.Time) bool {
	if target.Before(constraint) || target.After(constraint) {
		return true
	}

	return false
}
