package test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/http"
	"github.com/kilip/omed/finance/testutil"
	"github.com/stretchr/testify/suite"
)

type UserInjectorTestSuite struct {
	testutil.ApiTestSuite[core.AuthenticatedUser]
}

func (s *UserInjectorTestSuite) SetupTest() {
	s.ApiTestSuite.SetupTest()
	state := testutil.GetState()
	api := state.FiberApp

	api.Get("/test/rest", func(c fiber.Ctx) error {
		return http.OK(c, core.UserFromContext(c))
	})
	api.Get("/mcp/test", func(c fiber.Ctx) error {
		return http.OK(c, core.UserFromContext(c))
	})
}

func (s *UserInjectorTestSuite) TestAudienceRouting() {
	state := testutil.GetState()
	issuer := state.Config.AuthBaseUrl

	s.Run("token with aud=issuer is accepted on REST route", func() {
		s.RequestWithClaims("/test/rest", fiber.MethodGet, nil, jwt.MapClaims{
			"aud": issuer,
		})
		s.OK()
		resp := s.GetResponse()
		s.Equal(s.User.Name, resp.Data.Name)
	})

	s.Run("token with wrong issuer is rejected", func() {
		s.RequestWithClaims("/test/rest", fiber.MethodGet, nil, jwt.MapClaims{
			"iss": "http://wrong-issuer.com",
			"aud": issuer,
		})
		s.AssertStatus(fiber.StatusUnauthorized)
	})
}

func TestUserInjectorSuite(t *testing.T) {
	suite.Run(t, new(UserInjectorTestSuite))
}
