package service

// BuildSystemPrompt exports the unexported buildSystemPrompt function for tests in service_test.
func BuildSystemPrompt(snap FinanceSnapshot) string {
	return buildSystemPrompt(snap)
}

// DefaultAttachmentPrompt exports defaultAttachmentPrompt for tests in service_test.
const DefaultAttachmentPrompt = defaultAttachmentPrompt
