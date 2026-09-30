package model

import "time"

type WebResponse[T any] struct {
	Data T    `json:"data,omitempty"`
	Meta Meta `json:"meta"`
}

type NoContent struct {
	Meta Meta `json:"meta"`
}

type Cursor struct {
	Next    string `json:"next,omitempty"`
	Prev    string `json:"prev,omitempty"`
	HasMore bool   `json:"has_more"`
}

type Meta struct {
	RequestID string    `json:"requestId"`
	Timestamp time.Time `json:"timestamp"`
	Cursor    string    `json:"cursor,omitempty"`
}

type FieldError struct {
	Field string `json:"field"`
	Rule  string `json:"rule"`
}

type ErrorBody struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
}

type ErrorResponse struct {
	Error ErrorBody `json:"error"`
	Meta  Meta      `json:"meta"`
}
