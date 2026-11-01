package core

import (
	"github.com/kilip/omed/api/internal/platform/oauthx"
	"github.com/kilip/omed/api/internal/platform/oauthx/github"
	"github.com/kilip/omed/api/internal/platform/oauthx/google"
)

func newOAuthRegistry(cfg Config) *oauthx.Registry {
	reg := oauthx.NewRegistry()
	if cfg.GoogleID != "" && cfg.GoogleSecret != "" {
		reg.Register(google.New(google.Config{
			ClientID:     cfg.GoogleID,
			ClientSecret: cfg.GoogleSecret,
		}))
	}
	if cfg.GithubID != "" && cfg.GithubSecret != "" {
		reg.Register(github.New(github.Config{
			ClientID:     cfg.GithubID,
			ClientSecret: cfg.GithubSecret,
		}))
	}
	return reg
}
