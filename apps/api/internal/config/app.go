// Package config orchestrates application initialization, module lifecycle, Fiber HTTP configuration,
// authentication firewall setup, and graceful server shutdowns.
package config

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/core/service"
	"github.com/kilip/omed/api/internal/fin"
	"github.com/kilip/omed/api/internal/platform/authz"
	"golang.org/x/sync/errgroup"
)

// Omed manages application modules and the HTTP server lifecycle.
type Omed struct {
	cfg            Config
	state          State
	modules        []Module
	sessionService service.SessionService
}

// New creates a new Omed instance populated with runtime state and registered modules.
func New(state State) Omed {
	coreMod := core.New(core.Config{
		FiberApp:          state.FiberApp,
		Logger:            state.Logger,
		EntCli:            state.EntCli,
		AuthSecret:        state.Config.AuthSecret,
		ApiURL:            state.Config.ApiUrl,
		TrustedOrigins:    state.Config.TrustedOrigins,
		GoogleID:          state.Config.GoogleClientID,
		GoogleSecret:      state.Config.GoogleClientSecret,
		GithubID:          state.Config.GithubClientID,
		GithubSecret:      state.Config.GithubClientSecret,
		SessionCookieName: state.Config.SessionCookieName,
	})

	az, err := authz.New(coreMod.MemberService)
	if err != nil {
		log.Fatalf("failed to initialize authz: %v", err)
	}
	coreMod.SetAuthorizer(az)
	state.Authz = az

	finMod := fin.New(fin.Config{
		FiberApp: state.FiberApp,
		Logger:   state.Logger,
		EntCli:   state.EntCli,
	})
	finMod.SetAuthorizer(az)

	return Omed{
		state: state,
		modules: []Module{
			coreMod,
			finMod,
		},
		sessionService: coreMod.SessionService,
	}
}

func initBaseMiddleware(state State) {
	app := state.FiberApp
	app.Use(requestid.New())
}

// Boot initializes middlewares, sets up authentication firewalls, and triggers
// PreBoot and Boot hooks across all registered modules.
func (a *Omed) Boot() error {
	initBaseMiddleware(a.state)
	initFirewall(a.state, a.sessionService)

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
func (a *Omed) Run(ctx context.Context) error {
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
