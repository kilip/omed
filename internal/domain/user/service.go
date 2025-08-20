package user

import (
	"context"

	"github.com/kilip/omed/internal/dto"
)

type UserService interface {
	List(ctx context.Context, req dto.UserListRequest) ([]*User, error)
	Create(ctx context.Context, req dto.UserRequest) (*User, error)
	Update(ctx context.Context, req dto.UserRequest) (*User, error)
	Delete(ctx context.Context, id string) error
}
