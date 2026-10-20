package s3x

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"sync"
	"time"
)

type memoryObject struct {
	info ObjectInfo
	data []byte
}

// MemoryDriver stores buckets and objects in memory for unit/integration testing and local development.
type MemoryDriver struct {
	mu      sync.RWMutex
	buckets map[string]map[string]memoryObject
	closed  bool
}

// NewMemoryDriver constructs a fresh MemoryDriver instance.
func NewMemoryDriver() *MemoryDriver {
	return &MemoryDriver{
		buckets: make(map[string]map[string]memoryObject),
	}
}

// EnsureBucket creates an in-memory bucket if it does not already exist.
func (d *MemoryDriver) EnsureBucket(ctx context.Context, bucket string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if bucket == "" {
		return fmt.Errorf("s3x: bucket name is required")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.buckets[bucket]; !exists {
		d.buckets[bucket] = make(map[string]memoryObject)
	}
	return nil
}

// BucketExists reports whether a bucket exists in memory.
func (d *MemoryDriver) BucketExists(ctx context.Context, bucket string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	_, exists := d.buckets[bucket]
	return exists, nil
}

// DeleteBucket deletes an in-memory bucket. If the bucket does not exist, it returns nil.
// When force is false and the bucket contains objects, ErrBucketNotEmpty is returned.
func (d *MemoryDriver) DeleteBucket(ctx context.Context, bucket string, force bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	objs, exists := d.buckets[bucket]
	if !exists {
		return nil
	}
	if !force && len(objs) > 0 {
		return ErrBucketNotEmpty
	}
	delete(d.buckets, bucket)
	return nil
}

// PutObject stores an object in the specified in-memory bucket.
func (d *MemoryDriver) PutObject(ctx context.Context, bucket, key string, body io.Reader, _ int64, contentType string) (ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return ObjectInfo{}, err
	}
	if bucket == "" || key == "" {
		return ObjectInfo{}, fmt.Errorf("s3x: bucket and key are required")
	}
	if body == nil {
		return ObjectInfo{}, fmt.Errorf("s3x: object body is required")
	}

	data, err := io.ReadAll(body)
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("s3x: read object body: %w", err)
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	sum := sha256.Sum256(data)
	etag := hex.EncodeToString(sum[:16])

	d.mu.Lock()
	defer d.mu.Unlock()

	objs, exists := d.buckets[bucket]
	if !exists {
		return ObjectInfo{}, ErrBucketNotFound
	}

	info := ObjectInfo{
		Bucket:       bucket,
		Key:          key,
		Size:         int64(len(data)),
		ContentType:  contentType,
		ETag:         etag,
		LastModified: time.Now().UTC(),
	}
	objs[key] = memoryObject{
		info: info,
		data: data,
	}
	return info, nil
}

// GetObject retrieves an object and its body reader from the in-memory bucket.
func (d *MemoryDriver) GetObject(ctx context.Context, bucket, key string) (*Object, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	objs, exists := d.buckets[bucket]
	if !exists {
		return nil, ErrBucketNotFound
	}

	obj, ok := objs[key]
	if !ok {
		return nil, ErrObjectNotFound
	}

	cp := make([]byte, len(obj.data))
	copy(cp, obj.data)

	return &Object{
		ObjectInfo: obj.info,
		Body:       io.NopCloser(bytes.NewReader(cp)),
	}, nil
}

// StatObject returns metadata for an object in the in-memory bucket.
func (d *MemoryDriver) StatObject(ctx context.Context, bucket, key string) (ObjectInfo, error) {
	if err := ctx.Err(); err != nil {
		return ObjectInfo{}, err
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	objs, exists := d.buckets[bucket]
	if !exists {
		return ObjectInfo{}, ErrBucketNotFound
	}

	obj, ok := objs[key]
	if !ok {
		return ObjectInfo{}, ErrObjectNotFound
	}

	return obj.info, nil
}

// DeleteObject removes an object from the specified in-memory bucket.
func (d *MemoryDriver) DeleteObject(ctx context.Context, bucket, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	objs, exists := d.buckets[bucket]
	if !exists {
		return nil
	}

	delete(objs, key)
	return nil
}

// PresignGetObject generates a synthetic presigned URL for an object in the in-memory bucket.
func (d *MemoryDriver) PresignGetObject(ctx context.Context, bucket, key string, expires time.Duration) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	objs, exists := d.buckets[bucket]
	if !exists {
		return "", ErrBucketNotFound
	}
	if _, ok := objs[key]; !ok {
		return "", ErrObjectNotFound
	}

	u := fmt.Sprintf("http://memory.s3.local/%s/%s?X-Amz-Expires=%d",
		url.PathEscape(bucket),
		url.PathEscape(key),
		int(expires.Seconds()),
	)
	return u, nil
}

// Reset clears all buckets and objects from the in-memory driver.
func (d *MemoryDriver) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.buckets = make(map[string]map[string]memoryObject)
}

// Close marks the memory driver as closed.
func (d *MemoryDriver) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.closed = true
	return nil
}
