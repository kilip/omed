package config

import (
	"context"
	"log"

	_ "github.com/joho/godotenv/autoload"
	envconfig "github.com/sethvargo/go-envconfig"
)

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
	// AttachmentAllowedTypes lists allowed file extensions for attachments.
	AttachmentAllowedTypes []string `env:"OMED_ATTACHMENT_ALLOWED_TYPES,separator=,default=.pdf,.png,.jpg,.jpeg,.webp,.csv,.xls,.xlsx"`
	// Env is the runtime environment (development, test, production). Defaults to production.
	Env string `env:"OMED_ENV,default=production"`
}

// IsTestRoutesEnabled returns true when the environment is development or test.
func (c Config) IsTestRoutesEnabled() bool {
	return c.Env == "development" || c.Env == "test"
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

	if len(config.AttachmentAllowedTypes) == 0 {
		config.AttachmentAllowedTypes = []string{".pdf", ".png", ".jpg", ".jpeg", ".webp", ".csv", ".xls", ".xlsx"}
	}

	return config
}
