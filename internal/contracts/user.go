package contracts

import (
	"context"

	"github.com/kilip/omed/internal/dto"
	"github.com/kilip/omed/internal/entity"
)

type UserRepository interface {
	List(ctx context.Context, req dto.UserListRequest) ([]*entity.User, error)
	FindByID(ctx context.Context, id string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	Create(ctx context.Context, user *entity.User) error
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id string) error
}

type UserService interface {
	List(ctx context.Context, req dto.UserListRequest) ([]*entity.User, error)
	Create(ctx context.Context, req dto.UserRequest) (*entity.User, error)
	Update(ctx context.Context, req dto.UserRequest) (*entity.User, error)
	Delete(ctx context.Context, id string) error
}
