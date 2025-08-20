package api

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/kilip/omed/internal/infra/config"
	"github.com/kilip/omed/internal/infra/database"
	"github.com/kilip/omed/internal/infra/database/dal"
)

type Server struct {
	App *fiber.App
	Config config.Config
	Query *dal.Query

	// The api root router with /v1 prefix
	// This router should be secured automatically
	// by using auth middleware
	Router fiber.Router
}

func NewServer(conf config.Config) *Server {
	app := fiber.New(fiber.Config{
		Prefork: true,
	})

	query := dal.Use(database.NewGormDB(conf))
	router := app.Group("/v1")

	return &Server{
		App: app,
		Config: conf,
		Query: query,
		Router: router,
	}
}

func (s Server) Start(){
	listen := fmt.Sprintf(
		"%s:%d",
		s.Config.Api.Host,
		s.Config.Api.Port,
	)
	s.App.Listen(listen)
}
