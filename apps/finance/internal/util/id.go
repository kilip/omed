package util

import (
	"log"

	"github.com/google/uuid"
)

func GenerateID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		log.Fatalf("Can't create v7 id: %v", err)
	}

	return id
}
