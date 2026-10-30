package llmx_test

import (
	"context"
	"errors"
	"iter"
	"net/http"
	"testing"

	"github.com/kilip/omed/api/internal/platform/llmx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyAdapter struct{}

func (d *dummyAdapter) Chat(ctx context.Context, req llmx.Request) (*llmx.Response, error) {
	return &llmx.Response{}, nil
}

func (d *dummyAdapter) Stream(ctx context.Context, req llmx.Request) iter.Seq2[llmx.Event, error] {
	return func(yield func(llmx.Event, error) bool) {}
}

func (d *dummyAdapter) Capabilities() llmx.Capabilities {
	return llmx.Capabilities{}
}

func TestNewUnsupportedKind(t *testing.T) {
	_, err := llmx.New(llmx.Config{
		Kind: "non_existent_kind",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, llmx.ErrUnsupportedKind))
}

func TestRegisterAndNew(t *testing.T) {
	var capturedConfig llmx.Config
	llmx.Register("mock_kind", func(cfg llmx.Config) (llmx.Adapter, error) {
		capturedConfig = cfg
		return &dummyAdapter{}, nil
	})

	// When HTTPClient is nil, New should set a default HTTPClient
	adapter, err := llmx.New(llmx.Config{
		Kind: "mock_kind",
	})
	require.NoError(t, err)
	require.NotNil(t, adapter)
	assert.NotNil(t, capturedConfig.HTTPClient)

	// When custom HTTPClient is provided, New should preserve it
	customClient := &http.Client{}
	adapter2, err := llmx.New(llmx.Config{
		Kind:       "mock_kind",
		HTTPClient: customClient,
	})
	require.NoError(t, err)
	require.NotNil(t, adapter2)
	assert.Same(t, customClient, capturedConfig.HTTPClient)
}
