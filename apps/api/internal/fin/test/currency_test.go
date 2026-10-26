package test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type CurrencyApiSuite struct {
	testutil.ApiTestSuite[model.WorkspaceBaseCurrencyResponse]
}

func (s *CurrencyApiSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		SetBaseCurrency("IDR").
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *CurrencyApiSuite) Test_ListCurrencies() {
	ws := s.ensureWorkspace("Currency Test WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/currencies", ws.ID), fiber.MethodGet, nil)
	s.OK()

	var res httpx.WebResponse[[]model.CurrencyResponse]
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&res))
	s.Assert().NotEmpty(res.Data)

	foundUSD := false
	for _, c := range res.Data {
		if c.Code == "USD" {
			foundUSD = true
			s.Assert().Equal(2, c.Decimals)
			s.Assert().Equal("$", c.Symbol)
		}
	}
	s.Assert().True(foundUSD, "expected USD in list of currencies")
}

func (s *CurrencyApiSuite) Test_UpdateBaseCurrency_SuccessAndLock() {
	ws := s.ensureWorkspace("Base Currency WS", s.User.ID)
	client := testutil.GetEntCli()

	// Update base currency to USD
	s.Request(fmt.Sprintf("/workspaces/%d/fin/settings/base-currency", ws.ID), fiber.MethodPut, model.UpdateBaseCurrencyRequest{
		BaseCurrency: "USD",
	})
	s.OK()
	res := s.GetResponse()
	s.Equal("USD", res.Data.BaseCurrency)

	// Updating to unknown currency fails with 404
	s.Request(fmt.Sprintf("/workspaces/%d/fin/settings/base-currency", ws.ID), fiber.MethodPut, model.UpdateBaseCurrencyRequest{
		BaseCurrency: "XYZ",
	})
	s.AssertStatus(fiber.StatusNotFound)

	// Create period, accounts, and a POSTED entry
	wsCtx := base.WithWorkspaceID(s.Context, ws.ID)
	period, err := client.Period.Create().
		SetWorkspaceID(ws.ID).
		SetName("FY2026").
		SetStartDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetEndDate(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)).
		SetFiscalYear(2026).
		Save(wsCtx)
	s.Require().NoError(err)

	_, err = client.Account.Create().
		SetWorkspaceID(ws.ID).
		SetCode("1001").
		SetName("Cash").
		SetType("ASSET").
		SetNormalBalance("DEBIT").
		SetIsPostable(true).
		SetIsActive(true).
		Save(wsCtx)
	s.Require().NoError(err)

	_, err = client.Account.Create().
		SetWorkspaceID(ws.ID).
		SetCode("4001").
		SetName("Sales").
		SetType("REVENUE").
		SetNormalBalance("CREDIT").
		SetIsPostable(true).
		SetIsActive(true).
		Save(wsCtx)
	s.Require().NoError(err)

	tx, err := client.Transaction.Create().
		SetWorkspaceID(ws.ID).
		SetNumber("TRX-2026-999901").
		SetTransactionDate(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)).
		SetCurrency("USD").
		SetExchangeRate(1.0).
		Save(wsCtx)
	s.Require().NoError(err)

	now := time.Now()
	_, err = client.Entry.Create().
		SetWorkspaceID(ws.ID).
		SetTransactionID(tx.ID).
		SetPeriodID(period.ID).
		SetNumber("JE-2026-999901").
		SetEntryDate(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)).
		SetStatus("POSTED").
		SetPostedAt(now).
		SetPostedBy(s.User.ID).
		Save(wsCtx)
	s.Require().NoError(err)

	// Base currency update should now be rejected with 409 Conflict
	s.Request(fmt.Sprintf("/workspaces/%d/fin/settings/base-currency", ws.ID), fiber.MethodPut, model.UpdateBaseCurrencyRequest{
		BaseCurrency: "EUR",
	})
	s.AssertStatus(fiber.StatusConflict)
}

func (s *CurrencyApiSuite) Test_RBAC() {
	owner := s.User
	ws := s.ensureWorkspace("Currency RBAC WS", owner.ID)

	viewer := s.AsUser("viewer@example.com", "Viewer User")
	s.AddMember(ws.ID, viewer.ID, domain.RoleViewer)

	// Viewer can read currencies -> 200 OK
	s.Request(fmt.Sprintf("/workspaces/%d/fin/currencies", ws.ID), fiber.MethodGet, nil)
	s.OK()

	// Viewer cannot update base currency -> 403 Forbidden
	s.Request(fmt.Sprintf("/workspaces/%d/fin/settings/base-currency", ws.ID), fiber.MethodPut, model.UpdateBaseCurrencyRequest{
		BaseCurrency: "USD",
	})
	s.AssertStatus(fiber.StatusForbidden)

	// Accountant can update base currency -> 200 OK
	accountant := s.AsUser("accountant@example.com", "Accountant User")
	s.AddMember(ws.ID, accountant.ID, domain.RoleAccountant)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/settings/base-currency", ws.ID), fiber.MethodPut, model.UpdateBaseCurrencyRequest{
		BaseCurrency: "USD",
	})
	s.OK()
}

func TestCurrencyApiSuite(t *testing.T) {
	suite.Run(t, new(CurrencyApiSuite))
}
