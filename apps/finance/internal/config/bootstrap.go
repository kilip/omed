package config

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/healthcheck"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/http/middleware"
	"github.com/kilip/omed/finance/internal/repository"
	"github.com/kilip/omed/finance/internal/service"
	slogfiber "github.com/samber/slog-fiber"
)

type State struct {
	FiberApp  *fiber.App
	Config    Config
	Log       *slog.Logger
	EntClient *ent.Client
}

func initAuth(state State) {
	cfg := state.Config
	state.FiberApp.Use(middleware.UserInjector(cfg.AuthBaseUrl, cfg.AuthBaseUrl, cfg.AuthBaseUrl))

	users := repository.NewUserRepository(state.EntClient, state.Log)
	works := repository.NewWorkspaceRepository(state.EntClient, state.Log)
	userSnapshotService := service.NewUserSnapshotService(users, works, state.Log)
	state.FiberApp.Use(middleware.UserSnapshotMiddleware(userSnapshotService))
}

func initFiber(state State) error {

	state.FiberApp.Use(slogfiber.New(state.Log))

	// init healthcheck
	// Use the default probe on the conventional endpoints
	state.FiberApp.Get(healthcheck.LivenessEndpoint, healthcheck.New(healthcheck.Config{
		ResponseFormat: healthcheck.FormatJSON,
	}))

	// cors config
	state.FiberApp.Use(cors.New(cors.Config{
		AllowOrigins: state.Config.TrustedOrigins,
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
	}))

	// jwks config
	jwks, err := keyfunc.NewDefaultCtx(context.Background(), []string{state.Config.JWKSUrl})
	if err != nil {
		return fmt.Errorf("fetch JWKS from %s: %w", state.Config.JWKSUrl, err)
	}
	state.FiberApp.Use(jwtware.New(jwtware.Config{
		Next: func(c fiber.Ctx) bool {
			return c.Method() == fiber.MethodOptions
		},
		KeyFunc:   jwks.KeyfuncCtx(context.Background()),
		Extractor: extractors.FromAuthHeader("Bearer"),
		ErrorHandler: func(c fiber.Ctx, err error) error {
			state.Log.WarnContext(c, "JWKS Error", "error", err.Error())
			return fiber.NewError(fiber.StatusUnauthorized, err.Error())
		},
	}))

	return nil
}

func Bootstrap(state State) {
	initFiber(state)
	initAuth(state)
	loadController(state)
}

func WaitForJWKS(cfg Config, logger *slog.Logger) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := checkJWKS(ctx, cfg.JWKSUrl, logger); err != nil {
		log.Fatalf("auth not ready: %v", err)
	}
}

func checkJWKS(ctx context.Context, url string, log *slog.Logger) error {
	client := &http.Client{Timeout: 3 * time.Second}
	for i := 0; ; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err == nil {
			ok := resp.StatusCode == http.StatusOK
			resp.Body.Close()
			if ok {
				return nil
			}
			err = fmt.Errorf("status %d", resp.StatusCode)
		}
		wait := time.Duration(1<<min(i, 4)) * time.Second
		log.Warn("waiting for auth JWKS", "url", url, "in", wait, "error", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}
