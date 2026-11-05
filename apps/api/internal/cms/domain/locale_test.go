package domain_test

import (
	"testing"

	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeLocale(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		expectErr bool
	}{
		{name: "standard en", input: "en", expected: "en"},
		{name: "standard id", input: "id", expected: "id"},
		{name: "case insensitive en-US", input: "en-us", expected: "en-US"},
		{name: "whitespace trimming", input: "  id  ", expected: "id"},
		{name: "empty string", input: "", expectErr: true},
		{name: "whitespace only", input: "   ", expectErr: true},
		{name: "invalid bcp47 tag", input: "invalid---tag", expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := domain.NormalizeLocale(tt.input)
			if tt.expectErr {
				assert.ErrorIs(t, err, domain.ErrInvalidLocale)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, res)
			}
		})
	}
}

func TestIsLocaleSupported(t *testing.T) {
	supported := []string{"en", "id", "ja"}
	assert.True(t, domain.IsLocaleSupported("en", supported))
	assert.True(t, domain.IsLocaleSupported("EN", supported))
	assert.True(t, domain.IsLocaleSupported("id", supported))
	assert.False(t, domain.IsLocaleSupported("fr", supported))
}
