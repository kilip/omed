package service

import "context"
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
