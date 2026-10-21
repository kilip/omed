package service

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	coredomain "github.com/kilip/omed/api/internal/core/domain"
	coremodel "github.com/kilip/omed/api/internal/core/model"
)

// FileServiceContract defines the file operations required by CMS MediaService from the core module.
type FileServiceContract interface {
	Upload(c context.Context, req coremodel.UploadFileRequest) (coredomain.File, error)
	GetByID(c context.Context, id int64) (coredomain.File, error)
	PresignDownload(c context.Context, id int64) (string, error)
	Update(c context.Context, id int64, req coremodel.UpdateFileRequest) (coredomain.File, error)
	Delete(c context.Context, id int64) error
}

// MediaRepository defines the persistence contract required by MediaService.
type MediaRepository interface {
	GetByID(c context.Context, id int64) (domain.Media, error)
	Create(c context.Context, req model.CreateMediaRecordRequest) (domain.Media, error)
	Update(c context.Context, id int64, req model.UpdateMediaRequest) (domain.Media, error)
	Delete(c context.Context, id int64) error
	Search(c context.Context, req model.SearchMediaRequest) ([]domain.Media, string, error)
	IsReferenced(c context.Context, id int64) (bool, error)
}

// MediaConfig configures size and file type restrictions for CMS media uploads.
type MediaConfig struct {
	MaxSizeMB    int64
	AllowedTypes []string
}

// MediaService manages business logic and validation for CMS media files and metadata.
type MediaService struct {
	repo    MediaRepository
	fileSvc FileServiceContract
	cfg     MediaConfig
}

// NewMediaService constructs a new MediaService backed by the given repository, file service, and configuration.
func NewMediaService(repo MediaRepository, fileSvc FileServiceContract, cfg MediaConfig) MediaService {
	return MediaService{
		repo:    repo,
		fileSvc: fileSvc,
		cfg:     cfg,
	}
}

// Upload validates file size and extension, uploads to FileService, and persists CMS media metadata.
func (s MediaService) Upload(c context.Context, req model.UploadMediaRequest) (domain.Media, error) {
	if req.Reader == nil {
		return domain.Media{}, coredomain.ErrFileRequired
	}
	if req.Size <= 0 {
		return domain.Media{}, coredomain.ErrFileEmpty
	}
	if req.Size > s.maxSizeBytes() {
		return domain.Media{}, domain.ErrMediaTooLarge
	}

	name := strings.TrimSpace(req.Name)
	if !s.isAllowedType(name) {
		return domain.Media{}, domain.ErrMediaTypeNotAllowed
	}

	file, err := s.fileSvc.Upload(c, coremodel.UploadFileRequest{
		WorkspaceID: req.WorkspaceID,
		Name:        name,
		MimeType:    req.MimeType,
		Size:        req.Size,
		EntityType:  domain.MediaEntityType,
		EntityID:    0,
		Reader:      req.Reader,
	})
	if err != nil {
		return domain.Media{}, err
	}

	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}

	created, err := s.repo.Create(c, model.CreateMediaRecordRequest{
		WorkspaceID: req.WorkspaceID,
		FileID:      file.ID,
		Title:       req.Title,
		AltText:     req.AltText,
		Caption:     req.Caption,
		IsPublic:    isPublic,
	})
	if err != nil {
		// Compensation: delete uploaded file if media row insert fails
		_ = s.fileSvc.Delete(c, file.ID)
		return domain.Media{}, err
	}

	_, _ = s.fileSvc.Update(c, file.ID, coremodel.UpdateFileRequest{
		Name:       file.Name,
		EntityType: domain.MediaEntityType,
		EntityID:   created.ID,
	})

	if created.File == nil {
		created.File = &file
	}

	url, err := s.fileSvc.PresignDownload(c, file.ID)
	if err == nil {
		created.URL = url
	}

	return created, nil
}

// GetByID retrieves a media record by ID and populates a presigned download URL.
func (s MediaService) GetByID(c context.Context, id int64) (domain.Media, error) {
	m, err := s.repo.GetByID(c, id)
	if err != nil {
		return domain.Media{}, err
	}

	url, err := s.fileSvc.PresignDownload(c, m.FileID)
	if err == nil {
		m.URL = url
	}

	return m, nil
}

// Search searches active media records and populates presigned download URLs.
func (s MediaService) Search(c context.Context, req model.SearchMediaRequest) ([]domain.Media, string, error) {
	items, cursor, err := s.repo.Search(c, req)
	if err != nil {
		return nil, "", err
	}

	for i := range items {
		url, err := s.fileSvc.PresignDownload(c, items[i].FileID)
		if err == nil {
			items[i].URL = url
		}
	}

	return items, cursor, nil
}

// Update modifies metadata fields on an active media item and populates a presigned download URL.
func (s MediaService) Update(c context.Context, id int64, req model.UpdateMediaRequest) (domain.Media, error) {
	m, err := s.repo.Update(c, id, req)
	if err != nil {
		return domain.Media{}, err
	}

	url, err := s.fileSvc.PresignDownload(c, m.FileID)
	if err == nil {
		m.URL = url
	}

	return m, nil
}

// Delete checks if the media is referenced by any post, and deletes the metadata and file if unreferenced.
func (s MediaService) Delete(c context.Context, id int64) error {
	m, err := s.repo.GetByID(c, id)
	if err != nil {
		return err
	}

	referenced, err := s.repo.IsReferenced(c, id)
	if err != nil {
		return err
	}
	if referenced {
		return domain.ErrMediaInUse
	}

	if err := s.repo.Delete(c, id); err != nil {
		return err
	}

	return s.fileSvc.Delete(c, m.FileID)
}

func (s MediaService) maxSizeBytes() int64 {
	mb := s.cfg.MaxSizeMB
	if mb <= 0 {
		mb = 10
	}
	return mb * 1024 * 1024
}

func (s MediaService) isAllowedType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return false
	}
	allowed := s.cfg.AllowedTypes
	if len(allowed) == 0 {
		allowed = []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".pdf"}
	}
	for _, a := range allowed {
		a = strings.TrimSpace(strings.ToLower(a))
		if a != "" && !strings.HasPrefix(a, ".") {
			a = "." + a
		}
		if ext == a {
			return true
		}
	}
	return false
}
