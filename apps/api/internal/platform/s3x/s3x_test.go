package s3x_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kilip/omed/api/internal/platform/s3x"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type S3XSuite struct {
	suite.Suite
}

func (s *S3XSuite) Test_Config_DefaultsAndWorkspaceBucket() {
	cfg := s3x.Config{}
	s.Equal(s3x.DefaultBucketPrefix, cfg.NormalizedBucketPrefix())
	s.Equal(s3x.DefaultPresignTTL, cfg.NormalizedPresignTTL())
	s.Equal("omed-ws-42", cfg.WorkspaceBucket(42))

	customCfg := s3x.Config{
		BucketPrefix: " Custom-Prefix- ",
		PresignTTL:   5 * time.Minute,
	}
	s.Equal("custom-prefix-99", customCfg.WorkspaceBucket(99))
	s.Equal(5*time.Minute, customCfg.NormalizedPresignTTL())
}

func (s *S3XSuite) Test_MemoryDriver_FullLifecycle() {
	ctx := context.Background()
	st, err := s3x.New(s3x.Config{Driver: "memory"})
	s.Require().NoError(err)
	defer func() { _ = st.Close() }()

	// Invalid workspace ID
	_, err = st.EnsureWorkspaceBucket(ctx, 0)
	s.Error(err)
	s.Error(st.DeleteWorkspaceBucket(ctx, -1))

	// Ensure workspace bucket
	bucket, err := st.EnsureWorkspaceBucket(ctx, 10)
	s.Require().NoError(err)
	s.Equal("omed-ws-10", bucket)

	// Idempotent EnsureBucket
	s.Require().NoError(st.EnsureBucket(ctx, bucket))

	exists, err := st.BucketExists(ctx, bucket)
	s.Require().NoError(err)
	s.True(exists)

	// PutObject
	payload := []byte("hello omed s3 storage")
	info, err := st.PutObject(ctx, bucket, "docs/readme.txt", bytes.NewReader(payload), int64(len(payload)), "text/plain")
	s.Require().NoError(err)
	s.Equal(bucket, info.Bucket)
	s.Equal("docs/readme.txt", info.Key)
	s.Equal(int64(len(payload)), info.Size)
	s.Equal("text/plain", info.ContentType)
	s.NotEmpty(info.ETag)

	// StatObject
	stat, err := st.StatObject(ctx, bucket, "docs/readme.txt")
	s.Require().NoError(err)
	s.Equal(info.Size, stat.Size)
	s.Equal("text/plain", stat.ContentType)

	// GetObject
	obj, err := st.GetObject(ctx, bucket, "docs/readme.txt")
	s.Require().NoError(err)
	defer func() { _ = obj.Body.Close() }()
	readBytes, err := io.ReadAll(obj.Body)
	s.Require().NoError(err)
	s.Equal(payload, readBytes)

	// PresignGetObject
	presignedURL, err := st.PresignGetObject(ctx, bucket, "docs/readme.txt", 0)
	s.Require().NoError(err)
	s.Contains(presignedURL, "omed-ws-10")
	s.Contains(presignedURL, "X-Amz-Expires=900")

	// Non-force DeleteBucket fails when not empty
	err = st.DeleteBucket(ctx, bucket, false)
	s.ErrorIs(err, s3x.ErrBucketNotEmpty)

	// DeleteObject
	s.Require().NoError(st.DeleteObject(ctx, bucket, "docs/readme.txt"))
	_, err = st.GetObject(ctx, bucket, "docs/readme.txt")
	s.ErrorIs(err, s3x.ErrObjectNotFound)

	// Force DeleteWorkspaceBucket removes non-empty bucket
	_, err = st.PutObject(ctx, bucket, "temp.bin", bytes.NewReader([]byte{1, 2, 3}), 3, "")
	s.Require().NoError(err)
	s.Require().NoError(st.DeleteWorkspaceBucket(ctx, 10))

	exists, err = st.BucketExists(ctx, bucket)
	s.Require().NoError(err)
	s.False(exists)

	// Reset on MemoryDriver
	md, ok := st.Driver().(*s3x.MemoryDriver)
	s.Require().True(ok)
	_, err = st.EnsureWorkspaceBucket(ctx, 20)
	s.Require().NoError(err)
	md.Reset()
	exists, err = st.BucketExists(ctx, "omed-ws-20")
	s.Require().NoError(err)
	s.False(exists)
}

