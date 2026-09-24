package config

import (
	"log/slog"
	stdhttp "net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/finance/internal/delivery/http"
)

func GetFiber(config Config, logger *slog.Logger) *fiber.App {
	fiber := fiber.New(fiber.Config{
		AppName:      "finance",
		ErrorHandler: NewErrorHandler(logger),
	})

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
