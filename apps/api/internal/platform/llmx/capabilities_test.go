package llmx_test

import (
	"testing"

	"github.com/kilip/omed/api/internal/platform/llmx"
	"github.com/stretchr/testify/assert"
)

func TestDefaultCapabilities(t *testing.T) {
	expectedOpenAI := llmx.Capabilities{
		Vision:           true,
		ToolCalling:      true,
		StructuredOutput: true,
		Streaming:        true,
		Documents:        false,
	}
	expectedAnthropicGemini := llmx.Capabilities{
		Vision:           true,
		ToolCalling:      true,
		StructuredOutput: true,
		Streaming:        true,
		Documents:        true,
	}

	assert.Equal(t, expectedOpenAI, llmx.DefaultCapabilities(llmx.KindOpenAICompat))
	assert.Equal(t, expectedAnthropicGemini, llmx.DefaultCapabilities(llmx.KindAnthropic))
	assert.Equal(t, expectedAnthropicGemini, llmx.DefaultCapabilities(llmx.KindGemini))

	assert.Equal(t, llmx.Capabilities{}, llmx.DefaultCapabilities("unknown_kind"))
}

func TestDefaultCapabilities_Documents(t *testing.T) {
	assert.True(t, llmx.DefaultCapabilities(llmx.KindAnthropic).Documents)
	assert.True(t, llmx.DefaultCapabilities(llmx.KindGemini).Documents)
	assert.False(t, llmx.DefaultCapabilities(llmx.KindOpenAICompat).Documents)
	assert.False(t, llmx.DefaultCapabilities(llmx.KindMemory).Documents)
}

func TestMergeCapabilities(t *testing.T) {
	base := llmx.Capabilities{
		Vision:           true,
		ToolCalling:      true,
		StructuredOutput: true,
		Streaming:        true,
	}

	// nil override keeps base
	assert.Equal(t, base, llmx.MergeCapabilities(base, nil))

	// override completely replaces base
	override := &llmx.Capabilities{
		Vision:           false,
		ToolCalling:      false,
		StructuredOutput: false,
		Streaming:        false,
	}
	assert.Equal(t, *override, llmx.MergeCapabilities(base, override))

	customOverride := &llmx.Capabilities{
		Vision:           true,
		ToolCalling:      false,
		StructuredOutput: true,
		Streaming:        false,
	}
	emptyBase := llmx.Capabilities{}
	assert.Equal(t, *customOverride, llmx.MergeCapabilities(emptyBase, customOverride))
}
