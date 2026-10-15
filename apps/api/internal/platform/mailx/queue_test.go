package mailx_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/internal/platform/mailx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryQueue_EnqueueAndProcess(t *testing.T) {
	driver := mailx.NewMemoryDriver()
	q := mailx.NewMemoryQueue(driver, mailx.MemoryQueueConfig{
		BufferSize: 10,
		Workers:    2,
	})

	ctx := context.Background()
	require.NoError(t, q.Start(ctx))

	msg := mailx.NewMessage()
	msg.AddTo("recipient@example.com", "Test")
	msg.Subject = "Queue Test"
	msg.TextBody = "Background delivery test"

	err := q.Enqueue(ctx, msg)
	require.NoError(t, err)

	// Wait briefly for worker to consume
	require.Eventually(t, func() bool {
		return len(driver.SentMessages()) == 1
	}, 2*time.Second, 50*time.Millisecond)

	sent := driver.LastMessage()
	require.NotNil(t, sent)
	assert.Equal(t, "Queue Test", sent.Subject)

	// Stop gracefully
	stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	require.NoError(t, q.Stop(stopCtx))

	// Enqueue after stop fails
	assert.ErrorIs(t, q.Enqueue(ctx, msg), mailx.ErrQueueStopped)
}
