package test

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type MemberSuite struct {
	testutil.ApiTestSuite[model.MemberResponse]
}

func (s *MemberSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *MemberSuite) Test_Invite_Successfully() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/members", ws.ID), fiber.MethodPost, model.InviteMemberRequest{
		Email: "newmember@example.com",
		Roles: []string{"core:member"},
	})
	s.Created()

	res := s.GetResponse()
	s.Equal(ws.ID, res.Data.WorkspaceID)
	s.Equal("newmember@example.com", res.Data.User.Email)
	s.Equal([]string{"core:member"}, res.Data.Roles)
}

func (s *MemberSuite) Test_Invite_InvalidRole() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/members", ws.ID), fiber.MethodPost, model.InviteMemberRequest{
		Email: "newmember2@example.com",
		Roles: []string{"nonexistent:role"},
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *MemberSuite) Test_Invite_OwnerImmutable() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	// Owner attempts to invite themselves as a member
	s.Request(fmt.Sprintf("/workspaces/%d/members", ws.ID), fiber.MethodPost, model.InviteMemberRequest{
		Email: "test@example.com", // owner's email
		Roles: []string{"core:member"},
	})
	s.AssertStatus(fiber.StatusConflict)
}

func (s *MemberSuite) Test_Invite_ForbiddenForNormalMember() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	// Switch to non-owner member
	alice := s.AsUser("alice@example.com", "Alice")
	s.AddMember(ws.ID, alice.ID, "core:member")

	s.Request(fmt.Sprintf("/workspaces/%d/members", ws.ID), fiber.MethodPost, model.InviteMemberRequest{
		Email: "other@example.com",
		Roles: []string{"core:member"},
	})
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *MemberSuite) Test_UpdateRoles_Successfully() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	alice := s.AsUser("alice@example.com", "Alice")
	s.AddMember(ws.ID, alice.ID, "core:member")

	// Switch back to owner
	s.AsUser("test@example.com", "Test User")

	s.Request(fmt.Sprintf("/workspaces/%d/members/%d", ws.ID, alice.ID), fiber.MethodPut, model.UpdateMemberRequest{
		Roles: []string{"core:admin"},
	})
	s.OK()

	res := s.GetResponse()
	s.Equal([]string{"core:admin"}, res.Data.Roles)
}

func (s *MemberSuite) Test_UpdateRoles_OwnerImmutable() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/members/%d", ws.ID, s.User.ID), fiber.MethodPut, model.UpdateMemberRequest{
		Roles: []string{"core:member"},
	})
	s.AssertStatus(fiber.StatusConflict)
}

func (s *MemberSuite) Test_DeleteMember_Successfully() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	alice := s.AsUser("alice@example.com", "Alice")
	s.AddMember(ws.ID, alice.ID, "core:member")

	s.AsUser("test@example.com", "Test User")

	s.Request(fmt.Sprintf("/workspaces/%d/members/%d", ws.ID, alice.ID), fiber.MethodDelete, nil)
	s.OK()
}

func (s *MemberSuite) Test_SearchMembers_Successfully() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	alice := s.AsUser("alice@example.com", "Alice")
	s.AddMember(ws.ID, alice.ID, "core:member")

	s.AsUser("test@example.com", "Test User")

	s.Request(fmt.Sprintf("/workspaces/%d/members/search", ws.ID), fiber.MethodPost, model.SearchMemberRequest{})
	s.OK()

	res := s.PagedResponse()
	s.Len(res.Data, 1)
	s.Equal("alice@example.com", res.Data[0].User.Email)
}

func (s *MemberSuite) Test_Permissions_Owner() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/permissions", ws.ID), fiber.MethodGet, nil)
	s.OK()

	var resp httpx.WebResponse[model.PermissionsResponse]
	resp = testutil.DecodeOther[model.PermissionsResponse](s)
	s.True(resp.Data.Owner)
	s.Equal(ws.ID, resp.Data.WorkspaceID)
	s.Equal([]model.PermissionItem{{Object: "*", Action: "*"}}, resp.Data.Permissions)
}

func (s *MemberSuite) Test_Permissions_Member() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	alice := s.AsUser("alice@example.com", "Alice")
	s.AddMember(ws.ID, alice.ID, "core:member")

	s.Request(fmt.Sprintf("/workspaces/%d/permissions", ws.ID), fiber.MethodGet, nil)
	s.OK()

	var resp httpx.WebResponse[model.PermissionsResponse]
	resp = testutil.DecodeOther[model.PermissionsResponse](s)
	s.False(resp.Data.Owner)
	s.Equal([]string{"core:member"}, resp.Data.Roles)
	s.NotEmpty(resp.Data.Permissions)
}

func (s *MemberSuite) Test_Roles_Member() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	alice := s.AsUser("alice@example.com", "Alice")
	s.AddMember(ws.ID, alice.ID, "core:member")

	s.Request(fmt.Sprintf("/workspaces/%d/roles", ws.ID), fiber.MethodGet, nil)
	s.OK()

	resp := testutil.DecodeOther[[]model.RoleResponse](s)
	byID := map[string]model.RoleResponse{}
	for _, r := range resp.Data {
		byID[r.ID] = r
	}
	s.Equal(model.RoleResponse{
		ID:          "core:member",
		Name:        "Member",
		Description: "View the workspace and its members",
		Module:      "core",
	}, byID["core:member"])
	s.Equal("Admin", byID["core:admin"].Name)
}

func (s *MemberSuite) Test_NonMember_Gets404() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	s.AsUser("stranger@example.com", "Stranger")

	s.Request(fmt.Sprintf("/workspaces/%d", ws.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)

	s.Request(fmt.Sprintf("/workspaces/%d/permissions", ws.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)

	s.Request(fmt.Sprintf("/workspaces/%d/roles", ws.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func TestMemberSuite(t *testing.T) {
	suite.Run(t, new(MemberSuite))
}
