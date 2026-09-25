package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/model"
)

type WorkspaceRepository struct {
	client *ent.Client
}

func NewWorkspaceRepository(client *ent.Client) WorkspaceRepository {
	return WorkspaceRepository{
		client,
	}
}

func toWorkspaceResponse(ws ent.Workspace) *model.WorkspaceResponse {
	return &model.WorkspaceResponse{
		ID:        ws.ID,
		Name:      ws.Name,
		CreatedAt: ws.CreatedAt,
		UpdatedAt: ws.UpdatedAt,
	}
}

func (r WorkspaceRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.WorkspaceResponse, error) {
	ws, err := r.client.Workspace.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return toWorkspaceResponse(*ws), nil
}

func (r WorkspaceRepository) Create(ctx context.Context, authenticated model.AuthenticatedUser) (*model.WorkspaceResponse, error) {
	ws, err := r.client.Workspace.Create().
		SetID(authenticated.WorkspaceID).
		SetName(authenticated.WorkspaceName).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return toWorkspaceResponse(*ws), nil
}
