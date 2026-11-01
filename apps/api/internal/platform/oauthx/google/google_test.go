package google

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	oauth2google "golang.org/x/oauth2/google"
)

func TestDriver_Defaults(t *testing.T) {
	d := New(Config{
		ClientID:     "google-client-id",
		ClientSecret: "google-client-secret",
	})

	assert.Equal(t, ID, d.ID())
	assert.Equal(t, []string{
		"openid",
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	}, d.Scopes())

	authURL := d.GetAuthURL("state-123", "http://x/cb")
	assert.True(t, strings.HasPrefix(authURL, oauth2google.Endpoint.AuthURL), "expected auth URL to start with %s, got %s", oauth2google.Endpoint.AuthURL, authURL)
}

func TestDriver_WithEndpoint(t *testing.T) {
	customEndpoint := oauth2.Endpoint{
		AuthURL:  "https://custom-auth.example.com",
		TokenURL: "https://custom-token.example.com",
	}
	d := New(Config{
		ClientID:     "google-client-id",
		ClientSecret: "google-client-secret",
	}, WithEndpoint(customEndpoint))

	authURL := d.GetAuthURL("state-123", "http://x/cb")
	assert.True(t, strings.HasPrefix(authURL, "https://custom-auth.example.com"))
}

func TestFetchUserProfile_MapsFields(t *testing.T) {
	payload := `{
		"id": "12345",
		"email": "user@example.com",
		"name": "Jane Doe",
		"picture": "https://example.com/avatar.png",
		"locale": "id",
		"verified_email": true
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	d := New(
		Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
		WithUserInfoURL(srv.URL),
	)

	profile, err := d.FetchUserProfile(context.Background(), &oauth2.Token{AccessToken: "test-token"})
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, "12345", profile.ID)
	assert.Equal(t, "user@example.com", profile.Email)
	assert.Equal(t, "Jane Doe", profile.Name)
	assert.Equal(t, "https://example.com/avatar.png", profile.Avatar)
	assert.Equal(t, "id", profile.Locale)
	assert.True(t, profile.EmailVerified)
}

func TestFetchUserProfile_EmailVerifiedAlias(t *testing.T) {
	payload := `{
		"id": "12345",
		"email": "user@example.com",
		"name": "Jane Doe",
		"email_verified": true
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	d := New(
		Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
		WithUserInfoURL(srv.URL),
	)

	profile, err := d.FetchUserProfile(context.Background(), &oauth2.Token{AccessToken: "test-token"})
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.True(t, profile.EmailVerified)
}

func TestFetchUserProfile_Unverified(t *testing.T) {
	payload := `{
		"id": "12345",
		"email": "user@example.com",
		"verified_email": false
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	d := New(
		Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
		WithUserInfoURL(srv.URL),
	)

	profile, err := d.FetchUserProfile(context.Background(), &oauth2.Token{AccessToken: "test-token"})
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.False(t, profile.EmailVerified)
}

func TestFetchUserProfile_SendsBearerToken(t *testing.T) {
	var capturedAuthHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	defer srv.Close()

	d := New(
		Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
		WithUserInfoURL(srv.URL),
	)

	_, err := d.FetchUserProfile(context.Background(), &oauth2.Token{AccessToken: "tok"})
	require.NoError(t, err)
	assert.Equal(t, "Bearer tok", capturedAuthHeader)
}

func TestFetchUserProfile_NonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	d := New(
		Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
		WithUserInfoURL(srv.URL),
	)

	profile, err := d.FetchUserProfile(context.Background(), &oauth2.Token{AccessToken: "tok"})
	assert.Error(t, err)
	assert.Nil(t, profile)
	assert.Contains(t, err.Error(), "provider returned status 500")
}

func TestFetchUserProfile_NilToken(t *testing.T) {
	d := New(Config{ClientID: "cid", ClientSecret: "sec"})
	profile, err := d.FetchUserProfile(context.Background(), nil)
	assert.Error(t, err)
	assert.Nil(t, profile)
}
