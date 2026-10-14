package httpx_test

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/stretchr/testify/suite"
)

type ParamsSuite struct {
	suite.Suite
}

func (s *ParamsSuite) Test_GetID_Int64_Valid() {
	app := fiber.New()
	var (
		extractedID int64
		errReturned error
	)

	app.Get("/users/:id", func(c fiber.Ctx) error {
		extractedID, errReturned = httpx.GetID[int64](c)
		return nil
	})

	testCases := []struct {
		param    string
		expected int64
	}{
		{"42", 42},
		{"1", 1},
		{"0", 0},
		{"9223372036854775807", 9223372036854775807},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(fiber.MethodGet, fmt.Sprintf("/users/%s", tc.param), nil)
		_, err := app.Test(req)
		s.Require().NoError(err)
		s.NoError(errReturned)
		s.Equal(tc.expected, extractedID)
	}
}

func (s *ParamsSuite) Test_GetID_Int64_Invalid() {
	app := fiber.New()
	var (
		extractedID int64
		errReturned error
	)

	app.Get("/users/:id", func(c fiber.Ctx) error {
		extractedID, errReturned = httpx.GetID[int64](c)
		return nil
	})

	invalidParams := []string{
		"abc",
		"12.34",
		"undefined",
		"9999999999999999999999999999999",
	}

	for _, param := range invalidParams {
		req := httptest.NewRequest(fiber.MethodGet, fmt.Sprintf("/users/%s", param), nil)
		_, err := app.Test(req)
		s.Require().NoError(err)
		s.ErrorIs(errReturned, base.ErrInvalidID)
		s.Zero(extractedID)
	}
}

func (s *ParamsSuite) Test_GetID_Int64_MissingParam() {
	app := fiber.New()
	var (
		extractedID int64
		errReturned error
	)

	app.Get("/no-param", func(c fiber.Ctx) error {
		extractedID, errReturned = httpx.GetID[int64](c)
		return nil
	})

	req := httptest.NewRequest(fiber.MethodGet, "/no-param", nil)
	_, err := app.Test(req)
	s.Require().NoError(err)
	s.ErrorIs(errReturned, base.ErrInvalidID)
	s.Zero(extractedID)
}

func (s *ParamsSuite) Test_GetID_UUID_Valid() {
	app := fiber.New()
	var (
		extractedID uuid.UUID
		errReturned error
	)

	app.Get("/items/:id", func(c fiber.Ctx) error {
		extractedID, errReturned = httpx.GetID[uuid.UUID](c)
		return nil
	})

	validUUID := uuid.New()
	req := httptest.NewRequest(fiber.MethodGet, fmt.Sprintf("/items/%s", validUUID.String()), nil)
	_, err := app.Test(req)
	s.Require().NoError(err)
	s.NoError(errReturned)
	s.Equal(validUUID, extractedID)
}

func (s *ParamsSuite) Test_GetID_UUID_Invalid() {
	app := fiber.New()
	var (
		extractedID uuid.UUID
		errReturned error
	)

	app.Get("/items/:id", func(c fiber.Ctx) error {
		extractedID, errReturned = httpx.GetID[uuid.UUID](c)
		return nil
	})

	invalidParams := []string{
		"not-a-uuid",
		"12345",
		"undefined",
		"123e4567-e89b-12d3-a456",
	}

	for _, param := range invalidParams {
		req := httptest.NewRequest(fiber.MethodGet, fmt.Sprintf("/items/%s", param), nil)
		_, err := app.Test(req)
		s.Require().NoError(err)
		s.ErrorIs(errReturned, base.ErrInvalidID)
		s.Equal(uuid.Nil, extractedID)
	}
}

func (s *ParamsSuite) Test_GetID_UUID_MissingParam() {
	app := fiber.New()
	var (
		extractedID uuid.UUID
		errReturned error
	)

	app.Get("/no-param", func(c fiber.Ctx) error {
		extractedID, errReturned = httpx.GetID[uuid.UUID](c)
		return nil
	})

	req := httptest.NewRequest(fiber.MethodGet, "/no-param", nil)
	_, err := app.Test(req)
	s.Require().NoError(err)
	s.ErrorIs(errReturned, base.ErrInvalidID)
	s.Equal(uuid.Nil, extractedID)
}

func TestParamsSuite(t *testing.T) {
	suite.Run(t, &ParamsSuite{})
}
