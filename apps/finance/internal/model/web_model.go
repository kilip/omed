package model

import "time"

type WebResponse[T any] struct {
	Data T    `json:"data,omitempty"`
	Meta Meta `json:"meta"`
}

type NoContent struct {
	Meta Meta `json:"meta"`
}

type Meta struct {
	RequestID string    `json:"requestId"`
	Timestamp time.Time `json:"timestamp"`
	Cursor    string    `json:"omitempty"`
}

type Cursor struct {
	Next    string `json:"next,omitempty"`
	Prev    string `json:"prev,omitempty"`
	HasMore bool   `json:"has_more"`
}

type ErrorResponse struct {
	Errors error `json:"string"`
	Meta   Meta  `json:"meta"`
}
