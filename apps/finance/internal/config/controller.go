package config

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/http/controller"
	"github.com/kilip/omed/finance/internal/repository"
	"github.com/kilip/omed/finance/internal/service"
)

type Controller interface {
	InitRoutes(r fiber.Router)
}

func loadController(st State) {
	router := st.FiberApp.Group("/")

	accountR := repository.NewAccountRepository(st.EntClient, st.Log)
	entryR := repository.NewEntryRepository(st.EntClient, st.Log)
	periodR := repository.NewPeriodRepository(st.EntClient, st.Log)

	accountS := service.NewAccountService(accountR, entryR, st.Log)
	accountCtl := controller.NewAccountController(accountS)
	accountCtl.InitRoutes(router)

	periodS := service.NewPeriodService(periodR, st.Log)
	periodCtl := controller.NewPeriodController(periodS)
	periodCtl.InitRoutes(router)
}

