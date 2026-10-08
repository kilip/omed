package util

import (
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/core"
)

func GenerateID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		nr := errors.Join(core.ErrGenerateID, err)
		log.Fatal(nr.Error())
	}

	return id
}
