package config_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/kilip/omed/api/internal/config"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	suite.Suite
}

func (s *ConfigSuite) SetupTest() {
	s.T().Setenv("OMED_AUTH_SECRET", "test-secret-at-least-32-chars-long!")
}

func (s *ConfigSuite) Test_GetConfig_Defaults() {
	s.T().Setenv("OMED_AUTH_PATH", "")
	s.T().Setenv("OMED_SESSION_COOKIE_NAME", "")

	// Ensure default TrustedOrigins is tested
	originalOrigins, hadOrigins := os.LookupEnv("OMED_TRUSTED_ORIGINS")
	os.Unsetenv("OMED_TRUSTED_ORIGINS")
	s.T().Cleanup(func() {
		if hadOrigins {
			os.Setenv("OMED_TRUSTED_ORIGINS", originalOrigins)
		}
	})

	cfg := config.GetConfig()
	s.Equal("/auth", cfg.AuthPath)
	s.Equal("omed_session", cfg.SessionCookieName)
	s.Equal([]string{"http://localhost:3000"}, cfg.TrustedOrigins)
	s.Equal(int64(10), cfg.CMSMediaMaxSizeMB)
	s.Equal([]string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".pdf"}, cfg.CMSMediaAllowedTypes)
}

func (s *ConfigSuite) Test_GetConfig_CustomValues() {
	s.T().Setenv("OMED_AUTH_PATH", "/api/v1/auth")
	s.T().Setenv("OMED_SESSION_COOKIE_NAME", "custom_session_token")
	s.T().Setenv("OMED_TRUSTED_ORIGINS", "https://app1.example.com,https://app2.example.com")
	s.T().Setenv("OMED_CMS_MEDIA_MAX_SIZE_MB", "25")
	s.T().Setenv("OMED_CMS_MEDIA_ALLOWED_TYPES", ".png,.jpg")

	cfg := config.GetConfig()
	s.Equal("/api/v1/auth", cfg.AuthPath)
	s.Equal("custom_session_token", cfg.SessionCookieName)
	s.Equal([]string{"https://app1.example.com", "https://app2.example.com"}, cfg.TrustedOrigins)
	s.Equal(int64(25), cfg.CMSMediaMaxSizeMB)
	s.Equal([]string{".png", ".jpg"}, cfg.CMSMediaAllowedTypes)
}

func (s *ConfigSuite) Test_GetConfig_Fallbacks() {
	s.T().Setenv("OMED_CMS_MEDIA_MAX_SIZE_MB", "0")
	s.T().Setenv("OMED_CMS_MEDIA_ALLOWED_TYPES", "")

	cfg := config.GetConfig()
	s.Equal(int64(10), cfg.CMSMediaMaxSizeMB)
	s.Equal([]string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".pdf"}, cfg.CMSMediaAllowedTypes)
}

func (s *ConfigSuite) Test_Validate() {
	// Production mode requires at least 32 bytes
	prodCfg := config.Config{Env: "production", AuthSecret: ""}
	s.EqualError(prodCfg.Validate(), "OMED_AUTH_SECRET must be at least 32 bytes in production")

	prodCfgShort := config.Config{Env: "production", AuthSecret: "too-short"}
	s.EqualError(prodCfgShort.Validate(), "OMED_AUTH_SECRET must be at least 32 bytes in production")

	prodCfgValid := config.Config{Env: "production", AuthSecret: "12345678901234567890123456789012"}
	s.NoError(prodCfgValid.Validate())

	// Non-production modes allow empty or short secret
	devCfg := config.Config{Env: "development", AuthSecret: ""}
	s.NoError(devCfg.Validate())

	testCfg := config.Config{Env: "test", AuthSecret: "short"}
	s.NoError(testCfg.Validate())
}

func (s *ConfigSuite) Test_GetConfig_Production_Requires32ByteSecret() {
	var fatalMsg string
	restore := config.SetLogFatalf(func(format string, v ...any) {
		fatalMsg = fmt.Sprintf(format, v...)
		panic("fatal: " + fatalMsg)
	})
	defer restore()

	s.T().Setenv("OMED_ENV", "production")

	// Empty secret
	s.T().Setenv("OMED_AUTH_SECRET", "")
	s.Panics(func() {
		config.GetConfig()
	})
	s.Contains(fatalMsg, "OMED_AUTH_SECRET must be at least 32 bytes in production")

	// Shorter than 32 bytes
	fatalMsg = ""
	s.T().Setenv("OMED_AUTH_SECRET", "too-short-secret")
	s.Panics(func() {
		config.GetConfig()
	})
	s.Contains(fatalMsg, "OMED_AUTH_SECRET must be at least 32 bytes in production")

	// Exactly 32 bytes should pass
	s.T().Setenv("OMED_AUTH_SECRET", "12345678901234567890123456789012")
	s.NotPanics(func() {
		cfg := config.GetConfig()
		s.Equal("12345678901234567890123456789012", cfg.AuthSecret)
	})
}

func (s *ConfigSuite) Test_GetConfig_NonProduction_AllowsShortSecret() {
	for _, env := range []string{"development", "test"} {
		s.T().Setenv("OMED_ENV", env)
		s.T().Setenv("OMED_AUTH_SECRET", "short")
		s.NotPanics(func() {
			cfg := config.GetConfig()
			s.Equal("short", cfg.AuthSecret)
		})
	}
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}
