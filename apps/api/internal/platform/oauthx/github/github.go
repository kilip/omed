package github

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"
	oauth2github "golang.org/x/oauth2/github"

	"github.com/kilip/omed/api/internal/platform/oauthx"
)

// ID is the unique identifier for the GitHub OAuth driver.
const ID = "github"

const (
	defaultUserInfoURL = "https://api.github.com/user"
	defaultEmailsURL   = "https://api.github.com/user/emails"
)

var defaultScopes = []string{
	"read:user",
	"user:email",
}

// Config holds configuration for the GitHub OAuth driver.
type Config struct {
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
}

type settings struct {
	endpoint    oauth2.Endpoint
	userInfoURL string
	emailsURL   string
}

// Option configures optional parameters for the GitHub driver.
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

// WithEmailsURL overrides the default emails endpoint URL.
func WithEmailsURL(url string) Option {
	return func(s *settings) {
		s.emailsURL = url
	}
}

// Driver implements oauthx.Driver for GitHub OAuth2.
type Driver struct {
	*oauthx.Base
	userInfoURL string
	emailsURL   string
}

var _ oauthx.Driver = (*Driver)(nil)

// New constructs a new GitHub OAuth driver.
func New(cfg Config, opts ...Option) *Driver {
	s := settings{
		endpoint:    oauth2github.Endpoint,
		userInfoURL: defaultUserInfoURL,
		emailsURL:   defaultEmailsURL,
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
		emailsURL:   s.emailsURL,
	}
}

type githubUserInfo struct {
	ID            int64  `json:"id"`
	Login         string `json:"login"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	AvatarURL     string `json:"avatar_url"`
	EmailVerified *bool  `json:"email_verified"`
	Verified      *bool  `json:"verified"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// FetchUserProfile retrieves and normalizes the user profile from GitHub.
func (d *Driver) FetchUserProfile(ctx context.Context, token *oauth2.Token) (*oauthx.Profile, error) {
	if token == nil {
		return nil, errors.New("oauthx/github: nil token")
	}

	var raw githubUserInfo
	if err := d.GetJSON(ctx, d.userInfoURL, token.AccessToken, &raw); err != nil {
		return nil, err
	}

	userEmail := raw.Email
	emailVerified := false

	if raw.EmailVerified != nil {
		emailVerified = *raw.EmailVerified
	} else if raw.Verified != nil {
		emailVerified = *raw.Verified
	}

	if vEmail, ok := d.fetchVerifiedEmail(ctx, token.AccessToken); ok {
		userEmail = vEmail
		emailVerified = true
	}

	displayName := raw.Name
	if displayName == "" {
		displayName = raw.Login
	}

	return &oauthx.Profile{
		ID:            strconv.FormatInt(raw.ID, 10),
		Email:         userEmail,
		Name:          displayName,
		Avatar:        raw.AvatarURL,
		EmailVerified: emailVerified,
	}, nil
}

func (d *Driver) fetchVerifiedEmail(ctx context.Context, accessToken string) (string, bool) {
	var emails []githubEmail
	if err := d.GetJSON(ctx, d.emailsURL, accessToken, &emails); err != nil {
		return "", false
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, true
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email, true
		}
	}

	return "", false
}
