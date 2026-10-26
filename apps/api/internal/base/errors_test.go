package base_test

import (
	"errors"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/stretchr/testify/suite"
)

type ErrorMapSuite struct {
	suite.Suite
}

func (s *ErrorMapSuite) Test_Map_Successfully() {
	item := base.MapError(base.ErrInvalidID)
	s.Equal(base.ErrInvalidID.Error(), item.Code)
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
			err:            base.ErrUnknown,
			expectedStatus: fiber.StatusInternalServerError,
			expectedCode:   "UNKNOWN",
			expectedMsg:    "unknown error",
		},
		{
			err:            base.ErrInvalidUUID,
			expectedStatus: fiber.StatusBadRequest,
			expectedCode:   "INVALID_UUID",
			expectedMsg:    "invalid uuid",
		},
		{
			err:            base.ErrInvalidCursor,
			expectedStatus: fiber.StatusBadRequest,
			expectedCode:   "INVALID_CURSOR",
			expectedMsg:    "invalid cursor",
		},
		{
			err:            base.ErrUnauthorized,
			expectedStatus: fiber.StatusUnauthorized,
			expectedCode:   "UNAUTHORIZED",
			expectedMsg:    "unauthorized access",
		},
		{
			err:            base.ErrForbidden,
			expectedStatus: fiber.StatusForbidden,
			expectedCode:   "FORBIDDEN",
			expectedMsg:    "access forbidden",
		},
		{
			err:            base.ErrNotFound,
			expectedStatus: fiber.StatusNotFound,
			expectedCode:   "NOT_FOUND",
			expectedMsg:    "not found",
		},
		{
			err:            base.ErrWorkspaceRequired,
			expectedStatus: fiber.StatusBadRequest,
			expectedCode:   "workspace context required",
			expectedMsg:    "workspace context required",
		},
	}

	for _, tc := range testCases {
		item := base.MapError(tc.err)
		s.Equal(tc.expectedCode, item.Code)
		s.Equal(tc.expectedStatus, item.Status)
		s.Equal(tc.expectedMsg, item.Message)
	}
}

func (s *ErrorMapSuite) Test_Map_Unregistered() {
	item := base.MapError(errors.New("unregistered"))
	s.Equal(base.ErrUnknown.Error(), item.Code)
	s.Equal(fiber.StatusInternalServerError, item.Status)
	s.Equal("unknown error", item.Message)
}

func (s *ErrorMapSuite) Test_AddToErrorMap() {
	customErr := errors.New("CUSTOM_FORBIDDEN")
	base.AddToErrorMap(customErr, fiber.StatusForbidden, "forbidden action")

	item := base.MapError(customErr)
	s.Equal(customErr.Error(), item.Code)
	s.Equal(fiber.StatusForbidden, item.Status)
	s.Equal("forbidden action", item.Message)
}

func (s *ErrorMapSuite) Test_IsNotFound() {
	s.True(base.IsNotFound(base.ErrNotFound))
	s.False(base.IsNotFound(base.ErrUnknown))
	s.False(base.IsNotFound(errors.New("other error")))
	s.False(base.IsNotFound(nil))
}

func TestErrorMap(t *testing.T) {
	suite.Run(t, &ErrorMapSuite{})
}
