package test

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type ProviderSuite struct {
	testutil.ApiTestSuite[model.ProviderResponse]
	fakeServer *httptest.Server
	authFail   bool
}

func TestProviderSuite(t *testing.T) {
	suite.Run(t, new(ProviderSuite))
}

func TestProvider_ListModels(t *testing.T) {
	if f := flag.Lookup("test.run"); f == nil || f.Value.String() == "" {
		t.Skip("skipping duplicate suite run when no -run filter is specified")
	}
	suite.Run(t, new(ProviderSuite))
}

func (s *ProviderSuite) SetupSuite() {
	s.fakeServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authFail || r.Header.Get("Authorization") == "Bearer bad-key" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"message": "Invalid API key",
					"type":    "invalid_request_error",
					"code":    "invalid_api_key",
				},
			})
			return
		}

		if strings.HasSuffix(r.URL.Path, "/models") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"object": "list",
				"data": []map[string]any{
					{"id": "gpt-4o", "object": "model"},
					{"id": "gpt-4o-mini", "object": "model"},
				},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "gpt-4o-mini",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "pong",
					},
					"finish_reason": "stop",
				},
			},
		})
	}))

	s.ApiTestSuite.SetupSuite()
}

func (s *ProviderSuite) TearDownSuite() {
	if s.fakeServer != nil {
		s.fakeServer.Close()
	}
}

func (s *ProviderSuite) SetupTest() {
	s.authFail = false
	s.ApiTestSuite.SetupTest()
}

func (s *ProviderSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *ProviderSuite) TestCRUD() {
	ws := s.ensureWorkspace("Provider CRUD WS", s.User.ID)

	// 1. Create
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "OpenAI Test",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "gpt-4o-mini",
		APIKey:  "sk-test-secret-key-1234",
	})
	s.Created()
	created := s.GetResponse().Data
	s.Positive(created.ID)
	s.Equal("OpenAI Test", created.Name)
	s.Equal("openai_compat", created.Kind)
	s.Equal("gpt-4o-mini", created.Model)
	s.True(created.HasKey)
	s.Equal("1234", created.KeyHint)

	// 2. Get
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.OK()
	fetched := s.GetResponse().Data
	s.Equal(created.ID, fetched.ID)
	s.Equal("OpenAI Test", fetched.Name)

	// 3. List
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodGet, nil)
	s.OK()
	listResp := testutil.DecodeOther[[]model.ProviderResponse](s)
	s.NotEmpty(listResp.Data)

	// 4. Update
	newName := "OpenAI Renamed"
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws.ID, created.ID), fiber.MethodPatch, model.UpdateProviderRequest{
		Name: &newName,
	})
	s.OK()
	updated := s.GetResponse().Data
	s.Equal("OpenAI Renamed", updated.Name)

	// 5. Delete
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws.ID, created.ID), fiber.MethodDelete, nil)
	s.OK()

	// 6. Get after delete -> 404
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *ProviderSuite) TestSetDefault() {
	ws := s.ensureWorkspace("Provider SetDefault WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "Provider 1",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "m1",
		APIKey:  "key1",
	})
	s.Created()
	p1 := s.GetResponse().Data
	s.True(p1.IsDefault)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "Provider 2",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "m2",
		APIKey:  "key2",
	})
	s.Created()
	p2 := s.GetResponse().Data
	s.False(p2.IsDefault)

	// Set p2 as default
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d/default", ws.ID, p2.ID), fiber.MethodPost, nil)
	s.OK()
	s.True(s.GetResponse().Data.IsDefault)

	// Verify p1 is no longer default
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws.ID, p1.ID), fiber.MethodGet, nil)
	s.OK()
	s.False(s.GetResponse().Data.IsDefault)
}

