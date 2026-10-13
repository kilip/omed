// Package service provides business logic and orchestration for user and workspace entities.
package service

import (
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
}

// NewWorkspaceService constructs a new WorkspaceService instance using the provided repository.
func NewWorkspaceService(repo WorkspaceRepository) WorkspaceService {
	return WorkspaceService{base.NewService(repo)}
}

