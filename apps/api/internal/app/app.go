package app

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
	"github.com/kilip/omed/api/internal/user"
	"golang.org/x/sync/errgroup"
)

type App struct {
	cfg     core.Config
	state   core.State
	modules []core.Module
}

func New(state core.State) App {
	return App{
		state: state,
		modules: []core.Module{
			user.New(),
		},
	}
}

func initBaseMiddleware(state core.State) {
	app := state.FiberApp
	app.Use(requestid.New())
}

func (a *App) Boot() error {
	initBaseMiddleware(a.state)
	initFirewall(a.state)

	var err error

	for _, m := range a.modules {
		if _, ok := m.(core.HasPreBoot); ok {
			if preBootErr := m.(core.HasPreBoot).PreBoot(a.state); preBootErr != nil {
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

// internal/app/run.go
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

		st := &core.State{
			App:    fiber.New(fiber.Config{ErrorHandler: core.ErrorHandler(logger)}),
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

func WaitForJWKS(cfg core.Config, logger *slog.Logger) {
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
