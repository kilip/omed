package config

import (
	"context"
	"log"
	"log/slog"

	_ "github.com/joho/godotenv/autoload"
	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	AuthBaseUrl string `env:"AUTH_URL,default=http://localhost:9001"`
	JWKSUrl     string `env:"AUTH_JWKS_URL,default=http://localhost:9001/jwks"`
	Port        int    `env:"FIN_PORT,default=9002"`
	DatabaseUrl string `env:"FIN_DB_URL,required"`
	SearchLimit int    `env:"FIN_SEARCH_LIMIT,default=10"`
}

func GetConfig() Config {
	var config Config

	if err := envconfig.Process(context.Background(), &config); err != nil {
		log.Fatalf("Error while parsing config %s", err)
	}
	slog.Info("Config", "config", config)

	return config
}
