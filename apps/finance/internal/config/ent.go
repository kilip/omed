package config

import (
	"log"

	"github.com/kilip/omed/finance/ent"
)

func GetEntClient(cfg Config) *ent.Client {
	cl, err := ent.Open("postgres", cfg.DatabaseUrl)
	if err != nil {
		log.Fatalf("DB failed connection %s", err)
	}

	return cl
}
