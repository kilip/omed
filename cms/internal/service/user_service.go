package service

import (
	"context"

	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/cms/internal/entity"
	"github.com/kilip/omed/cms/internal/model"
	"github.com/kilip/omed/cms/internal/repository"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	DB *gorm.DB
	Log *logrus.Logger
	Validate *validator.Validate
	Repository *repository.UserRepository
}

func NewUserService(db *gorm.DB, log *logrus.Logger, validate *validator.Validate, userRepository *repository.UserRepository) *UserService{
	return &UserService{
		DB: db,
		Log: log,
		Validate: validate,
		Repository: userRepository,
	}
}

func (c *UserService) Register(ctx context.Context, request *model.RegisterUserRequest) (*model.UserResponse, error){
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	err := c.Validate.Struct(request)
	if err != nil {
		c.Log.Warnf("Request body invalid: %+v", err)
		return nil, fiber.ErrBadRequest
	}

	total, err := c.Repository.CountByEmail(c.DB, request.Email)
	if err != nil {
		c.Log.Warnf("Failed to count user from database: %+v", err)
		return nil, fiber.ErrInternalServerError
	}

	if total > 0 {
		c.Log.Warnf("User already exists: %+v", err)
		return nil, fiber.ErrConflict
	}

	password, err := bcrypt.GenerateFromPassword([]byte(request.PlainPassword), bcrypt.DefaultCost)
	if err != nil {
		c.Log.Warnf("Failed to hash password: %+v", err)
		return nil, fiber.ErrInternalServerError
	}

	user := &entity.User{
		Name: request.Name,
		Email: request.Email,
		Password: string(password),
	}
	
	if err := c.Repository.Create(tx, user); err != nil {
		c.Log.Warnf("Failed create user to database : %+v", err)
		return nil, fiber.ErrInternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		c.Log.Warnf("Failed commit transaction : %+v", err)
		return nil, fiber.ErrInternalServerError
	}

	return model.UserToResponse(user), nil

}

func (c *UserService) Login(ctx context.Context, request *model.LoginRequest) (*model.LoginResponse, error){
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := c.Validate.Struct(request); err != nil {
		c.Log.Warnf("Invalid request body  : %+v", err)
		return nil, fiber.ErrBadRequest
	}

	user := new(entity.User)
	if err := c.Repository.FindByEmail(tx, user, request.Email); err != nil {
		c.Log.Warnf("Failed find user by email : %+v", err)
		return nil, fiber.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)); err != nil {
		c.Log.Warnf("Failed to compare user password with bcrype hash : %+v", err)
		return nil, fiber.ErrUnauthorized
	}
	
	token, err := c.Repository.CreateToken(tx, user)
	if  err != nil {
		c.Log.Warnf("Failed create token for user : %+v", err)
		return nil, fiber.ErrInternalServerError
	}

	if err := tx.Commit().Error; err != nil {
		c.Log.Warnf("Failed commit transaction : %+v", err)
		return nil, fiber.ErrInternalServerError
	}

	return model.CreateLoginResponse(user, token), nil
}
