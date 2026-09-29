package testutil

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/config"
)

//go:generate go run go.uber.org/mock/mockgen@latest -destination=cmocks/mock_slog_handler.go -package=cmocks log/slog Handler

var jwksMock JWKSMock
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

	cfg = config.GetConfig()

	// force test environment config
	cfg.JWKSUrl = "http://localhost:4321/jwks"

	jwksMock = *NewJWKSMock()
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	api = config.GetFiber()
	entClient = createTestDB()

	state = config.State{
		Config:    cfg,
		Log:       logger,
		Api:       api,
		EntClient: entClient,
	}

	config.Bootstrap(state)
}

func GetState() config.State {
	return state
}
