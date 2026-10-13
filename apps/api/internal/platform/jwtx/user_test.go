package jwtx_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MicahParks/jwkset"
	jwtware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/jwtx"
	"github.com/stretchr/testify/suite"
)

type UserSuite struct {
	suite.Suite
	secretKey []byte
	issuer    string
	audience  string
}

func (s *UserSuite) SetupSuite() {
	s.secretKey = []byte("super-secret-key-for-testing-1234")
	s.issuer = "https://auth.example.com"
	s.audience = "https://api.example.com"
}

func (s *UserSuite) createTokenWithClaims(claims jwt.Claims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.secretKey)
	s.Require().NoError(err)
	return tokenStr
}

func (s *UserSuite) createToken(claims jwt.MapClaims) string {
	return s.createTokenWithClaims(claims)
}

func (s *UserSuite) setupApp(claimsFactory jwt.Claims, capturedUser *base.UserContext) *fiber.App {
	app := fiber.New()

	jwtCfg := jwtware.Config{
		KeyFunc: func(token *jwt.Token) (any, error) {
			return s.secretKey, nil
		},
	}
	if claimsFactory != nil {
		jwtCfg.Claims = claimsFactory
	}

	app.Use(jwtware.New(jwtCfg))
	app.Use(jwtx.UserInjector(s.issuer, s.audience))

	app.All("/api/user", func(c fiber.Ctx) error {
		if capturedUser != nil {
			if u, ok := c.Locals(base.USER_CONTEXT_KEY).(base.UserContext); ok {
				*capturedUser = u
			}
		}
		return c.SendStatus(fiber.StatusOK)
	})
	app.All("/internal/test", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	return app
}

func (s *UserSuite) Test_UserInjector_Bypass_Options() {
	app := fiber.New()
	app.Use(jwtx.UserInjector(s.issuer, s.audience))
	app.All("/api/user", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(fiber.MethodOptions, "/api/user", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *UserSuite) Test_UserInjector_Bypass_InternalPath() {
	app := fiber.New()
	app.Use(jwtx.UserInjector(s.issuer, s.audience))
	app.All("/internal/test", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/internal/test", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *UserSuite) Test_UserInjector_NoTokenInContext() {
	app := fiber.New()
	app.Use(jwtx.UserInjector(s.issuer, s.audience))
	app.Get("/api/user", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/api/user", nil)
	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *UserSuite) Test_UserInjector_ClaimsNotMapClaims() {
	app := s.setupApp(&jwt.RegisteredClaims{}, nil)

	regClaims := &jwt.RegisteredClaims{
		Issuer:   s.issuer,
		Audience: jwt.ClaimStrings{s.audience},
		Subject:  "123",
	}
	tokenStr := s.createTokenWithClaims(regClaims)

	req := httptest.NewRequest(fiber.MethodGet, "/api/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *UserSuite) Test_UserInjector_IssuerMismatch() {
	app := s.setupApp(nil, nil)

	claims := jwt.MapClaims{
		"iss": "https://wrong-issuer.com",
		"aud": s.audience,
		"sub": "123",
	}
	tokenStr := s.createToken(claims)

	req := httptest.NewRequest(fiber.MethodGet, "/api/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *UserSuite) Test_UserInjector_MissingIssuer() {
	app := s.setupApp(nil, nil)

	claims := jwt.MapClaims{
		"aud": s.audience,
		"sub": "123",
	}
	tokenStr := s.createToken(claims)

	req := httptest.NewRequest(fiber.MethodGet, "/api/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *UserSuite) Test_UserInjector_AudienceMismatch() {
	app := s.setupApp(nil, nil)

	claims := jwt.MapClaims{
		"iss": s.issuer,
		"aud": "https://wrong-audience.com",
		"sub": "123",
	}
	tokenStr := s.createToken(claims)

	req := httptest.NewRequest(fiber.MethodGet, "/api/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *UserSuite) Test_UserInjector_MissingAudience() {
	app := s.setupApp(nil, nil)

	claims := jwt.MapClaims{
		"iss": s.issuer,
		"sub": "123",
	}
	tokenStr := s.createToken(claims)

	req := httptest.NewRequest(fiber.MethodGet, "/api/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *UserSuite) Test_UserInjector_AudienceSliceContainsExpected() {
	var capturedUser base.UserContext
	app := s.setupApp(nil, &capturedUser)

	claims := jwt.MapClaims{
		"iss":  s.issuer,
		"aud":  []any{"other-aud", s.audience},
		"sub":  "123",
		"name": "Slice Aud User",
	}
	tokenStr := s.createToken(claims)

	req := httptest.NewRequest(fiber.MethodGet, "/api/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.Equal(int64(123), capturedUser.ID)
	s.Equal("Slice Aud User", capturedUser.Name)
}

func (s *UserSuite) Test_UserInjector_InvalidSubject_NonNumeric() {
	app := s.setupApp(nil, nil)

	claims := jwt.MapClaims{
		"iss": s.issuer,
		"aud": s.audience,
		"sub": "not-a-number",
	}
	tokenStr := s.createToken(claims)

	req := httptest.NewRequest(fiber.MethodGet, "/api/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

func (s *UserSuite) Test_UserInjector_EmptySubject() {
	var capturedUser base.UserContext
	app := s.setupApp(nil, &capturedUser)

	claims := jwt.MapClaims{
		"iss":  s.issuer,
		"aud":  s.audience,
		"sub":  "",
		"name": "No Sub User",
	}
	tokenStr := s.createToken(claims)

	req := httptest.NewRequest(fiber.MethodGet, "/api/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.Equal(int64(0), capturedUser.ID)
	s.Equal("No Sub User", capturedUser.Name)
}

func (s *UserSuite) Test_UserInjector_EmailVerified_SnakeCase() {
	var capturedUser base.UserContext
	app := s.setupApp(nil, &capturedUser)

	claims := jwt.MapClaims{
		"iss":            s.issuer,
		"aud":            s.audience,
		"sub":            "456",
		"name":           "Snake User",
		"email_verified": true,
	}
	tokenStr := s.createToken(claims)

	req := httptest.NewRequest(fiber.MethodGet, "/api/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.Equal(int64(456), capturedUser.ID)
	s.True(capturedUser.EmailVerified)
}

func (s *UserSuite) Test_UserInjector_EmailVerified_CamelCase() {
	var capturedUser base.UserContext
	app := s.setupApp(nil, &capturedUser)

	claims := jwt.MapClaims{
		"iss":           s.issuer,
		"aud":           s.audience,
		"sub":           "789",
		"name":          "Camel User",
		"emailVerified": true,
	}
	tokenStr := s.createToken(claims)

	req := httptest.NewRequest(fiber.MethodGet, "/api/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.Equal(int64(789), capturedUser.ID)
	s.True(capturedUser.EmailVerified)
}

func (s *UserSuite) Test_UserInjector_EmailVerified_False() {
	var capturedUser base.UserContext
	app := s.setupApp(nil, &capturedUser)

	claims := jwt.MapClaims{
		"iss":            s.issuer,
		"aud":            s.audience,
		"sub":            "111",
		"email_verified": false,
	}
	tokenStr := s.createToken(claims)

	req := httptest.NewRequest(fiber.MethodGet, "/api/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.False(capturedUser.EmailVerified)
}

func (s *UserSuite) Test_UserInjector_UnmarshalError() {
	app := s.setupApp(nil, nil)

	claims := jwt.MapClaims{
		"iss":   s.issuer,
		"aud":   s.audience,
		"sub":   "222",
		"roles": "not-an-array-of-roles",
	}
	tokenStr := s.createToken(claims)

	req := httptest.NewRequest(fiber.MethodGet, "/api/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusInternalServerError, resp.StatusCode)
}

func (s *UserSuite) Test_UserInjector_Success_FullUserContext() {
	var capturedUser base.UserContext
	app := s.setupApp(nil, &capturedUser)

	claims := jwt.MapClaims{
		"iss":            s.issuer,
		"aud":            s.audience,
		"sub":            "9999",
		"name":           "Alice Smith",
		"email_verified": true,
		"roles":          []string{string(base.UserRoleAdmin), string(base.UserRoleUser)},
	}
	tokenStr := s.createToken(claims)

	req := httptest.NewRequest(fiber.MethodGet, "/api/user", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.Equal(int64(9999), capturedUser.ID)
	s.Equal("Alice Smith", capturedUser.Name)
	s.True(capturedUser.EmailVerified)
	s.Equal([]base.UserRole{base.UserRoleAdmin, base.UserRoleUser}, capturedUser.Roles)
}

func (s *UserSuite) Test_Integration_HandlerAndUserInjector() {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	s.Require().NoError(err)
	keyID := "integration-key-id"

	jwk, err := jwkset.NewJWKFromKey(&privateKey.PublicKey, jwkset.JWKOptions{
		Metadata: jwkset.JWKMetadataOptions{
			KID: keyID,
		},
	})
	s.Require().NoError(err)

	jwksJSON, err := json.Marshal(jwkset.JWKSMarshal{
		Keys: []jwkset.JWKMarshal{jwk.Marshal()},
	})
	s.Require().NoError(err)

	jwksServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(jwksJSON)
	}))
	defer jwksServer.Close()

	handler, err := jwtx.Handler(jwksServer.URL, slog.Default())
	s.Require().NoError(err)

	var capturedUser base.UserContext
	app := fiber.New()
	app.Use(handler)
	app.Use(jwtx.UserInjector(s.issuer, s.audience))
	app.Get("/api/me", func(c fiber.Ctx) error {
		if u, ok := c.Locals(base.USER_CONTEXT_KEY).(base.UserContext); ok {
			capturedUser = u
		}
		return c.SendStatus(fiber.StatusOK)
	})

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":            s.issuer,
		"aud":            s.audience,
		"sub":            "5555",
		"name":           "Integration User",
		"email_verified": true,
		"roles":          []string{string(base.UserRoleUser)},
		"exp":            time.Now().Add(time.Hour).Unix(),
	})
	token.Header["kid"] = keyID
	tokenStr, err := token.SignedString(privateKey)
	s.Require().NoError(err)

	req := httptest.NewRequest(fiber.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)

	resp, err := app.Test(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.Equal(int64(5555), capturedUser.ID)
	s.Equal("Integration User", capturedUser.Name)
	s.True(capturedUser.EmailVerified)
	s.Equal([]base.UserRole{base.UserRoleUser}, capturedUser.Roles)
}

func TestUserSuite(t *testing.T) {
	suite.Run(t, &UserSuite{})
}

var _ = io.Discard

