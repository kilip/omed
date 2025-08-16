package test

import (
	"os"

	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/cms/internal/config"
	"github.com/kilip/omed/cms/internal/entity"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

var conf *viper.Viper
var log	*logrus.Logger
var app *fiber.App
var db *gorm.DB
var validate *validator.Validate

func init(){
	os.Setenv("OMED_APP_ENV", "test")
	conf = config.NewConfig()
	log = config.NewLogger(conf)
	validate = config.NewValidator(conf)
	db = config.NewDatabase(conf, log)
	app = config.NewFiber(conf)

	db.AutoMigrate(&entity.User{}, &entity.UserToken{})

	config.Bootstrap(&config.Omed{
		Config: conf,
		Log: log,
		Validate: validate,
		DB: db,
		App: app,
	})
}
