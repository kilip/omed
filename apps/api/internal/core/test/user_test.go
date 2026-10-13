package test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"uuid"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/workspace"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
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

	// Verify personal workspace was auto-created
	client := testutil.GetEntCli()
	ws, err := client.Workspace.Query().Where(workspace.OwnerIDEQ(res.Data.ID), workspace.PersonalEQ(true)).Only(u.Context)
	u.Require().NoError(err)
	u.Equal(testUser.Name+"'s Workspace", ws.Name)
}

func (u *UserSuite) Test_NonSuperadmin_Forbidden() {
	u.AsUser("normal@example.com", "Normal User", base.UserRoleUser)
	u.Request("/users", fiber.MethodPost, model.CreateUserRequest{
		Name:  "Someone",
		Email: "someone@example.com",
	})
	u.AssertStatus(fiber.StatusForbidden)
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

func (u *UserSuite) getErrorResponse() httpx.ErrorResponse {
	var res httpx.ErrorResponse
	u.T().Helper()
	defer u.HttpResponse.Body.Close()
	u.Require().NoError(json.NewDecoder(u.HttpResponse.Body).Decode(&res))
	return res
}

func (u *UserSuite) Test_Create_ValidationError_RequiredFields() {
	u.Request("/users", fiber.MethodPost, model.CreateUserRequest{})
	u.AssertStatus(fiber.StatusUnprocessableEntity)

	errRes := u.getErrorResponse()
	u.Equal("VALIDATION_FAILED", errRes.Error.Code)

	fieldRules := make(map[string]string)
	for _, f := range errRes.Error.Fields {
		fieldRules[f.Field] = f.Rule
	}
	u.Equal("required", fieldRules["Name"])
	u.Equal("required", fieldRules["Email"])
}

func (u *UserSuite) Test_Create_ValidationError_InvalidEmail() {
	u.Request("/users", fiber.MethodPost, model.CreateUserRequest{
		Name:  "Test User",
		Email: "invalid-email-format",
	})
	u.AssertStatus(fiber.StatusUnprocessableEntity)

	errRes := u.getErrorResponse()
	u.Equal("VALIDATION_FAILED", errRes.Error.Code)
	u.Require().Len(errRes.Error.Fields, 1)
	u.Equal("Email", errRes.Error.Fields[0].Field)
	u.Equal("email", errRes.Error.Fields[0].Rule)
}

func (u *UserSuite) Test_Create_ValidationError_InvalidAvatarURL() {
	u.Request("/users", fiber.MethodPost, model.CreateUserRequest{
		Name:   "Test User",
		Email:  "test@example.com",
		Avatar: "not-a-valid-url",
	})
	u.AssertStatus(fiber.StatusUnprocessableEntity)

	errRes := u.getErrorResponse()
	u.Equal("VALIDATION_FAILED", errRes.Error.Code)
	u.Require().Len(errRes.Error.Fields, 1)
	u.Equal("Avatar", errRes.Error.Fields[0].Field)
	u.Equal("url", errRes.Error.Fields[0].Rule)
}

func (u *UserSuite) Test_Update_ValidationError_RequiredName() {
	user := u.EnsureUser(testUser)
	u.Request(fmt.Sprintf("/users/%d", user.ID), fiber.MethodPut, model.UpdateUserRequest{
		Name: "",
	})
	u.AssertStatus(fiber.StatusUnprocessableEntity)

	errRes := u.getErrorResponse()
	u.Equal("VALIDATION_FAILED", errRes.Error.Code)
	u.Require().Len(errRes.Error.Fields, 1)
	u.Equal("Name", errRes.Error.Fields[0].Field)
	u.Equal("required", errRes.Error.Fields[0].Rule)
}

func (u *UserSuite) Test_Update_ValidationError_InvalidAvatarURL() {
	user := u.EnsureUser(testUser)
	u.Request(fmt.Sprintf("/users/%d", user.ID), fiber.MethodPut, model.UpdateUserRequest{
		Name:   "Valid Name",
		Avatar: "not-a-valid-url",
	})
	u.AssertStatus(fiber.StatusUnprocessableEntity)

	errRes := u.getErrorResponse()
	u.Equal("VALIDATION_FAILED", errRes.Error.Code)
	u.Require().Len(errRes.Error.Fields, 1)
	u.Equal("Avatar", errRes.Error.Fields[0].Field)
	u.Equal("url", errRes.Error.Fields[0].Rule)
}

func (u *UserSuite) Test_Search_ValidationError_InvalidLimit() {
	u.Request("/users/search", fiber.MethodPost, model.SearchRequest{
		Limit: -1,
	})
	u.AssertStatus(fiber.StatusUnprocessableEntity)

	errRes := u.getErrorResponse()
	u.Equal("VALIDATION_FAILED", errRes.Error.Code)
	u.Require().Len(errRes.Error.Fields, 1)
	u.Equal("Limit", errRes.Error.Fields[0].Field)
	u.Equal("min", errRes.Error.Fields[0].Rule)

	u.Request("/users/search", fiber.MethodPost, model.SearchRequest{
		Limit: 101,
	})
	u.AssertStatus(fiber.StatusUnprocessableEntity)

	errRes = u.getErrorResponse()
	u.Equal("VALIDATION_FAILED", errRes.Error.Code)
	u.Require().Len(errRes.Error.Fields, 1)
	u.Equal("Limit", errRes.Error.Fields[0].Field)
	u.Equal("max", errRes.Error.Fields[0].Rule)
}

func TestUserSuite(t *testing.T) {
	suite.Run(t, &UserSuite{})
}
