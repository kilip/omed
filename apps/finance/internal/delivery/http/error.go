package http

import "github.com/gofiber/fiber/v3"

// ErrorResponse represents a standard API error following RFC 7807
// swagger:model
type ErrorResponse struct {
	Type     string            `json:"type,omitempty" example:"https://omed.dev/errors/validation-failed"`
	Title    string            `json:"title" example:"Validation Failed"`
	Status   int               `json:"status" example:"400"`
	Detail   string            `json:"detail,omitempty" example:"field 'name' is required"`
	Instance string            `json:"instance,omitempty" example:"/accounts"`
	Errors   map[string]string `json:"errors,omitempty"` // field-level validation errors
	TraceID  string            `json:"trace_id,omitempty" example:"a1b2c3d4"`
}

func NewErrorResponse(status int, title, detail string) *ErrorResponse {
	return &ErrorResponse{
		Title:  title,
		Status: status,
		Detail: detail,
	}
}

func (e *ErrorResponse) Error() string {
	return e.Detail
}

func ErrBadRequest(detail string) *ErrorResponse {
	return NewErrorResponse(fiber.StatusBadRequest, "Invalid Request", detail)
}

func ErrUnauthorized(detail string) *ErrorResponse {
	return NewErrorResponse(fiber.StatusUnauthorized, "Unauthorized", detail)
}

func ErrForbidden(detail string) *ErrorResponse {
	return NewErrorResponse(fiber.StatusForbidden, "Forbidden", detail)
}

func ErrNotFound(detail string) *ErrorResponse {
	return NewErrorResponse(fiber.StatusNotFound, "Not Found", detail)
}

func ErrConflict(detail string) *ErrorResponse {
	return NewErrorResponse(fiber.StatusConflict, "Conflict", detail)
}

func ErrValidation(detail string, fieldErrors map[string]string) *ErrorResponse {
	e := NewErrorResponse(fiber.StatusUnprocessableEntity, "Validation Failed", detail)
	e.Errors = fieldErrors
	return e
}

func ErrInternal(detail string) *ErrorResponse {
	return NewErrorResponse(fiber.StatusInternalServerError, "Internal Server Error", detail)
}
