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
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/http/middleware"
	"github.com/kilip/omed/finance/internal/repository"
	"github.com/kilip/omed/finance/internal/service"
)

// State the state of application
type State struct {
	Config    Config
	Api       *fiber.App
	Log       *slog.Logger
	EntClient *ent.Client
}

func configureMiddleware(state State) error {
	jwks, err := keyfunc.NewDefaultCtx(context.Background(), []string{state.Config.JWKSUrl})
	if err != nil {
		return fmt.Errorf("fetch JWKS from %s: %w", state.Config.JWKSUrl, err)
	}

	state.Api.Use(jwtware.New(jwtware.Config{
		KeyFunc:   jwks.KeyfuncCtx(context.Background()),
		Extractor: extractors.FromAuthHeader("Bearer"),
		ErrorHandler: func(c fiber.Ctx, err error) error {
			state.Log.WarnContext(c, "JWKS Error", "error", err.Error())
			return err
		},
	}))
	state.Api.Use(middleware.UserInjector(state.Config.AuthBaseUrl))

	users := repository.NewUserRepository(state.EntClient, state.Log)
	works := repository.NewWorkspaceRepository(state.EntClient, state.Log)
	authSvc := service.NewAuthService(users, works, state.Log)
	state.Api.Use(middleware.AuthSnapshotMiddleware(authSvc))

	return nil
}

func WaitForJWKS(state State) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := checkJWKS(ctx, state.Config.JWKSUrl, state.Log); err != nil {
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

func Bootstrap(state State) {
	configureMiddleware(state)
	bootControllers(state)
}
