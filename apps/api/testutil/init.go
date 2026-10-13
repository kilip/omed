package testutil

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/app"
	"github.com/kilip/omed/api/internal/core"
)

//go:generate go run go.uber.org/mock/mockgen@latest -destination=cmocks/mock_slog_handler.go -package=cmocks log/slog Handler

var cfg core.Config
var state core.State
var api *fiber.App
var logger *slog.Logger
var entClient *ent.Client
var jwksMock JWKSMock

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

	if os.Getenv("API_INTERNAL_KEY") == "" {
		_ = os.Setenv("API_INTERNAL_KEY", "test-internal-key-for-domus-api-at-least-32-chars")
	}

	cfg = core.GetConfig()

	mock := NewJWKSMock()
	jwksMock = *mock

	// force test environment config
	cfg.JWKSUrl = mock.URL

	_, b, _, _ := runtime.Caller(0)
	logPath := filepath.Join(filepath.Dir(b), "../tmp/test.log")
	_ = os.MkdirAll(filepath.Dir(logPath), 0755)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	var logWriter io.Writer = logFile
	if err != nil {
		logWriter = os.Stdout
	}

	logger = slog.New(slog.NewJSONHandler(logWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	api = app.GetFiber(cfg, logger)
	entClient = CreateTestDB()

	// config.ConfigureEntCli(entClient)
	// mockStorage := storage.NewMemoryStorage()
	state = core.State{
		FiberApp: api,
		Config:   cfg,
		Logger:   logger,
		EntCli:   entClient,
		// Authz:    config.GetAuthz(entClient),
		// Storage:  mockStorage,
	}

	a := app.New(state)
	a.Boot()

}

func GetState() core.State {
	return state
}

func GetEntCli() *ent.Client {
	return state.EntCli
}
