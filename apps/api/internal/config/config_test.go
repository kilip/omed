package config_test

import (
	"os"
	"testing"

	"github.com/kilip/omed/api/internal/config"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	suite.Suite
}

func (s *ConfigSuite) Test_GetConfig_Defaults() {
	s.T().Setenv("OMED_AUTH_URL", "http://auth.local")
	s.T().Setenv("OMED_AUTH_PATH", "")
	s.T().Setenv("OMED_JWKS_URL", "")

	// Ensure default TrustedOrigins is tested
	originalOrigins, hadOrigins := os.LookupEnv("OMED_TRUSTED_ORIGINS")
	os.Unsetenv("OMED_TRUSTED_ORIGINS")
	s.T().Cleanup(func() {
		if hadOrigins {
			os.Setenv("OMED_TRUSTED_ORIGINS", originalOrigins)
		}
	})

	cfg := config.GetConfig()
	s.Equal("http://auth.local", cfg.AuthUrl)
	s.Equal("/auth", cfg.AuthPath)
	s.Equal("http://auth.local/auth/jwks", cfg.JWKSUrl)
	s.Equal([]string{"http://localhost:3001"}, cfg.TrustedOrigins)
}

func (s *ConfigSuite) Test_GetConfig_CustomValues() {
	s.T().Setenv("OMED_AUTH_URL", "https://custom-auth.example.com")
	s.T().Setenv("OMED_AUTH_PATH", "/api/v1/auth")
	s.T().Setenv("OMED_JWKS_URL", "https://custom-auth.example.com/oauth/keys")
	s.T().Setenv("OMED_TRUSTED_ORIGINS", "https://app1.example.com,https://app2.example.com")

	cfg := config.GetConfig()
	s.Equal("https://custom-auth.example.com", cfg.AuthUrl)
	s.Equal("/api/v1/auth", cfg.AuthPath)
	s.Equal("https://custom-auth.example.com/oauth/keys", cfg.JWKSUrl)
	s.Equal([]string{"https://app1.example.com", "https://app2.example.com"}, cfg.TrustedOrigins)
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}
