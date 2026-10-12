package test

import (
	"fmt"
	"testing"
	"uuid"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/user/model"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

var testUser = ent.User{
	Name:   "Test User",
	Email:  "test.get@example.com",
	Avatar: "https://example.com/test-avatar.jpg",
}

type UserSuite struct {
	testutil.ApiTestSuite[model.UserResponse]
}

func (u *UserSuite) Test_GetByID_Successfully() {
	user := u.EnsureUser(testUser)
	u.Request(fmt.Sprintf("/users/%d", user.ID), fiber.MethodGet, nil)
	u.OK()
}

func (u *UserSuite) Test_GetByID_NotExistID() {
	u.Request("/users/9999", fiber.MethodGet, nil)
	u.AssertStatus(fiber.StatusNotFound)
}

func (u *UserSuite) Test_GetByID_InvalidTypeID() {
	u.Request("/users/not-existing", fiber.MethodGet, nil)
	u.AssertStatus(fiber.StatusBadRequest)

	uuid := uuid.NewV7()
	u.Request(fmt.Sprintf("/users/%s", uuid.String()), fiber.MethodGet, nil)
	u.AssertStatus(fiber.StatusBadRequest)
}

func (u *UserSuite) Test_Create_Successfully() {
	u.Request("/users", fiber.MethodPost, model.CreateUserRequest{
		Name:   testUser.Name,
		Email:  testUser.Email,
		Avatar: testUser.Avatar,
	})
	u.Created()
	res := u.GetResponse()

	u.Equal(testUser.Name, res.Data.Name)
	u.Equal(testUser.Avatar, res.Data.Avatar)
	u.NotNil(res.Data.CreatedAt)
	u.NotNil(res.Data.UpdatedAt)
}

func (u *UserSuite) Test_Update_Successfully() {
	user := u.EnsureUser(testUser)
	user.Name = "Test Updated User Name"
	u.Request(fmt.Sprintf("/users/%d", user.ID), fiber.MethodPut, model.UpdateUserRequest{
		Name:   user.Name,
		Avatar: user.Avatar,
	})
	u.OK()

	res := u.GetResponse().Data

	u.Equal(user.Name, res.Name)
	u.Greater(res.UpdatedAt, user.UpdatedAt)
}

func (u *UserSuite) Test_Update_NonExistingID() {
	u.Request("/users/9999", fiber.MethodPut, model.UpdateUserRequest{
		Name:   testUser.Name,
		Avatar: testUser.Avatar,
	})
}

func (u *UserSuite) Test_Update_InvalidID() {
	u.Request("/users/invalid-id", fiber.MethodPut, model.UpdateUserRequest{
		Name:   testUser.Name,
		Avatar: testUser.Avatar,
	})
	u.AssertStatus(fiber.StatusBadRequest)
}

func (u *UserSuite) Test_Delete_Successfully() {
	user := u.EnsureUser(testUser)
	u.Request(fmt.Sprintf("/users/%d", user.ID), fiber.MethodDelete, nil)
	u.OK()
}

func (u *UserSuite) Test_Delete_InvalidID() {
	u.Request("/users/invalid-id", fiber.MethodDelete, nil)
	u.AssertStatus(fiber.StatusBadRequest)
}

func TestUserSuite(t *testing.T) {
	suite.Run(t, &UserSuite{})
}
