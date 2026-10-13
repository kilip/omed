// Package httpx provides standard HTTP envelope structures, response helpers,
// parameter parsers, validation mapping, and generic controller execution helpers.
package httpx

import "time"

// WebResponse is the standard JSON envelope wrapping successful API responses.
type WebResponse[T any] struct {
	// Data contains the response payload.
	Data T `json:"data,omitempty"`
	// Meta contains request metadata such as request ID and timestamps.
	Meta Meta `json:"meta"`
}

// NoContent represents an empty successful response containing only metadata.
type NoContent struct {
	// Meta contains request metadata.
	Meta Meta `json:"meta"`
}

// Cursor contains pagination cursor pointers and status.
type Cursor struct {
	Next    string `json:"next,omitempty"`
	Prev    string `json:"prev,omitempty"`
	HasMore bool   `json:"has_more"`
}

// Meta encapsulates operational metadata included in all API responses.
type Meta struct {
	RequestID string    `json:"requestId"`
	Timestamp time.Time `json:"timestamp"`
	Cursor    string    `json:"cursor,omitempty"`
}

// FieldError details a validation error on a specific struct field.
type FieldError struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
}

// ErrorBody describes the details and machine-readable code of an error response.
type ErrorBody struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
}

// ErrorResponse is the standard JSON envelope wrapping error responses.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
	Meta  Meta      `json:"meta"`
}
