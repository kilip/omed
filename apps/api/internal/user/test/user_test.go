package test

import (
	"fmt"
	"strings"
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

func (u *UserSuite) Test_Search_ByName_Successfully() {
	u.EnsureUser(ent.User{
		Name:  "Alice Wonderland",
		Email: "alice.search@example.com",
	})
	u.EnsureUser(ent.User{
		Name:  "Bob Builder",
		Email: "bob.search@example.com",
	})

	u.Request("/users/search", fiber.MethodPost, model.SearchRequest{
		Name: "alice",
	})
	u.OK()

	res := u.PagedResponse()
	u.NotEmpty(res.Data)
	for _, usr := range res.Data {
		u.Contains(strings.ToLower(usr.Name), "alice")
	}
}

func (u *UserSuite) Test_Search_EmptyName_ReturnsAll() {
	u.EnsureUser(ent.User{
		Name:  "Charlie Chaplin",
		Email: "charlie.search@example.com",
	})

	u.Request("/users/search", fiber.MethodPost, model.SearchRequest{})
	u.OK()

	res := u.PagedResponse()
	u.NotEmpty(res.Data)
}

func (u *UserSuite) Test_Search_Pagination_Cursor() {
	u.EnsureUser(ent.User{
		Name:  "PaginatedUser One",
		Email: "page1@example.com",
	})
	u.EnsureUser(ent.User{
		Name:  "PaginatedUser Two",
		Email: "page2@example.com",
	})
	u.EnsureUser(ent.User{
		Name:  "PaginatedUser Three",
		Email: "page3@example.com",
	})

	// Page 1 with limit 2
	u.Request("/users/search", fiber.MethodPost, model.SearchRequest{
		Name:  "PaginatedUser",
		Limit: 2,
	})
	u.OK()

	page1 := u.PagedResponse()
	u.Len(page1.Data, 2)
	u.NotEmpty(page1.Meta.Cursor)

	// Page 2 with cursor from Page 1
	u.Request("/users/search", fiber.MethodPost, model.SearchRequest{
		Name:   "PaginatedUser",
		Cursor: page1.Meta.Cursor,
		Limit:  2,
	})
	u.OK()

	page2 := u.PagedResponse()
	u.Len(page2.Data, 1)
	u.Empty(page2.Meta.Cursor)

	// Ensure descending ID order
	u.Greater(page1.Data[1].ID, page2.Data[0].ID)
}

func (u *UserSuite) Test_Search_InvalidCursor() {
	u.Request("/users/search", fiber.MethodPost, model.SearchRequest{
		Cursor: "not-a-number",
	})
	u.AssertStatus(fiber.StatusBadRequest)
}

func (u *UserSuite) Test_Search_NoResults() {
	u.Request("/users/search", fiber.MethodPost, model.SearchRequest{
		Name: "NonExistentUserXYZ987",
	})
	u.OK()

	res := u.PagedResponse()
	u.Empty(res.Data)
	u.Empty(res.Meta.Cursor)
}

func TestUserSuite(t *testing.T) {
	suite.Run(t, &UserSuite{})
}
