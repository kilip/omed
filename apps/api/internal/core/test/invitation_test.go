package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/internal/platform/mailx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type InvitationSuite struct {
	testutil.ApiTestSuite[model.InvitationResponse]
}

func (s *InvitationSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *InvitationSuite) Test_Create_Successfully() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	expiresAt := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	s.Request(fmt.Sprintf("/workspaces/%d/invitations", ws.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles:     []string{"core:member"},
		MaxUses:   5,
		ExpiresAt: &expiresAt,
	})
	s.Created()

	res := s.GetResponse()
	s.Equal(ws.ID, res.Data.WorkspaceID)
	s.Equal([]string{"core:member"}, res.Data.Roles)
	s.Equal(5, res.Data.MaxUses)
	s.Equal(0, res.Data.UsesCount)
	s.NotEmpty(res.Data.Token)
	s.False(res.Data.IsRevoked)
}

func (s *InvitationSuite) Test_Create_InvalidRole() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/invitations", ws.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles: []string{"invalid:role"},
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *InvitationSuite) Test_Create_PersonalWorkspace_Conflict() {
	owner := s.AsUser("pw-owner-inv@example.com", "PW Owner Inv")
	client := testutil.GetEntCli()
	pw, err := client.Workspace.Create().
		SetName("Personal Workspace").
		SetOwnerID(owner.ID).
		SetPersonal(true).
		Save(s.Context)
	s.Require().NoError(err)

	s.Request(fmt.Sprintf("/workspaces/%d/invitations", pw.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles: []string{"core:member"},
	})
	s.AssertStatus(fiber.StatusConflict)

	errRes := s.GetErrorResponse()
	s.Equal("WORKSPACE_PERSONAL_MEMBER", errRes.Error.Code)
}

func (s *InvitationSuite) Test_Create_ForbiddenForNormalMember() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	bob := s.AsUser("bob@example.com", "Bob")
	s.AddMember(ws.ID, bob.ID, "core:member")

	s.Request(fmt.Sprintf("/workspaces/%d/invitations", ws.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles: []string{"core:member"},
	})
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *InvitationSuite) Test_Search_Successfully() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	// Create 2 invitations
	s.Request(fmt.Sprintf("/workspaces/%d/invitations", ws.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles: []string{"core:member"},
	})
	s.Created()
	s.Request(fmt.Sprintf("/workspaces/%d/invitations", ws.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles: []string{"core:admin"},
	})
	s.Created()

	s.Request(fmt.Sprintf("/workspaces/%d/invitations/search", ws.ID), fiber.MethodPost, model.SearchInvitationRequest{
		Limit: 10,
	})
	s.OK()

	var listResp httpx.WebResponse[[]model.InvitationResponse]
	listResp = testutil.DecodeOther[[]model.InvitationResponse](s)
	s.GreaterOrEqual(len(listResp.Data), 2)
}

func (s *InvitationSuite) Test_GetPublic_Successfully() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/invitations", ws.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles: []string{"core:member"},
	})
	s.Created()
	token := s.GetResponse().Data.Token

	// Access public preview without auth credentials
	s.Request(fmt.Sprintf("/invitations/%s", token), fiber.MethodGet, nil)
	s.OK()

	var pubResp httpx.WebResponse[model.PublicInvitationResponse]
	pubResp = testutil.DecodeOther[model.PublicInvitationResponse](s)
	s.Equal(ws.ID, pubResp.Data.WorkspaceID)
	s.Equal("Acme Corp", pubResp.Data.WorkspaceName)
	s.Equal([]string{"core:member"}, pubResp.Data.Roles)
	s.True(pubResp.Data.IsValid)
}

func (s *InvitationSuite) Test_GetPublic_NotFound() {
	s.Request("/invitations/non-existent-token-xyz", fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *InvitationSuite) Test_Accept_Successfully() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/invitations", ws.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles: []string{"core:member"},
	})
	s.Created()
	token := s.GetResponse().Data.Token

	// Switch to a new user
	charlie := s.AsUser("charlie@example.com", "Charlie")

	s.Request(fmt.Sprintf("/invitations/%s/accept", token), fiber.MethodPost, nil)
	s.Created()

	var memberResp httpx.WebResponse[model.MemberResponse]
	memberResp = testutil.DecodeOther[model.MemberResponse](s)
	s.Equal(ws.ID, memberResp.Data.WorkspaceID)
	s.Equal(charlie.ID, memberResp.Data.User.ID)
	s.Equal([]string{"core:member"}, memberResp.Data.Roles)
}

func (s *InvitationSuite) Test_Accept_AlreadyMember() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/invitations", ws.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles: []string{"core:member"},
	})
	s.Created()
	token := s.GetResponse().Data.Token

	dave := s.AsUser("dave@example.com", "Dave")
	s.AddMember(ws.ID, dave.ID, "core:member")

	// Dave tries to accept
	s.Request(fmt.Sprintf("/invitations/%s/accept", token), fiber.MethodPost, nil)
	s.AssertStatus(fiber.StatusConflict)
}

func (s *InvitationSuite) Test_Accept_OwnerForbidden() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/invitations", ws.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles: []string{"core:member"},
	})
	s.Created()
	token := s.GetResponse().Data.Token

	// Workspace owner tries to accept their own invite link
	s.Request(fmt.Sprintf("/invitations/%s/accept", token), fiber.MethodPost, nil)
	s.AssertStatus(fiber.StatusConflict)
}

