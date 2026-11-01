package test

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type McpSuite struct {
	testutil.ApiTestSuite[any]
}

func (s *McpSuite) Test_Mcp_ValidPAT_ToolsListIncludesFinTools() {
	pat := s.EnsurePAT(s.User.ID)
	s.SetHeader("Accept", "application/json, text/event-stream")

	s.RequestWithToken("/mcp", fiber.MethodPost, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]any{},
	}, pat)
	s.OK()

	bodyBytes, err := io.ReadAll(s.GetHttpResponse().Body)
	s.Require().NoError(err)
	s.Contains(string(bodyBytes), "fin_period_overview")
}

func (s *McpSuite) Test_Mcp_ExpiredPAT_ReturnsLink() {
	rawBytes := make([]byte, 32)
	_, err := rand.Read(rawBytes)
	s.Require().NoError(err)
	plaintext := "omed_pat_" + base64.RawURLEncoding.EncodeToString(rawBytes)
	prefix := plaintext[:12]
	h := sha256.Sum256([]byte(plaintext))
	tokenHash := hex.EncodeToString(h[:])

	ctx := s.Context
	_, err = testutil.GetEntCli().PersonalAccessToken.Create().
		SetUserID(s.User.ID).
		SetName("Expired Token").
		SetPrefix(prefix).
		SetTokenHash(tokenHash).
		SetExpiresAt(time.Now().Add(-1 * time.Hour)).
		Save(ctx)
	s.Require().NoError(err)

	s.RequestWithToken("/mcp", fiber.MethodPost, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]any{},
	}, plaintext)
	s.AssertStatus(fiber.StatusUnauthorized)

	errResp := s.GetErrorResponse()
	s.Equal("TOKEN_EXPIRED", errResp.Error.Code)
	s.True(strings.HasSuffix(errResp.Error.Message, "/core/access-tokens"))
}

func (s *McpSuite) Test_Mcp_RevokedPAT_GenericUnauthorized() {
	rawBytes := make([]byte, 32)
	_, err := rand.Read(rawBytes)
	s.Require().NoError(err)
	plaintext := "omed_pat_" + base64.RawURLEncoding.EncodeToString(rawBytes)
	prefix := plaintext[:12]
	h := sha256.Sum256([]byte(plaintext))
	tokenHash := hex.EncodeToString(h[:])

	ctx := s.Context
	now := time.Now()
	_, err = testutil.GetEntCli().PersonalAccessToken.Create().
		SetUserID(s.User.ID).
		SetName("Revoked Token").
		SetPrefix(prefix).
		SetTokenHash(tokenHash).
		SetExpiresAt(time.Now().Add(24 * time.Hour)).
		SetRevokedAt(now).
		Save(ctx)
	s.Require().NoError(err)

	s.RequestWithToken("/mcp", fiber.MethodPost, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]any{},
	}, plaintext)
	s.AssertStatus(fiber.StatusUnauthorized)

	errResp := s.GetErrorResponse()
	s.Equal("UNAUTHORIZED", errResp.Error.Code)
	s.Equal("invalid token", errResp.Error.Message)
}

func (s *McpSuite) Test_Mcp_SessionCookie_Rejected() {
	s.ClearCookies()
	s.SetCookie(&http.Cookie{
		Name:  "omed_session",
		Value: s.SessionToken,
	})

	s.RequestWithToken("/mcp", fiber.MethodPost, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]any{},
	}, "")
	s.AssertStatus(fiber.StatusUnauthorized)

	errResp := s.GetErrorResponse()
	s.Equal("UNAUTHORIZED", errResp.Error.Code)
	s.Equal("invalid token", errResp.Error.Message)
}

func (s *McpSuite) Test_Mcp_SessionTokenAsBearer_Rejected() {
	s.RequestWithToken("/mcp", fiber.MethodPost, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]any{},
	}, s.SessionToken)
	s.AssertStatus(fiber.StatusUnauthorized)

	errResp := s.GetErrorResponse()
	s.Equal("UNAUTHORIZED", errResp.Error.Code)
	s.Equal("invalid token", errResp.Error.Message)
}

func TestMcpSuite(t *testing.T) {
	suite.Run(t, new(McpSuite))
}
