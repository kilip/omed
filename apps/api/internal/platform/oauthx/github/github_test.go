package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	oauth2github "golang.org/x/oauth2/github"
)

func TestDriver_Defaults(t *testing.T) {
	d := New(Config{
		ClientID:     "github-client-id",
		ClientSecret: "github-client-secret",
	})

	assert.Equal(t, ID, d.ID())
	assert.Equal(t, []string{"read:user", "user:email"}, d.Scopes())

	authURL := d.GetAuthURL("state-123", "http://x/cb")
	assert.True(t, strings.HasPrefix(authURL, oauth2github.Endpoint.AuthURL),
		"expected auth URL to start with %s, got %s", oauth2github.Endpoint.AuthURL, authURL)
}

func TestDriver_WithOptions(t *testing.T) {
	customEndpoint := oauth2.Endpoint{
		AuthURL:  "https://custom-auth.example.com",
		TokenURL: "https://custom-token.example.com",
	}
	d := New(
		Config{
			ClientID:     "github-client-id",
			ClientSecret: "github-client-secret",
		},
		WithEndpoint(customEndpoint),
		WithUserInfoURL("https://custom.example.com/user"),
		WithEmailsURL("https://custom.example.com/user/emails"),
	)

	authURL := d.GetAuthURL("state-123", "http://x/cb")
	assert.True(t, strings.HasPrefix(authURL, "https://custom-auth.example.com"))
	assert.Equal(t, "https://custom.example.com/user", d.userInfoURL)
	assert.Equal(t, "https://custom.example.com/user/emails", d.emailsURL)
}

