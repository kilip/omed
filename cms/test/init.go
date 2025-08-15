package test

import (
	"github.com/go-playground/validator"
	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/cms/internal/config"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var log	*logrus.Logger
var app *fiber.App
var db *gorm.DB
var validate *validator.Validate

func init(){
	c := config.NewConfig()
	log = config.NewLogger(c)
	validate = config.NewValidator(c)
	db = config.NewDatabase(c, log)
	app = config.NewFiber(c)

	config.Bootstrap(&config.Omed{
		Config: c,
		Log: log,
		Validate: validate,
		DB: db,
		App: app,
	})
}
