package http

import "github.com/gofiber/fiber/v2"

type RouteConfig struct {
	App *fiber.App
	UserController *UserController
}

func (c RouteConfig) Setup(){
	c.SetupUserRoutes()
}

func (c RouteConfig) SetupUserRoutes(){
	c.App.Post("/users", c.UserController.Register)
}