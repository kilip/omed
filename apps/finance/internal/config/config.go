package config

import (
	"context"
	"log"

	_ "github.com/joho/godotenv/autoload"
	"github.com/sethvargo/go-envconfig"
)

var config Config
var configLoaded bool = false

type Config struct {
	App struct {
		Port    int    `env:"PORT, default=3001"`
		JWKSUrl string `env:"AUTH_JWKS_URL, default=http://localhost:3000/api/auth/jwks"`
	}
	DB struct {
		URL      string `env:"DB_URL"`
		Host     string `env:"DB_HOST"`
		Port     int    `env:"DB_PORT"`
		User     string `env:"DB_USER"`
		Password string `env:"DB_PASSWORD"`
		Name     string `env:"DB_NAME"`
		SSLMode  string `env:"DB_SSL"`
	}
}

func GetConfig() Config {
	if !configLoaded {
		ctx := context.Background()

		if err := envconfig.Process(ctx, &config); err != nil {
			log.Fatalf("Error while parsing config %s", err)
		}
		configLoaded = true
	}

	return config
}
