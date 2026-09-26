package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestUser struct {
	UserID             string
	Name               string
	WorkspaceID        string
	UserRole           []model.UserRole
	UpdatedAt          time.Time
	WorkspaceRole      []model.WorkspaceRole
	WorkspaceName      string
	WorkspaceUpdatedAt time.Time
}

type ApiRequest struct {
	Path   string
	Method string
	Body   any
}

type ApiTester struct {
	user     *TestUser
	Response *http.Response
}

func NewApiTester() *ApiTester {
	return &ApiTester{
		user: &TestUser{
			Name:               "Test User",
			UserID:             shared.GenerateID().String(),
			WorkspaceID:        shared.GenerateID().String(),
			UserRole:           []model.UserRole{model.UserRoleUser},
			UpdatedAt:          time.Now(),
			WorkspaceRole:      []model.WorkspaceRole{model.WorkspaceRoleMember},
			WorkspaceName:      "Test Workspace",
			WorkspaceUpdatedAt: time.Now(),
		},
	}
}

func (a *ApiTester) WithWorkspaceRole(role model.WorkspaceRole) {
	a.user.WorkspaceRole = []model.WorkspaceRole{model.WorkspaceRole(role)}
}

func (a *ApiTester) Execute(T *testing.T, request ApiRequest) {
	var token string

	T.Helper()
	if a.user != nil {
		token = SignToken(T, jwt.MapClaims{
			"id":                     a.user.UserID,
			"name":                   a.user.Name,
			"role":                   a.user.UserRole,
			"updatedAt":              a.user.UpdatedAt,
			"activeTeamID":           a.user.WorkspaceID,
			"activeTeamName":         a.user.WorkspaceName,
			"activeTeamUpdatedAt":    a.user.WorkspaceUpdatedAt,
			"activeOrganizationRole": a.user.WorkspaceRole,
		})

		assert.NotEmpty(T, token)
	}

	var body io.Reader

	if request.Body != nil {
		body = JSONBody(T, request.Body)
	}

	req := httptest.NewRequest(request.Method, request.Path, body)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := fiberApp.Test(req)
	require.NoError(T, err)

	a.Response = resp
}

func (a *ApiTester) OK(t *testing.T) {
	assert.Equal(t, fiber.StatusOK, a.Response.StatusCode)
}

func DecodeResponse[T any](t *testing.T, resp *http.Response, env *model.WebResponse[T]) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(env))
}

func JSONBody(t *testing.T, v any) io.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(b)
}
