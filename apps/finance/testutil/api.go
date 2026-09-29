package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ApiTestSuite[T any] struct {
	suite.Suite
	httpResponse *http.Response
	User         *shared.AuthenticatedUser
}

func (s *ApiTestSuite[T]) SetupTest() {
	s.User = &shared.AuthenticatedUser{
		Name:           "Test User",
		ID:             shared.GenerateID(),
		WorkspaceID:    shared.GenerateID(),
		WorkspaceRoles: []shared.WorkspaceRole{shared.WorkspaceRoleOwner},
		WorkspaceName:  "Test Workspace",
	}
}
func (s *ApiTestSuite[T]) jsonBody(v any) io.Reader {
	s.T().Helper()
	b, err := json.Marshal(v)
	require.NoError(s.T(), err)
	return bytes.NewReader(b)
}

func (s *ApiTestSuite[T]) Request(path string, method string, requestBody any) {
	s.T().Helper()

	token := SignToken(s.T(), jwt.MapClaims{
		"id":                      s.User.ID,
		"name":                    s.User.Name,
		"activeTeamID":            s.User.WorkspaceID,
		"activeTeamName":          s.User.WorkspaceName,
		"activeOrganizationRoles": s.User.WorkspaceRoles,
	})

	var body io.Reader
	if requestBody != nil {
		body = s.jsonBody(requestBody)
	}

	req := httptest.NewRequest(method, path, body)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+token)

	var err error
	s.httpResponse = nil
	s.httpResponse, err = api.Test(req)
	require.NoError(s.T(), err)
}

func (s *ApiTestSuite[T]) GetResponse() model.WebResponse[T] {
	var env model.WebResponse[T]
	s.T().Helper()
	defer s.httpResponse.Body.Close()
	require.NoError(s.T(), json.NewDecoder(s.httpResponse.Body).Decode(&env))
	return env
}

func (s *ApiTestSuite[T]) PagedResponse() model.WebResponse[[]T] {
	var env model.WebResponse[[]T]
	s.T().Helper()
	defer s.httpResponse.Body.Close()
	require.NoError(s.T(), json.NewDecoder(s.httpResponse.Body).Decode(&env))
	return env
}

func (s *ApiTestSuite[T]) AssertStatus(code int) {
	s.T().Helper()
	s.Equal(code, s.httpResponse.StatusCode)
}

func (s *ApiTestSuite[T]) OK() {
	s.T().Helper()
	s.AssertStatus(fiber.StatusOK)
}

func (s *ApiTestSuite[T]) Created() {
	s.T().Helper()
	s.AssertStatus(fiber.StatusCreated)
}

func (s *ApiTestSuite[T]) NoContent() {
	s.T().Helper()
	s.AssertStatus(fiber.StatusNoContent)
}
