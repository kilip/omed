package llmx_test

import (
	"testing"

	"github.com/kilip/omed/api/internal/platform/llmx"
	"github.com/stretchr/testify/assert"
)

func TestDefaultCapabilities(t *testing.T) {
	expectedAllTrue := llmx.Capabilities{
		Vision:           true,
		ToolCalling:      true,
		StructuredOutput: true,
		Streaming:        true,
	}

	assert.Equal(t, expectedAllTrue, llmx.DefaultCapabilities(llmx.KindOpenAICompat))
	assert.Equal(t, expectedAllTrue, llmx.DefaultCapabilities(llmx.KindAnthropic))
	assert.Equal(t, expectedAllTrue, llmx.DefaultCapabilities(llmx.KindGemini))

	assert.Equal(t, llmx.Capabilities{}, llmx.DefaultCapabilities("unknown_kind"))
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
