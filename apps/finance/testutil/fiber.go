package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRequest describes a single HTTP call to make against the test app.
type TestRequest struct {
	Method  string
	Path    string
	Body    any               // optional, will be JSON-encoded
	Headers map[string]string // optional
	Token   string
}

// DoRequest sends req against app and returns the raw *http.Response.
// Fails the test immediately on transport-level errors.
func DoRequest(t *testing.T, req TestRequest) *http.Response {
	t.Helper()

	var bodyReader io.Reader
	if req.Body != nil {
		b, err := json.Marshal(req.Body)
		require.NoError(t, err, "failed to marshal request body")
		bodyReader = bytes.NewReader(b)
	}

	httpReq := httptest.NewRequest(req.Method, req.Path, bodyReader)
	if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	if req.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.Token)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := app.Test(httpReq) // -1 = no timeout
	require.NoError(t, err, "app.Test failed")
	return resp
}

func AssertStatus(t *testing.T, resp *http.Response, want int) {
	t.Helper()
	require.Equal(t, want, resp.StatusCode, "unexpected status code")
}

// DecodeJSON reads resp.Body and unmarshals it into target.
func DecodeJSON(t *testing.T, resp *http.Response, target any) {
	t.Helper()
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")
	require.NoError(t, json.Unmarshal(b, target), "failed to unmarshal response body: %s", string(b))
}
