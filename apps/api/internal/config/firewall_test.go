package config_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/config"
	"github.com/stretchr/testify/suite"
)

type FirewallSuite struct {
	suite.Suite
	mockAuth *testAuthMock
	logger   *slog.Logger
}

type testAuthMock struct {
	server *httptest.Server
	key    *rsa.PrivateKey
	kid    string
	url    string
}

func newTestAuthMock(t *testing.T) *testAuthMock {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}
	kid := "test-firewall-key"

	jwk, err := jwkset.NewJWKFromKey(&key.PublicKey, jwkset.JWKOptions{
		Metadata: jwkset.JWKMetadataOptions{
			KID: kid,
		},
	})
	if err != nil {
		t.Fatalf("failed to create JWK: %v", err)
	}

	jwksJSON, err := json.Marshal(jwkset.JWKSMarshal{
		Keys: []jwkset.JWKMarshal{jwk.Marshal()},
	})
	if err != nil {
		t.Fatalf("failed to marshal JWKS: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwksJSON)
	}))

	return &testAuthMock{
		server: srv,
		key:    key,
		kid:    kid,
		url:    srv.URL + "/jwks",
	}
}

func (m *testAuthMock) signToken(t *testing.T, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = m.kid
	signed, err := token.SignedString(m.key)
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}
	return signed
}

func (s *FirewallSuite) SetupSuite() {
	s.mockAuth = newTestAuthMock(s.T())
	s.logger = slog.Default()
}

func (s *FirewallSuite) TearDownSuite() {
	if s.mockAuth != nil && s.mockAuth.server != nil {
		s.mockAuth.server.Close()
	}
}

func (s *FirewallSuite) Test_InitFirewall() {
	authURL := "http://auth.local"
	cfg := config.Config{
		JWKSUrl: s.mockAuth.url,
		AuthUrl: authURL,
	}
	app := config.GetFiber(cfg, s.logger)

	state := config.State{
		FiberApp: app,
		Logger:   s.logger,
		Config:   cfg,
	}

	config.InitFirewall(state)

	var capturedUser base.UserContext
	app.All("/internal/health", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})
	app.All("/api/protected", func(c fiber.Ctx) error {
		if u, ok := c.Locals(base.USER_CONTEXT_KEY).(base.UserContext); ok {
			capturedUser = u
		}
		return c.SendString("protected ok")
	})

	// 1. Bypass /internal/*
	reqInternal := httptest.NewRequest(fiber.MethodGet, "/internal/health", nil)
	respInternal, err := app.Test(reqInternal)
	s.Require().NoError(err)
	defer respInternal.Body.Close()
	s.Equal(fiber.StatusOK, respInternal.StatusCode)

	// 2. Bypass OPTIONS
	reqOpt := httptest.NewRequest(fiber.MethodOptions, "/api/protected", nil)
	respOpt, err := app.Test(reqOpt)
	s.Require().NoError(err)
	defer respOpt.Body.Close()
	s.Equal(fiber.StatusOK, respOpt.StatusCode)

	// 3. Unauthorized request without token
	reqNoToken := httptest.NewRequest(fiber.MethodGet, "/api/protected", nil)
	respNoToken, err := app.Test(reqNoToken)
	s.Require().NoError(err)
	defer respNoToken.Body.Close()
	s.Equal(fiber.StatusUnauthorized, respNoToken.StatusCode)

	// 4. Authorized request with valid token
	token := s.mockAuth.signToken(s.T(), jwt.MapClaims{
		"sub": "99",
		"iss": authURL,
		"aud": []string{authURL},
		"exp": time.Now().Add(time.Hour).Unix(),
	})

	reqAuth := httptest.NewRequest(fiber.MethodGet, "/api/protected", nil)
	reqAuth.Header.Set("Authorization", "Bearer "+token)

	respAuth, err := app.Test(reqAuth)
	s.Require().NoError(err)
	defer respAuth.Body.Close()
	s.Equal(fiber.StatusOK, respAuth.StatusCode)
	s.Equal(int64(99), capturedUser.ID)
}

func TestFirewallSuite(t *testing.T) {
	suite.Run(t, new(FirewallSuite))
}
