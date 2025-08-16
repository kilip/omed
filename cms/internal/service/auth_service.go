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

type AuthService struct {
	DB *gorm.DB
	Users *repository.UserRepository
	Tokens *repository.TokenRepository
	Validate *validator.Validate
	Log *logrus.Logger
}

func NewAuthService(
	db *gorm.DB,
	log *logrus.Logger,
	validate *validator.Validate,
	users *repository.UserRepository,
	tokens *repository.TokenRepository,
) *AuthService {
	return &AuthService{
		DB: db,
		Users: users,
		Tokens: tokens,
		Validate: validate,
		Log: log,
	}
}

func (c *AuthService) Login(ctx context.Context, request *model.LoginRequest) (*model.LoginResponse, error){
	tx := c.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := c.Validate.Struct(request); err != nil {
		c.Log.Warnf("Invalid request body  : %+v", err)
		return nil, fiber.ErrBadRequest
	}

	user := new(entity.User)
	if err := c.Users.FindByEmail(tx, user, request.Email); err != nil {
		c.Log.Warnf("Failed find user by email : %+v", err)
		return nil, fiber.ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)); err != nil {
		c.Log.Warnf("Failed to compare user password with bcrype hash : %+v", err)
		return nil, fiber.ErrUnauthorized
	}
	
	token, err := c.Tokens.CreateToken(tx, user)
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

func (svc *AuthService) Verify(ctx context.Context, request *model.VerifyUserRequest) (*model.AuthResponse, error){
	tx := svc.DB.WithContext(ctx).Begin()
	defer tx.Rollback()

	if err := svc.Validate.Struct(request); err != nil {
		svc.Log.Warnf("Invalid request body : %+v", err)
		return nil, fiber.ErrBadRequest
	}
	
	token := new(entity.UserToken)
	if err := svc.Tokens.FindByToken(tx, token, request.Token); err != nil {
		svc.Log.Warnf("Failed find user by token: %+v", err)
		return nil, fiber.ErrNotFound
	}

	if err := tx.Commit().Error; err != nil {
		svc.Log.Warnf("Failed commit transaction: %+v", err)
		return nil, fiber.ErrInternalServerError
	}
	
	user := new(entity.User)
	if err := svc.Users.FindById(svc.DB, user, token.UserID); err != nil {
		svc.Log.Warnf("Failed find user by id: %+v", err)
		return nil, fiber.ErrInternalServerError
	}


	
	response := &model.AuthResponse{
		UserID: token.UserID,
		Name: user.Name,
		Avatar: user.Avatar,
	}

	return response, nil
}

