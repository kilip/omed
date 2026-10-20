package test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type UserSuite struct {
	testutil.ApiTestSuite[model.UserResponse]
}

func (s *UserSuite) Test_GetMe_Successfully() {
	s.Request("/users/me", fiber.MethodGet, nil)
	s.OK()

	res := s.GetResponse()
	s.Equal(s.User.ID, res.Data.ID)
	s.Equal(s.User.Name, res.Data.Name)
	s.NotEmpty(res.Data.Locale)
	s.NotEmpty(res.Data.Currency)
}

func (s *UserSuite) Test_UpdateMe_Successfully() {
	req := model.UpdateUserRequest{
		Name:     "Updated Name",
		Locale:   "id",
		Currency: "IDR",
	}

	s.Request("/users/me", fiber.MethodPut, req)
	s.OK()

	res := s.GetResponse()
	s.Equal(s.User.ID, res.Data.ID)
	s.Equal("Updated Name", res.Data.Name)
	s.Equal("id", res.Data.Locale)
	s.Equal("IDR", res.Data.Currency)

	// Verify persistence via GET /users/me
	s.Request("/users/me", fiber.MethodGet, nil)
	s.OK()
	getRes := s.GetResponse()
	s.Equal("Updated Name", getRes.Data.Name)
	s.Equal("id", getRes.Data.Locale)
	s.Equal("IDR", getRes.Data.Currency)
}

func (s *UserSuite) Test_UpdateMe_InvalidBody() {
	s.Request("/users/me", fiber.MethodPut, "invalid-json")
	s.AssertStatus(fiber.StatusBadRequest)
}

func (s *UserSuite) Test_GetMe_Unauthorized() {
	s.RequestWithToken("/users/me", fiber.MethodGet, nil, "")
	s.AssertStatus(fiber.StatusUnauthorized)
}

func TestUserSuite(t *testing.T) {
	suite.Run(t, new(UserSuite))
}
