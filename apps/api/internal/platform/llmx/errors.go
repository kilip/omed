package llmx

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrAuth            = errors.New("llmx: authentication failed")
	ErrRateLimit       = errors.New("llmx: rate limited")
	ErrModelNotFound   = errors.New("llmx: model not found")
	ErrBadRequest      = errors.New("llmx: bad request")
	ErrUnavailable     = errors.New("llmx: service unavailable")
	ErrBlockedURL      = errors.New("llmx: blocked url")
	ErrUnsupportedKind = errors.New("llmx: unsupported kind")
)

// MapHTTPError maps an HTTP status code and response body to a sanitized error wrapping a sentinel.
func MapHTTPError(status int, body []byte, secrets ...string) error {
	var sentinel error
	switch status {
	case 401, 403:
		sentinel = ErrAuth
	case 404:
		sentinel = ErrModelNotFound
	case 429:
		sentinel = ErrRateLimit
	case 400, 422:
		sentinel = ErrBadRequest
	case 500, 502, 503:
		sentinel = ErrUnavailable
	default:
		sentinel = ErrUnavailable
	}

	msg := string(body)
	for _, secret := range secrets {
		if secret != "" {
			msg = strings.ReplaceAll(msg, secret, "[REDACTED]")
		}
	}

	if msg == "" {
		return fmt.Errorf("%w: status %d", sentinel, status)
	}
	return fmt.Errorf("%w: status %d: %s", sentinel, status, msg)
}
