package mailx_test

import (
	"context"
	"testing"

	"github.com/kilip/omed/api/internal/platform/mailx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryDriver_SendAndInspect(t *testing.T) {
	ctx := context.Background()
	driver := mailx.NewMemoryDriver()

	assert.Empty(t, driver.SentMessages())
	assert.Nil(t, driver.LastMessage())

	msg := mailx.NewMessage()
	msg.AddTo("user@example.com", "Test User")
	msg.Subject = "Test Subject"
	msg.TextBody = "Hello world"

	err := driver.Send(ctx, msg)
	require.NoError(t, err)

	sent := driver.SentMessages()
	require.Len(t, sent, 1)
	assert.Equal(t, "Test Subject", sent[0].Subject)
	assert.Equal(t, "user@example.com", sent[0].To[0].Address)

	last := driver.LastMessage()
	require.NotNil(t, last)
	assert.Equal(t, "Test Subject", last.Subject)

	driver.Reset()
	assert.Empty(t, driver.SentMessages())
	assert.Nil(t, driver.LastMessage())

	require.NoError(t, driver.Close())
}
