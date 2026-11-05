package test

import (
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/cms/model"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type SettingSuite struct {
	testutil.ApiTestSuite[model.CmsSettingResponse]
}

func TestSettingSuite(t *testing.T) {
	suite.Run(t, new(SettingSuite))
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

func (s *SettingSuite) Test_Get_And_Update_Settings() {
	ws := s.ensureWorkspace("Setting WS", s.User.ID)

	// 1. Get default settings
	s.Request(fmt.Sprintf("/workspaces/%d/cms/settings", ws.ID), fiber.MethodGet, nil)
	s.OK()
	getResp := s.GetResponse()
	s.Equal("en", getResp.Data.DefaultLocale)
	s.Equal([]string{"en", "id"}, getResp.Data.SupportedLocales)

	// 2. Update settings: add Japanese
	newLocales := []string{"en", "id", "ja"}
	s.Request(fmt.Sprintf("/workspaces/%d/cms/settings", ws.ID), fiber.MethodPut, model.UpdateCmsSettingRequest{
		SupportedLocales: newLocales,
	})
	s.OK()
	updateResp := s.GetResponse()
	s.Equal([]string{"en", "id", "ja"}, updateResp.Data.SupportedLocales)

	// 3. Update default to ja
	ja := "ja"
	s.Request(fmt.Sprintf("/workspaces/%d/cms/settings", ws.ID), fiber.MethodPut, model.UpdateCmsSettingRequest{
		DefaultLocale: &ja,
	})
	s.OK()
	updateResp2 := s.GetResponse()
	s.Equal("ja", updateResp2.Data.DefaultLocale)

	// 4. Update with default not in supported -> 400
	fr := "fr"
	s.Request(fmt.Sprintf("/workspaces/%d/cms/settings", ws.ID), fiber.MethodPut, model.UpdateCmsSettingRequest{
		DefaultLocale: &fr,
	})
	s.AssertStatus(fiber.StatusBadRequest)
}
