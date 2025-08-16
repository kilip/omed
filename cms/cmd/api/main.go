package main

import (
	"fmt"

	"github.com/kilip/omed/cms/internal/config"
)

func main(){

	conf := config.NewConfig()
	log := config.NewLogger(conf)
	db := config.NewDatabase(conf, log)
	validate := config.NewValidator(conf)
	app := config.NewFiber(conf)

	config.Bootstrap(&config.Omed{
		Config: conf,
		Log: log,
		DB: db,
		Validate: validate,
		App: app,
	})

	// start api server
	port := conf.GetInt("web.port")
	err := app.Listen(fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
