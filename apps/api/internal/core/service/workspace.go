// Package service provides business logic and orchestration for user and workspace entities.
package service

import (
	"context"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
)

// WorkspaceRepository defines the persistence contract required by WorkspaceService.
type WorkspaceRepository interface {
	base.ServiceRepository[int64, domain.Workspace, model.CreateWorkspaceRequest, model.UpdateWorkspaceRequest, model.SearchWorkspaceRequest]
}

// WorkspaceService provides business operations and lifecycle management for workspaces.
type WorkspaceService struct {
	base.Service[int64, domain.Workspace, model.CreateWorkspaceRequest, model.UpdateWorkspaceRequest, model.SearchWorkspaceRequest]
	repo WorkspaceRepository
}

// Create makes the authenticated user the owner of the created workspace.
func (s WorkspaceService) Create(c context.Context, req model.CreateWorkspaceRequest) (domain.Workspace, error) {
	req.OwnerID = base.UserFromContext(c).ID
	return s.Service.Create(c, req)
}

// Search restricts results to workspaces the authenticated user owns or belongs to.
func (s WorkspaceService) Search(c context.Context, req model.SearchWorkspaceRequest) ([]domain.Workspace, string, error) {
	if user := base.UserFromContext(c); user.ID != 0 {
		req.MemberID = user.ID
	}
	return s.Service.Search(c, req)
}

// Delete prevents deleting a personal workspace.
func (s WorkspaceService) Delete(c context.Context, id int64) error {
	ws, err := s.repo.GetByID(c, id)
	if err != nil {
		return err
	}
	if ws.Personal {
		return domain.ErrPersonalWorkspace
	}
	return s.repo.Delete(c, id)
}

// NewWorkspaceService constructs a new WorkspaceService instance using the provided repository.
func NewWorkspaceService(repo WorkspaceRepository) WorkspaceService {
	return WorkspaceService{
		Service: base.NewService(repo),
		repo:    repo,
	}
}
