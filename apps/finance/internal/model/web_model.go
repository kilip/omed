package model

import "time"

type WebResponse[T any] struct {
	Errors string `json:"errors,omitempty"`
	Data   T      `json:"data,omitempty"`
	Meta   Meta   `json:"meta"`
}

type Meta struct {
	RequestID string    `json:"requestId"`
	Timestamp time.Time `json:"timestamp"`
	Cursor    *Cursor   `json:"omitempty"`
}

type Cursor struct {
	Next    string `json:"next,omitempty"`
	Prev    string `json:"prev,omitempty"`
	HasMore bool   `json:"has_more"`
}
