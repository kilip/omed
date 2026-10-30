package llmx_test

import (
	"context"
	"sync"
	"testing"

	"github.com/kilip/omed/api/internal/platform/llmx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryReplaysInOrder(t *testing.T) {
	ctx := context.Background()
	r1 := llmx.Response{
		Message: llmx.Message{
			Role:  llmx.RoleAssistant,
			Parts: []llmx.Part{{Text: "first reply"}},
		},
		Usage: llmx.Usage{InputTokens: 10, OutputTokens: 5},
	}
	r2 := llmx.Response{
		Message: llmx.Message{
			Role:  llmx.RoleAssistant,
			Parts: []llmx.Part{{Text: "second reply"}},
		},
		Usage: llmx.Usage{InputTokens: 20, OutputTokens: 10},
	}

	mem := llmx.NewMemory(r1, r2)

	// Adapter interface compliance check
	var _ llmx.Adapter = mem

	// First call -> r1
	resp1, err := mem.Chat(ctx, llmx.Request{Model: "m1"})
	require.NoError(t, err)
	assert.Equal(t, r1, *resp1)

	// Second call -> r2
	resp2, err := mem.Chat(ctx, llmx.Request{Model: "m2"})
	require.NoError(t, err)
	assert.Equal(t, r2, *resp2)

	// Third call -> empty response when exhausted
	resp3, err := mem.Chat(ctx, llmx.Request{Model: "m3"})
	require.NoError(t, err)
	assert.Equal(t, llmx.Response{}, *resp3)
}

func TestMemoryRecordsRequests(t *testing.T) {
	ctx := context.Background()
	mem := llmx.NewMemory()

	req1 := llmx.Request{Model: "gpt-4o", Messages: []llmx.Message{{Role: llmx.RoleUser, Parts: []llmx.Part{{Text: "hi"}}}}}
	req2 := llmx.Request{Model: "claude-3-5-sonnet", Messages: []llmx.Message{{Role: llmx.RoleUser, Parts: []llmx.Part{{Text: "hello"}}}}}

	_, err := mem.Chat(ctx, req1)
	require.NoError(t, err)

	_, err = mem.Chat(ctx, req2)
	require.NoError(t, err)

	reqs := mem.Requests()
	require.Len(t, reqs, 2)
	assert.Equal(t, req1, reqs[0])
	assert.Equal(t, req2, reqs[1])

	// Concurrency safety check
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = mem.Chat(ctx, llmx.Request{Model: "concurrent"})
			_ = mem.Requests()
		}(i)
	}
	wg.Wait()

	assert.Len(t, mem.Requests(), 22)
}

func TestMemoryStreamEmitsTextThenDone(t *testing.T) {
	ctx := context.Background()
	expectedUsage := llmx.Usage{InputTokens: 12, OutputTokens: 8}
	reply := llmx.Response{
		Message: llmx.Message{
			Role:  llmx.RoleAssistant,
			Parts: []llmx.Part{{Text: "streamed response content"}},
		},
		Usage:      expectedUsage,
		StopReason: "stop",
	}

	mem := llmx.NewMemory(reply)

	req := llmx.Request{Model: "test-model"}
	var events []llmx.Event

	for event, err := range mem.Stream(ctx, req) {
		require.NoError(t, err)
		events = append(events, event)
	}

	require.Len(t, events, 2)
	assert.Equal(t, llmx.EventText, events[0].Type)
	assert.Equal(t, "streamed response content", events[0].Text)

	assert.Equal(t, llmx.EventDone, events[1].Type)
	require.NotNil(t, events[1].Usage)
	assert.Equal(t, expectedUsage, *events[1].Usage)

	// Also verify the stream recorded the request
	assert.Len(t, mem.Requests(), 1)
	assert.Equal(t, req, mem.Requests()[0])
}

func TestMemoryCapabilities(t *testing.T) {
	mem := llmx.NewMemory()
	expected := llmx.DefaultCapabilities(llmx.KindOpenAICompat)
	assert.Equal(t, expected, mem.Capabilities())
}
