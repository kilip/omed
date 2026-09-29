package seed_coa_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	seed_coa "github.com/kilip/omed/finance/internal/shared/seed/coa"
)

func TestLoad(t *testing.T) {
	t.Run("success loading freelancer.en", func(t *testing.T) {
		accounts, err := seed_coa.Load("freelancer", "en")
		assert.NoError(t, err)
		assert.NotEmpty(t, accounts)

		// Check first account
		assert.Equal(t, "1000", accounts[0].Code)
		assert.Equal(t, "Assets", accounts[0].Name)
		assert.Equal(t, "asset", accounts[0].Type)
		assert.Nil(t, accounts[0].ParentCode)
	})

	t.Run("success loading freelancer.id", func(t *testing.T) {
		accounts, err := seed_coa.Load("freelancer", "id")
		assert.NoError(t, err)
		assert.NotEmpty(t, accounts)
	})

	t.Run("not found profile or lang", func(t *testing.T) {
		accounts, err := seed_coa.Load("unknown", "en")
		assert.Error(t, err)
		assert.Nil(t, accounts)
	})
}
