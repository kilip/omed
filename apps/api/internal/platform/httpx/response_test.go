package httpx_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/stretchr/testify/suite"
)

type ResponseSuite struct {
	suite.Suite
}

type testSampleData struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (s *ResponseSuite) Test_NewMeta_WithRequestID() {
	app := fiber.New()
	app.Use(requestid.New())

	var capturedMeta httpx.Meta
	app.Get("/meta", func(c fiber.Ctx) error {
		capturedMeta = httpx.NewMeta(c, "test-cursor-123")
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/meta", nil)
	req.Header.Set("X-Request-ID", "custom-req-id-abc")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	s.Equal("custom-req-id-abc", capturedMeta.RequestID)
	s.Equal("test-cursor-123", capturedMeta.Cursor)
	s.WithinDuration(time.Now(), capturedMeta.Timestamp, 5*time.Second)
}

func (s *ResponseSuite) Test_NewMeta_WithoutRequestID() {
	app := fiber.New()

	var capturedMeta httpx.Meta
	app.Get("/meta", func(c fiber.Ctx) error {
		capturedMeta = httpx.NewMeta(c, "")
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/meta", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	s.Equal(fiber.StatusOK, resp.StatusCode)

	s.Empty(capturedMeta.RequestID)
	s.Empty(capturedMeta.Cursor)
	s.WithinDuration(time.Now(), capturedMeta.Timestamp, 5*time.Second)
}

func (s *ResponseSuite) Test_Success() {
	app := fiber.New()
	data := testSampleData{ID: 10, Name: "Sample"}

	app.Get("/success", func(c fiber.Ctx) error {
		return httpx.Success(c, fiber.StatusAccepted, data)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/success", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusAccepted, resp.StatusCode)

	var result httpx.WebResponse[testSampleData]
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&result))
	s.Equal(data, result.Data)
	s.WithinDuration(time.Now(), result.Meta.Timestamp, 5*time.Second)
}

func (s *ResponseSuite) Test_OK() {
	app := fiber.New()
	data := testSampleData{ID: 1, Name: "Test OK"}

	app.Get("/ok", func(c fiber.Ctx) error {
		return httpx.OK(c, data)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/ok", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)

	var result httpx.WebResponse[testSampleData]
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&result))
	s.Equal(data, result.Data)
	s.WithinDuration(time.Now(), result.Meta.Timestamp, 5*time.Second)
}

func (s *ResponseSuite) Test_Created() {
	app := fiber.New()
	data := testSampleData{ID: 2, Name: "Test Created"}

	app.Post("/created", func(c fiber.Ctx) error {
		return httpx.Created(c, data)
	})

	req := httptest.NewRequest(fiber.MethodPost, "/created", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusCreated, resp.StatusCode)

	var result httpx.WebResponse[testSampleData]
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&result))
	s.Equal(data, result.Data)
	s.WithinDuration(time.Now(), result.Meta.Timestamp, 5*time.Second)
}

func (s *ResponseSuite) Test_WithCursor() {
	app := fiber.New()
	items := []testSampleData{
		{ID: 1, Name: "Item 1"},
		{ID: 2, Name: "Item 2"},
	}

	app.Get("/cursor", func(c fiber.Ctx) error {
		return httpx.WithCursor(c, items, "cursor_next_token")
	})

	req := httptest.NewRequest(fiber.MethodGet, "/cursor", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)

	var result httpx.WebResponse[[]testSampleData]
	s.Require().NoError(json.NewDecoder(resp.Body).Decode(&result))
	s.Equal(items, result.Data)
	s.Equal("cursor_next_token", result.Meta.Cursor)
	s.WithinDuration(time.Now(), result.Meta.Timestamp, 5*time.Second)
}

func TestResponseSuite(t *testing.T) {
	suite.Run(t, &ResponseSuite{})
}

