package service

import (
	"context"
	"time"
)

// BuildChatSystemPrompt exports buildChatSystemPrompt for tests in service_test.
func BuildChatSystemPrompt(now time.Time) string {
	return buildChatSystemPrompt(now)
}
// BuildSystemPrompt exports the unexported buildSystemPrompt function for tests in service_test.
func BuildSystemPrompt(snap FinanceSnapshot) string {
	return buildSystemPrompt(snap)
}

// DefaultAttachmentPrompt exports defaultAttachmentPrompt for tests in service_test.
const DefaultAttachmentPrompt = defaultAttachmentPrompt

// ExportMapLLMError exports mapLLMError for tests in service_test.
func ExportMapLLMError(err error, chatCtx context.Context) error {
	return mapLLMError(err, chatCtx)
}
