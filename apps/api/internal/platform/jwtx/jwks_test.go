package jwtx_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MicahParks/jwkset"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/jwtx"
	"github.com/stretchr/testify/suite"
)

type JWKSSuite struct {
	suite.Suite
	privateKey *rsa.PrivateKey
	keyID      string
	jwksServer *httptest.Server
	logBuf     *bytes.Buffer
	logger     *slog.Logger
}

func (s *JWKSSuite) SetupSuite() {
	var err error
	s.privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	s.Require().NoError(err)
	s.keyID = "test-jwks-key-id"

	jwk, err := jwkset.NewJWKFromKey(&s.privateKey.PublicKey, jwkset.JWKOptions{
		Metadata: jwkset.JWKMetadataOptions{
			KID: s.keyID,
		},
	})
	s.Require().NoError(err)

	jwksJSON, err := json.Marshal(jwkset.JWKSMarshal{
		Keys: []jwkset.JWKMarshal{jwk.Marshal()},
	})
	s.Require().NoError(err)

	s.jwksServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwksJSON)
	}))
}

func (s *JWKSSuite) TearDownSuite() {
	if s.jwksServer != nil {
		s.jwksServer.Close()
	}
}

func (s *JWKSSuite) SetupTest() {
	s.logBuf = &bytes.Buffer{}
	s.logger = slog.New(slog.NewTextHandler(s.logBuf, nil))
}

func (s *JWKSSuite) createToken(claims jwt.MapClaims, key *rsa.PrivateKey, kid string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	tokenString, err := token.SignedString(key)
	s.Require().NoError(err)
	return tokenString
}

func (s *JWKSSuite) setupApp(handler fiber.Handler) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			if errors.Is(err, base.ErrUnauthorized) {
				return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
			}
			return fiber.DefaultErrorHandler(c, err)
		},
	})
	app.Use(handler)
	app.All("/api/protected", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	app.All("/internal/health", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})
	return app
}

func (s *JWKSSuite) Test_Handler_InvalidURL() {
	handler, err := jwtx.Handler("://invalid-url", s.logger)
	s.Error(err)
	s.Nil(handler)
	s.Contains(err.Error(), "fetch JWKS from ://invalid-url")
}

func (s *JWKSSuite) Test_Handler_Success_ValidToken() {
	handler, err := jwtx.Handler(s.jwksServer.URL, s.logger)
	s.Require().NoError(err)
	s.Require().NotNil(handler)

	app := s.setupApp(handler)

	claims := jwt.MapClaims{
		"sub": "12345",
		"iss": "test-issuer",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr := s.createToken(claims, s.privateKey, s.keyID)

	req := httptest.NewRequest(fiber.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *JWKSSuite) Test_Handler_Bypass_Options() {
	handler, err := jwtx.Handler(s.jwksServer.URL, s.logger)
	s.Require().NoError(err)

	app := s.setupApp(handler)

	req := httptest.NewRequest(fiber.MethodOptions, "/api/protected", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *JWKSSuite) Test_Handler_Bypass_InternalPath() {
	handler, err := jwtx.Handler(s.jwksServer.URL, s.logger)
	s.Require().NoError(err)

	app := s.setupApp(handler)

	req := httptest.NewRequest(fiber.MethodGet, "/internal/health", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *JWKSSuite) Test_Handler_MissingAuthHeader() {
	handler, err := jwtx.Handler(s.jwksServer.URL, s.logger)
	s.Require().NoError(err)

	app := s.setupApp(handler)

	req := httptest.NewRequest(fiber.MethodGet, "/api/protected", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
	s.Contains(s.logBuf.String(), "JWKS Error")
}

func (s *JWKSSuite) Test_Handler_MalformedToken() {
	handler, err := jwtx.Handler(s.jwksServer.URL, s.logger)
	s.Require().NoError(err)

	app := s.setupApp(handler)

	req := httptest.NewRequest(fiber.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer not-a-valid-jwt-token")

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
	s.Contains(s.logBuf.String(), "JWKS Error")
}

func (s *JWKSSuite) Test_Handler_WrongSigningKey() {
	handler, err := jwtx.Handler(s.jwksServer.URL, s.logger)
	s.Require().NoError(err)

	app := s.setupApp(handler)

	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	s.Require().NoError(err)

	claims := jwt.MapClaims{
		"sub": "12345",
		"iss": "test-issuer",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr := s.createToken(claims, wrongKey, s.keyID)

	req := httptest.NewRequest(fiber.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
	s.Contains(s.logBuf.String(), "JWKS Error")
}

func (s *JWKSSuite) Test_Handler_UnknownKID() {
	handler, err := jwtx.Handler(s.jwksServer.URL, s.logger)
	s.Require().NoError(err)

	app := s.setupApp(handler)

	claims := jwt.MapClaims{
		"sub": "12345",
		"iss": "test-issuer",
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	tokenStr := s.createToken(claims, s.privateKey, "unknown-kid")

	req := httptest.NewRequest(fiber.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
	s.Contains(s.logBuf.String(), "JWKS Error")
}

func (s *JWKSSuite) Test_Handler_ExpiredToken() {
	handler, err := jwtx.Handler(s.jwksServer.URL, s.logger)
	s.Require().NoError(err)

	app := s.setupApp(handler)

	claims := jwt.MapClaims{
		"sub": "12345",
		"iss": "test-issuer",
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	}
	tokenStr := s.createToken(claims, s.privateKey, s.keyID)

	req := httptest.NewRequest(fiber.MethodGet, "/api/protected", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
	s.Contains(s.logBuf.String(), "JWKS Error")
}

func TestJWKSSuite(t *testing.T) {
	suite.Run(t, &JWKSSuite{})
}

// Unused import guard
var _ = io.Discard
