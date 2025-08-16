package config

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/cms/internal/delivery/http/controller"
	"github.com/kilip/omed/cms/internal/delivery/http/middleware"
	"github.com/kilip/omed/cms/internal/delivery/http/route"
	"github.com/kilip/omed/cms/internal/repository"
	"github.com/kilip/omed/cms/internal/service"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type Omed struct {
	DB *gorm.DB
	App *fiber.App
	Log *logrus.Logger
	Config *viper.Viper
	Validate *validator.Validate
}

func Bootstrap(omed *Omed){
	users := repository.NewUserRepository(omed.Log)
	tokens := repository.NewTokenRepository(omed.Log)
	userService := service.NewUserService(omed.DB, omed.Log, omed.Validate, users)
	authService := service.NewAuthService(omed.DB, omed.Log, omed.Validate, users, tokens)
	
	userController := controller.NewUserController(userService, omed.Log)
	authController := controller.NewAuthController(authService, omed.Log)
	authMiddleware := middleware.AuthMiddleware(authService)

	routeConfig := route.RouteConfig{
		App: omed.App,
		UserController: userController,
		AuthController: authController,
		AuthMiddleware: authMiddleware,
	}

	routeConfig.Setup()
}
