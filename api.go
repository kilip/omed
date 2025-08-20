package main

import (
	"github.com/kilip/omed/internal/delivery/api"
	"github.com/kilip/omed/internal/delivery/api/controller"
	"github.com/kilip/omed/internal/utils"
)

func main() {
	conf := utils.NewConfig()
	server := api.NewServer(conf)

	controller.NewUserController(server)

	server.Start()

}
