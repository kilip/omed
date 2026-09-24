package config

import (
	"log/slog"
	"os"
)

func GetLogger(config Config) *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	slog.SetDefault(logger)

	return logger
}
