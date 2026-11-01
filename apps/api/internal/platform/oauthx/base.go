package oauthx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

// BaseConfig holds common configuration for OAuth2 drivers.
type BaseConfig struct {
	ClientID     string
	ClientSecret string
	Endpoint     oauth2.Endpoint
	Scopes       []string
	HTTPClient   *http.Client
}

// Base provides shared OAuth2 driver logic.
type Base struct {
	id           string
	clientID     string
	clientSecret string
	endpoint     oauth2.Endpoint
	scopes       []string
	httpClient   *http.Client
}

// NewBase constructs a new Base OAuth2 driver helper.
func NewBase(id string, cfg BaseConfig) *Base {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	scopes := make([]string, len(cfg.Scopes))
	copy(scopes, cfg.Scopes)

	return &Base{
		id:           id,
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
		endpoint:     cfg.Endpoint,
		scopes:       scopes,
		httpClient:   client,
	}
}

// ID returns the provider identifier.
func (b *Base) ID() string {
	return b.id
}

// Scopes returns a copy of the configured OAuth scopes.
func (b *Base) Scopes() []string {
	out := make([]string, len(b.scopes))
	copy(out, b.scopes)
	return out
}

func (b *Base) oauth2Config(redirectURL string) oauth2.Config {
	return oauth2.Config{
		ClientID:     b.clientID,
		ClientSecret: b.clientSecret,
		Endpoint:     b.endpoint,
		RedirectURL:  redirectURL,
		Scopes:       b.scopes,
	}
}

func (b *Base) client() *http.Client {
	if b.httpClient != nil {
		return b.httpClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// GetAuthURL builds the provider authorization URL with offline access.
func (b *Base) GetAuthURL(state, redirectURL string) string {
	cfg := b.oauth2Config(redirectURL)
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCode exchanges an authorization code for a token using the driver's HTTP client.
func (b *Base) ExchangeCode(ctx context.Context, code, redirectURL string) (*oauth2.Token, error) {
	cfg := b.oauth2Config(redirectURL)
	exchangeCtx := context.WithValue(ctx, oauth2.HTTPClient, b.client())
	return cfg.Exchange(exchangeCtx, code)
}

// GetJSON performs an authenticated GET request with the bearer token and decodes the JSON response.
func (b *Base) GetJSON(ctx context.Context, url, accessToken string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := b.client().Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("provider returned status %d: %s", resp.StatusCode, string(body))
	}

	return json.Unmarshal(body, out)
}
