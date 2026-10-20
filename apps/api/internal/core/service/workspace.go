// Package service provides business logic and orchestration for user and workspace entities.
package service

import (
	"context"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/s3x"
)

// WorkspaceRepository defines the persistence contract required by WorkspaceService.
type WorkspaceRepository interface {
	base.ServiceRepository[int64, domain.Workspace, model.CreateWorkspaceRequest, model.UpdateWorkspaceRequest, model.SearchWorkspaceRequest]
}

// WorkspaceService provides business operations and lifecycle management for workspaces.
type WorkspaceService struct {
	base.Service[int64, domain.Workspace, model.CreateWorkspaceRequest, model.UpdateWorkspaceRequest, model.SearchWorkspaceRequest]
	repo    WorkspaceRepository
	storage s3x.Storage
}

// Create makes the authenticated user the owner of the created workspace and provisions its S3 bucket.
func (s WorkspaceService) Create(c context.Context, req model.CreateWorkspaceRequest) (domain.Workspace, error) {
	req.OwnerID = base.UserFromContext(c).ID
	ws, err := s.Service.Create(c, req)
	if err != nil {
		return ws, err
	}
	if s.storage != nil && ws.ID > 0 {
		if _, err := s.storage.EnsureWorkspaceBucket(c, ws.ID); err != nil {
			_ = s.repo.Delete(c, ws.ID)
			return domain.Workspace{}, err
		}
	}
	return ws, nil
}

// Search restricts results to workspaces the authenticated user owns or belongs to.
func (s WorkspaceService) Search(c context.Context, req model.SearchWorkspaceRequest) ([]domain.Workspace, string, error) {
	if user := base.UserFromContext(c); user.ID != 0 {
		req.MemberID = user.ID
	}
	return s.Service.Search(c, req)
}

// Delete prevents deleting a personal workspace and removes the workspace's S3 bucket upon deletion.
func (s WorkspaceService) Delete(c context.Context, id int64) error {
	ws, err := s.repo.GetByID(c, id)
	if err != nil {
		return err
	}
	if ws.Personal {
		return domain.ErrPersonalWorkspace
	}
	if err := s.repo.Delete(c, id); err != nil {
		return err
	}
	if s.storage != nil {
		return s.storage.DeleteWorkspaceBucket(c, id)
	}
	return nil
}

// NewWorkspaceService constructs a new WorkspaceService instance using the provided repository and optional S3 storage.
func NewWorkspaceService(repo WorkspaceRepository, storage ...s3x.Storage) WorkspaceService {
	var st s3x.Storage
	if len(storage) > 0 {
		st = storage[0]
	}
	return WorkspaceService{
		Service: base.NewService(repo),
		repo:    repo,
		storage: st,
	}
}
