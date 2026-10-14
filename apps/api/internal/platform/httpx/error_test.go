package httpx_test

import (
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/stretchr/testify/suite"
)

type ErrorSuite struct {
	suite.Suite
	validate *validator.Validate
}

func (s *ErrorSuite) SetupTest() {
	s.validate = validator.New()
}

type testSampleValidation struct {
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
}

func (s *ErrorSuite) Test_MapError_ValidationErrors() {
	invalidData := testSampleValidation{
		Name:  "",
		Email: "not-an-email",
	}

	err := s.validate.Struct(invalidData)
	s.Require().Error(err)

	status, body := httpx.MapError(err)
	s.Equal(fiber.StatusUnprocessableEntity, status)
	s.Equal("VALIDATION_FAILED", body.Code)
	s.Equal("validation failed", body.Message)
	s.Require().Len(body.Fields, 2)

	fieldsMap := make(map[string]string)
	for _, f := range body.Fields {
		fieldsMap[f.Field] = f.Rule
	}

	s.Equal("required", fieldsMap["Name"])
	s.Equal("email", fieldsMap["Email"])
}

func (s *ErrorSuite) Test_MapError_PredefinedBaseErrors() {
	testCases := []struct {
		name           string
		err            error
		expectedStatus int
		expectedCode   string
		expectedMsg    string
	}{
		{
			name:           "ErrInvalidID",
			err:            base.ErrInvalidID,
			expectedStatus: fiber.StatusBadRequest,
			expectedCode:   base.ErrInvalidID.Error(),
			expectedMsg:    "invalid id",
		},
		{
			name:           "ErrNotFound",
			err:            base.ErrNotFound,
			expectedStatus: fiber.StatusNotFound,
			expectedCode:   base.ErrNotFound.Error(),
			expectedMsg:    "not found",
		},
		{
			name:           "ErrUnauthorized",
			err:            base.ErrUnauthorized,
			expectedStatus: fiber.StatusUnauthorized,
			expectedCode:   base.ErrUnauthorized.Error(),
			expectedMsg:    "unauthorized access",
		},
		{
			name:           "ErrInvalidUUID",
			err:            base.ErrInvalidUUID,
			expectedStatus: fiber.StatusBadRequest,
			expectedCode:   base.ErrInvalidUUID.Error(),
			expectedMsg:    "invalid uuid",
		},
		{
			name:           "ErrInvalidCursor",
			err:            base.ErrInvalidCursor,
			expectedStatus: fiber.StatusBadRequest,
			expectedCode:   base.ErrInvalidCursor.Error(),
			expectedMsg:    "invalid cursor",
		},
		{
			name:           "ErrUnknown",
			err:            base.ErrUnknown,
			expectedStatus: fiber.StatusInternalServerError,
			expectedCode:   base.ErrUnknown.Error(),
			expectedMsg:    "unknown error",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			status, body := httpx.MapError(tc.err)
			s.Equal(tc.expectedStatus, status)
			s.Equal(tc.expectedCode, body.Code)
			s.Equal(tc.expectedMsg, body.Message)
			s.Empty(body.Fields)
		})
	}
}

func (s *ErrorSuite) Test_MapError_UnregisteredError() {
	unknownErr := errors.New("database connection reset")
	status, body := httpx.MapError(unknownErr)

	s.Equal(fiber.StatusInternalServerError, status)
	s.Equal(base.ErrUnknown.Error(), body.Code)
	s.Equal("unknown error", body.Message)
	s.Empty(body.Fields)
}

func (s *ErrorSuite) Test_MapError_CustomRegisteredError() {
	customErr := errors.New("PAYMENT_REQUIRED")
	base.AddToErrorMap(customErr, fiber.StatusPaymentRequired, "payment required")

	status, body := httpx.MapError(customErr)
	s.Equal(fiber.StatusPaymentRequired, status)
	s.Equal(customErr.Error(), body.Code)
	s.Equal("payment required", body.Message)
	s.Empty(body.Fields)
}

func TestErrorSuite(t *testing.T) {
	suite.Run(t, &ErrorSuite{})
}
