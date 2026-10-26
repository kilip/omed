// Package config orchestrates application initialization, module lifecycle, Fiber HTTP configuration,
// authentication firewall setup, and graceful server shutdowns.
package config

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/cms"
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/core/service"
	"github.com/kilip/omed/api/internal/fin"
	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/kilip/omed/api/internal/platform/mailx"
	"github.com/kilip/omed/api/internal/platform/s3x"
	"github.com/kilip/omed/api/internal/testctl"
	"golang.org/x/sync/errgroup"
)

// Omed manages application modules and the HTTP server lifecycle.
type Omed struct {
	cfg            Config
	state          State
	modules        []Module
	sessionService service.SessionService
}

// State returns the application runtime state.
func (a Omed) State() State {
	return a.state
}

// New creates a new Omed instance populated with runtime state and registered modules.
func New(state State) Omed {
	if state.Mailer == nil {
		mailer, err := mailx.New(mailx.Config{
			Driver:      state.Config.MailerDriver,
			FromAddress: state.Config.MailerFromAddress,
			FromName:    state.Config.MailerFromName,
			SMTP: mailx.SMTPConfig{
				Host:        state.Config.MailerHost,
				Port:        state.Config.MailerPort,
				Username:    state.Config.MailerUsername,
				Password:    state.Config.MailerPassword,
				InsecureTLS: state.Config.MailerInsecureTLS,
			},
			Logger: state.Logger,
		})
		if err != nil {
			log.Fatalf("failed to initialize mailer: %v", err)
		}
		state.Mailer = mailer
	}

	if state.Storage == nil {
		storage, err := s3x.New(s3x.Config{
			Driver:       state.Config.S3Driver,
			BucketPrefix: state.Config.S3BucketPrefix,
			S3: s3x.S3Config{
				Endpoint:       state.Config.S3Endpoint,
				AccessKey:      state.Config.S3AccessKey,
				SecretKey:      state.Config.S3SecretKey,
				Region:         state.Config.S3Region,
				ForcePathStyle: state.Config.S3ForcePathStyle,
			},
			Logger: state.Logger,
		})
		if err != nil {
			log.Fatalf("failed to initialize s3 storage: %v", err)
		}
		state.Storage = storage
	}

	coreMod := core.New(core.Config{
		FiberApp:            state.FiberApp,
		Logger:              state.Logger,
		EntCli:              state.EntCli,
		AuthSecret:          state.Config.AuthSecret,
		ApiURL:              state.Config.ApiUrl,
		ApiUrl:              state.Config.ApiUrl,
		Env:                 state.Config.Env,
		WebURL:              state.Config.WebUrl,
		TrustedOrigins:      state.Config.TrustedOrigins,
		GoogleID:            state.Config.GoogleClientID,
		GoogleSecret:        state.Config.GoogleClientSecret,
		GithubID:            state.Config.GithubClientID,
		GithubSecret:        state.Config.GithubClientSecret,
		SessionCookieName:   state.Config.SessionCookieName,
		Mailer:              state.Mailer,
		Storage:             state.Storage,
		IsTestRoutesEnabled: state.Config.IsTestRoutesEnabled(),
	})

	az, err := authz.New(coreMod.MemberService)
	if err != nil {
		log.Fatalf("failed to initialize authz: %v", err)
	}
	coreMod.SetAuthorizer(az)
	state.Authz = az

	finMod := fin.New(fin.Config{
		FiberApp:               state.FiberApp,
		Logger:                 state.Logger,
		EntCli:                 state.EntCli,
		FileService:            coreMod.FileService,
		AttachmentMaxSizeMB:    state.Config.AttachmentMaxSizeMB,
		AttachmentAllowedTypes: state.Config.AttachmentAllowedTypes,
	})
	finMod.SetAuthorizer(az)

	cmsMod := cms.New(cms.Config{
		FiberApp:             state.FiberApp,
		Logger:               state.Logger,
		EntCli:               state.EntCli,
		FileService:          coreMod.FileService,
		ApiURL:               state.Config.ApiUrl,
		CMSMediaMaxSizeMB:    state.Config.CMSMediaMaxSizeMB,
		CMSMediaAllowedTypes: state.Config.CMSMediaAllowedTypes,
	})
	cmsMod.SetAuthorizer(az)

	return Omed{
		state: state,
		modules: []Module{
			coreMod,
			finMod,
			cmsMod,
		},
		sessionService: coreMod.SessionService,
	}
}

// PingDB verifies connectivity to the database using the Ent client.
func PingDB(ctx context.Context, client *ent.Client) error {
	if client == nil {
		return errors.New("database client is not initialized")
	}
	_, err := client.User.Query().Limit(1).Count(ctx)
	return err
}

// RegisterHealthRoutes mounts the /healthz and /readyz probe routes.
func RegisterHealthRoutes(r fiber.Router, entCli *ent.Client, logger *slog.Logger) {
	r.Get("/healthz", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
		})
	})

	r.Get("/readyz", func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		if err := PingDB(ctx, entCli); err != nil {
			if logger != nil {
				logger.WarnContext(c, "readiness probe failed", "error", err.Error())
			}
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "unavailable",
				"error":  err.Error(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
		})
	})
}

func initBaseMiddleware(state State) {
	app := state.FiberApp
	app.Use(requestid.New())
}

// Boot initializes middlewares, sets up authentication firewalls, and triggers
// PreBoot and Boot hooks across all registered modules.
func (a *Omed) Boot() error {
	initBaseMiddleware(a.state)
	RegisterHealthRoutes(a.state.FiberApp, a.state.EntCli, a.state.Logger)
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

	if a.state.Config.IsTestRoutesEnabled() && a.state.EntCli != nil {
		if a.state.Logger != nil {
			a.state.Logger.Warn("⚠️ WARNING: Test endpoints enabled (/test/*). DO NOT ENABLE IN PRODUCTION!")
		}
		ctl := testctl.NewController(a.state.EntCli, a.state.Mailer, a.state.Config.SessionCookieName)
		ctl.RegisterRoutes(a.state.FiberApp)
	}

	return err
}

// Run boots the application and starts the HTTP server listening on the configured port.
// It listens for context cancellation to perform graceful shutdown.
func (a *Omed) Run(ctx context.Context) error {
	if err := a.Boot(); err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)

	port := a.state.Config.Port
	if port <= 0 {
		port = 8001
	}
	addr := fmt.Sprintf(":%d", port)

	// HTTP server
	g.Go(func() error {
		return a.state.FiberApp.Listen(addr, fiber.ListenConfig{
			ListenerNetwork: "tcp",
		})
	})

	// Graceful shutdown: wait for context cancellation or error from another goroutine
	g.Go(func() error {
		<-ctx.Done()
		sctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 1. Shutdown Fiber first with context timeout
		err := a.state.FiberApp.ShutdownWithContext(sctx)

		// 2. Then close mailer, storage, and Ent database client
		if a.state.Mailer != nil {
			_ = a.state.Mailer.Close()
		}
		if a.state.Storage != nil {
			_ = a.state.Storage.Close()
		}
		if a.state.EntCli != nil {
			_ = a.state.EntCli.Close()
		}
		return err
	})

	return g.Wait()
}
