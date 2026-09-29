package config

import (
	"github.com/kilip/omed/finance/internal/http/controller"
	"github.com/kilip/omed/finance/internal/repository"
	"github.com/kilip/omed/finance/internal/service"
)

func bootControllers(state State) {
	accountR := repository.NewAccountRepository(state.EntClient, state.Log)
	accountS := service.NewAccountService(accountR, state.Log)
	accCtl := controller.NewAccountController(accountS)
	accCtl.Register(state.Api.Group("/"))

	ledgerPeriodR := repository.NewLedgerPeriodRepository(state.EntClient, state.Log)
	ledgerPeriodS := service.NewLedgerPeriodService(ledgerPeriodR, state.Log)
	lpCtl := controller.NewLedgerPeriodController(ledgerPeriodS)
	lpCtl.Register(state.Api.Group("/"))
}