func (s *InvitationSuite) Test_Accept_Expired() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	// Create expired invitation in db
	past := time.Now().Add(-2 * time.Hour)
	client := testutil.GetEntCli()
	inv, err := client.WorkspaceInvitation.Create().
		SetWorkspaceID(ws.ID).
		SetToken("expired-token-123").
		SetRoles([]string{"core:member"}).
		SetExpiresAt(past).
		Save(s.Context)
	s.Require().NoError(err)

	s.AsUser("eve@example.com", "Eve")
	s.Request(fmt.Sprintf("/invitations/%s/accept", inv.Token), fiber.MethodPost, nil)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *InvitationSuite) Test_Accept_MaxUsesReached() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/invitations", ws.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles:   []string{"core:member"},
		MaxUses: 1,
	})
	s.Created()
	token := s.GetResponse().Data.Token

	// First user accepts: succeeds
	s.AsUser("user1@example.com", "User One")
	s.Request(fmt.Sprintf("/invitations/%s/accept", token), fiber.MethodPost, nil)
	s.Created()

	// Second user accepts: limit reached
	s.AsUser("user2@example.com", "User Two")
	s.Request(fmt.Sprintf("/invitations/%s/accept", token), fiber.MethodPost, nil)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *InvitationSuite) Test_Accept_PersonalWorkspace_Conflict() {
	owner := s.EnsureUser(ent.User{Name: "PW Owner Accept", Email: "pw-owner-accept@example.com"})
	client := testutil.GetEntCli()
	pw, err := client.Workspace.Create().
		SetName("Personal Workspace").
		SetOwnerID(owner.ID).
		SetPersonal(true).
		Save(s.Context)
	s.Require().NoError(err)

	inv, err := client.WorkspaceInvitation.Create().
		SetWorkspaceID(pw.ID).
		SetToken("personal-token-test").
		SetRoles([]string{"core:member"}).
		Save(s.Context)
	s.Require().NoError(err)

	s.AsUser("bob@example.com", "Bob")
	s.Request(fmt.Sprintf("/invitations/%s/accept", inv.Token), fiber.MethodPost, nil)
	s.AssertStatus(fiber.StatusConflict)

	errRes := s.GetErrorResponse()
	s.Equal("WORKSPACE_PERSONAL_MEMBER", errRes.Error.Code)
}

func (s *InvitationSuite) Test_Revoke_Successfully() {
	ws := s.ensureWorkspace("Acme Corp", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/invitations", ws.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles: []string{"core:member"},
	})
	s.Created()
	inv := s.GetResponse().Data

	// Revoke
	s.Request(fmt.Sprintf("/workspaces/%d/invitations/%d", ws.ID, inv.ID), fiber.MethodDelete, nil)
	s.OK()

	// Trying to accept revoked link
	s.AsUser("frank@example.com", "Frank")
	s.Request(fmt.Sprintf("/invitations/%s/accept", inv.Token), fiber.MethodPost, nil)
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *InvitationSuite) Test_Create_WithEmail_DispatchesEmail() {
	mailer := testutil.GetMailer()
	s.Require().NotNil(mailer)
	memDriver, ok := mailer.Driver().(*mailx.MemoryDriver)
	s.Require().True(ok)
	memDriver.Reset()

	ws := s.ensureWorkspace("Email Dispatch WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/invitations", ws.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles: []string{"core:member"},
		Email: "invited.member@example.com",
	})
	s.Created()

	s.Eventually(func() bool {
		return len(memDriver.SentMessages()) == 1
	}, 2*time.Second, 50*time.Millisecond)

	msg := memDriver.LastMessage()
	s.Require().NotNil(msg)
	s.Equal("invited.member@example.com", msg.To[0].Address)
	s.Contains(msg.Subject, "Email Dispatch WS")
	s.Contains(msg.HTMLBody, "Email Dispatch WS")
	s.Contains(msg.HTMLBody, s.GetResponse().Data.Token)
}

func (s *InvitationSuite) Test_Send_Successfully() {
	mailer := testutil.GetMailer()
	s.Require().NotNil(mailer)
	memDriver, ok := mailer.Driver().(*mailx.MemoryDriver)
	s.Require().True(ok)
	memDriver.Reset()

	ws := s.ensureWorkspace("Resend WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/invitations", ws.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles: []string{"core:member"},
	})
	s.Created()
	inv := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/invitations/%d/send", ws.ID, inv.ID), fiber.MethodPost, model.SendInvitationRequest{
		Email: "resend.target@example.com",
	})
	s.OK()

	s.Eventually(func() bool {
		return len(memDriver.SentMessages()) == 1
	}, 2*time.Second, 50*time.Millisecond)

	msg := memDriver.LastMessage()
	s.Require().NotNil(msg)
	s.Equal("resend.target@example.com", msg.To[0].Address)
	s.Contains(msg.Subject, "Resend WS")
	s.Contains(msg.HTMLBody, inv.Token)
}

func (s *InvitationSuite) Test_Send_InvalidEmail() {
	ws := s.ensureWorkspace("Invalid Email WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/invitations", ws.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles: []string{"core:member"},
	})
	s.Created()
	inv := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/invitations/%d/send", ws.ID, inv.ID), fiber.MethodPost, model.SendInvitationRequest{
		Email: "not-an-email",
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *InvitationSuite) Test_Send_Revoked() {
	ws := s.ensureWorkspace("Revoked Send WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/invitations", ws.ID), fiber.MethodPost, model.CreateInvitationRequest{
		Roles: []string{"core:member"},
	})
	s.Created()
	inv := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/invitations/%d", ws.ID, inv.ID), fiber.MethodDelete, nil)
	s.OK()

	s.Request(fmt.Sprintf("/workspaces/%d/invitations/%d/send", ws.ID, inv.ID), fiber.MethodPost, model.SendInvitationRequest{
		Email: "revoked@example.com",
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func TestInvitationSuite(t *testing.T) {
	suite.Run(t, new(InvitationSuite))
}
