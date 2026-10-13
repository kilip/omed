package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	"entgo.io/ent/dialect/sql"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/user"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// ApiTestSuite provides an end-to-end HTTP integration testing harness with embedded Fiber test client,
// automatic test user provisioning, JWT signing, cookie management, and response assertions.
type ApiTestSuite[T any] struct {
	suite.Suite
	HttpResponse *http.Response
	User         *base.UserContext
	Cookies      []*http.Cookie
	Context      context.Context
}

// SetupTest prepares test state before each test case runs, resetting cookies and provisioning a default test user.
func (s *ApiTestSuite[T]) SetupTest() {
	s.Cookies = nil
	usr := s.EnsureUser(ent.User{
		Name:  "Test User",
		Email: "test@example.com",
	})

	s.User = &base.UserContext{
		ID:            usr.ID,
		Name:          "Test User",
		EmailVerified: true,
		Roles:         []base.UserRole{base.UserRoleAdmin},
	}

	s.Context = context.Background()
}

// AsUser switches the active test user to the specified email and optional global roles.
func (s *ApiTestSuite[T]) AsUser(email, name string, globalRoles ...base.UserRole) *ent.User {
	usr := s.EnsureUser(ent.User{
		Name:  name,
		Email: email,
	})
	roles := globalRoles
	if len(roles) == 0 {
		roles = []base.UserRole{base.UserRoleUser}
	}
	s.User = &base.UserContext{
		ID:            usr.ID,
		Name:          name,
		EmailVerified: true,
		Roles:         roles,
	}
	return usr
}

// AddMember adds a user as a member of a workspace with the given roles in the test DB.
func (s *ApiTestSuite[T]) AddMember(wid, uid int64, roles ...string) {
	ctx := context.Background()
	_, err := entClient.WorkspaceMember.Create().
		SetWorkspaceID(wid).
		SetUserID(uid).
		SetRoles(roles).
		Save(ctx)
	require.NoError(s.T(), err)
}

// SetCookie appends or updates an HTTP cookie to be sent with subsequent test requests.
func (s *ApiTestSuite[T]) SetCookie(cookie *http.Cookie) {
	for i, c := range s.Cookies {
		if c.Name == cookie.Name {
			s.Cookies[i] = cookie
			return
		}
	}
	s.Cookies = append(s.Cookies, cookie)
}

// EnsureUser creates or updates a user in the test database and returns the resulting user entity.
func (s *ApiTestSuite[T]) EnsureUser(req ent.User) *ent.User {
	ctx := context.Background()
	id, err := entClient.User.Create().
		SetName(req.Name).
		SetEmail(req.Email).
		SetNillableAvatar(&req.Avatar).
		OnConflict(
			sql.ConflictColumns(user.FieldEmail),
		).
		UpdateNewValues().
		ID(ctx)
	if err != nil {
		require.NoError(s.T(), err)
	}

	usr, err := entClient.User.Get(ctx, id)
	if err != nil {
		require.NoError(s.T(), err)
	}
	return usr
}

// ClearCookies clears all stored cookies from the test suite.
func (s *ApiTestSuite[T]) ClearCookies() {
	s.Cookies = nil
}

func (s *ApiTestSuite[T]) jsonBody(v any) io.Reader {
	s.T().Helper()
	b, err := json.Marshal(v)
	require.NoError(s.T(), err)
	return bytes.NewReader(b)
}

// RequestWithToken sends an HTTP request with an explicit Bearer JWT token and request body.
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
	for _, c := range s.Cookies {
		req.AddCookie(c)
	}

	var err error
	s.HttpResponse = nil
	s.HttpResponse, err = api.Test(req)
	require.NoError(s.T(), err)
}

// RequestWithClaims signs a JWT token with the current test user context and additional claims, then executes the request.
func (s *ApiTestSuite[T]) RequestWithClaims(path string, method string, requestBody any, extraClaims jwt.MapClaims) {
	s.T().Helper()

	claims := jwt.MapClaims{
		"id":             s.User.ID,
		"sub":            s.User.ID,
		"name":           s.User.Name,
		"email_verified": true,
		"roles":          s.User.Roles,
	}
	for k, v := range extraClaims {
		claims[k] = v
	}

	token := SignToken(s.T(), claims)
	s.RequestWithToken(path, method, requestBody, token)
}

