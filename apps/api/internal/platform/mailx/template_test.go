package mailx_test

import (
	"strings"
	"testing"

	"github.com/kilip/omed/api/internal/platform/mailx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateEngine_RenderInvitation(t *testing.T) {
	engine := mailx.NewTemplateEngine()

	data := map[string]any{
		"WorkspaceName": "Acme Corp",
		"InviterName":   "Alice Admin",
		"InviteURL":     "http://localhost:3000/invite/token123",
	}

	htmlBody, textBody, err := engine.Render("invitation", data)
	require.NoError(t, err)

	assert.Contains(t, htmlBody, "Alice Admin")
	assert.Contains(t, htmlBody, "Acme Corp")
	assert.Contains(t, htmlBody, "http://localhost:3000/invite/token123")
	assert.True(t, strings.HasPrefix(htmlBody, "<!DOCTYPE html>"))

	assert.Contains(t, textBody, "Alice Admin")
	assert.Contains(t, textBody, "Acme Corp")
	assert.Contains(t, textBody, "http://localhost:3000/invite/token123")
}

func TestTemplateEngine_NotFound(t *testing.T) {
	engine := mailx.NewTemplateEngine()
	_, _, err := engine.Render("non_existent", nil)
	assert.Error(t, err)
}
