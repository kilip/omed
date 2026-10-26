package test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/account"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type SettingApiSuite struct {
	testutil.ApiTestSuite[model.FXAccountsSettingResponse]
}

func (s *SettingApiSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		SetBaseCurrency("IDR").
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *SettingApiSuite) ensureAccount(wsID int64, code, name, accType, normalBalance string) int64 {
	client := testutil.GetEntCli()
	ctx := base.WithWorkspaceID(s.Context, wsID)
	acc, err := client.Account.Create().
		SetWorkspaceID(wsID).
		SetCode(code).
		SetName(name).
		SetType(account.Type(accType)).
		SetNormalBalance(account.NormalBalance(normalBalance)).
		SetIsActive(true).
		SetIsPostable(true).
		Save(ctx)
	s.Require().NoError(err)
	return acc.ID
}

func (s *SettingApiSuite) Test_FXAccounts_GetAndPut() {
	ws := s.ensureWorkspace("Settings Test WS", s.User.ID)

	// 1. Initially unconfigured returns 200 with nil fields
	s.Request(fmt.Sprintf("/workspaces/%d/fin/settings/fx-accounts", ws.ID), fiber.MethodGet, nil)
	s.OK()

	var getRes httpx.WebResponse[model.FXAccountsSettingResponse]
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&getRes))
	s.Nil(getRes.Data.FXGainAccountID)
	s.Nil(getRes.Data.FXLossAccountID)
	s.Nil(getRes.Data.FXRoundingAccountID)

	// 2. Create accounts and update settings
	gainID := s.ensureAccount(ws.ID, "7210", "FX Gain", domain.AccountTypeRevenue, domain.NormalBalanceCredit)
	lossID := s.ensureAccount(ws.ID, "7310", "FX Loss", domain.AccountTypeExpense, domain.NormalBalanceDebit)
	roundID := s.ensureAccount(ws.ID, "7400", "FX Rounding", domain.AccountTypeExpense, domain.NormalBalanceDebit)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/settings/fx-accounts", ws.ID), fiber.MethodPut, model.UpdateFXAccountsRequest{
		FXGainAccountID:     &gainID,
		FXLossAccountID:     &lossID,
		FXRoundingAccountID: &roundID,
	})
	s.OK()

	var putRes httpx.WebResponse[model.FXAccountsSettingResponse]
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&putRes))
	s.Equal(&gainID, putRes.Data.FXGainAccountID)
	s.Equal(&lossID, putRes.Data.FXLossAccountID)
	s.Equal(&roundID, putRes.Data.FXRoundingAccountID)

	// 3. GET returns the updated settings
	s.Request(fmt.Sprintf("/workspaces/%d/fin/settings/fx-accounts", ws.ID), fiber.MethodGet, nil)
	s.OK()

	var getRes2 httpx.WebResponse[model.FXAccountsSettingResponse]
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&getRes2))
	s.Equal(&gainID, getRes2.Data.FXGainAccountID)
	s.Equal(&lossID, getRes2.Data.FXLossAccountID)
	s.Equal(&roundID, getRes2.Data.FXRoundingAccountID)
}

func TestSettingApiSuite(t *testing.T) {
	suite.Run(t, new(SettingApiSuite))
}
