package google

import (
	"context"
	"errors"
	"net/http"

	"golang.org/x/oauth2"
	oauth2google "golang.org/x/oauth2/google"

	"github.com/kilip/omed/api/internal/platform/oauthx"
)

// ID is the unique identifier for the Google OAuth driver.
const ID = "google"

const defaultUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"

var defaultScopes = []string{
	"openid",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
}

// Config holds configuration for the Google OAuth driver.
type Config struct {
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
}

type settings struct {
	endpoint    oauth2.Endpoint
	userInfoURL string
}

// Option configures optional parameters for the Google driver.
type Option func(*settings)

// WithEndpoint overrides the default OAuth2 endpoint.
func WithEndpoint(endpoint oauth2.Endpoint) Option {
	return func(s *settings) {
		s.endpoint = endpoint
	}
}

// WithUserInfoURL overrides the default user info endpoint URL.
func WithUserInfoURL(url string) Option {
	return func(s *settings) {
		s.userInfoURL = url
	}
}

// Driver implements oauthx.Driver for Google OAuth2.
type Driver struct {
	*oauthx.Base
	userInfoURL string
}

var _ oauthx.Driver = (*Driver)(nil)

// New constructs a new Google OAuth driver.
func New(cfg Config, opts ...Option) *Driver {
	s := settings{
		endpoint:    oauth2google.Endpoint,
		userInfoURL: defaultUserInfoURL,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&s)
		}
	}

	base := oauthx.NewBase(ID, oauthx.BaseConfig{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint:     s.endpoint,
		Scopes:       defaultScopes,
		HTTPClient:   cfg.HTTPClient,
	})

	return &Driver{
		Base:        base,
		userInfoURL: s.userInfoURL,
	}
}

type googleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// FetchUserProfile retrieves and normalizes the user profile from Google.
func (d *Driver) FetchUserProfile(ctx context.Context, token *oauth2.Token) (*oauthx.Profile, error) {
	if token == nil {
		return nil, errors.New("oauthx/google: nil token")
	}

	var raw googleUserInfo
	if err := d.GetJSON(ctx, d.userInfoURL, token.AccessToken, &raw); err != nil {
		return nil, err
	}

	return &oauthx.Profile{
		ID:            raw.ID,
		Email:         raw.Email,
		Name:          raw.Name,
		Avatar:        raw.Picture,
		EmailVerified: raw.VerifiedEmail || raw.EmailVerified,
		Locale:        raw.Locale,
	}, nil
}
