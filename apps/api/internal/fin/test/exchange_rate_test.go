package test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/internal/platform/money"
	"github.com/kilip/omed/api/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type ExchangeRateApiSuite struct {
	testutil.ApiTestSuite[model.ExchangeRateResponse]
}

func (s *ExchangeRateApiSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		SetBaseCurrency("IDR").
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *ExchangeRateApiSuite) Test_ExchangeRate_Lifecycle() {
	ws := s.ensureWorkspace("ExchangeRate WS", s.User.ID)

	// 1. Create USD -> IDR
	s.Request(fmt.Sprintf("/workspaces/%d/fin/exchange-rates", ws.ID), fiber.MethodPost, model.CreateExchangeRateRequest{
		FromCurrency:  "USD",
		ToCurrency:    "IDR",
		Rate:          dec(16200),
		EffectiveDate: parseApiDate("2026-01-01"),
	})
	s.Created()
	created := s.GetResponse().Data
	s.Equal("USD", created.FromCurrency)
	s.Equal("IDR", created.ToCurrency)
	s.True(created.Rate.Equal(dec(16200)))

	// 2. Get by ID
	s.Request(fmt.Sprintf("/workspaces/%d/fin/exchange-rates/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.OK()
	fetched := s.GetResponse().Data
	s.Equal(created.ID, fetched.ID)

	// 3. Update rate
	s.Request(fmt.Sprintf("/workspaces/%d/fin/exchange-rates/%d", ws.ID, created.ID), fiber.MethodPut, model.UpdateExchangeRateRequest{
		Rate:          dec(16300),
		EffectiveDate: parseApiDate("2026-01-01"),
	})
	s.OK()
	updated := s.GetResponse().Data
	s.True(updated.Rate.Equal(dec(16300)))

	// 4. Lookup direct rate
	s.Request(fmt.Sprintf("/workspaces/%d/fin/exchange-rates/lookup?from=USD&to=IDR&date=2026-01-10", ws.ID), fiber.MethodGet, nil)
	s.OK()
	var lookupResp httpx.WebResponse[model.ExchangeRateLookupResponse]
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&lookupResp))
	s.True(lookupResp.Data.Rate.Equal(dec(16300)))

	// 5. Lookup inverse rate (IDR -> USD)
	s.Request(fmt.Sprintf("/workspaces/%d/fin/exchange-rates/lookup?from=IDR&to=USD&date=2026-01-10", ws.ID), fiber.MethodGet, nil)
	s.OK()
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&lookupResp))
	expectedInv := money.Round(decimal.NewFromInt(1).Div(dec(16300)), 8)
	s.True(lookupResp.Data.Rate.Equal(expectedInv))

	// 6. Search
	usd := "USD"
	s.Request(fmt.Sprintf("/workspaces/%d/fin/exchange-rates/search", ws.ID), fiber.MethodPost, model.SearchExchangeRateRequest{
		FromCurrency: &usd,
	})
	s.OK()
	var searchResp httpx.WebResponse[[]model.ExchangeRateResponse]
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&searchResp))
	s.Len(searchResp.Data, 1)

	// 7. Delete
	s.Request(fmt.Sprintf("/workspaces/%d/fin/exchange-rates/%d", ws.ID, created.ID), fiber.MethodDelete, nil)
	s.OK()

	// 8. Get after delete -> 404
	s.Request(fmt.Sprintf("/workspaces/%d/fin/exchange-rates/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *ExchangeRateApiSuite) Test_RBAC() {
	owner := s.User
	ws := s.ensureWorkspace("ExchangeRate RBAC WS", owner.ID)

	viewer := s.AsUser("viewer@example.com", "Viewer User")
	s.AddMember(ws.ID, viewer.ID, domain.RoleViewer)

	// Viewer cannot create -> 403 Forbidden
	s.Request(fmt.Sprintf("/workspaces/%d/fin/exchange-rates", ws.ID), fiber.MethodPost, model.CreateExchangeRateRequest{
		FromCurrency:  "USD",
		ToCurrency:    "IDR",
		Rate:          dec(16200),
		EffectiveDate: parseApiDate("2026-01-01"),
	})
	s.AssertStatus(fiber.StatusForbidden)

	// Accountant can create -> 201 Created
	accountant := s.AsUser("accountant@example.com", "Accountant User")
	s.AddMember(ws.ID, accountant.ID, domain.RoleAccountant)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/exchange-rates", ws.ID), fiber.MethodPost, model.CreateExchangeRateRequest{
		FromCurrency:  "USD",
		ToCurrency:    "IDR",
		Rate:          dec(16200),
		EffectiveDate: parseApiDate("2026-01-01"),
	})
	s.Created()
	rateID := s.GetResponse().Data.ID

	// Viewer can read and lookup -> 200 OK
	s.AsUser("viewer@example.com", "Viewer User")
	s.Request(fmt.Sprintf("/workspaces/%d/fin/exchange-rates/%d", ws.ID, rateID), fiber.MethodGet, nil)
	s.OK()

	s.Request(fmt.Sprintf("/workspaces/%d/fin/exchange-rates/lookup?from=USD&to=IDR", ws.ID), fiber.MethodGet, nil)
	s.OK()

	// Accountant can delete -> 200 OK
	s.AsUser("accountant@example.com", "Accountant User")
	s.Request(fmt.Sprintf("/workspaces/%d/fin/exchange-rates/%d", ws.ID, rateID), fiber.MethodDelete, nil)
	s.OK()
}

func TestExchangeRateApiSuite(t *testing.T) {
	suite.Run(t, new(ExchangeRateApiSuite))
}
