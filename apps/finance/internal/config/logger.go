package config

import (
	"log/slog"
	"os"
)

func GetLogger(cfg Config) *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))

	return logger
}
