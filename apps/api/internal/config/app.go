// Package config orchestrates application initialization, module lifecycle, Fiber HTTP configuration,
// authentication firewall setup, and graceful server shutdowns.
package config

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/platform/authz"
	"golang.org/x/sync/errgroup"
)

// App manages application modules and the HTTP server lifecycle.
type App struct {
	cfg     Config
	state   State
	modules []Module
}

// New creates a new App instance populated with runtime state and registered modules.
func New(state State) App {
	coreMod := core.New(core.Config{
		FiberApp: state.FiberApp,
		Logger:   state.Logger,
		EntCli:   state.EntCli,
	})

	az, err := authz.New(coreMod.MemberService)
	if err != nil {
		log.Fatalf("failed to initialize authz: %v", err)
	}
	coreMod.SetAuthorizer(az)
	state.Authz = az

	return App{
		state: state,
		modules: []Module{
			coreMod,
		},
	}
}

func initBaseMiddleware(state State) {
	app := state.FiberApp
	app.Use(requestid.New())
}

// Boot initializes middlewares, sets up authentication firewalls, and triggers
// PreBoot and Boot hooks across all registered modules.
func (a *App) Boot() error {
	initBaseMiddleware(a.state)
	initFirewall(a.state)

	var err error

	for _, m := range a.modules {
		if _, ok := m.(HasPreBoot); ok {
			if preBootErr := m.(HasPreBoot).PreBoot(); preBootErr != nil {
				err = errors.Join(preBootErr, err)
			}
		}
	}

	for _, m := range a.modules {
		if bootErr := m.Boot(); bootErr != nil {
			return errors.Join(bootErr, err)
		}
	}

	return err
}

// Run boots the application and starts the HTTP server listening on port 8002.
// It listens for context cancellation to perform graceful shutdown.
func (a *App) Run(ctx context.Context) error {
	a.Boot()

	g, ctx := errgroup.WithContext(ctx)

	// HTTP server
	g.Go(func() error {
		return a.state.FiberApp.Listen(":8002")
	})

	// Shutdown: nunggu signal / error dari goroutine lain
	g.Go(func() error {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return a.state.FiberApp.ShutdownWithContext(sctx)
	})

	return g.Wait()
}

// WaitForJWKS blocks until the JWKS endpoint is reachable and returns HTTP 200,
// or until an OS termination signal is received.
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