func (s *ProviderSuite) TestPermissionMatrix() {
	ws := s.ensureWorkspace("Provider Perms WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "Seed Provider",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "m1",
		APIKey:  "k1",
	})
	s.Created()
	provider := s.GetResponse().Data

	// 1. Unauthenticated -> 401
	s.SessionToken = ""
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusUnauthorized)
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/models", ws.ID), fiber.MethodPost, model.ListModelsRequest{
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		APIKey:  "k1",
	})
	s.AssertStatus(fiber.StatusUnauthorized)
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d/models", ws.ID, provider.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusUnauthorized)

	// 2. Member with no role -> 403
	norole := s.AsUser("norole-prov@example.com", "No Role User")
	s.AddMember(ws.ID, norole.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusForbidden)
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/models", ws.ID), fiber.MethodPost, model.ListModelsRequest{
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		APIKey:  "k1",
	})
	s.AssertStatus(fiber.StatusForbidden)
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d/models", ws.ID, provider.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusForbidden)
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "Fail",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "m",
		APIKey:  "k",
	})
	s.AssertStatus(fiber.StatusForbidden)

	// 3. Viewer role: GET ok, POST/PATCH/DELETE/test 403
	viewer := s.AsUser("viewer-prov@example.com", "Viewer User")
	s.AddMember(ws.ID, viewer.ID, domain.RoleViewer)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws.ID, provider.ID), fiber.MethodGet, nil)
	s.OK()

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "Viewer Fail",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "m",
		APIKey:  "k",
	})
	s.AssertStatus(fiber.StatusForbidden)

	newName := "Viewer Fail Update"
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws.ID, provider.ID), fiber.MethodPatch, model.UpdateProviderRequest{
		Name: &newName,
	})
	s.AssertStatus(fiber.StatusForbidden)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws.ID, provider.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusForbidden)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d/test", ws.ID, provider.ID), fiber.MethodPost, nil)
	s.AssertStatus(fiber.StatusForbidden)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d/models", ws.ID, provider.ID), fiber.MethodGet, nil)
	s.OK()

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/models", ws.ID), fiber.MethodPost, model.ListModelsRequest{
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		APIKey:  "k",
	})
	s.AssertStatus(fiber.StatusForbidden)

	// 4. Admin role: all operations ok
	admin := s.AsUser("admin-prov@example.com", "Admin User")
	s.AddMember(ws.ID, admin.ID, domain.RoleAdmin)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws.ID, provider.ID), fiber.MethodGet, nil)
	s.OK()

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d/models", ws.ID, provider.ID), fiber.MethodGet, nil)
	s.OK()

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/models", ws.ID), fiber.MethodPost, model.ListModelsRequest{
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		APIKey:  "k",
	})
	s.OK()

	adminUpdatedName := "Admin Update"
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws.ID, provider.ID), fiber.MethodPatch, model.UpdateProviderRequest{
		Name: &adminUpdatedName,
	})
	s.OK()
}

func (s *ProviderSuite) TestCrossWorkspaceIsolation() {
	ws1 := s.ensureWorkspace("Provider WS 1", s.User.ID)
	ws2 := s.ensureWorkspace("Provider WS 2", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws1.ID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "WS1 Provider",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "m1",
		APIKey:  "key1",
	})
	s.Created()
	p1 := s.GetResponse().Data

	// Access p1 from ws2 -> 404
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws2.ID, p1.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)

	// Update p1 from ws2 -> 404
	newName := "Hacked"
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws2.ID, p1.ID), fiber.MethodPatch, model.UpdateProviderRequest{
		Name: &newName,
	})
	s.AssertStatus(fiber.StatusNotFound)

	// Delete p1 from ws2 -> 404
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws2.ID, p1.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *ProviderSuite) TestKeyNeverInAnyResponse() {
	ws := s.ensureWorkspace("Provider Key Leaks WS", s.User.ID)

	rawKey := "sk-super-secret-unleaked-key-9999"

	// Create
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "Leaky Provider",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "m1",
		APIKey:  rawKey,
	})
	s.Created()
	s.NotContains(string(s.HttpResponse.Header.Get("Content-Type")), rawKey)
	respCreate := s.GetResponse()
	bCreate, _ := json.Marshal(respCreate)
	s.NotContains(string(bCreate), rawKey)
	created := respCreate.Data
	s.True(created.HasKey)
	s.Equal("9999", created.KeyHint)

	// Get
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.OK()
	respGet := s.GetResponse()
	bGet, _ := json.Marshal(respGet)
	s.NotContains(string(bGet), rawKey)
	get := respGet.Data
	s.True(get.HasKey)
	s.Equal("9999", get.KeyHint)

	// List
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodGet, nil)
	s.OK()
	listResp := testutil.DecodeOther[[]model.ProviderResponse](s)
	bList, _ := json.Marshal(listResp)
	s.NotContains(string(bList), rawKey)
	s.NotEmpty(listResp.Data)
	s.True(listResp.Data[0].HasKey)
	s.Equal("9999", listResp.Data[0].KeyHint)

	// Update
	newName := "Updated Name"
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws.ID, created.ID), fiber.MethodPatch, model.UpdateProviderRequest{
		Name: &newName,
	})
	s.OK()
	respUpdate := s.GetResponse()
	bUpdate, _ := json.Marshal(respUpdate)
	s.NotContains(string(bUpdate), rawKey)
	updated := respUpdate.Data
	s.True(updated.HasKey)
	s.Equal("9999", updated.KeyHint)

	// ListModels (existing)
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d/models", ws.ID, created.ID), fiber.MethodGet, nil)
	s.OK()
	respModels := testutil.DecodeOther[[]string](s)
	bModels, _ := json.Marshal(respModels)
	s.NotContains(string(bModels), rawKey)

	// ListModelsConfig
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/models", ws.ID), fiber.MethodPost, model.ListModelsRequest{
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		APIKey:  rawKey,
	})
	s.OK()
	respConfigModels := testutil.DecodeOther[[]string](s)
	bConfigModels, _ := json.Marshal(respConfigModels)
	s.NotContains(string(bConfigModels), rawKey)
}