// Request executes an authenticated HTTP request using the default test user credentials.
func (s *ApiTestSuite[T]) Request(path string, method string, requestBody any) {
	s.T().Helper()
	s.RequestWithClaims(path, method, requestBody, nil)
}

// RequestMultipart sends a multipart/form-data upload request authenticated with the test user.
func (s *ApiTestSuite[T]) RequestMultipart(path string, method string, fieldName string, filename string, content []byte) {
	s.T().Helper()
	s.RequestMultipartWithClaims(path, method, fieldName, filename, content, nil)
}

// RequestMultipartWithClaims sends a multipart/form-data request with custom claims.
func (s *ApiTestSuite[T]) RequestMultipartWithClaims(path string, method string, fieldName string, filename string, content []byte, extraClaims jwt.MapClaims) {
	s.T().Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if fieldName != "" {
		part, err := w.CreateFormFile(fieldName, filename)
		require.NoError(s.T(), err)
		_, err = part.Write(content)
		require.NoError(s.T(), err)
	}
	require.NoError(s.T(), w.Close())

	req := httptest.NewRequest(method, path, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())

	claims := jwt.MapClaims{
		"id":             s.User.ID,
		"sub":            s.User.ID,
		"name":           s.User.Name,
		"email_verified": true,
		"roles":          s.User.Roles,
	}
	for k, v := range extraClaims {
		claims[k] = v
	}

	token := SignToken(s.T(), claims)
	req.Header.Set("Authorization", "Bearer "+token)
	for _, c := range s.Cookies {
		req.AddCookie(c)
	}

	var err error
	s.HttpResponse = nil
	s.HttpResponse, err = api.Test(req)
	require.NoError(s.T(), err)
}

// GetResponse decodes the HTTP response JSON body into a WebResponse[T].
func (s *ApiTestSuite[T]) GetResponse() httpx.WebResponse[T] {
	var env httpx.WebResponse[T]
	s.T().Helper()
	defer s.HttpResponse.Body.Close()
	require.NoError(s.T(), json.NewDecoder(s.HttpResponse.Body).Decode(&env))
	return env
}

// PagedResponse decodes the HTTP response JSON body into a cursor-paginated WebResponse[[]T].
func (s *ApiTestSuite[T]) PagedResponse() httpx.WebResponse[[]T] {
	var env httpx.WebResponse[[]T]
	s.T().Helper()
	defer s.HttpResponse.Body.Close()
	require.NoError(s.T(), json.NewDecoder(s.HttpResponse.Body).Decode(&env))
	return env
}

// DecodeOther decodes the response body into an arbitrary WebResponse[R] envelope.
func DecodeOther[R any](s interface{ GetHttpResponse() *http.Response }) httpx.WebResponse[R] {
	var env httpx.WebResponse[R]
	resp := s.GetHttpResponse()
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&env)
	return env
}

// GetHttpResponse returns the raw recorded *http.Response from the last executed request.
func (s *ApiTestSuite[T]) GetHttpResponse() *http.Response {
	return s.HttpResponse
}

// AssertStatus asserts that the recorded response status code matches the expected code.
func (s *ApiTestSuite[T]) AssertStatus(code int) {
	s.T().Helper()
	s.Equal(code, s.HttpResponse.StatusCode)
}

// OK asserts that the recorded HTTP response has status 200 OK.
func (s *ApiTestSuite[T]) OK() {
	s.T().Helper()
	s.AssertStatus(fiber.StatusOK)
}

// Created asserts that the recorded HTTP response has status 201 Created.
func (s *ApiTestSuite[T]) Created() {
	s.T().Helper()
	s.AssertStatus(fiber.StatusCreated)
}

// NoContent asserts that the recorded HTTP response has status 204 No Content.
func (s *ApiTestSuite[T]) NoContent() {
	s.T().Helper()
	s.AssertStatus(fiber.StatusNoContent)
}
