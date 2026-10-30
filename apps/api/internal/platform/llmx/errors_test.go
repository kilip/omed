package llmx_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/kilip/omed/api/internal/platform/llmx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapHTTPError(t *testing.T) {
	tests := []struct {
		status   int
		expected error
	}{
		{status: 401, expected: llmx.ErrAuth},
		{status: 403, expected: llmx.ErrAuth},
		{status: 404, expected: llmx.ErrModelNotFound},
		{status: 429, expected: llmx.ErrRateLimit},
		{status: 400, expected: llmx.ErrBadRequest},
		{status: 422, expected: llmx.ErrBadRequest},
		{status: 500, expected: llmx.ErrUnavailable},
		{status: 502, expected: llmx.ErrUnavailable},
		{status: 503, expected: llmx.ErrUnavailable},
		{status: 504, expected: llmx.ErrUnavailable}, // default
		{status: 418, expected: llmx.ErrUnavailable}, // default
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.status), func(t *testing.T) {
			err := llmx.MapHTTPError(tt.status, []byte("some error details"))
			require.Error(t, err)
			assert.True(t, errors.Is(err, tt.expected), "expected errors.Is(err, %v) to be true, got %v", tt.expected, err)
		})
	}
}

func TestMapHTTPErrorRedactsSecret(t *testing.T) {
	secret := "sk-secret-1234567890abcdef"
	secret2 := "bearer-token-xyz"
	body := []byte(fmt.Sprintf(`{"error":{"message":"invalid key %s or auth %s"}}`, secret, secret2))

	err := llmx.MapHTTPError(401, body, secret, secret2)
	require.Error(t, err)
	assert.True(t, errors.Is(err, llmx.ErrAuth))

	errStr := err.Error()
	assert.NotContains(t, errStr, secret)
	assert.NotContains(t, errStr, secret2)
	assert.Contains(t, errStr, "[REDACTED]")
}
