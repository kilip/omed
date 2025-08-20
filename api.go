package main

import (
	"github.com/kilip/omed/internal/delivery/http"
	"github.com/kilip/omed/internal/utils"
)

func main() {
	conf := utils.NewConfig()
	server := http.NewServer(conf)

	if err := server.Start(); err != nil {
		panic(err)
	}
}
