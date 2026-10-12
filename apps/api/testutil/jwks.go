package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"log"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

const (
	testKID           = "test-key-1"
	jwksURLEnvVarName = "JWKS_URL"
)

// JWKSMock bundles a fake JWKS HTTP server with the private key used to sign
// tokens. Point jwtware.Config.JWKSetURLs at .URL and use .SignToken to mint tokens.
type JWKSMock struct {
	Server *httptest.Server
	URL    string
	key    *rsa.PrivateKey
}

// NewJWKSMock spins up a fake JWKS server serving a single RSA public key under KID testKID.
func NewJWKSMock() *JWKSMock {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("failed to generate RSA key: %v", err)
	}

	jwks := buildJWKS(key, testKID)

	mux := http.NewServeMux()
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(jwks)
	})

	srv := httptest.NewServer(mux)

	return &JWKSMock{
		Server: srv,
		URL:    srv.URL + "/jwks",
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
		"iss": state.Config.AuthUrl,
		"aud": state.Config.AuthUrl,
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
