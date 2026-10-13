package config

import (
	"log/slog"
	"os"
)

// GetLogger creates and configures the default JSON structured logger (slog) writing to stdout.
func GetLogger() *slog.Logger {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	return logger
}
