package testutil

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Role mirrors the subject names in authz/policy.csv.
type Role string

const (
	RoleAdmin      Role = "admin"
	RoleMember     Role = "member"
	RoleFinance    Role = "finance"
	RoleOwner      Role = "owner"
	RoleSuperAdmin Role = "superadmin"
)

// EndpointSuite is the generic base for HTTP endpoint tests. It reuses the
// package-level app/db/enforcer/jwksMock wired up in init(), so no per-suite
// bootstrap is needed. Embed it:
//
//	type AccountSuite struct{ testutil.EndpointSuite }
//	func TestAccountSuite(t *testing.T) { suite.Run(t, new(AccountSuite)) }
//
//	func (s *AccountSuite) TestCreate_Forbidden() {
//		token := s.Token(testutil.RoleMember)
//		resp := s.Do(fiber.MethodPost, "/accounts", token, testutil.JSONBody(s.T(), payload))
//		s.Equal(fiber.StatusForbidden, resp.StatusCode)
//	}
//
// Every test method starts against an empty DB and a fresh WorkspaceID/UserID.
type EndpointSuite struct {
	suite.Suite

	WorkspaceID uuid.UUID
	UserID      uuid.UUID
}

func (s *EndpointSuite) SetupTest() {
	s.WorkspaceID = uuid.New()
	s.UserID = uuid.New()
	s.truncateAll()
}

func (s *EndpointSuite) TearDownTest() {
	s.truncateAll()
}

// App returns the shared, already-bootstrapped fiber app.
func (s *EndpointSuite) App() *fiber.App { return app }

// DB returns the shared ent client, for seeding/asserting outside HTTP.
func (s *EndpointSuite) DB() *ent.Client { return db }

// Token signs a JWT for a fake authenticated user in this test's workspace,
// carrying the given org-level roles (what RequirePermission/casbin checks).
func (s *EndpointSuite) Token(roles ...Role) string {
	roleStrs := make([]string, len(roles))
	for i, r := range roles {
		roleStrs[i] = string(r)
	}

	return SignToken(s.T(), jwt.MapClaims{
		"id":                     s.UserID.String(),
		"name":                   "Test User",
		"avatar":                 "",
		"activeTeamId":           s.WorkspaceID.String(),
		"activeTeamName":         "Test Workspace",
		"activeOrganizationRole": roleStrs,
		"role":                   roleStrs,
	})
}

// Do performs an HTTP request against the shared app. Pass an empty token
// to hit the endpoint unauthenticated.
func (s *EndpointSuite) Do(method, path, token string, body io.Reader) *http.Response {
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := app.Test(req)
	s.Require().NoError(err)
	return resp
}

// JSONBody marshals v for use as a request body.
func JSONBody(t *testing.T, v any) io.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewReader(b)
}

// DecodeEnvelope decodes a success envelope response body into T.
func DecodeEnvelope[T any](t *testing.T, resp *http.Response, env *model.Envelope[T]) {
	t.Helper()
	defer resp.Body.Close()
	require.NoError(t, json.NewDecoder(resp.Body).Decode(env))
}

// DecodeError decodes an error envelope response body.
func DecodeError(t *testing.T, resp *http.Response) model.ErrorEnvelope {
	t.Helper()
	defer resp.Body.Close()

	var env model.ErrorEnvelope
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&env))
	return env
}

var rawDB *sql.DB

// truncateAll wipes every table so each test starts clean, without needing
// to hardcode entity names — new ent schemas are picked up automatically.
// Uses a raw connection (not the ent client) so it bypasses WorkspaceHook/
// WorkspaceInterceptor, which require an authenticated user in ctx.
func (s *EndpointSuite) truncateAll() {
	if rawDB == nil {
		conn, err := sql.Open("sqlite3", "file:ent?mode=memory&cache=shared&_fk=1")
		s.Require().NoError(err)
		conn.SetMaxOpenConns(1) // PRAGMA is per-connection; pin to one
		rawDB = conn
	}

	ctx := context.Background()
	s.Require().NoError(rawDB.PingContext(ctx))

	_, err := rawDB.ExecContext(ctx, "PRAGMA foreign_keys = OFF")
	s.Require().NoError(err)
	defer rawDB.ExecContext(ctx, "PRAGMA foreign_keys = ON")

	rows, err := rawDB.QueryContext(ctx, `
		SELECT name FROM sqlite_master
		WHERE type = 'table' AND name NOT LIKE 'sqlite_%'
	`)
	s.Require().NoError(err)

	var tables []string
	for rows.Next() {
		var name string
		s.Require().NoError(rows.Scan(&name))
		tables = append(tables, name)
	}
	s.Require().NoError(rows.Err())
	rows.Close()

	for _, t := range tables {
		_, err := rawDB.ExecContext(ctx, `DELETE FROM "`+t+`"`)
		s.Require().NoError(err)
	}
}
