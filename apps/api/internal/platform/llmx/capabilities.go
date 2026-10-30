package llmx

// DefaultCapabilities returns the default capabilities for a given provider kind.
// For KindOpenAICompat, KindAnthropic, and KindGemini, all four capabilities are true.
func DefaultCapabilities(k Kind) Capabilities {
	switch k {
	case KindOpenAICompat, KindAnthropic, KindGemini:
		return Capabilities{
			Vision:           true,
			ToolCalling:      true,
			StructuredOutput: true,
			Streaming:        true,
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
