package mailx_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/internal/platform/mailx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMailer_Send_Validation(t *testing.T) {
	mailer, err := mailx.New(mailx.Config{
		Driver:      "memory",
		FromAddress: "system@omed.local",
		FromName:    "Omed System",
	})
	require.NoError(t, err)
	defer mailer.Close()

	ctx := context.Background()

	// Missing recipient
	msg := mailx.NewMessage()
	msg.Subject = "No recipient"
	msg.TextBody = "Test"
	assert.ErrorIs(t, mailer.Send(ctx, msg), mailx.ErrNoRecipient)

	// Missing subject
	msg = mailx.NewMessage()
	msg.AddTo("user@example.com", "User")
	msg.TextBody = "Test"
	assert.ErrorIs(t, mailer.Send(ctx, msg), mailx.ErrNoSubject)

	// Missing body
	msg = mailx.NewMessage()
	msg.AddTo("user@example.com", "User")
	msg.Subject = "Subject"
	assert.ErrorIs(t, mailer.Send(ctx, msg), mailx.ErrNoBody)

	// Invalid email
	msg = mailx.NewMessage()
	msg.AddTo("invalid-email-format", "User")
	msg.Subject = "Subject"
	msg.TextBody = "Test"
	assert.ErrorIs(t, mailer.Send(ctx, msg), mailx.ErrInvalidAddress)
}

func TestMailer_Send_And_SendAsync(t *testing.T) {
	mailer, err := mailx.New(mailx.Config{
		Driver:      "memory",
		FromAddress: "system@omed.local",
		FromName:    "Omed System",
	})
	require.NoError(t, err)
	defer mailer.Close()

	ctx := context.Background()
	driver, ok := mailer.Driver().(*mailx.MemoryDriver)
	require.True(t, ok)

	// Sync Send
	msg := mailx.NewMessage()
	msg.AddTo("john@example.com", "John")
	msg.Subject = "Welcome"
	msg.TextBody = "Welcome to our platform"
	err = mailer.Send(ctx, msg)
	require.NoError(t, err)

	require.Len(t, driver.SentMessages(), 1)
	assert.Equal(t, "system@omed.local", driver.LastMessage().From.Address)

	// Async Send Template
	tmplData := map[string]any{
		"WorkspaceName": "Finance Workspace",
		"InviterName":   "Boss",
		"InviteURL":     "http://localhost:3000/invite/tok123",
	}

	asyncMsg := mailx.NewMessage()
	asyncMsg.AddTo("jane@example.com", "Jane")
	asyncMsg.Subject = "Invitation to Workspace"

	err = mailer.SendTemplateAsync(ctx, "invitation", tmplData, asyncMsg)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return len(driver.SentMessages()) == 2
	}, 2*time.Second, 50*time.Millisecond)

	last := driver.LastMessage()
	assert.Equal(t, "jane@example.com", last.To[0].Address)
	assert.Contains(t, last.HTMLBody, "Finance Workspace")
}
