package app

import (
	"log"

	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/platform/jwtx"
)

func initFirewall(state core.State) {
	app := state.FiberApp
	cfg := state.Config

	handler, err := jwtx.Handler(cfg.JWKSUrl, state.Logger)
	if err != nil {
		log.Fatalf("Can't init firewall %+v", err)
	}
	app.Use(handler)
	app.Use(jwtx.UserInjector(cfg.AuthUrl, cfg.AuthUrl))
}
