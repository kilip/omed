package test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"uuid"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type WorkspaceSuite struct {
	testutil.ApiTestSuite[model.WorkspaceResponse]
}

func (s *WorkspaceSuite) ensureWorkspace(name, desc string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetDescription(desc).
		SetOwnerID(ownerID).
		SetCreatedByID(ownerID).
		SetUpdatedByID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *WorkspaceSuite) getErrorResponse() httpx.ErrorResponse {
	var res httpx.ErrorResponse
	s.T().Helper()
	defer s.HttpResponse.Body.Close()
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&res))
	return res
}

func (s *WorkspaceSuite) Test_GetByID_Successfully() {
	ws := s.ensureWorkspace("Engineering", "Engineering Workspace", s.User.ID)
	s.Request(fmt.Sprintf("/workspaces/%d", ws.ID), fiber.MethodGet, nil)
	s.OK()

	res := s.GetResponse()
	s.Equal(ws.ID, res.Data.ID)
	s.Equal("Engineering", res.Data.Name)
	s.Equal("Engineering Workspace", res.Data.Description)
	s.Equal(s.User.ID, res.Data.Owner.ID)
	s.Equal(s.User.Name, res.Data.Owner.Name)
	s.Equal(s.User.Name, res.Data.CreatedBy.Name)
	s.Equal(s.User.Name, res.Data.UpdatedBy.Name)
}

