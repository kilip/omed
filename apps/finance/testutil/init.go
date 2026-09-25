package testutil

import (
	"context"
	"log"
	"log/slog"
	"path/filepath"
	"runtime"

	"github.com/casbin/casbin/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/config"
	_ "github.com/mattn/go-sqlite3"
)

var app *fiber.App
var db *ent.Client
var logger *slog.Logger
var cfg config.Config
var jwksMock *JWKSMock
var enforcer *casbin.Enforcer

func loadEnvForTest() {
	// _, b, _, _ returns the absolute path of the current file
	_, b, _, _ := runtime.Caller(0)

	// Get the directory of this file, then navigate up to the project root
	// Adjust "../" depending on how deep this file is nested from the root
	basepath := filepath.Join(filepath.Dir(b), "../.env.test")

	_ = godotenv.Load(basepath)
}

func createTestDB() *ent.Client {
	// Open an in-memory SQLite client with automatic schema migration
	client, err := ent.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
	if err != nil {
		log.Fatalf("Can't create test db client: %v", err)
	}

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed to creating schema resources: %v", err)
	}

	return client
}

func init() {
	loadEnvForTest()
	jwksMock = NewJWKSMock()
	cfg = config.GetConfig()
	logger = config.GetLogger(cfg)
	db = createTestDB()
	app = config.GetFiber(cfg, logger)
	enforcer = config.GetEnforcer()
	config.ConfigureDBClient(db)

	config.Bootstrap(config.BootstrapConfig{
		Config:   cfg,
		Logger:   logger,
		DB:       db,
		Fiber:    app,
		Enforcer: enforcer,
	})
}
