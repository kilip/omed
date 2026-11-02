package llmx

// DefaultCapabilities returns the default capabilities for a given provider kind.
// For KindAnthropic and KindGemini, all capabilities including Documents are true.
// For KindOpenAICompat, Vision, ToolCalling, StructuredOutput, and Streaming are true, Documents is false.
func DefaultCapabilities(k Kind) Capabilities {
	switch k {
	case KindAnthropic, KindGemini:
		return Capabilities{
			Vision:           true,
			ToolCalling:      true,
			StructuredOutput: true,
			Streaming:        true,
			Documents:        true,
		}
	case KindOpenAICompat:
		return Capabilities{
			Vision:           true,
			ToolCalling:      true,
			StructuredOutput: true,
			Streaming:        true,
			Documents:        false,
		}
	default:
		return Capabilities{}
	}
}

// MergeCapabilities returns base if override is nil, or *override if non-nil.
func MergeCapabilities(base Capabilities, override *Capabilities) Capabilities {
	if override == nil {
		return base
	}
	return *override
}
