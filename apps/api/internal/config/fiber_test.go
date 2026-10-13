package config_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/config"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/stretchr/testify/suite"
)

type FiberSuite struct {
	suite.Suite
	logBuf *bytes.Buffer
	logger *slog.Logger
	cfg    config.Config
}

func (s *FiberSuite) SetupTest() {
	s.logBuf = new(bytes.Buffer)
	s.logger = slog.New(slog.NewJSONHandler(s.logBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	s.cfg = config.Config{
		TrustedOrigins: []string{"http://trusted.com", "http://localhost:3000"},
	}
}

type testPayload struct {
	Name string `json:"name" validate:"required"`
	Age  int    `json:"age" validate:"gte=18"`
}

func (s *FiberSuite) Test_GetFiber_StructValidator_Valid() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/validate", func(c fiber.Ctx) error {
		var p testPayload
		if err := c.Bind().Body(&p); err != nil {
			return err
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
	})

	body := `{"name":"John Doe","age":25}`
	req := httptest.NewRequest(fiber.MethodPost, "/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *FiberSuite) Test_GetFiber_StructValidator_Invalid() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/validate", func(c fiber.Ctx) error {
		var p testPayload
		if err := c.Bind().Body(&p); err != nil {
			return err
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
	})

	body := `{"name":"","age":15}`
	req := httptest.NewRequest(fiber.MethodPost, "/validate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusUnprocessableEntity, resp.StatusCode)

	var errResp httpx.ErrorResponse
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	s.Require().NoError(err)
	s.Equal("VALIDATION_FAILED", errResp.Error.Code)
	s.Require().NotEmpty(errResp.Error.Fields)
}

func (s *FiberSuite) Test_GetFiber_CORS_AllowedOrigin() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Get("/hello", func(c fiber.Ctx) error {
		return c.SendString("world")
	})

	req := httptest.NewRequest(fiber.MethodGet, "/hello", nil)
	req.Header.Set("Origin", "http://trusted.com")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.Equal("http://trusted.com", resp.Header.Get("Access-Control-Allow-Origin"))
}

func (s *FiberSuite) Test_GetFiber_CORS_DisallowedOrigin() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Get("/hello", func(c fiber.Ctx) error {
		return c.SendString("world")
	})

	req := httptest.NewRequest(fiber.MethodGet, "/hello", nil)
	req.Header.Set("Origin", "http://untrusted.com")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.Empty(resp.Header.Get("Access-Control-Allow-Origin"))
}

func (s *FiberSuite) Test_GetFiber_CORS_Preflight() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Post("/data", func(c fiber.Ctx) error {
		return c.SendString("created")
	})

	req := httptest.NewRequest(fiber.MethodOptions, "/data", nil)
	req.Header.Set("Origin", "http://trusted.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal("http://trusted.com", resp.Header.Get("Access-Control-Allow-Origin"))
	s.Contains(resp.Header.Get("Access-Control-Allow-Methods"), "POST")
}

func (s *FiberSuite) Test_GetFiber_RecoverMiddleware() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Get("/panic", func(c fiber.Ctx) error {
		panic("unexpected crash")
	})

	req := httptest.NewRequest(fiber.MethodGet, "/panic", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusInternalServerError, resp.StatusCode)
}

func (s *FiberSuite) Test_ErrorHandler_MappedError() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Get("/not-found", func(c fiber.Ctx) error {
		return base.ErrNotFound
	})
	app.Get("/invalid-id", func(c fiber.Ctx) error {
		return base.ErrInvalidID
	})

	// Test NotFound
	reqNF := httptest.NewRequest(fiber.MethodGet, "/not-found", nil)
	respNF, err := app.Test(reqNF)
	s.Require().NoError(err)
	defer respNF.Body.Close()

	s.Equal(fiber.StatusNotFound, respNF.StatusCode)
	var errRespNF httpx.ErrorResponse
	s.Require().NoError(json.NewDecoder(respNF.Body).Decode(&errRespNF))
	s.Equal("NOT_FOUND", errRespNF.Error.Code)
	s.Equal("not found", errRespNF.Error.Message)

	// Test InvalidID
	reqID := httptest.NewRequest(fiber.MethodGet, "/invalid-id", nil)
	respID, err := app.Test(reqID)
	s.Require().NoError(err)
	defer respID.Body.Close()

	s.Equal(fiber.StatusBadRequest, respID.StatusCode)
	var errRespID httpx.ErrorResponse
	s.Require().NoError(json.NewDecoder(respID.Body).Decode(&errRespID))
	s.Equal("INVALID_ID", errRespID.Error.Code)
	s.Equal("invalid id", errRespID.Error.Message)
}

func (s *FiberSuite) Test_ErrorHandler_UnknownError() {
	app := config.GetFiber(s.cfg, s.logger)
	app.Get("/unknown", func(c fiber.Ctx) error {
		return errors.New("database disconnected unexpectedly")
	})

	req := httptest.NewRequest(fiber.MethodGet, "/unknown", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusInternalServerError, resp.StatusCode)
	var errResp httpx.ErrorResponse
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&errResp))
	s.Equal(base.ErrUnknown.Error(), errResp.Error.Code)
	s.Equal("unknown error", errResp.Error.Message)

	// Verify logged
	logged := s.logBuf.String()
	s.Contains(logged, "unknown error")
	s.Contains(logged, "database disconnected unexpectedly")
}

func TestFiberSuite(t *testing.T) {
	suite.Run(t, new(FiberSuite))
}
