// Package core provides user and workspace management endpoints, domain logic, and data persistence.
package core

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/core/repository"
	"github.com/kilip/omed/api/internal/core/service"
	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/kilip/omed/api/internal/platform/mailx"
	"github.com/kilip/omed/api/internal/platform/s3x"
)

type Config struct {
	FiberApp          *fiber.App
	Logger            *slog.Logger
	EntCli            *ent.Client
	AuthSecret        string
	ApiURL            string
	WebURL            string
	TrustedOrigins    []string
	GoogleID          string
	GoogleSecret      string
	GithubID          string
	GithubSecret      string
	SessionTTL        time.Duration
	SessionCookieName string
	Mailer            mailx.Mailer
	Storage           s3x.Storage
	Env               string
	ApiUrl            string
}

// CoreModule represents the core module implementing base.Module.
type CoreModule struct {
	cfg Config
	// UserService handles user business logic.
	UserService service.UserService
	// WorkspaceService handles workspace business logic.
	WorkspaceService service.WorkspaceService
	// MemberService handles workspace member business logic.
	MemberService service.MemberService
	// InvitationService handles workspace invitation business logic.
	InvitationService service.InvitationService
	// FileService handles workspace file storage business logic.
	FileService service.FileService
	// SessionService handles session business logic.
	SessionService service.SessionService
	// OAuthService handles third-party OAuth2 authentication.
	OAuthService service.OAuthService
	// Authorizer handles workspace-scoped authorization checks.
	Authorizer *authz.Authorizer
}

// UserModule is an alias for CoreModule.
type UserModule = CoreModule

// New constructs a new CoreModule instance.
func New(cfg Config) *CoreModule {
	if cfg.ApiUrl == "" && cfg.ApiURL != "" {
		cfg.ApiUrl = cfg.ApiURL
	} else if cfg.ApiURL == "" && cfg.ApiUrl != "" {
		cfg.ApiURL = cfg.ApiUrl
	}
	if cfg.Storage == nil {
		cfg.Storage, _ = s3x.New(s3x.Config{Driver: "memory", Logger: cfg.Logger})
	}

	users := repository.NewUserRepository(cfg.EntCli, cfg.Storage)
	workspaces := repository.NewWorkspaceRepository(cfg.EntCli)
	members := repository.NewMemberRepository(cfg.EntCli)
	invitations := repository.NewInvitationRepository(cfg.EntCli)
	files := repository.NewFileRepository(cfg.EntCli)
	accounts := repository.NewAccountRepository(cfg.EntCli)
	sessions := repository.NewSessionRepository(cfg.EntCli)

	userSvc := service.NewUserService(users)
	workspaceSvc := service.NewWorkspaceService(workspaces, cfg.Storage)
	memberSvc := service.NewMemberService(members, workspaceSvc, userSvc)
	invitationSvc := service.NewInvitationService(invitations, workspaceSvc, members, userSvc, cfg.Mailer, cfg.WebURL)
	fileSvc := service.NewFileService(files, cfg.Storage)
	sessionSvc := service.NewSessionService(sessions, users, cfg.SessionTTL)
	oauthSvc := service.NewOAuthService(service.OAuthServiceParams{
		Accounts:       accounts,
		Users:          users,
		Sessions:       sessionSvc,
		AuthSecret:     cfg.AuthSecret,
		ApiURL:         cfg.ApiURL,
		TrustedOrigins: cfg.TrustedOrigins,
		GoogleID:       cfg.GoogleID,
		GoogleSecret:   cfg.GoogleSecret,
		GithubID:       cfg.GithubID,
		GithubSecret:   cfg.GithubSecret,
	})

	mod := &CoreModule{
		cfg:               cfg,
		UserService:       userSvc,
		WorkspaceService:  workspaceSvc,
		MemberService:     memberSvc,
		InvitationService: invitationSvc,
		FileService:       fileSvc,
		SessionService:    sessionSvc,
		OAuthService:      oauthSvc,
	}

	InitEntCli(cfg.EntCli)
	return mod
}

// SetAuthorizer configures the authorizer used by the core module.
func (u *CoreModule) SetAuthorizer(az *authz.Authorizer) {
	u.Authorizer = az
	u.MemberService.SetAuthorizer(az)
	u.InvitationService.SetAuthorizer(az)
}

// PreBoot executes pre-initialization tasks for the core module.
func (u *CoreModule) PreBoot() error {
	return nil
}

// Boot initializes repositories, services, and registers routes for the core module.
func (u *CoreModule) Boot() error {
	initRoutes(*u, u.cfg.FiberApp, u.Authorizer)
	return nil
}
