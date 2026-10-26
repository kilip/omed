package config_test

import (
	"fmt"
	"os"
	"testing"
	"time"

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
	s.Require().NoError(os.Unsetenv("OMED_TRUSTED_ORIGINS"))
	s.T().Cleanup(func() {
		if hadOrigins {
			s.Require().NoError(os.Setenv("OMED_TRUSTED_ORIGINS", originalOrigins))
		}
	})

	originalRateLimit, hadRateLimit := os.LookupEnv("OMED_RATE_LIMIT_ENABLED")
	s.Require().NoError(os.Unsetenv("OMED_RATE_LIMIT_ENABLED"))
	s.T().Cleanup(func() {
		if hadRateLimit {
			s.Require().NoError(os.Setenv("OMED_RATE_LIMIT_ENABLED", originalRateLimit))
		}
	})

	originalReadTimeout, hadReadTimeout := os.LookupEnv("OMED_READ_TIMEOUT")
	s.Require().NoError(os.Unsetenv("OMED_READ_TIMEOUT"))
	s.T().Cleanup(func() {
		if hadReadTimeout {
			s.Require().NoError(os.Setenv("OMED_READ_TIMEOUT", originalReadTimeout))
		}
	})

	originalWriteTimeout, hadWriteTimeout := os.LookupEnv("OMED_WRITE_TIMEOUT")
	s.Require().NoError(os.Unsetenv("OMED_WRITE_TIMEOUT"))
	s.T().Cleanup(func() {
		if hadWriteTimeout {
			s.Require().NoError(os.Setenv("OMED_WRITE_TIMEOUT", originalWriteTimeout))
		}
	})

	originalIdleTimeout, hadIdleTimeout := os.LookupEnv("OMED_IDLE_TIMEOUT")
	s.Require().NoError(os.Unsetenv("OMED_IDLE_TIMEOUT"))
	s.T().Cleanup(func() {
		if hadIdleTimeout {
			s.Require().NoError(os.Setenv("OMED_IDLE_TIMEOUT", originalIdleTimeout))
		}
	})

	originalCmsMediaMaxSize, hadCmsMediaMaxSize := os.LookupEnv("OMED_CMS_MEDIA_MAX_SIZE_MB")
	s.Require().NoError(os.Unsetenv("OMED_CMS_MEDIA_MAX_SIZE_MB"))
	s.T().Cleanup(func() {
		if hadCmsMediaMaxSize {
			s.Require().NoError(os.Setenv("OMED_CMS_MEDIA_MAX_SIZE_MB", originalCmsMediaMaxSize))
		}
	})

	originalPort, hadPort := os.LookupEnv("OMED_API_PORT")
	s.Require().NoError(os.Unsetenv("OMED_API_PORT"))
	s.T().Cleanup(func() {
		if hadPort {
			s.Require().NoError(os.Setenv("OMED_API_PORT", originalPort))
		}
	})

	cfg := config.GetConfig()
	s.Equal(8001, cfg.Port)
	s.Equal("/auth", cfg.AuthPath)
	s.Equal("omed_session", cfg.SessionCookieName)
	s.Equal([]string{"http://localhost:3000"}, cfg.TrustedOrigins)
	s.Equal(10*time.Second, cfg.ReadTimeout)
	s.Equal(30*time.Second, cfg.WriteTimeout)
	s.Equal(120*time.Second, cfg.IdleTimeout)
	s.True(cfg.RateLimitEnabled)
	s.Equal(100, cfg.RateLimitGeneralMax)
	s.Equal(1*time.Minute, cfg.RateLimitGeneralExpiration)
	s.Equal(10, cfg.RateLimitSensitiveMax)
	s.Equal(1*time.Minute, cfg.RateLimitSensitiveExpiration)
	s.Equal(int64(25), cfg.CMSMediaMaxSizeMB)
	s.Equal([]string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".pdf"}, cfg.CMSMediaAllowedTypes)
}

func (s *ConfigSuite) Test_GetConfig_CustomValues() {
	s.T().Setenv("OMED_API_PORT", "9001")
	s.T().Setenv("OMED_AUTH_PATH", "/api/v1/auth")
	s.T().Setenv("OMED_SESSION_COOKIE_NAME", "custom_session_token")
	s.T().Setenv("OMED_TRUSTED_ORIGINS", "https://app1.example.com,https://app2.example.com")
	s.T().Setenv("OMED_READ_TIMEOUT", "15s")
	s.T().Setenv("OMED_WRITE_TIMEOUT", "45s")
	s.T().Setenv("OMED_IDLE_TIMEOUT", "60s")
	s.T().Setenv("OMED_RATE_LIMIT_ENABLED", "false")
	s.T().Setenv("OMED_RATE_LIMIT_GENERAL_MAX", "250")
	s.T().Setenv("OMED_RATE_LIMIT_GENERAL_EXPIRATION", "2m")
	s.T().Setenv("OMED_RATE_LIMIT_SENSITIVE_MAX", "20")
	s.T().Setenv("OMED_RATE_LIMIT_SENSITIVE_EXPIRATION", "30s")
	s.T().Setenv("OMED_CMS_MEDIA_MAX_SIZE_MB", "50")
	s.T().Setenv("OMED_CMS_MEDIA_ALLOWED_TYPES", ".png,.jpg")

	cfg := config.GetConfig()
	s.Equal(9001, cfg.Port)
	s.Equal("/api/v1/auth", cfg.AuthPath)
	s.Equal("custom_session_token", cfg.SessionCookieName)
	s.Equal([]string{"https://app1.example.com", "https://app2.example.com"}, cfg.TrustedOrigins)
	s.Equal(15*time.Second, cfg.ReadTimeout)
	s.Equal(45*time.Second, cfg.WriteTimeout)
	s.Equal(60*time.Second, cfg.IdleTimeout)
	s.False(cfg.RateLimitEnabled)
	s.Equal(250, cfg.RateLimitGeneralMax)
	s.Equal(2*time.Minute, cfg.RateLimitGeneralExpiration)
	s.Equal(20, cfg.RateLimitSensitiveMax)
	s.Equal(30*time.Second, cfg.RateLimitSensitiveExpiration)
	s.Equal(int64(50), cfg.CMSMediaMaxSizeMB)
	s.Equal([]string{".png", ".jpg"}, cfg.CMSMediaAllowedTypes)
}

func (s *ConfigSuite) Test_GetConfig_Fallbacks() {
	s.T().Setenv("OMED_CMS_MEDIA_MAX_SIZE_MB", "0")
	s.T().Setenv("OMED_CMS_MEDIA_ALLOWED_TYPES", "")

	cfg := config.GetConfig()
	s.Equal(int64(25), cfg.CMSMediaMaxSizeMB)
	s.Equal([]string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".pdf"}, cfg.CMSMediaAllowedTypes)
}

func (s *ConfigSuite) Test_GetConfig_Port_Fallback() {
	s.T().Setenv("OMED_API_PORT", "0")
	cfg := config.GetConfig()
	s.Equal(8001, cfg.Port)

	s.T().Setenv("OMED_API_PORT", "-5")
	cfg = config.GetConfig()
	s.Equal(8001, cfg.Port)
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
