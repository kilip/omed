package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/s3x"
)

// FileRepository defines the persistence contract required by FileService.
type FileRepository interface {
	GetByID(c context.Context, id int64) (domain.File, error)
	Search(c context.Context, req model.SearchFileRequest) ([]domain.File, string, error)
	Create(c context.Context, req model.CreateFileRecordRequest) (domain.File, error)
	Update(c context.Context, id int64, req model.UpdateFileRequest) (domain.File, error)
	Delete(c context.Context, id int64) error
}

// FileService orchestrates workspace file uploads, downloads, presigned URLs, and lifecycle in S3 and PostgreSQL.
type FileService struct {
	repo    FileRepository
	storage s3x.Storage
}

// NewFileService constructs a new FileService backed by the given FileRepository and s3x.Storage.
func NewFileService(repo FileRepository, storage s3x.Storage) FileService {
	return FileService{
		repo:    repo,
		storage: storage,
	}
}

// Upload ensures the workspace S3 bucket exists, uploads the file stream, and records file metadata in the database.
func (s FileService) Upload(c context.Context, req model.UploadFileRequest) (domain.File, error) {
	if req.Reader == nil {
		return domain.File{}, domain.ErrFileRequired
	}
	if req.Size <= 0 {
		return domain.File{}, domain.ErrFileEmpty
	}

	wsID := req.WorkspaceID
	if wsID <= 0 {
		wsID = base.WorkspaceIDFromContext(c)
	}
	if wsID <= 0 {
		return domain.File{}, base.ErrInvalidID
	}

	c = base.WithWorkspaceID(c, wsID)

	bucket, err := s.storage.EnsureWorkspaceBucket(c, wsID)
	if err != nil {
		return domain.File{}, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "unnamed"
	}

	key := generateObjectKey(name, req.EntityType)
	mimeType := strings.TrimSpace(req.MimeType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	info, err := s.storage.PutObject(c, bucket, key, req.Reader, req.Size, mimeType)
	if err != nil {
		return domain.File{}, err
	}

	created, err := s.repo.Create(c, model.CreateFileRecordRequest{
		WorkspaceID: wsID,
		Name:        name,
		Bucket:      bucket,
		Key:         key,
		MimeType:    info.ContentType,
		Size:        info.Size,
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
	})
	if err != nil {
		_ = s.storage.DeleteObject(c, bucket, key)
		return domain.File{}, err
	}

	return created, nil
}

// GetByID retrieves file metadata by ID within the current workspace.
func (s FileService) GetByID(c context.Context, id int64) (domain.File, error) {
	return s.repo.GetByID(c, id)
}

// Search searches active files within the current workspace with cursor pagination.
func (s FileService) Search(c context.Context, req model.SearchFileRequest) ([]domain.File, string, error) {
	return s.repo.Search(c, req)
}

// Download retrieves file metadata and opens a download stream from the workspace's S3 bucket.
func (s FileService) Download(c context.Context, id int64) (domain.File, *s3x.Object, error) {
	f, err := s.repo.GetByID(c, id)
	if err != nil {
		return domain.File{}, nil, err
	}

	obj, err := s.storage.GetObject(c, f.Bucket, f.Key)
	if err != nil {
		if errors.Is(err, s3x.ErrObjectNotFound) || errors.Is(err, s3x.ErrBucketNotFound) {
			return domain.File{}, nil, base.ErrNotFound
		}
		return domain.File{}, nil, err
	}

	return f, obj, nil
}

// PresignDownload generates a temporary presigned GET URL for downloading the file directly from S3.
func (s FileService) PresignDownload(c context.Context, id int64) (string, error) {
	f, err := s.repo.GetByID(c, id)
	if err != nil {
		return "", err
	}

	url, err := s.storage.PresignGetObject(c, f.Bucket, f.Key, 0)
	if err != nil {
		if errors.Is(err, s3x.ErrObjectNotFound) || errors.Is(err, s3x.ErrBucketNotFound) {
			return "", base.ErrNotFound
		}
		return "", err
	}

	return url, nil
}

// Update modifies file metadata (such as display name or linked entity).
func (s FileService) Update(c context.Context, id int64, req model.UpdateFileRequest) (domain.File, error) {
	return s.repo.Update(c, id, req)
}

// Delete removes the underlying S3 object and soft-deletes the database metadata record.
func (s FileService) Delete(c context.Context, id int64) error {
	f, err := s.repo.GetByID(c, id)
	if err != nil {
		return err
	}

	if err := s.storage.DeleteObject(c, f.Bucket, f.Key); err != nil {
		return err
	}

	return s.repo.Delete(c, id)
}

func generateObjectKey(filename, entityType string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	id, err := uuid.NewV7()
	if err != nil {
		id = uuid.New()
	}

	prefix := "files"
	if cleaned := sanitizeEntityPrefix(entityType); cleaned != "" {
		prefix = cleaned
	}

	return prefix + "/" + id.String() + ext
}

func sanitizeEntityPrefix(entityType string) string {
	entityType = strings.TrimSpace(strings.ToLower(entityType))
	if entityType == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range entityType {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_':
			b.WriteRune(r)
		case r == ':' || r == '/':
			b.WriteRune('/')
		}
	}
	return strings.Trim(b.String(), "/")
}