func (s *ProviderSuite) TestTestEndpointOK() {
	ws := s.ensureWorkspace("Provider Test OK WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "Test OK Provider",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "gpt-4o-mini",
		APIKey:  "valid-key",
	})
	s.Created()
	p := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d/test", ws.ID, p.ID), fiber.MethodPost, nil)
	s.OK()
	testResult := testutil.DecodeOther[model.TestResult](s).Data
	s.True(testResult.OK)
	s.Empty(testResult.Code)
	s.GreaterOrEqual(testResult.LatencyMs, int64(0))
}

func (s *ProviderSuite) TestTestEndpointAuthFailure() {
	ws := s.ensureWorkspace("Provider Test Auth Fail WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "Test Auth Fail Provider",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "gpt-4o-mini",
		APIKey:  "bad-key",
	})
	s.Created()
	p := s.GetResponse().Data

	s.authFail = true
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d/test", ws.ID, p.ID), fiber.MethodPost, nil)
	s.OK()
	testResult := testutil.DecodeOther[model.TestResult](s).Data
	s.False(testResult.OK)
	s.Equal("AUTH_FAILED", testResult.Code)
}

func (s *ProviderSuite) TestTestConfigBeforeSave() {
	ws := s.ensureWorkspace("Provider Test Config WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/test", ws.ID), fiber.MethodPost, model.TestProviderRequest{
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "gpt-4o-mini",
		APIKey:  "valid-key",
	})
	s.OK()
	testResult := testutil.DecodeOther[model.TestResult](s).Data
	s.True(testResult.OK)
}

func (s *ProviderSuite) TestDeleteDefaultLeavesNoDefault() {
	ws := s.ensureWorkspace("Provider Del Default WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "Only Default Provider",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "m1",
		APIKey:  "k1",
	})
	s.Created()
	p := s.GetResponse().Data
	s.True(p.IsDefault)

	// Delete
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws.ID, p.ID), fiber.MethodDelete, nil)
	s.OK()

	// List should be empty
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodGet, nil)
	s.OK()
	listResp := testutil.DecodeOther[[]model.ProviderResponse](s)
	s.Empty(listResp.Data)
}

func (s *ProviderSuite) TestUpdateBaseURLWithoutKey422() {
	ws := s.ensureWorkspace("Provider Update BaseURL WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "BaseURL Provider",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "m1",
		APIKey:  "initial-key",
	})
	s.Created()
	p := s.GetResponse().Data

	newURL := "http://127.0.0.1:9999/v1"
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d", ws.ID, p.ID), fiber.MethodPatch, model.UpdateProviderRequest{
		BaseURL: &newURL,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *ProviderSuite) TestProvider_ListModels_ExistingAndConfig() {
	ws := s.ensureWorkspace("Provider ListModels WS", s.User.ID)

	// 1. POST /workspaces/:wid/agent/providers/models with valid payload -> 200 OK with []string
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/models", ws.ID), fiber.MethodPost, model.ListModelsRequest{
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		APIKey:  "valid-key",
	})
	s.OK()
	modelsResp := testutil.DecodeOther[[]string](s).Data
	s.Equal([]string{"gpt-4o", "gpt-4o-mini"}, modelsResp)

	// 2. GET /workspaces/:wid/agent/providers/:id/models -> 200 OK with []string
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "Saved Models Provider",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "gpt-4o-mini",
		APIKey:  "valid-key",
	})
	s.Created()
	created := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/%d/models", ws.ID, created.ID), fiber.MethodGet, nil)
	s.OK()
	savedModelsResp := testutil.DecodeOther[[]string](s).Data
	s.Equal([]string{"gpt-4o", "gpt-4o-mini"}, savedModelsResp)

	// 3. POST /workspaces/:wid/agent/providers/models with blocked URL -> 422 or 400
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/models", ws.ID), fiber.MethodPost, model.ListModelsRequest{
		Kind:    "openai_compat",
		BaseURL: "ftp://127.0.0.1:9999",
		APIKey:  "valid-key",
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)

	// Missing required fields -> 422 (validation error)
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers/models", ws.ID), fiber.MethodPost, model.ListModelsRequest{
		Kind:   "",
		APIKey: "",
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}
