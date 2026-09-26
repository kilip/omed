package testutil

import (
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
	"github.com/stretchr/testify/assert"
)

type TestUser struct {
	UserID        string
	Name          string
	WorkspaceID   string
	UserRole      []model.UserRole
	WorkspaceRole []model.WorkspaceRole
}

type ApiRequest struct {
	Path   string
	Method string
	Body   any
}

type ApiTester struct {
	user *TestUser
}

func (t *ApiTester) WithUser(user *TestUser) {
	if t.user == nil {
		t.user = &TestUser{
			Name:          "Test User",
			UserID:        shared.GenerateID().String(),
			WorkspaceID:   shared.GenerateID().String(),
			UserRole:      []model.UserRole{model.UserRoleUser},
			WorkspaceRole: []model.WorkspaceRole{model.WorkspaceRoleMember},
		}
	}

	if user.WorkspaceRole != nil {
		t.user.WorkspaceRole = user.WorkspaceRole
	}
}

func (t *ApiTester) WithWorkspaceRole(role model.WorkspaceRole) {
	t.WithUser(&TestUser{
		WorkspaceRole: []model.WorkspaceRole{model.WorkspaceRole(role)},
	})
}

func (t *ApiTester) Execute(T *testing.T, request ApiRequest) {
	var token string

	T.Helper()
	if t.user != nil {
		token = SignToken(T, jwt.MapClaims{
			"id":                     t.user.UserID,
			"role":                   t.user.UserRole,
			"activeTeamID":           t.user.WorkspaceID,
			"activeOrganizationRole": t.user.WorkspaceRole,
		})

		assert.NotEmpty(T, token)
	}

}
