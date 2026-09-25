package http

import (
	"entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/model"
)

type PingResponse struct {
	User           model.AuthenticatedUser `json:"user"`
	DatabaseStatus string                  `json:"databaseStatus"`
}

type HealthController struct {
	databaseUrl string
}

func NewHealthController(app *fiber.App, databaseUrl string) *HealthController {
	ctl := &HealthController{
		databaseUrl,
	}
	ctl.initRoutes(app)

	return ctl
}

func (ctl *HealthController) initRoutes(app *fiber.App) {
	app.Get("/ping", ctl.ping)
}

// @Summary		Ping
// @Description	Check finance health
// @Tags			health
// @Accept			json
// @Produce		json
// @Success		200		{object}	model.Envelope[http.PingResponse]
// @Failure		500		{object}	http.ErrorResponse	"Internal server error"
// @Security		BearerAuth
// @Router			/ping [get]
func (ctl *HealthController) ping(c fiber.Ctx) error {
	dbStatus := "ok"

	user := GetUser(c)
	conn, err := sql.Open("postgres", ctl.databaseUrl)
	if err != nil {
		dbStatus = err.Error()
	}
	defer conn.Close()

	response := &PingResponse{
		User:           user,
		DatabaseStatus: dbStatus,
	}

	return Success(c, fiber.StatusOK, response)
}
