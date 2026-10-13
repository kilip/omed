package config_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/kilip/omed/api/internal/config"
	"github.com/stretchr/testify/suite"
)

type LoggerSuite struct {
	suite.Suite
}

func (s *LoggerSuite) Test_GetLogger() {
	logger := config.GetLogger()
	s.Require().NotNil(logger)

	// LevelInfo should be enabled by default
	s.True(logger.Enabled(context.Background(), slog.LevelInfo))
	// LevelDebug should not be enabled by default
	s.False(logger.Enabled(context.Background(), slog.LevelDebug))

	// Ensure logging does not panic
	s.NotPanics(func() {
		logger.Info("test info log", "key", "val")
	})
}

func TestLoggerSuite(t *testing.T) {
	suite.Run(t, new(LoggerSuite))
}
