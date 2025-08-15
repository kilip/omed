package config

import (
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/cms/internal/delivery/http"
	"github.com/kilip/omed/cms/internal/repository"
	"github.com/kilip/omed/cms/internal/service"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Omed struct {
	DB *gorm.DB
	App *fiber.App
	Log *logrus.Logger
	Config *OmedConfig
	Validate *validator.Validate
}

func Bootstrap(omed *Omed){
	userRepository := repository.NewUserRepository(omed.Log)
	userService := service.NewUserService(omed.DB, omed.Log, omed.Validate, userRepository)

	userController := http.NewUserController(userService, omed.Log)

	routeConfig := http.RouteConfig{
		App: omed.App,
		UserController: userController,
	}

	routeConfig.Setup()
}