func (s *S3XSuite) Test_MemoryDriver_ErrorCases() {
	ctx := context.Background()
	md := s3x.NewMemoryDriver()

	s.Error(md.EnsureBucket(ctx, ""))
	_, err := md.PutObject(ctx, "", "k", bytes.NewReader(nil), 0, "")
	s.Error(err)
	_, err = md.PutObject(ctx, "b", "k", nil, 0, "")
	s.Error(err)
	_, err = md.PutObject(ctx, "missing", "k", bytes.NewReader([]byte("x")), 1, "")
	s.ErrorIs(err, s3x.ErrBucketNotFound)
	_, err = md.GetObject(ctx, "missing", "k")
	s.ErrorIs(err, s3x.ErrBucketNotFound)
	_, err = md.StatObject(ctx, "missing", "k")
	s.ErrorIs(err, s3x.ErrBucketNotFound)
	_, err = md.PresignGetObject(ctx, "missing", "k", time.Minute)
	s.ErrorIs(err, s3x.ErrBucketNotFound)
	s.NoError(md.DeleteObject(ctx, "missing", "k"))
	s.NoError(md.DeleteBucket(ctx, "missing", false))

	cctx, cancel := context.WithCancel(ctx)
	cancel()
	s.Error(md.EnsureBucket(cctx, "b"))
	_, err = md.BucketExists(cctx, "b")
	s.Error(err)
	s.Error(md.DeleteBucket(cctx, "b", true))
	_, err = md.PutObject(cctx, "b", "k", bytes.NewReader(nil), 0, "")
	s.Error(err)
	_, err = md.GetObject(cctx, "b", "k")
	s.Error(err)
	_, err = md.StatObject(cctx, "b", "k")
	s.Error(err)
	s.Error(md.DeleteObject(cctx, "b", "k"))
	_, err = md.PresignGetObject(cctx, "b", "k", time.Minute)
	s.Error(err)
}

// mockS3Server provides a minimal path-style S3 HTTP server for testing S3Driver deterministically.
type mockS3Server struct {
	mu      sync.Mutex
	buckets map[string]map[string]struct {
		data        []byte
		contentType string
	}
}

func newMockS3Server() *httptest.Server {
	srv := &mockS3Server{
		buckets: make(map[string]map[string]struct {
			data        []byte
			contentType string
		}),
	}
	return httptest.NewServer(http.HandlerFunc(srv.handle))
}

