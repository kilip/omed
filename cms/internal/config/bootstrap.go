package config

import (
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
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

}
