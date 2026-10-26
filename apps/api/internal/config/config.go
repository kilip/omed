package config

import (
	"context"
	"errors"
	"log"
	"time"

	_ "github.com/joho/godotenv/autoload"
	envconfig "github.com/sethvargo/go-envconfig"
)

var logFatalf = log.Fatalf

// Config holds the application-level environment configuration settings.
type Config struct {
	// AuthPath is the subpath mounted for authentication endpoints.
	AuthPath string `env:"OMED_AUTH_PATH"`
	// SessionCookieName is the cookie name used to store and retrieve the session token.
	SessionCookieName string `env:"OMED_SESSION_COOKIE_NAME,default=omed_session"`
	// TrustedOrigins lists allowed CORS origins for browser clients.
	TrustedOrigins []string `env:"OMED_TRUSTED_ORIGINS,separator=,default=http://localhost:3000"`
	// DatabaseUrl database server to be use in omed
	DatabaseUrl string `env:"OMED_DB_URL"`
	// AuthSecret is used to sign/verify tokens and OAuth state.
	AuthSecret string `env:"OMED_AUTH_SECRET"`
	// GoogleClientID is the OAuth2 client ID for Google authentication.
	GoogleClientID string `env:"OMED_GOOGLE_ID"`
	// GoogleClientSecret is the OAuth2 client secret for Google authentication.
	GoogleClientSecret string `env:"OMED_GOOGLE_SECRET"`
	// GithubClientID is the OAuth2 client ID for GitHub authentication.
	GithubClientID string `env:"OMED_GITHUB_ID"`
	// GithubClientSecret is the OAuth2 client secret for GitHub authentication.
	GithubClientSecret string `env:"OMED_GITHUB_SECRET"`
	// ApiUrl is the base URL for the API service.
	ApiUrl string `env:"OMED_API_URL,default=http://localhost:8001"`
	// WebUrl is the base URL for the web frontend client.
	WebUrl string `env:"OMED_WEB_URL,default=http://localhost:3000"`
	// MailerDriver specifies the email transport driver (memory, smtp). Defaults to memory.
	MailerDriver string `env:"OMED_MAILER_DRIVER,default=memory"`
	// MailerHost is the SMTP server hostname.
	MailerHost string `env:"OMED_MAILER_HOST,default=localhost"`
	// MailerPort is the SMTP server port.
	MailerPort int `env:"OMED_MAILER_PORT,default=1025"`
	// MailerUsername is the SMTP auth username.
	MailerUsername string `env:"OMED_MAILER_USER"`
	// MailerPassword is the SMTP auth password.
	MailerPassword string `env:"OMED_MAILER_PASSWORD"`
	// MailerFromAddress is the default sender email address.
	MailerFromAddress string `env:"OMED_MAILER_FROM_ADDRESS,default=no-reply@omed.local"`
	// MailerFromName is the default sender display name.
	MailerFromName string `env:"OMED_MAILER_FROM_NAME,default=Omed"`
	// MailerInsecureTLS allows self-signed TLS certificates for SMTP.
	MailerInsecureTLS bool `env:"OMED_MAILER_INSECURE_TLS,default=false"`
	// S3Driver specifies the object storage driver (memory, s3). Defaults to memory.
	S3Driver string `env:"OMED_S3_DRIVER,default=memory"`
	// S3Endpoint is the S3/RustFS endpoint URL.
	S3Endpoint string `env:"OMED_S3_ENDPOINT,default=http://localhost:9000"`
	// S3AccessKey is the S3 access key ID.
	S3AccessKey string `env:"OMED_S3_ACCESS_KEY"`
	// S3SecretKey is the S3 secret access key.
	S3SecretKey string `env:"OMED_S3_SECRET_KEY"`
	// S3Region is the S3 region name.
	S3Region string `env:"OMED_S3_REGION,default=us-east-1"`
	// S3ForcePathStyle enables path-style S3 addressing (required for RustFS/MinIO).
	S3ForcePathStyle bool `env:"OMED_S3_FORCE_PATH_STYLE,default=true"`
	// S3BucketPrefix is the prefix prepended to workspace IDs for workspace bucket names.
	S3BucketPrefix string `env:"OMED_S3_BUCKET_PREFIX,default=omed-ws-"`
	// AttachmentMaxSizeMB is the maximum allowed attachment file size in megabytes. Defaults to 10.
	AttachmentMaxSizeMB int64 `env:"OMED_ATTACHMENT_MAX_SIZE_MB,default=10"`
	// CmsMediaMaxSizeMB is the maximum allowed CMS media file size in megabytes. Defaults to 25.
	CmsMediaMaxSizeMB int64 `env:"OMED_CMS_MEDIA_MAX_SIZE_MB,default=25"`
	// AttachmentAllowedTypes lists allowed file extensions for attachments.
	AttachmentAllowedTypes []string `env:"OMED_ATTACHMENT_ALLOWED_TYPES,separator=,default=.pdf,.png,.jpg,.jpeg,.webp,.csv,.xls,.xlsx"`
	// CMSMediaMaxSizeMB is the maximum allowed CMS media file size in megabytes. Defaults to 10.
	CMSMediaMaxSizeMB int64 `env:"OMED_CMS_MEDIA_MAX_SIZE_MB,default=10"`
	// CMSMediaAllowedTypes lists allowed file extensions for CMS media uploads.
	CMSMediaAllowedTypes []string `env:"OMED_CMS_MEDIA_ALLOWED_TYPES,separator=,default=.png,.jpg,.jpeg,.webp,.gif,.pdf"`
	// Env is the runtime environment (development, test, production). Defaults to production.
	Env string `env:"OMED_ENV,default=production"`
	// ReadTimeout is the maximum duration for reading the entire request. Defaults to 10s.
	ReadTimeout time.Duration `env:"OMED_READ_TIMEOUT,default=10s"`
	// WriteTimeout is the maximum duration before timing out writes of the response. Defaults to 30s.
	WriteTimeout time.Duration `env:"OMED_WRITE_TIMEOUT,default=30s"`
	// IdleTimeout is the maximum amount of time to wait for the next request when keep-alive is enabled. Defaults to 120s.
	IdleTimeout time.Duration `env:"OMED_IDLE_TIMEOUT,default=120s"`
	// RateLimitEnabled toggles HTTP rate limiting. Defaults to true.
	RateLimitEnabled bool `env:"OMED_RATE_LIMIT_ENABLED,default=true"`
	// RateLimitGeneralMax is the maximum requests allowed per general expiration window. Defaults to 100.
	RateLimitGeneralMax int `env:"OMED_RATE_LIMIT_GENERAL_MAX,default=100"`
	// RateLimitGeneralExpiration is the expiration window for general routes. Defaults to 1m.
	RateLimitGeneralExpiration time.Duration `env:"OMED_RATE_LIMIT_GENERAL_EXPIRATION,default=1m"`
	// RateLimitSensitiveMax is the maximum requests allowed per sensitive expiration window. Defaults to 10.
	RateLimitSensitiveMax int `env:"OMED_RATE_LIMIT_SENSITIVE_MAX,default=10"`
	// RateLimitSensitiveExpiration is the expiration window for sensitive routes. Defaults to 1m.
	RateLimitSensitiveExpiration time.Duration `env:"OMED_RATE_LIMIT_SENSITIVE_EXPIRATION,default=1m"`
}

