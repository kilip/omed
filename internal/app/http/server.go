package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/internal/utils"
)

type Server struct {
	conf        utils.Config
	app         *fiber.App
	controllers []Controller
}

func (s *Server) Start() error {
	s.loadRoutes()
	return s.app.Listen(":3000")
}

func NewServer(conf utils.Config) *Server {
	app := fiber.New(fiber.Config{})
	return &Server{conf, app, make([]Controller, 0)}
}

func (s *Server) AddController(controller Controller) {
	s.controllers = append(s.controllers, controller)
}

func (s *Server) loadRoutes() {
	s.app.Get("/", func(c *fiber.Ctx) error {
		return c.Status(200).JSON(fiber.Map{
			"success": true,
		})
	})

	for _, controller := range s.controllers {
		controller.LoadRoutes(s.app)
	}
}
