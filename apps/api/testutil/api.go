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
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ApiTestSuite[T any] struct {
	suite.Suite
	HttpResponse *http.Response
	User         *core.UserContext
	Cookies      []*http.Cookie
	Context      context.Context
}

func (s *ApiTestSuite[T]) SetupTest() {
	s.Cookies = nil
	usr := s.EnsureUser(ent.User{
		Name:  "Test User",
		Email: "test@example.com",
	})

	s.User = &core.UserContext{
		ID:            usr.ID,
		Name:          "Test User",
		EmailVerified: true,
		Roles:         []core.UserRole{core.UserRoleAdmin},
	}

	s.Context = context.Background()
}

func (s *ApiTestSuite[T]) SetCookie(cookie *http.Cookie) {
	for i, c := range s.Cookies {
		if c.Name == cookie.Name {
			s.Cookies[i] = cookie
			return
		}
	}
	s.Cookies = append(s.Cookies, cookie)
}

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

func (s *ApiTestSuite[T]) ClearCookies() {
	s.Cookies = nil
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
	for _, c := range s.Cookies {
		req.AddCookie(c)
	}

	var err error
	s.HttpResponse = nil
	s.HttpResponse, err = api.Test(req)
	require.NoError(s.T(), err)
}

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

func (s *ApiTestSuite[T]) Request(path string, method string, requestBody any) {
	s.T().Helper()
	s.RequestWithClaims(path, method, requestBody, nil)
}

func (s *ApiTestSuite[T]) RequestMultipart(path string, method string, fieldName string, filename string, content []byte) {
	s.T().Helper()
	s.RequestMultipartWithClaims(path, method, fieldName, filename, content, nil)
}

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

func (s *ApiTestSuite[T]) GetResponse() httpx.WebResponse[T] {
	var env httpx.WebResponse[T]
	s.T().Helper()
	defer s.HttpResponse.Body.Close()
	require.NoError(s.T(), json.NewDecoder(s.HttpResponse.Body).Decode(&env))
	return env
}

func (s *ApiTestSuite[T]) PagedResponse() httpx.WebResponse[[]T] {
	var env httpx.WebResponse[[]T]
	s.T().Helper()
	defer s.HttpResponse.Body.Close()
	require.NoError(s.T(), json.NewDecoder(s.HttpResponse.Body).Decode(&env))
	return env
}

func DecodeOther[R any](s interface{ GetHttpResponse() *http.Response }) httpx.WebResponse[R] {
	var env httpx.WebResponse[R]
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
