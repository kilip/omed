package httpx_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModel_WebResponseSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	resp := httpx.WebResponse[string]{
		Data: "test-data",
		Meta: httpx.Meta{
			RequestID: "req-123",
			Timestamp: now,
			Cursor:    "cursor-abc",
		},
	}

	bytes, err := json.Marshal(resp)
	require.NoError(t, err)

	var decoded httpx.WebResponse[string]
	require.NoError(t, json.Unmarshal(bytes, &decoded))

	assert.Equal(t, "test-data", decoded.Data)
	assert.Equal(t, "req-123", decoded.Meta.RequestID)
	assert.Equal(t, "cursor-abc", decoded.Meta.Cursor)
	assert.True(t, now.Equal(decoded.Meta.Timestamp))
}

func TestModel_NoContentSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	noContent := httpx.NoContent{
		Meta: httpx.Meta{
			RequestID: "req-456",
			Timestamp: now,
		},
	}

	bytes, err := json.Marshal(noContent)
	require.NoError(t, err)

	var decoded httpx.NoContent
	require.NoError(t, json.Unmarshal(bytes, &decoded))

	assert.Equal(t, "req-456", decoded.Meta.RequestID)
	assert.Empty(t, decoded.Meta.Cursor)
}

func TestModel_CursorSerialization(t *testing.T) {
	cursor := httpx.Cursor{
		Next:    "next-cur",
		Prev:    "prev-cur",
		HasMore: true,
	}

	bytes, err := json.Marshal(cursor)
	require.NoError(t, err)

	var decoded httpx.Cursor
	require.NoError(t, json.Unmarshal(bytes, &decoded))

	assert.Equal(t, "next-cur", decoded.Next)
	assert.Equal(t, "prev-cur", decoded.Prev)
	assert.True(t, decoded.HasMore)
}

func TestModel_ErrorResponseSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	errResp := httpx.ErrorResponse{
		Error: httpx.ErrorBody{
			Code:    "VALIDATION_FAILED",
			Message: "validation failed",
			Fields: []httpx.FieldError{
				{Field: "email", Rule: "required"},
			},
		},
		Meta: httpx.Meta{
			RequestID: "req-err-789",
			Timestamp: now,
		},
	}

	bytes, err := json.Marshal(errResp)
	require.NoError(t, err)

	var decoded httpx.ErrorResponse
	require.NoError(t, json.Unmarshal(bytes, &decoded))

	assert.Equal(t, "VALIDATION_FAILED", decoded.Error.Code)
	assert.Equal(t, "validation failed", decoded.Error.Message)
	require.Len(t, decoded.Error.Fields, 1)
	assert.Equal(t, "email", decoded.Error.Fields[0].Field)
	assert.Equal(t, "required", decoded.Error.Fields[0].Rule)
	assert.Equal(t, "req-err-789", decoded.Meta.RequestID)
}
