package testutil

import (
	"log/slog"
	"path/filepath"
	"runtime"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/config"
)

var cfg config.Config
var state config.State
var fiberApp *fiber.App
var logger *slog.Logger
var jwksMock *JWKSMock
var entClient *ent.Client

func loadEnvForTest() {
	// _, b, _, _ returns the absolute path of the current file
	_, b, _, _ := runtime.Caller(0)

	// Get the directory of this file, then navigate up to the project root
	// Adjust "../" depending on how deep this file is nested from the root
	basepath := filepath.Join(filepath.Dir(b), "../.env")

	_ = godotenv.Load(basepath)
}

func init() {
	loadEnvForTest()
	jwksMock = NewJWKSMock()
	cfg = config.GetConfig()
	logger = config.GetLogger(cfg)
	fiberApp = config.GetFiber(cfg, logger)
	entClient = config.GetEntClient(cfg)

	state = config.State{
		Config:    cfg,
		Fiber:     fiberApp,
		Logger:    logger,
		EntClient: entClient,
	}

	config.Bootstrap(state)
}
