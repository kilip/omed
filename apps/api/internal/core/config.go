package core

import (
	"context"
	"fmt"
	"log"

	_ "github.com/joho/godotenv/autoload"
	envconfig "github.com/sethvargo/go-envconfig"
)

type Config struct {
	AuthUrl        string   `env:"OMED_AUTH_URL"`
	AuthPath       string   `env:"OMED_AUTH_PATH"`
	JWKSUrl        string   `env:"OMED_JWKS_URL"`
	TrustedOrigins []string `env:"OMED_TRUSTED_ORIGINS,separator=,default=http://localhost:3001"`
}

func GetConfig() Config {
	var config Config

	if err := envconfig.Process(context.Background(), &config); err != nil {
		log.Fatalf("Error while parsing config: %s", err)
	}

	if config.AuthPath == "" {
		config.AuthPath = "/auth"
	}

	if config.JWKSUrl == "" {
		config.JWKSUrl = fmt.Sprintf("%s%s/jwks", config.AuthUrl, config.AuthPath)
	}

	return config
}
