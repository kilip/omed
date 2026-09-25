package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/model"
)

type UserRepository interface {
	GetByID(context.Context, uuid.UUID) (*model.UserResponse, error)
	Create(context.Context, model.AuthenticatedUser) (*model.UserResponse, error)
}

type WorkspaceRepository interface {
	GetByID(context.Context, uuid.UUID) (*model.WorkspaceResponse, error)
	Create(context.Context, model.AuthenticatedUser) (*model.WorkspaceResponse, error)
}

type WorkspaceService struct {
	userR      UserRepository
	workspaceR WorkspaceRepository
}

func NewWorkspaceService(userR UserRepository, workspaceR WorkspaceRepository) WorkspaceService {
	return WorkspaceService{
		userR,
		workspaceR,
	}
}

func (s WorkspaceService) EnsureWorkspace(ctx context.Context, authenticated model.AuthenticatedUser) error {
	// ensure user exists
	user, err := s.userR.GetByID(ctx, authenticated.ID)
	if err != nil {
		return err
	}
	if user == nil {
		_, err := s.userR.Create(ctx, authenticated)
		if err != nil {
			return err
		}
	}

	// ensure workspace exists
	ws, err := s.workspaceR.GetByID(ctx, authenticated.WorkspaceID)
	if err != nil {
		return err
	}
	if ws == nil {
		if _, err := s.workspaceR.Create(ctx, authenticated); err != nil {
			return err
		}

	}

	return nil

}
