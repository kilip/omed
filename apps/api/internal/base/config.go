package base

import (
	"context"
	"fmt"
	"log"

	_ "github.com/joho/godotenv/autoload"
	envconfig "github.com/sethvargo/go-envconfig"
)

// Config holds the application-level environment configuration settings.
type Config struct {
	// AuthUrl is the base URL for the authentication service.
	AuthUrl string `env:"OMED_AUTH_URL"`
	// AuthPath is the subpath mounted for authentication endpoints.
	AuthPath string `env:"OMED_AUTH_PATH"`
	// JWKSUrl is the JWKS endpoint used to verify signed JSON Web Tokens.
	JWKSUrl string `env:"OMED_JWKS_URL"`
	// TrustedOrigins lists allowed CORS origins for browser clients.
	TrustedOrigins []string `env:"OMED_TRUSTED_ORIGINS,separator=,default=http://localhost:3001"`
}

// GetConfig reads environment variables and returns a populated Config instance.
// It applies default fallback values for AuthPath and JWKSUrl if they are omitted.
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