func (m *mockS3Server) handle(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.SplitN(path, "/", 2)
	bucket := parts[0]
	key := ""
	if len(parts) > 1 {
		key = parts[1]
	}

	// Bucket-level operations
	if key == "" {
		// Batch DeleteObjects: POST /{bucket}?delete
		if r.Method == http.MethodPost && r.URL.Query().Has("delete") {
			m.buckets[bucket] = make(map[string]struct {
				data        []byte
				contentType string
			})
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><DeleteResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></DeleteResult>`))
			return
		}

		// ListObjectsV2: GET /{bucket}?list-type=2
		if r.Method == http.MethodGet {
			objs, ok := m.buckets[bucket]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/xml")
			var sb strings.Builder
			sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><IsTruncated>false</IsTruncated>`)
			for k, v := range objs {
				fmt.Fprintf(&sb, `<Contents><Key>%s</Key><Size>%d</Size></Contents>`, k, len(v.data))
			}
			sb.WriteString(`</ListBucketResult>`)
			_, _ = w.Write([]byte(sb.String()))
			return
		}

		switch r.Method {
		case http.MethodHead:
			if _, ok := m.buckets[bucket]; !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusOK)
		case http.MethodPut:
			if _, ok := m.buckets[bucket]; !ok {
				m.buckets[bucket] = make(map[string]struct {
					data        []byte
					contentType string
				})
			}
			w.WriteHeader(http.StatusOK)
		case http.MethodDelete:
			delete(m.buckets, bucket)
			w.WriteHeader(http.StatusNoContent)
		}
		return
	}

	// Object-level operations
	objs, ok := m.buckets[bucket]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	switch r.Method {
	case http.MethodPut:
		data, _ := io.ReadAll(r.Body)
		ct := r.Header.Get("Content-Type")
		if ct == "" {
			ct = "application/octet-stream"
		}
		objs[key] = struct {
			data        []byte
			contentType string
		}{data: data, contentType: ct}
		w.Header().Set("ETag", `"mock-etag-123"`)
		w.WriteHeader(http.StatusOK)
	case http.MethodHead:
		obj, exists := objs[key]
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", obj.contentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(obj.data)))
		w.Header().Set("ETag", `"mock-etag-123"`)
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		obj, exists := objs[key]
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", obj.contentType)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(obj.data)))
		w.Header().Set("ETag", `"mock-etag-123"`)
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(obj.data)
	case http.MethodDelete:
		delete(objs, key)
		w.WriteHeader(http.StatusNoContent)
	}
}

func (s *S3XSuite) Test_S3Driver_FullLifecycle() {
	srv := newMockS3Server()
	defer srv.Close()

	ctx := context.Background()
	st, err := s3x.New(s3x.Config{
		Driver: "s3",
		S3: s3x.S3Config{
			Endpoint:       srv.URL,
			AccessKey:      "testkey",
			SecretKey:      "testsecret",
			Region:         "us-east-1",
			ForcePathStyle: true,
		},
	})
	require.NoError(s.T(), err)
	defer func() { _ = st.Close() }()

	bucket, err := st.EnsureWorkspaceBucket(ctx, 77)
	s.Require().NoError(err)
	s.Equal("omed-ws-77", bucket)

	// Second EnsureWorkspaceBucket is idempotent
	_, err = st.EnsureWorkspaceBucket(ctx, 77)
	s.Require().NoError(err)

	data := []byte("invoice pdf content")
	info, err := st.PutObject(ctx, bucket, "invoices/inv-01.pdf", bytes.NewReader(data), int64(len(data)), "application/pdf")
	s.Require().NoError(err)
	s.Equal("mock-etag-123", info.ETag)

	stat, err := st.StatObject(ctx, bucket, "invoices/inv-01.pdf")
	s.Require().NoError(err)
	s.Equal(int64(len(data)), stat.Size)
	s.Equal("application/pdf", stat.ContentType)

	obj, err := st.GetObject(ctx, bucket, "invoices/inv-01.pdf")
	s.Require().NoError(err)
	body, err := io.ReadAll(obj.Body)
	_ = obj.Body.Close()
	s.Require().NoError(err)
	s.Equal(data, body)

	urlStr, err := st.PresignGetObject(ctx, bucket, "invoices/inv-01.pdf", 10*time.Minute)
	s.Require().NoError(err)
	s.Contains(urlStr, "omed-ws-77/invoices/inv-01.pdf")

	s.Require().NoError(st.DeleteObject(ctx, bucket, "invoices/inv-01.pdf"))
	_, err = st.GetObject(ctx, bucket, "invoices/inv-01.pdf")
	s.ErrorIs(err, s3x.ErrObjectNotFound)

	// Re-upload and test force DeleteWorkspaceBucket
	_, err = st.PutObject(ctx, bucket, "invoices/inv-02.pdf", bytes.NewReader(data), int64(len(data)), "application/pdf")
	s.Require().NoError(err)
	s.Require().NoError(st.DeleteWorkspaceBucket(ctx, 77))

	exists, err := st.BucketExists(ctx, bucket)
	s.Require().NoError(err)
	s.False(exists)
}

func TestS3XSuite(t *testing.T) {
	suite.Run(t, new(S3XSuite))
}
