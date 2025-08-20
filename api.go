package main

import (
	"fmt"

	"github.com/kilip/omed/internal/delivery/api"
	"github.com/kilip/omed/internal/delivery/api/controller"
	"github.com/kilip/omed/internal/infra/database"
	"github.com/kilip/omed/internal/infra/database/dal"
	"github.com/kilip/omed/internal/infra/database/repository"
	"github.com/kilip/omed/internal/service"
	"github.com/kilip/omed/internal/utils"
)

func main() {
	conf := utils.NewConfig()
	server := api.NewServer(conf)
	query := dal.Use(database.NewGormDB(conf))

	userR := repository.NewUserRepository(query)
	userS := service.NewUserService(userR)
	controller.NewUserController(server, userS)

	listen := fmt.Sprintf("%s:%d", conf.Api.Host, conf.Api.Port)
	server.Listen(listen)
}
