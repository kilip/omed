package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/kilip/omed/internal/dto"
)

type UserService interface {
	Create(ctx context.Context, req dto.UserRequest) (*User, error)
	Update(ctx context.Context, req dto.UserRequest) (*User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
