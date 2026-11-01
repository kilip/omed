package core

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/platform/oauthx/github"
	"github.com/kilip/omed/api/internal/platform/oauthx/google"
)

func TestNewOAuthRegistry_RegistersConfiguredProviders(t *testing.T) {
	cfg := Config{
		GoogleID:     "gid",
		GoogleSecret: "gsec",
		GithubID:     "ghid",
		GithubSecret: "ghsec",
	}

	reg := newOAuthRegistry(cfg)

	assert.Equal(t, []string{"github", "google"}, reg.IDs())
}

func TestNewOAuthRegistry_SkipsIncompleteCredentials(t *testing.T) {
	testCases := []struct {
		name string
		cfg  Config
	}{
		{
			name: "empty config",
			cfg:  Config{},
		},
		{
			name: "google id only",
			cfg:  Config{GoogleID: "gid"},
		},
		{
			name: "google secret only",
			cfg:  Config{GoogleSecret: "gsec"},
		},
		{
			name: "github id only",
			cfg:  Config{GithubID: "ghid"},
		},
		{
			name: "github secret only",
			cfg:  Config{GithubSecret: "ghsec"},
		},
		{
			name: "google id and github secret only",
			cfg:  Config{GoogleID: "gid", GithubSecret: "ghsec"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reg := newOAuthRegistry(tc.cfg)
			assert.Empty(t, reg.IDs())
		})
	}
}

func TestOAuthDriverIDsMatchDomainConstants(t *testing.T) {
	assert.Equal(t, domain.ProviderGoogle, google.ID)
	assert.Equal(t, domain.ProviderGithub, github.ID)
}
