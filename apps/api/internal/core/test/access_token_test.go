package test

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type AccessTokenSuite struct {
	testutil.ApiTestSuite[model.CreatedAccessTokenResponse]
}

func (s *AccessTokenSuite) Test_Lifecycle_CreateListRevoke() {
	// 1. Create personal access token
	s.Request("/users/me/access-tokens", fiber.MethodPost, model.CreateAccessTokenRequest{
		Name:          "my token",
		ExpiresInDays: 30,
	})
	s.Created()

	createdResp := s.GetResponse()
	s.True(strings.HasPrefix(createdResp.Data.Token, "omed_pat_"))
	s.Equal("my token", createdResp.Data.Name)
	s.Equal(createdResp.Data.Token[:12], createdResp.Data.Prefix)
	s.Equal("active", createdResp.Data.Status)
	s.False(createdResp.Data.ExpiresAt.IsZero())
	s.False(createdResp.Data.CreatedAt.IsZero())
	tokenID := createdResp.Data.ID
	s.Require().Positive(tokenID)

	// 2. List tokens: status is active and no "token" field
	s.Request("/users/me/access-tokens", fiber.MethodGet, nil)
	s.OK()

	rawBody, err := io.ReadAll(s.GetHttpResponse().Body)
	s.Require().NoError(err)
	s.NotContains(string(rawBody), `"token":`)

	var listResp httpx.WebResponse[[]model.AccessTokenResponse]
	s.Require().NoError(json.Unmarshal(rawBody, &listResp))
	s.Require().Len(listResp.Data, 1)
	s.Equal(tokenID, listResp.Data[0].ID)
	s.Equal("active", listResp.Data[0].Status)

	// 3. Revoke token
	s.Request(fmt.Sprintf("/users/me/access-tokens/%d", tokenID), fiber.MethodDelete, nil)
	s.OK()

	// 4. List tokens: status is revoked
	s.Request("/users/me/access-tokens", fiber.MethodGet, nil)
	s.OK()

	var listAfterRevoke httpx.WebResponse[[]model.AccessTokenResponse]
	s.Require().NoError(json.NewDecoder(s.GetHttpResponse().Body).Decode(&listAfterRevoke))
	s.Require().Len(listAfterRevoke.Data, 1)
	s.Equal(tokenID, listAfterRevoke.Data[0].ID)
	s.Equal("revoked", listAfterRevoke.Data[0].Status)
}

func (s *AccessTokenSuite) Test_List_NoTokens_ReturnsEmptyArray() {
	s.AsUser("notokens@example.com", "No Tokens")
	s.Request("/users/me/access-tokens", fiber.MethodGet, nil)
	s.OK()

	rawBody, err := io.ReadAll(s.GetHttpResponse().Body)
	s.Require().NoError(err)
	s.Contains(string(rawBody), `"data":[]`)
}

func (s *AccessTokenSuite) Test_Create_InvalidExpiry_Returns422() {
	s.Request("/users/me/access-tokens", fiber.MethodPost, model.CreateAccessTokenRequest{
		Name:          "bad expiry token",
		ExpiresInDays: 45,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *AccessTokenSuite) Test_Create_LimitReached_Returns422() {
	s.AsUser("limituser@example.com", "Limit User")
	for i := 1; i <= 20; i++ {
		s.Request("/users/me/access-tokens", fiber.MethodPost, model.CreateAccessTokenRequest{
			Name:          fmt.Sprintf("token-%d", i),
			ExpiresInDays: 30,
		})
		s.Created()
	}

	s.Request("/users/me/access-tokens", fiber.MethodPost, model.CreateAccessTokenRequest{
		Name:          "token-21",
		ExpiresInDays: 30,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *AccessTokenSuite) Test_Revoke_OtherUsersToken_Returns404() {
	s.AsUser("usera@example.com", "User A")
	s.Request("/users/me/access-tokens", fiber.MethodPost, model.CreateAccessTokenRequest{
		Name:          "token-a",
		ExpiresInDays: 30,
	})
	s.Created()
	created := s.GetResponse()

	s.AsUser("userb@example.com", "User B")
	s.Request(fmt.Sprintf("/users/me/access-tokens/%d", created.Data.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *AccessTokenSuite) Test_PAT_RejectedOnRest() {
	s.Request("/users/me/access-tokens", fiber.MethodPost, model.CreateAccessTokenRequest{
		Name:          "pat-token",
		ExpiresInDays: 30,
	})
	s.Created()
	created := s.GetResponse()
	pat := created.Data.Token
	s.Require().NotEmpty(pat)

	s.RequestWithToken("/users/me", fiber.MethodGet, nil, pat)
	s.AssertStatus(fiber.StatusUnauthorized)
}

func (s *AccessTokenSuite) Test_PAT_CannotManageTokens() {
	s.Request("/users/me/access-tokens", fiber.MethodPost, model.CreateAccessTokenRequest{
		Name:          "pat-token-manager",
		ExpiresInDays: 30,
	})
	s.Created()
	created := s.GetResponse()
	pat := created.Data.Token
	s.Require().NotEmpty(pat)

	s.RequestWithToken("/users/me/access-tokens", fiber.MethodPost, model.CreateAccessTokenRequest{
		Name:          "nested-token",
		ExpiresInDays: 30,
	}, pat)
	s.AssertStatus(fiber.StatusUnauthorized)
}

func TestAccessTokenSuite(t *testing.T) {
	suite.Run(t, new(AccessTokenSuite))
}