func TestFetchUserProfile_PrefersPrimaryVerifiedEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user":
			_, _ = w.Write([]byte(`{
				"id": 101,
				"login": "octocat",
				"name": "Mona Lisa Octocat",
				"email": "profile@example.com"
			}`))
		case "/user/emails":
			_, _ = w.Write([]byte(`[
				{"email": "n@x.com", "primary": false, "verified": true},
				{"email": "p@x.com", "primary": true, "verified": true}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	d := New(
		Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
		WithUserInfoURL(srv.URL+"/user"),
		WithEmailsURL(srv.URL+"/user/emails"),
	)

	profile, err := d.FetchUserProfile(context.Background(), &oauth2.Token{AccessToken: "test-token"})
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, "p@x.com", profile.Email)
	assert.True(t, profile.EmailVerified)
	assert.Equal(t, "101", profile.ID)
	assert.Equal(t, "Mona Lisa Octocat", profile.Name)
}

func TestFetchUserProfile_FallsBackToVerifiedNonPrimary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user":
			_, _ = w.Write([]byte(`{
				"id": 102,
				"login": "octocat",
				"email": ""
			}`))
		case "/user/emails":
			_, _ = w.Write([]byte(`[
				{"email": "u@x.com", "primary": true, "verified": false},
				{"email": "v@x.com", "primary": false, "verified": true}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	d := New(
		Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
		WithUserInfoURL(srv.URL+"/user"),
		WithEmailsURL(srv.URL+"/user/emails"),
	)

	profile, err := d.FetchUserProfile(context.Background(), &oauth2.Token{AccessToken: "test-token"})
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, "v@x.com", profile.Email)
	assert.True(t, profile.EmailVerified)
}

func TestFetchUserProfile_NoVerifiedEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user":
			_, _ = w.Write([]byte(`{
				"id": 103,
				"login": "octocat",
				"email": "e@x.com"
			}`))
		case "/user/emails":
			_, _ = w.Write([]byte(`[
				{"email": "other@x.com", "primary": true, "verified": false},
				{"email": "unverified@x.com", "primary": false, "verified": false}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	d := New(
		Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
		WithUserInfoURL(srv.URL+"/user"),
		WithEmailsURL(srv.URL+"/user/emails"),
	)

	profile, err := d.FetchUserProfile(context.Background(), &oauth2.Token{AccessToken: "test-token"})
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, "e@x.com", profile.Email)
	assert.False(t, profile.EmailVerified)
}

func TestFetchUserProfile_EmailsEndpointFailureIgnored(t *testing.T) {
	t.Run("status 500", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/user":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"id": 104,
					"login": "octocat",
					"email": "e@x.com"
				}`))
			case "/user/emails":
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("server error"))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		d := New(
			Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
			WithUserInfoURL(srv.URL+"/user"),
			WithEmailsURL(srv.URL+"/user/emails"),
		)

		profile, err := d.FetchUserProfile(context.Background(), &oauth2.Token{AccessToken: "test-token"})
		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, "e@x.com", profile.Email)
		assert.False(t, profile.EmailVerified)
	})

	t.Run("body not-json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/user":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{
					"id": 105,
					"login": "octocat",
					"email": "e@x.com"
				}`))
			case "/user/emails":
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte("not-json"))
			default:
				http.NotFound(w, r)
			}
		}))
		defer srv.Close()

		d := New(
			Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
			WithUserInfoURL(srv.URL+"/user"),
			WithEmailsURL(srv.URL+"/user/emails"),
		)

		profile, err := d.FetchUserProfile(context.Background(), &oauth2.Token{AccessToken: "test-token"})
		require.NoError(t, err)
		require.NotNil(t, profile)
		assert.Equal(t, "e@x.com", profile.Email)
		assert.False(t, profile.EmailVerified)
	})
}

func TestFetchUserProfile_NameFallsBackToLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user":
			_, _ = w.Write([]byte(`{
				"id": 106,
				"login": "octo",
				"name": ""
			}`))
		case "/user/emails":
			_, _ = w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	d := New(
		Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
		WithUserInfoURL(srv.URL+"/user"),
		WithEmailsURL(srv.URL+"/user/emails"),
	)

	profile, err := d.FetchUserProfile(context.Background(), &oauth2.Token{AccessToken: "test-token"})
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, "octo", profile.Name)
}

func TestFetchUserProfile_IDFormattedBase10(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user":
			_, _ = w.Write([]byte(`{
				"id": 2002,
				"login": "octo"
			}`))
		case "/user/emails":
			_, _ = w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	d := New(
		Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
		WithUserInfoURL(srv.URL+"/user"),
		WithEmailsURL(srv.URL+"/user/emails"),
	)

	profile, err := d.FetchUserProfile(context.Background(), &oauth2.Token{AccessToken: "test-token"})
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, "2002", profile.ID)
}

func TestFetchUserProfile_VerifiedFieldPrecedence(t *testing.T) {
	tests := []struct {
		name             string
		userPayload      string
		expectedVerified bool
	}{
		{
			name:             "email_verified true",
			userPayload:      `{"id": 1, "login": "octo", "email": "a@x.com", "email_verified": true}`,
			expectedVerified: true,
		},
		{
			name:             "only verified true",
			userPayload:      `{"id": 1, "login": "octo", "email": "a@x.com", "verified": true}`,
			expectedVerified: true,
		},
		{
			name:             "email_verified false with verified true",
			userPayload:      `{"id": 1, "login": "octo", "email": "a@x.com", "email_verified": false, "verified": true}`,
			expectedVerified: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/user":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(tt.userPayload))
				case "/user/emails":
					http.NotFound(w, r)
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()

			d := New(
				Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
				WithUserInfoURL(srv.URL+"/user"),
				WithEmailsURL(srv.URL+"/user/emails"),
			)

			profile, err := d.FetchUserProfile(context.Background(), &oauth2.Token{AccessToken: "test-token"})
			require.NoError(t, err)
			require.NotNil(t, profile)
			assert.Equal(t, tt.expectedVerified, profile.EmailVerified)
		})
	}
}

func TestFetchUserProfile_UserInfoNonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	d := New(
		Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
		WithUserInfoURL(srv.URL),
		WithEmailsURL(srv.URL+"/emails"),
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

func TestFetchUserProfile_AvatarMapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/user":
			_, _ = w.Write([]byte(`{
				"id": 1,
				"login": "octocat",
				"avatar_url": "https://avatars.githubusercontent.com/u/1"
			}`))
		case "/user/emails":
			_, _ = w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	d := New(
		Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
		WithUserInfoURL(srv.URL+"/user"),
		WithEmailsURL(srv.URL+"/user/emails"),
	)

	profile, err := d.FetchUserProfile(context.Background(), &oauth2.Token{AccessToken: "test-token"})
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, "https://avatars.githubusercontent.com/u/1", profile.Avatar)
}

func TestFetchUserProfile_SendsBearerToken(t *testing.T) {
	var userAuthHeader, emailsAuthHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			userAuthHeader = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id": 1, "login": "octo"}`))
		case "/user/emails":
			emailsAuthHeader = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	d := New(
		Config{ClientID: "cid", ClientSecret: "sec", HTTPClient: srv.Client()},
		WithUserInfoURL(srv.URL+"/user"),
		WithEmailsURL(srv.URL+"/user/emails"),
	)

	_, err := d.FetchUserProfile(context.Background(), &oauth2.Token{AccessToken: "secret-token"})
	require.NoError(t, err)
	assert.Equal(t, "Bearer secret-token", userAuthHeader)
	assert.Equal(t, "Bearer secret-token", emailsAuthHeader)
}
