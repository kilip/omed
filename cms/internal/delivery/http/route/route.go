package route

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/cms/internal/delivery/http/controller"
)

type RouteConfig struct {
	App *fiber.App
	UserController *controller.UserController
	AuthController *controller.AuthController
	AuthMiddleware fiber.Handler
}

func (c RouteConfig) Setup(){
	c.SetupGuestRoutes()
	c.SetupUserRoutes()
}

func (c RouteConfig) SetupGuestRoutes(){
	c.App.Post("/auth/login", c.AuthController.Login)
	c.App.Post("/register", c.UserController.Register)
}

func (c RouteConfig) SetupUserRoutes(){
	c.App.Use(c.AuthMiddleware)
	c.App.Get("/me", c.AuthController.Profile)
	
	c.App.Put("/users", c.UserController.Update)
}