func (s *WorkspaceSuite) Test_GetByID_NotExistID() {
	s.Request("/workspaces/9999", fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *WorkspaceSuite) Test_GetByID_InvalidTypeID() {
	s.Request("/workspaces/not-existing", fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusBadRequest)

	u := uuid.NewV7()
	s.Request(fmt.Sprintf("/workspaces/%s", u.String()), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusBadRequest)
}

func (s *WorkspaceSuite) Test_Create_Successfully() {
	s.Request("/workspaces", fiber.MethodPost, model.CreateWorkspaceRequest{
		Name:        "Product Design",
		Description: "Design Team Workspace",
	})
	s.Created()

	res := s.GetResponse()
	s.Positive(res.Data.ID)
	s.Equal("Product Design", res.Data.Name)
	s.Equal("Design Team Workspace", res.Data.Description)
	s.Equal(s.User.ID, res.Data.Owner.ID)
	s.Equal(s.User.Name, res.Data.Owner.Name)
	s.Equal(s.User.Name, res.Data.CreatedBy.Name)
	s.Equal(s.User.Name, res.Data.UpdatedBy.Name)
	s.False(res.Data.CreatedAt.IsZero())
	s.False(res.Data.UpdatedAt.IsZero())
}

func (s *WorkspaceSuite) Test_Create_ValidationError_RequiredName() {
	s.Request("/workspaces", fiber.MethodPost, model.CreateWorkspaceRequest{})
	s.AssertStatus(fiber.StatusUnprocessableEntity)

	errRes := s.getErrorResponse()
	s.Equal("VALIDATION_FAILED", errRes.Error.Code)
	s.Require().Len(errRes.Error.Fields, 1)
	s.Equal("Name", errRes.Error.Fields[0].Field)
	s.Equal("required", errRes.Error.Fields[0].Rule)
}

func (s *WorkspaceSuite) Test_Update_Successfully() {
	ws := s.ensureWorkspace("Old Name", "Old Desc", s.User.ID)
	s.Request(fmt.Sprintf("/workspaces/%d", ws.ID), fiber.MethodPut, model.UpdateWorkspaceRequest{
		Name:        "New Workspace Name",
		Description: "New Workspace Description",
	})
	s.OK()

	res := s.GetResponse().Data
	s.Equal("New Workspace Name", res.Name)
	s.Equal("New Workspace Description", res.Description)
	s.GreaterOrEqual(res.UpdatedAt, ws.UpdatedAt)
}

func (s *WorkspaceSuite) Test_Update_NonExistingID() {
	s.Request("/workspaces/9999", fiber.MethodPut, model.UpdateWorkspaceRequest{
		Name: "Non Existent",
	})
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *WorkspaceSuite) Test_Update_InvalidID() {
	s.Request("/workspaces/invalid-id", fiber.MethodPut, model.UpdateWorkspaceRequest{
		Name: "Invalid",
	})
	s.AssertStatus(fiber.StatusBadRequest)
}

func (s *WorkspaceSuite) Test_Update_ValidationError_RequiredName() {
	ws := s.ensureWorkspace("Valid Name", "Desc", s.User.ID)
	s.Request(fmt.Sprintf("/workspaces/%d", ws.ID), fiber.MethodPut, model.UpdateWorkspaceRequest{
		Name: "",
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)

	errRes := s.getErrorResponse()
	s.Equal("VALIDATION_FAILED", errRes.Error.Code)
	s.Require().Len(errRes.Error.Fields, 1)
	s.Equal("Name", errRes.Error.Fields[0].Field)
	s.Equal("required", errRes.Error.Fields[0].Rule)
}

func (s *WorkspaceSuite) Test_Delete_Successfully() {
	ws := s.ensureWorkspace("To Delete", "Desc", s.User.ID)
	s.Request(fmt.Sprintf("/workspaces/%d", ws.ID), fiber.MethodDelete, nil)
	s.OK()

	// Verify not found after deletion
	s.Request(fmt.Sprintf("/workspaces/%d", ws.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *WorkspaceSuite) Test_Delete_InvalidID() {
	s.Request("/workspaces/invalid-id", fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusBadRequest)
}

func (s *WorkspaceSuite) Test_Search_ByName_Successfully() {
	s.ensureWorkspace("Alpha Workspace", "A", s.User.ID)
	s.ensureWorkspace("Beta Workspace", "B", s.User.ID)

	s.Request("/workspaces/search", fiber.MethodPost, model.SearchWorkspaceRequest{
		Name: "alpha",
	})
	s.OK()

	res := s.PagedResponse()
	s.NotEmpty(res.Data)
	for _, w := range res.Data {
		s.Contains(strings.ToLower(w.Name), "alpha")
	}
}

func (s *WorkspaceSuite) Test_Search_EmptyName_ReturnsAll() {
	s.ensureWorkspace("Search All 1", "", s.User.ID)

	s.Request("/workspaces/search", fiber.MethodPost, model.SearchWorkspaceRequest{})
	s.OK()

	res := s.PagedResponse()
	s.NotEmpty(res.Data)
}

func (s *WorkspaceSuite) Test_Search_Pagination_Cursor() {
	s.ensureWorkspace("Paginated One", "", s.User.ID)
	s.ensureWorkspace("Paginated Two", "", s.User.ID)
	s.ensureWorkspace("Paginated Three", "", s.User.ID)

	// Page 1 with limit 2
	s.Request("/workspaces/search", fiber.MethodPost, model.SearchWorkspaceRequest{
		Name:  "Paginated",
		Limit: 2,
	})
	s.OK()

	page1 := s.PagedResponse()
	s.Len(page1.Data, 2)
	s.NotEmpty(page1.Meta.Cursor)

	// Page 2 with cursor from Page 1
	s.Request("/workspaces/search", fiber.MethodPost, model.SearchWorkspaceRequest{
		Name:   "Paginated",
		Cursor: page1.Meta.Cursor,
		Limit:  2,
	})
	s.OK()

	page2 := s.PagedResponse()
	s.Len(page2.Data, 1)
	s.Empty(page2.Meta.Cursor)

	s.Greater(page1.Data[1].ID, page2.Data[0].ID)
}

func (s *WorkspaceSuite) Test_Search_InvalidCursor() {
	s.Request("/workspaces/search", fiber.MethodPost, model.SearchWorkspaceRequest{
		Cursor: "not-a-number",
	})
	s.AssertStatus(fiber.StatusBadRequest)
}

func (s *WorkspaceSuite) Test_Search_NoResults() {
	s.Request("/workspaces/search", fiber.MethodPost, model.SearchWorkspaceRequest{
		Name: "NonExistentWorkspaceXYZ987",
	})
	s.OK()

	res := s.PagedResponse()
	s.Empty(res.Data)
	s.Empty(res.Meta.Cursor)
}

func (s *WorkspaceSuite) Test_Search_ValidationError_InvalidLimit() {
	s.Request("/workspaces/search", fiber.MethodPost, model.SearchWorkspaceRequest{
		Limit: -1,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)

	errRes := s.getErrorResponse()
	s.Equal("VALIDATION_FAILED", errRes.Error.Code)
	s.Require().Len(errRes.Error.Fields, 1)
	s.Equal("Limit", errRes.Error.Fields[0].Field)
	s.Equal("min", errRes.Error.Fields[0].Rule)

	s.Request("/workspaces/search", fiber.MethodPost, model.SearchWorkspaceRequest{
		Limit: 101,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)

	errRes = s.getErrorResponse()
	s.Equal("VALIDATION_FAILED", errRes.Error.Code)
	s.Require().Len(errRes.Error.Fields, 1)
	s.Equal("Limit", errRes.Error.Fields[0].Field)
	s.Equal("max", errRes.Error.Fields[0].Rule)
}

func TestWorkspaceSuite(t *testing.T) {
	suite.Run(t, &WorkspaceSuite{})
}
