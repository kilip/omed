package main

import (
	"fmt"

	"github.com/kilip/omed/cms/internal/config"
)

func main(){

	c := config.NewConfig()
	log := config.NewLogger(c)
	db := config.NewDatabase(c, log)
	validate := config.NewValidator(c)
	app := config.NewFiber(c)

	config.Bootstrap(&config.Omed{
		Config: c,
		Log: log,
		DB: db,
		Validate: validate,
		App: app,
	})

	// start api server
	port := c.Web.Port
	err := app.Listen(fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
