// Package cms provides content management system endpoints, domain logic, and data persistence.
package cms

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/cms/repository"
	"github.com/kilip/omed/api/internal/cms/service"
	"github.com/kilip/omed/api/internal/platform/authz"
)

// FileServiceContract defines the file operations required by CMS from the core module.
type FileServiceContract = service.FileServiceContract

// Config encapsulates configuration and dependencies for CmsModule.
type Config struct {
	FiberApp             *fiber.App
	Logger               *slog.Logger
	EntCli               *ent.Client
	FileService          FileServiceContract
	ApiURL               string
	CMSMediaMaxSizeMB    int64
	CMSMediaAllowedTypes []string
}

// CmsModule represents the cms module implementing config.Module.
type CmsModule struct {
	cfg          Config
	Authorizer   *authz.Authorizer
	MediaService service.MediaService
	PostService  service.PostService
}

// New constructs a new CmsModule instance.
func New(cfg Config) *CmsModule {
	mediaRepo := repository.NewMediaRepository(cfg.EntCli)
	mediaSvc := service.NewMediaService(mediaRepo, cfg.FileService, service.MediaConfig{
		MaxSizeMB:    cfg.CMSMediaMaxSizeMB,
		AllowedTypes: cfg.CMSMediaAllowedTypes,
	})

	postRepo := repository.NewPostRepository(cfg.EntCli)
	postSvc := service.NewPostService(postRepo, service.WithPostMediaChecker(mediaRepo))

	return &CmsModule{
		cfg:          cfg,
		MediaService: mediaSvc,
		PostService:  postSvc,
	}
}

// SetAuthorizer sets the Casbin authorizer instance.
func (m *CmsModule) SetAuthorizer(az *authz.Authorizer) {
	m.Authorizer = az
}

// PreBoot executes early initialization tasks before Boot.
func (m *CmsModule) PreBoot() error {
	return nil
}

// Boot registers routes for the cms module.
func (m *CmsModule) Boot() error {
	initRoutes(*m, m.cfg.FiberApp, m.Authorizer)
	return nil
}
