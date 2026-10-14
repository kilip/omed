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

// Create enforces that non-superadmin users become the owner of the created workspace.
func (s WorkspaceService) Create(c context.Context, req model.CreateWorkspaceRequest) (domain.Workspace, error) {
	user := base.UserFromContext(c)
	if !user.HasRole(base.UserRoleAdmin) {
		req.OwnerID = user.ID
	}
	return s.Service.Create(c, req)
}

// Search restricts results for non-superadmins to workspaces they own or belong to.
func (s WorkspaceService) Search(c context.Context, req model.SearchWorkspaceRequest) ([]domain.Workspace, string, error) {
	user := base.UserFromContext(c)
	if !user.HasRole(base.UserRoleAdmin) && user.ID != 0 {
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
