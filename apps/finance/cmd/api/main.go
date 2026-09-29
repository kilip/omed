package main

import (
	"github.com/gofiber/fiber/v3"
)

func main() {
	app := fiber.New()

	fail := app.Listen(":9002", fiber.ListenConfig{
		EnablePrefork: true,
	})
	if fail != nil {
		panic(fail)
	}
}