// IsTestRoutesEnabled returns true when the environment is development or test.
func (c Config) IsTestRoutesEnabled() bool {
	return c.Env == "development" || c.Env == "test"
}

// Validate checks application configuration integrity.
// In production mode, AuthSecret must be at least 32 bytes long.
func (c Config) Validate() error {
	if c.Env == "production" && len(c.AuthSecret) < 32 {
		return errors.New("OMED_AUTH_SECRET must be at least 32 bytes in production")
	}
	return nil
}

// GetConfig reads environment variables and returns a populated Config instance.
// It applies default fallback values for AuthPath and SessionCookieName if they are omitted.
func GetConfig() Config {
	var config Config

	if err := envconfig.Process(context.Background(), &config); err != nil {
		log.Fatalf("Error while parsing config: %s", err)
	}

	if config.Env == "" {
		config.Env = "production"
	}

	if err := config.Validate(); err != nil {
		logFatalf("%s", err)
	}

	if config.AuthPath == "" {
		config.AuthPath = "/auth"
	}

	if config.SessionCookieName == "" {
		config.SessionCookieName = "omed_session"
	}

	if config.WebUrl == "" {
		config.WebUrl = "http://localhost:3000"
	}

	if config.MailerDriver == "" {
		config.MailerDriver = "memory"
	}

	if config.MailerPort <= 0 {
		config.MailerPort = 1025
	}

	if config.MailerFromAddress == "" {
		config.MailerFromAddress = "no-reply@omed.local"
	}

	if config.MailerFromName == "" {
		config.MailerFromName = "Omed"
	}

	if config.S3Driver == "" {
		config.S3Driver = "memory"
	}

	if config.S3Endpoint == "" {
		config.S3Endpoint = "http://localhost:9000"
	}

	if config.S3Region == "" {
		config.S3Region = "us-east-1"
	}

	if config.S3BucketPrefix == "" {
		config.S3BucketPrefix = "omed-ws-"
	}

	if config.AttachmentMaxSizeMB <= 0 {
		config.AttachmentMaxSizeMB = 10
	}

	if config.CmsMediaMaxSizeMB <= 0 {
		config.CmsMediaMaxSizeMB = 25
	}

	if len(config.AttachmentAllowedTypes) == 0 {
		config.AttachmentAllowedTypes = []string{".pdf", ".png", ".jpg", ".jpeg", ".webp", ".csv", ".xls", ".xlsx"}
	}

	if config.CMSMediaMaxSizeMB <= 0 {
		config.CMSMediaMaxSizeMB = 10
	}

	if len(config.CMSMediaAllowedTypes) == 0 {
		config.CMSMediaAllowedTypes = []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".pdf"}
	}

	if config.ReadTimeout <= 0 {
		config.ReadTimeout = 10 * time.Second
	}

	if config.WriteTimeout <= 0 {
		config.WriteTimeout = 30 * time.Second
	}

	if config.IdleTimeout <= 0 {
		config.IdleTimeout = 120 * time.Second
	}

	if config.RateLimitGeneralMax <= 0 {
		config.RateLimitGeneralMax = 100
	}

	if config.RateLimitGeneralExpiration <= 0 {
		config.RateLimitGeneralExpiration = 1 * time.Minute
	}

	if config.RateLimitSensitiveMax <= 0 {
		config.RateLimitSensitiveMax = 10
	}

	if config.RateLimitSensitiveExpiration <= 0 {
		config.RateLimitSensitiveExpiration = 1 * time.Minute
	}

	return config
}
