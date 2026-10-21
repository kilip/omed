package domain_test

import (
	"testing"

	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/stretchr/testify/assert"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple title",
			input:    "Hello World",
			expected: "hello-world",
		},
		{
			name:     "mixed case with numbers and punctuation",
			input:    "Hello, World! 2026: Year of Antigravity?",
			expected: "hello-world-2026-year-of-antigravity",
		},
		{
			name:     "multiple spaces and hyphens",
			input:    "  Hello   ---   World  --  Test  ",
			expected: "hello-world-test",
		},
		{
			name:     "unicode diacritics / accents",
			input:    "Crème Brûlée & Café au Lait",
			expected: "creme-brulee-cafe-au-lait",
		},
		{
			name:     "german umlauts",
			input:    "Über die Brücke nach München",
			expected: "uber-die-brucke-nach-munchen",
		},
		{
			name:     "empty string fallback",
			input:    "",
			expected: domain.DefaultSlugFallback,
		},
		{
			name:     "whitespace only fallback",
			input:    "   \t\n  ",
			expected: domain.DefaultSlugFallback,
		},
		{
			name:     "punctuation only fallback",
			input:    "???---!!!...",
			expected: domain.DefaultSlugFallback,
		},
		{
			name:     "cjk characters fallback",
			input:    "你好世界",
			expected: domain.DefaultSlugFallback,
		},
		{
			name:     "emoji only fallback",
			input:    "🚀🎉✨",
			expected: domain.DefaultSlugFallback,
		},
		{
			name:     "emoji mixed with words",
			input:    "🚀 Space Launch 2026! 🎉",
			expected: "space-launch-2026",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := domain.Slugify(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestAppendSlugSuffix(t *testing.T) {
	tests := []struct {
		name     string
		slug     string
		suffix   int
		expected string
	}{
		{
			name:     "standard slug with suffix",
			slug:     "my-post",
			suffix:   2,
			expected: "my-post-2",
		},
		{
			name:     "empty slug with suffix",
			slug:     "",
			suffix:   3,
			expected: "post-3",
		},
		{
			name:     "slug with higher number",
			slug:     "article",
			suffix:   42,
			expected: "article-42",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := domain.AppendSlugSuffix(tc.slug, tc.suffix)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
