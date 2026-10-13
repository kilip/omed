package service

import (
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/user/domain"
	"github.com/kilip/omed/api/internal/user/model"
	"github.com/kilip/omed/api/internal/user/repository"
)

type UserService struct {
	core.Service[int64, domain.User, model.CreateUserRequest, model.UpdateUserRequest, model.SearchRequest]
}

func NewUserService(repo repository.UserRepository) UserService {
	return UserService{core.NewService(repo)}
}
