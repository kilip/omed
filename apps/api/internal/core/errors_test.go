package core_test

import (
	"errors"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/core"
	"github.com/stretchr/testify/suite"
)

type ErrorMapSuite struct {
	suite.Suite
}

func (s *ErrorMapSuite) Test_Map_Successfully() {
	item := core.MapError(core.ErrInvalidID)
	s.Equal(core.ErrInvalidID.Error(), item.Code)
	s.Equal(fiber.StatusBadRequest, item.Status)
	s.Equal("invalid id", item.Message)
}

func (s *ErrorMapSuite) Test_Map_PredefinedErrors() {
	testCases := []struct {
		err            error
		expectedStatus int
		expectedCode   string
		expectedMsg    string
	}{
		{
			err:            core.ErrUnknown,
			expectedStatus: fiber.StatusInternalServerError,
			expectedCode:   "UNKNOWN",
			expectedMsg:    "unknown error",
		},
		{
			err:            core.ErrInvalidUUID,
			expectedStatus: fiber.StatusBadRequest,
			expectedCode:   "INVALID_UUID",
			expectedMsg:    "invalid uuid",
		},
		{
			err:            core.ErrInvalidCursor,
			expectedStatus: fiber.StatusBadRequest,
			expectedCode:   "INVALID_CURSOR",
			expectedMsg:    "invalid cursor",
		},
		{
			err:            core.ErrUnauthorized,
			expectedStatus: fiber.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
			expectedMsg:    "unauthorized access",
		},
		{
			err:            core.ErrNotFound,
			expectedStatus: fiber.StatusNotFound,
			expectedCode:   "NOT_FOUND",
			expectedMsg:    "not found",
		},
	}

	for _, tc := range testCases {
		item := core.MapError(tc.err)
		s.Equal(tc.expectedCode, item.Code)
		s.Equal(tc.expectedStatus, item.Status)
		s.Equal(tc.expectedMsg, item.Message)
	}
}

func (s *ErrorMapSuite) Test_Map_Unregistered() {
	item := core.MapError(errors.New("unregistered"))
	s.Equal(core.ErrUnknown.Error(), item.Code)
	s.Equal(fiber.StatusInternalServerError, item.Status)
	s.Equal("unknown error", item.Message)
}

func (s *ErrorMapSuite) Test_AddToErrorMap() {
	customErr := errors.New("CUSTOM_FORBIDDEN")
	core.AddToErrorMap(customErr, fiber.StatusForbidden, "forbidden action")

	item := core.MapError(customErr)
	s.Equal(customErr.Error(), item.Code)
	s.Equal(fiber.StatusForbidden, item.Status)
	s.Equal("forbidden action", item.Message)
}

func (s *ErrorMapSuite) Test_IsNotFound() {
	s.True(core.IsNotFound(core.ErrNotFound))
	s.False(core.IsNotFound(core.ErrUnknown))
	s.False(core.IsNotFound(errors.New("other error")))
	s.False(core.IsNotFound(nil))
}

func TestErrorMap(t *testing.T) {
	suite.Run(t, &ErrorMapSuite{})
}
