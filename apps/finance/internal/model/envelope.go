package model

type Envelope[T any] struct {
    Success bool        `json:"success"`
    Data    T `json:"data,omitempty"`
    Error   *ErrorInfo  `json:"error,omitempty"`
    Meta    Meta        `json:"meta"`
}

type ErrorEnvelope struct {
    Success bool       `json:"success"`
    Error   *ErrorInfo `json:"error"`
    Meta    Meta       `json:"meta"`
}

type ErrorInfo struct {
    Code    string      `json:"code"`
    Message string      `json:"message"`
    Details interface{} `json:"details,omitempty"`
}

type Meta struct {
    RequestID string  `json:"request_id"`
    Timestamp string  `json:"timestamp"`
    Cursor    *Cursor `json:"cursor,omitempty"`
}

type Cursor struct {
    Next    string `json:"next,omitempty"`
    Prev    string `json:"prev,omitempty"`
    HasMore bool   `json:"has_more"`
}
