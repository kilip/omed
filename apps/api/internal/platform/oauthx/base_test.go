package oauthx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type countingTransport struct {
	base  http.RoundTripper
	count atomic.Int32
}

func (c *countingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.count.Add(1)
	return c.base.RoundTrip(req)
}

func TestBase_ID(t *testing.T) {
	b := NewBase("google", BaseConfig{})
	assert.Equal(t, "google", b.ID())
}

func TestBase_GetAuthURL_IncludesParams(t *testing.T) {
	b := NewBase("test", BaseConfig{
		ClientID: "cid",
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://provider.example.com/oauth/auth",
			TokenURL: "https://provider.example.com/oauth/token",
		},
	})

	redirectURL := "https://api.example.com/auth/callback/test"
	got := b.GetAuthURL("st", redirectURL)

	assert.True(t, strings.HasPrefix(got, "https://provider.example.com/oauth/auth"))
	parsed, err := url.Parse(got)
	require.NoError(t, err)

	q := parsed.Query()
	assert.Equal(t, "cid", q.Get("client_id"))
	assert.Equal(t, "st", q.Get("state"))
	assert.Equal(t, "offline", q.Get("access_type"))
	assert.Equal(t, redirectURL, q.Get("redirect_uri"))
}

func TestBase_ExchangeCode_UsesInjectedHTTPClient(t *testing.T) {
	var requestedRedirectURI string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())
		requestedRedirectURI = r.Form.Get("redirect_uri")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"custom-token","token_type":"bearer"}`))
	}))
	defer srv.Close()

	transport := &countingTransport{base: http.DefaultTransport}
	customClient := &http.Client{Transport: transport}

	b := NewBase("test", BaseConfig{
		ClientID:     "cid",
		ClientSecret: "sec",
		Endpoint: oauth2.Endpoint{
			TokenURL: srv.URL,
		},
		HTTPClient: customClient,
	})

	redirectURL := "https://api.example.com/callback"
	token, err := b.ExchangeCode(context.Background(), "test-code", redirectURL)
	require.NoError(t, err)
	assert.Equal(t, int32(1), transport.count.Load())
	assert.Equal(t, "custom-token", token.AccessToken)
	assert.Equal(t, redirectURL, requestedRedirectURI)
}

func TestBase_ExchangeCode_ProviderRejects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()

	b := NewBase("test", BaseConfig{
		Endpoint: oauth2.Endpoint{
			TokenURL: srv.URL,
		},
		HTTPClient: srv.Client(),
	})

	token, err := b.ExchangeCode(context.Background(), "bad-code", "https://example.com/cb")
	assert.Error(t, err)
	assert.Nil(t, token)
}

func TestBase_GetJSON_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer tok", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"user-1","name":"Alice"}`))
	}))
	defer srv.Close()

	b := NewBase("test", BaseConfig{
		HTTPClient: srv.Client(),
	})

	var got struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	err := b.GetJSON(context.Background(), srv.URL, "tok", &got)
	require.NoError(t, err)
	assert.Equal(t, "user-1", got.ID)
	assert.Equal(t, "Alice", got.Name)
}

func TestBase_GetJSON_NonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("boom"))
	}))
	defer srv.Close()

	b := NewBase("test", BaseConfig{
		HTTPClient: srv.Client(),
	})

	var got map[string]any
	err := b.GetJSON(context.Background(), srv.URL, "tok", &got)
	require.Error(t, err)
	assert.EqualError(t, err, "provider returned status 502: boom")
}

func TestBase_GetJSON_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	b := NewBase("test", BaseConfig{
		HTTPClient: srv.Client(),
	})

	var got map[string]any
	err := b.GetJSON(context.Background(), srv.URL, "tok", &got)
	assert.Error(t, err)
}

func TestBase_Scopes_ReturnsCopy(t *testing.T) {
	original := []string{"email", "profile"}
	b := NewBase("test", BaseConfig{
		Scopes: original,
	})

	got := b.Scopes()
	assert.Equal(t, original, got)

	got[0] = "mutated"
	second := b.Scopes()
	assert.Equal(t, []string{"email", "profile"}, second)
}

func TestNewBase_DefaultsHTTPClientTimeout(t *testing.T) {
	b := NewBase("test", BaseConfig{
		HTTPClient: nil,
	})

	require.NotNil(t, b.httpClient)
	assert.Equal(t, 10*time.Second, b.httpClient.Timeout)
}
