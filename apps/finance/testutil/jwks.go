package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const (
	testKID           = "test-key-1"
	defaultJWKSURL    = "http://localhost:1234/jwks"
	jwksURLEnvVarName = "JWKS_URL"
)

// JWKSMock bundles a fake JWKS HTTP server with the private key used to sign
// tokens. Point jwtware.Config.JWKSetURLs at .URL and use .SignToken to mint tokens.
type JWKSMock struct {
	Server *httptest.Server
	URL    string
	key    *rsa.PrivateKey
}

// NewJWKSMock spins up a fake JWKS server matching the JWKS_URL env var
// (falls back to http://localhost:3000/api/auth/jwks if unset), serving a
// single RSA public key under KID testKID. Call defer mock.Server.Close()
// in your test.
//
// Since it binds to a fixed host:port taken from JWKS_URL, only one test
// using this helper can run at a time per process — run serially (not
// t.Parallel()) or with -p 1.
func NewJWKSMock() *JWKSMock {
	//t.Helper()

	rawURL := state.Config.DatabaseUrl
	if rawURL == "" {
		rawURL = defaultJWKSURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	//require.NoError(t, err, "invalid %s: %q", jwksURLEnvVarName, rawURL)

	addr := parsed.Host
	path := parsed.Path
	if path == "" {
		path = "/"
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	//require.NoError(t, err, "failed to generate RSA key")

	jwks := buildJWKS(key, testKID)

	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	})

	lis, err := net.Listen("tcp", addr)
	//require.NoError(t, err, "failed to bind %s (is it already in use?)", addr)

	srv := httptest.NewUnstartedServer(mux)
	srv.Listener.Close()
	srv.Listener = lis
	srv.Start()

	return &JWKSMock{
		Server: srv,
		URL:    rawURL,
		key:    key,
	}
}

// SignToken creates an RS256 JWT signed with the mock's private key, with
// "kid" set so the middleware can match it against the JWKS. Merges extraClaims
// on top of a default exp (1h from now).
func SignToken(t *testing.T, extraClaims jwt.MapClaims) string {
	t.Helper()

	claims := jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	for k, v := range extraClaims {
		claims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = testKID

	signed, err := token.SignedString(jwksMock.key)
	require.NoError(t, err, "failed to sign token")
	return signed
}

// buildJWKS converts an RSA public key into a minimal JWKS document.
// Exponent/modulus are base64url-encoded per RFC 7517.
func buildJWKS(key *rsa.PrivateKey, kid string) map[string]any {
	pub := key.PublicKey
	return map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"use": "sig",
				"alg": "RS256",
				"kid": kid,
				"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
			},
		},
	}
}
