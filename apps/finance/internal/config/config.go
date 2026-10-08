package config

import (
	"context"
	"log"

	_ "github.com/joho/godotenv/autoload"
	envconfig "github.com/sethvargo/go-envconfig"
)

type Config struct {
	Port           int      `env:"FIN_PORT,default=8002"`
	AuthBaseUrl    string   `env:"AUTH_BASE_URL,default=http://localhost:8001"`
	JWKSUrl        string   `env:"AUTH_JWKS_URL,default=http://localhost:8001/jwks"`
	DatabaseUrl    string   `env:"FIN_DB_URL,default=postgresql://omed:omed@localhost:5432/omed"`
	TrustedOrigins []string `env:"FIN_TRUSTED_ORIGINS,delimiter= ,default=http://localhost:3001 http://localhost:8002"`
}

func GetConfig() Config {
	var config Config

	if err := envconfig.Process(context.Background(), &config); err != nil {
		log.Fatalf("Error while parsing config: %s", err)
	}

	return config
}
