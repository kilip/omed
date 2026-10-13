package authz_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/kilip/omed/api/internal/platform/authz/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

//go:generate go run go.uber.org/mock/mockgen -source context.go -destination mock/mock_resolver.go -package mock

type AuthorizerSuite struct {
	suite.Suite
	resolver *mock.MockMembershipResolver
	az       *authz.Authorizer
	user     *base.UserContext
}

func (s *AuthorizerSuite) SetupTest() {
	ctl := gomock.NewController(s.T())
	s.resolver = mock.NewMockMembershipResolver(ctl)

	reg := newFinRegistry(s.T())
	az, err := authz.NewWithRegistry(reg, s.resolver)
	s.Require().NoError(err)
	s.az = az
	s.user = &base.UserContext{ID: 7, Roles: []base.UserRole{base.UserRoleUser}}
}

func (s *AuthorizerSuite) app(handlers ...fiber.Handler) *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			return c.Status(base.MapError(err).Status).SendString(err.Error())
		},
	})
	app.Use(func(c fiber.Ctx) error {
		if s.user != nil {
			c.Locals(base.USER_CONTEXT_KEY, *s.user)
		}
		return c.Next()
	})
	rest := make([]any, 0, len(handlers))
	for _, h := range handlers[1:] {
		rest = append(rest, h)
	}
	rest = append(rest, func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	app.Get("/workspaces/:wid/invoices", handlers[0], rest...)
	return app
}

func (s *AuthorizerSuite) status(app *fiber.App, path string) int {
	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, path, nil))
	s.Require().NoError(err)
	return resp.StatusCode
}

func (s *AuthorizerSuite) Test_Require_AllowedByRole() {
	s.resolver.EXPECT().ResolveMembership(gomock.Any(), int64(7), int64(1)).
		Return(authz.Membership{WorkspaceID: 1, UserID: 7, Roles: []string{"fin:accountant"}}, nil)
	s.Equal(fiber.StatusOK, s.status(s.app(s.az.Require("fin:invoice", "read")), "/workspaces/1/invoices"))
}

func (s *AuthorizerSuite) Test_Require_Forbidden() {
	s.resolver.EXPECT().ResolveMembership(gomock.Any(), int64(7), int64(1)).
		Return(authz.Membership{WorkspaceID: 1, UserID: 7, Roles: []string{"fin:viewer"}}, nil)
	s.Equal(fiber.StatusForbidden, s.status(s.app(s.az.Require("fin:invoice", "read")), "/workspaces/1/invoices"))
}

func (s *AuthorizerSuite) Test_Require_OwnerBypass() {
	s.resolver.EXPECT().ResolveMembership(gomock.Any(), int64(7), int64(1)).
		Return(authz.Membership{WorkspaceID: 1, UserID: 7, IsOwner: true}, nil)
	s.Equal(fiber.StatusOK, s.status(s.app(s.az.Require("fin:invoice", "delete")), "/workspaces/1/invoices"))
}

func (s *AuthorizerSuite) Test_Require_SuperadminNotMember() {
	s.user.Roles = []base.UserRole{base.UserRoleAdmin}
	s.resolver.EXPECT().ResolveMembership(gomock.Any(), int64(7), int64(1)).
		Return(authz.Membership{}, base.ErrNotFound)
	s.Equal(fiber.StatusNotFound, s.status(s.app(s.az.Require("fin:invoice", "read")), "/workspaces/1/invoices"))
}

func (s *AuthorizerSuite) Test_Require_InvalidWorkspaceID() {
	app := s.app(s.az.Require("fin:invoice", "read"))
	s.Equal(fiber.StatusBadRequest, s.status(app, "/workspaces/abc/invoices"))
	s.Equal(fiber.StatusBadRequest, s.status(app, "/workspaces/0/invoices"))
}

func (s *AuthorizerSuite) Test_Require_NoUser() {
	s.user = nil
	s.Equal(fiber.StatusUnauthorized, s.status(s.app(s.az.Require("fin:invoice", "read")), "/workspaces/1/invoices"))
}

