package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared/util"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ApiTestSuite[T any] struct {
	suite.Suite
	HttpResponse *http.Response
	User         *core.AuthenticatedUser
}

func (s *ApiTestSuite[T]) SetupTest() {
	s.User = &core.AuthenticatedUser{
		Name:           "Test User",
		ID:             util.GenerateID(),
		WorkspaceID:    util.GenerateID(),
		WorkspaceRoles: []core.WorkspaceRole{core.WorkspaceRoleOwner},
		WorkspaceName:  "Test Workspace",
	}
}
func (s *ApiTestSuite[T]) jsonBody(v any) io.Reader {
	s.T().Helper()
	b, err := json.Marshal(v)
	require.NoError(s.T(), err)
	return bytes.NewReader(b)
}

func (s *ApiTestSuite[T]) RequestWithToken(path string, method string, requestBody any, token string) {
	s.T().Helper()

	var body io.Reader
	if requestBody != nil {
		body = s.jsonBody(requestBody)
	}

	req := httptest.NewRequest(method, path, body)

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	var err error
	s.HttpResponse = nil
	s.HttpResponse, err = api.Test(req)
	require.NoError(s.T(), err)
}

func (s *ApiTestSuite[T]) RequestWithClaims(path string, method string, requestBody any, extraClaims jwt.MapClaims) {
	s.T().Helper()

	claims := jwt.MapClaims{
		"id":                   s.User.ID,
		"name":                 s.User.Name,
		"activeWorkspaceId":    s.User.WorkspaceID,
		"activeWorkspaceName":  s.User.WorkspaceName,
		"activeWorkspaceRoles": s.User.WorkspaceRoles,
	}
	for k, v := range extraClaims {
		claims[k] = v
	}

	token := SignToken(s.T(), claims)
	s.RequestWithToken(path, method, requestBody, token)
}

func (s *ApiTestSuite[T]) Request(path string, method string, requestBody any) {
	s.T().Helper()
	s.RequestWithClaims(path, method, requestBody, nil)
}

func (s *ApiTestSuite[T]) GetResponse() model.WebResponse[T] {
	var env model.WebResponse[T]
	s.T().Helper()
	defer s.HttpResponse.Body.Close()
	require.NoError(s.T(), json.NewDecoder(s.HttpResponse.Body).Decode(&env))
	return env
}

func (s *ApiTestSuite[T]) PagedResponse() model.WebResponse[[]T] {
	var env model.WebResponse[[]T]
	s.T().Helper()
	defer s.HttpResponse.Body.Close()
	require.NoError(s.T(), json.NewDecoder(s.HttpResponse.Body).Decode(&env))
	return env
}

func DecodeOther[R any](s interface{ GetHttpResponse() *http.Response }) model.WebResponse[R] {
	var env model.WebResponse[R]
	resp := s.GetHttpResponse()
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&env)
	return env
}

func (s *ApiTestSuite[T]) GetHttpResponse() *http.Response {
	return s.HttpResponse
}

func (s *ApiTestSuite[T]) AssertStatus(code int) {
	s.T().Helper()
	s.Equal(code, s.HttpResponse.StatusCode)
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
