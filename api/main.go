package main

import (
	"github.com/joho/godotenv"
	"github.com/kilip/omed/api/core"
)

func main(){
	godotenv.Load()

	app := core.Bootstrap()

	app.Listen("0.0.0.0:3001")
}