func (s *AuthorizerSuite) Test_Member_ResolvesOnceAcrossMiddlewares() {
	s.resolver.EXPECT().ResolveMembership(gomock.Any(), int64(7), int64(1)).
		Return(authz.Membership{WorkspaceID: 1, UserID: 7, Roles: []string{"fin:accountant"}}, nil).
		Times(1)

	var canCreate, canDelete bool
	var perms []authz.Permission
	check := func(c fiber.Ctx) error {
		canCreate = s.az.Can(c, "fin:invoice", "create")
		canDelete = s.az.Can(c, "fin:invoice", "delete")
		perms = s.az.Permissions(c)
		return c.Next()
	}
	app := s.app(s.az.Member(), s.az.Require("fin:invoice", "read"), check)
	s.Equal(fiber.StatusOK, s.status(app, "/workspaces/1/invoices"))
	s.True(canCreate)
	s.False(canDelete)
	s.Len(perms, 3)
}

func (s *AuthorizerSuite) Test_RequireOwner() {
	s.resolver.EXPECT().ResolveMembership(gomock.Any(), int64(7), int64(1)).
		Return(authz.Membership{WorkspaceID: 1, UserID: 7, Roles: []string{"fin:admin"}}, nil)
	s.Equal(fiber.StatusForbidden, s.status(s.app(s.az.RequireOwner()), "/workspaces/1/invoices"))

	s.resolver.EXPECT().ResolveMembership(gomock.Any(), int64(7), int64(2)).
		Return(authz.Membership{WorkspaceID: 2, UserID: 7, IsOwner: true}, nil)
	s.Equal(fiber.StatusOK, s.status(s.app(s.az.RequireOwner()), "/workspaces/2/invoices"))
}

func (s *AuthorizerSuite) Test_RequireOwner_NotMember() {
	s.resolver.EXPECT().ResolveMembership(gomock.Any(), int64(7), int64(1)).
		Return(authz.Membership{}, base.ErrNotFound)
	s.Equal(fiber.StatusNotFound, s.status(s.app(s.az.RequireOwner()), "/workspaces/1/invoices"))
}

func (s *AuthorizerSuite) Test_RequireGlobal() {
	app := s.app(authz.RequireGlobal(base.UserRoleAdmin))
	s.Equal(fiber.StatusForbidden, s.status(app, "/workspaces/1/invoices"))

	s.user.Roles = []base.UserRole{base.UserRoleAdmin}
	s.Equal(fiber.StatusOK, s.status(app, "/workspaces/1/invoices"))

	s.user = nil
	s.Equal(fiber.StatusUnauthorized, s.status(app, "/workspaces/1/invoices"))
}

func (s *AuthorizerSuite) Test_HasRole() {
	s.True(s.az.HasRole("fin:accountant"))
	s.False(s.az.HasRole("core:admin"))
}

func TestAuthorizerSuite(t *testing.T) {
	suite.Run(t, new(AuthorizerSuite))
}

func TestCan_WithoutMembership(t *testing.T) {
	az, err := authz.NewWithRegistry(newFinRegistry(t), nil)
	require.NoError(t, err)
	assert.False(t, az.Can(context.Background(), "fin:invoice", "read"))
	assert.Empty(t, az.Permissions(context.Background()))
}

func TestCan_ContextWithMembership(t *testing.T) {
	az, err := authz.NewWithRegistry(newFinRegistry(t), nil)
	require.NoError(t, err)

	ctx := authz.ContextWithMembership(context.Background(), authz.Membership{WorkspaceID: 1, IsOwner: true})
	assert.True(t, az.Can(ctx, "fin:anything", "delete"))
	assert.Equal(t, []authz.Permission{{Object: "*", Action: "*"}}, az.Permissions(ctx))

	m, ok := authz.MembershipFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, int64(1), m.WorkspaceID)

	_, ok = authz.MembershipFromContext(nil) //nolint:staticcheck // nil context is part of the contract
	assert.False(t, ok)
}

func TestNew_DefaultRegistry(t *testing.T) {
	az, err := authz.New(nil)
	require.NoError(t, err)
	assert.NotNil(t, az)
}
