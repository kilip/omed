package service

import (
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/user/domain"
	"github.com/kilip/omed/api/internal/user/model"
)

//go:generate go run go.uber.org/mock/mockgen -source=user.go -destination=mock/mock_user.go -package=mock

type UserRepository interface {
	core.ServiceRepository[int64, domain.User, model.CreateUserRequest, model.UpdateUserRequest, model.SearchRequest]
}

type UserService struct {
	core.Service[int64, domain.User, model.CreateUserRequest, model.UpdateUserRequest, model.SearchRequest]
}

func NewUserService(repo UserRepository) UserService {
	return UserService{core.NewService(repo)}
}
