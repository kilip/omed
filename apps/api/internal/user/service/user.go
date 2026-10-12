package service

import (
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/user/model"
	"github.com/kilip/omed/api/internal/user/repository"
)

type UserService struct {
	core.Service[int64, model.UserResponse, model.CreateUserRequest, model.UpdateUserRequest, model.SearchRequest]
}

func NewUserService(repo repository.UserRepository) UserService {
	svc := UserService{}
	svc.SetRepository(repo)
	return svc
}
