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
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/user"
	"golang.org/x/sync/errgroup"
)

// App manages application modules and the HTTP server lifecycle.
type App struct {
	cfg     base.Config
	state   base.State
	modules []base.Module
}

// New creates a new App instance populated with runtime state and registered modules.
func New(state base.State) App {
	return App{
		state: state,
		modules: []base.Module{
			user.New(),
		},
	}
}

func initBaseMiddleware(state base.State) {
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
		if _, ok := m.(base.HasPreBoot); ok {
			if preBootErr := m.(base.HasPreBoot).PreBoot(a.state); preBootErr != nil {
				err = errors.Join(preBootErr, err)
			}
		}
	}

	for _, m := range a.modules {
		if bootErr := m.Boot(a.state); bootErr != nil {
			return errors.Join(bootErr, err)
		}
	}

	return nil
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
	/*
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

		db, err := openDB(ctx, a.cfg.DBURL) // ent client
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		defer db.Close()

		st := &base.State{
			App:    fiber.New(fiber.Config{ErrorHandler: base.ErrorHandler(logger)}),
			Logger: logger,
			DB:     db,
		}

		for _, m := range a.modules {
			if err := m.Register(st); err != nil {
				return fmt.Errorf("register %T: %w", m, err)
			}
		}

		g, ctx := errgroup.WithContext(ctx)

		// HTTP server
		g.Go(func() error {
			return st.App.Listen(a.cfg.APIAddr)
		})

		// Shutdown: nunggu signal / error dari goroutine lain
		g.Go(func() error {
			<-ctx.Done()
			sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			return st.App.ShutdownWithContext(sctx)
		})

		// Worker (Kafka dll) nanti tinggal tambah g.Go di sini,
		// atau loop dari module yang implement interface Runner.

		return g.Wait()
	*/
}

// WaitForJWKS blocks until the JWKS endpoint is reachable and returns HTTP 200,
// or until an OS termination signal is received.
func WaitForJWKS(cfg base.Config, logger *slog.Logger) {
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
