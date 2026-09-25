package config

import (
	"log/slog"
	stdhttp "net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/kilip/omed/finance/internal/delivery/http"
	"github.com/kilip/omed/finance/internal/util"
)

func GetFiber(config Config, logger *slog.Logger) *fiber.App {
	fiber := fiber.New(fiber.Config{
		AppName:      "finance",
		ErrorHandler: NewErrorHandler(logger),
	})

	fiber.Use(requestid.New(requestid.Config{
		Header:    "X-Request-ID",           // "X-Request-ID"
		Generator: util.GenerateID().String, // atau uuid.NewString kalau pakai google/uuid
	}))

	fiber.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://*.itstoni.com", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Orign", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))
	return fiber
}

func NewErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		var errResp *http.ErrorResponse

		switch e := err.(type) {
		case *http.ErrorResponse:
			errResp = e
		case *fiber.Error:
			errResp = http.NewErrorResponse(e.Code, stdhttp.StatusText(e.Code), e.Message)
		default:
			errResp = http.ErrInternal(err.Error())
		}

		errResp.Instance = c.Path()
		errResp.TraceID = c.GetRespHeader("X-Trace-Id")

		if errResp.Status >= 500 {
			logger.Error("internal error", "err", err, "path", c.Path())
		}

		return c.Status(errResp.Status).JSON(errResp)
	}
}
