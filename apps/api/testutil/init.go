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
	"github.com/kilip/omed/api/internal/config"
	"github.com/kilip/omed/api/internal/platform/mailx"
	"github.com/kilip/omed/api/internal/platform/s3x"
)

//go:generate go run go.uber.org/mock/mockgen@latest -destination=cmocks/mock_slog_handler.go -package=cmocks log/slog Handler

var cfg config.Config
var state config.State
var api *fiber.App
var logger *slog.Logger
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

	if os.Getenv("API_INTERNAL_KEY") == "" {
		_ = os.Setenv("API_INTERNAL_KEY", "test-internal-key-for-domus-api-at-least-32-chars")
	}

	if os.Getenv("OMED_AUTH_SECRET") == "" {
		_ = os.Setenv("OMED_AUTH_SECRET", "test-secret-at-least-32-chars-long!")
	}

	cfg = config.GetConfig()

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
	api = config.GetFiber(cfg, logger)
	entClient = CreateTestDB()

	// config.ConfigureEntCli(entClient)
	// mockStorage := storage.NewMemoryStorage()
	state = config.State{
		FiberApp: api,
		Config:   cfg,
		Logger:   logger,
		EntCli:   entClient,
		// Authz:    config.GetAuthz(entClient),
		// Storage:  mockStorage,
	}

	a := config.New(state)
	a.Boot()
	state = a.State()
}

// GetState returns the global initialized testing State instance.
func GetState() config.State {
	return state
}

// GetEntCli returns the global in-memory test Ent client.
func GetEntCli() *ent.Client {
	return state.EntCli
}

// GetMailer returns the global initialized testing Mailer instance.
func GetMailer() mailx.Mailer {
	return state.Mailer
}

// GetStorage returns the global initialized testing Storage instance.
func GetStorage() s3x.Storage {
	return state.Storage
}
