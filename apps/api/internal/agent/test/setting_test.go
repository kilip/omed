package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type SettingSuite struct {
	testutil.ApiTestSuite[model.SettingResponse]
	fakeServer *httptest.Server
}

func TestSettingsSuite(t *testing.T) {
	suite.Run(t, new(SettingSuite))
}

func (s *SettingSuite) SetupSuite() {
	s.fakeServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	}))

	s.ApiTestSuite.SetupSuite()
}

func (s *SettingSuite) TearDownSuite() {
	if s.fakeServer != nil {
		s.fakeServer.Close()
	}
}

func (s *SettingSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *SettingSuite) TestSettings_DefaultsDisabled() {
	ws := s.ensureWorkspace("Settings Defaults WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/settings", ws.ID), fiber.MethodGet, nil)
	s.OK()

	res := s.GetResponse().Data
	s.False(res.FinancialEnabled)
	s.Nil(res.FinancialProviderID)
}

func (s *SettingSuite) TestSettings_EnableWithProvider() {
	ws := s.ensureWorkspace("Settings Enable Provider WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", ws.ID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "OpenAI Vision Provider",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "gpt-4o",
		APIKey:  "sk-test-secret-key-setting",
	})
	s.Created()
	prov := testutil.DecodeOther[model.ProviderResponse](s).Data
	s.Positive(prov.ID)

	enabled := true
	s.Request(fmt.Sprintf("/workspaces/%d/agent/settings", ws.ID), fiber.MethodPatch, model.UpdateSettingRequest{
		FinancialEnabled:    &enabled,
		FinancialProviderID: &prov.ID,
	})
	s.OK()
	patchRes := s.GetResponse().Data
	s.True(patchRes.FinancialEnabled)
	s.Require().NotNil(patchRes.FinancialProviderID)
	s.Equal(prov.ID, *patchRes.FinancialProviderID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/settings", ws.ID), fiber.MethodGet, nil)
	s.OK()
	getRes := s.GetResponse().Data
	s.True(getRes.FinancialEnabled)
	s.Require().NotNil(getRes.FinancialProviderID)
	s.Equal(prov.ID, *getRes.FinancialProviderID)
}

func (s *SettingSuite) TestSettings_EnableWithoutProvider422() {
	ws := s.ensureWorkspace("Settings Enable No Provider WS", s.User.ID)

	enabled := true
	s.Request(fmt.Sprintf("/workspaces/%d/agent/settings", ws.ID), fiber.MethodPatch, model.UpdateSettingRequest{
		FinancialEnabled: &enabled,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
	errResp := s.GetErrorResponse()
	s.Equal("FINANCIAL_AGENT_NO_PROVIDER", errResp.Error.Code)
}

func (s *SettingSuite) TestSettings_ViewerCannotPatch() {
	ws := s.ensureWorkspace("Settings Viewer WS", s.User.ID)

	viewer := s.AsUser("viewer-setting@example.com", "Viewer User")
	s.AddMember(ws.ID, viewer.ID, domain.RoleViewer)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/settings", ws.ID), fiber.MethodGet, nil)
	s.OK()

	enabled := false
	s.Request(fmt.Sprintf("/workspaces/%d/agent/settings", ws.ID), fiber.MethodPatch, model.UpdateSettingRequest{
		FinancialEnabled: &enabled,
	})
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *SettingSuite) TestSettings_FinancialUserCannotRead() {
	ws := s.ensureWorkspace("Settings Financial User WS", s.User.ID)

	finUser := s.AsUser("finuser-setting@example.com", "Financial User")
	s.AddMember(ws.ID, finUser.ID, domain.RoleFinancialUser)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/settings", ws.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *SettingSuite) TestSettings_WorkspaceIsolation() {
	wsA := s.ensureWorkspace("Settings WS A", s.User.ID)
	wsB := s.ensureWorkspace("Settings WS B", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", wsA.ID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "WS A Provider",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "gpt-4o",
		APIKey:  "sk-test-secret-key-ws-a",
	})
	s.Created()
	provA := testutil.DecodeOther[model.ProviderResponse](s).Data

	enabled := true
	s.Request(fmt.Sprintf("/workspaces/%d/agent/settings", wsA.ID), fiber.MethodPatch, model.UpdateSettingRequest{
		FinancialEnabled:    &enabled,
		FinancialProviderID: &provA.ID,
	})
	s.OK()

	s.Request(fmt.Sprintf("/workspaces/%d/agent/settings", wsB.ID), fiber.MethodGet, nil)
	s.OK()
	resB := s.GetResponse().Data
	s.False(resB.FinancialEnabled)
	s.Nil(resB.FinancialProviderID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/settings", wsB.ID), fiber.MethodPatch, model.UpdateSettingRequest{
		FinancialEnabled:    &enabled,
		FinancialProviderID: &provA.ID,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
	errResp := s.GetErrorResponse()
	s.Equal("FINANCIAL_AGENT_NO_PROVIDER", errResp.Error.Code)
}
