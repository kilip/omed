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

func TestListModels_MemoryAdapter(t *testing.T) {
	ctx := context.Background()
	models, err := llmx.ListModels(ctx, llmx.Config{
		Kind: llmx.Kind("memory"),
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(models) == 0 {
		t.Fatalf("expected non-empty models list")
	}
	assert.Equal(t, []string{"memory-default", "memory-fast"}, models)
}

func TestListModels_UnsupportedKind(t *testing.T) {
	_, err := llmx.ListModels(context.Background(), llmx.Config{
		Kind: "unknown_provider",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, llmx.ErrUnsupportedKind))
}

func TestListModels_NotModelLister(t *testing.T) {
	llmx.Register("no_lister_kind", func(cfg llmx.Config) (llmx.Adapter, error) {
		return &dummyAdapter{}, nil
	})

	_, err := llmx.ListModels(context.Background(), llmx.Config{
		Kind: "no_lister_kind",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, llmx.ErrUnsupportedKind))
}

type errorListerAdapter struct {
	dummyAdapter
	err error
}

func (e *errorListerAdapter) ListModels(ctx context.Context) ([]string, error) {
	return nil, e.err
}

func TestListModels_RedactsAPIKey(t *testing.T) {
	secretKey := "sk-secret-leaked-key-12345"
	llmx.Register("leaky_kind", func(cfg llmx.Config) (llmx.Adapter, error) {
		return &errorListerAdapter{
			err: errors.New("failed with key: " + cfg.APIKey),
		}, nil
	})

	_, err := llmx.ListModels(context.Background(), llmx.Config{
		Kind:   "leaky_kind",
		APIKey: secretKey,
	})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), secretKey)
	assert.Contains(t, err.Error(), "[REDACTED]")
}
