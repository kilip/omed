// Package s3x provides S3-compatible object storage abstractions, workspace bucket management,
// and in-memory / AWS S3 transport drivers.
package s3x

import (
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	// DefaultBucketPrefix is the default prefix for per-workspace S3 buckets.
	DefaultBucketPrefix = "omed-ws-"
	// DefaultPresignTTL is the default expiration duration for presigned download URLs.
	DefaultPresignTTL = 15 * time.Minute
	// DefaultRegion is the default S3 region when omitted.
	DefaultRegion = "us-east-1"
)

// S3Config holds connection settings for an S3-compatible object storage server.
type S3Config struct {
	Endpoint       string
	AccessKey      string
	SecretKey      string
	Region         string
	ForcePathStyle bool
}

// Config encapsulates configuration for the s3x storage subsystem.
type Config struct {
	Driver       string // "s3", "memory"
	BucketPrefix string
	PresignTTL   time.Duration
	S3           S3Config
	Logger       *slog.Logger
}

// NormalizedBucketPrefix returns the configured bucket prefix, falling back to DefaultBucketPrefix.
func (c Config) NormalizedBucketPrefix() string {
	prefix := strings.ToLower(strings.TrimSpace(c.BucketPrefix))
	if prefix == "" {
		return DefaultBucketPrefix
	}
	return prefix
}

// NormalizedPresignTTL returns the configured presign TTL, falling back to DefaultPresignTTL.
func (c Config) NormalizedPresignTTL() time.Duration {
	if c.PresignTTL <= 0 {
		return DefaultPresignTTL
	}
	return c.PresignTTL
}

// WorkspaceBucket returns the deterministic S3 bucket name for a given workspace ID.
func (c Config) WorkspaceBucket(workspaceID int64) string {
	return fmt.Sprintf("%s%d", c.NormalizedBucketPrefix(), workspaceID)
}
