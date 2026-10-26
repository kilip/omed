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
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type PeriodApiSuite struct {
	testutil.ApiTestSuite[model.PeriodResponse]
}

func (s *PeriodApiSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *PeriodApiSuite) getListResponse() httpx.WebResponse[[]model.PeriodResponse] {
	var res httpx.WebResponse[[]model.PeriodResponse]
	s.T().Helper()
	defer func() { _ = s.HttpResponse.Body.Close() }()
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&res))
	return res
}

func parseApiDate(str string) model.Date {
	t, _ := time.Parse("2006-01-02", str)
	return model.Date(t)
}

func (s *PeriodApiSuite) Test_Create_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods", ws.ID), fiber.MethodPost, model.CreatePeriodRequest{
		Name:        "January 2026",
		StartDate:   parseApiDate("2026-01-01"),
		EndDate:     parseApiDate("2026-01-31"),
		FiscalYear:  2026,
		Description: "First month of year",
	})
	s.Created()

	res := s.GetResponse()
	s.Equal("January 2026", res.Data.Name)
	s.Equal(domain.PeriodStatusOpen, res.Data.Status)
	s.Equal(2026, res.Data.FiscalYear)
	s.Equal("First month of year", res.Data.Description)
	s.Equal(parseApiDate("2026-01-01"), res.Data.StartDate)
	s.Equal(parseApiDate("2026-01-31"), res.Data.EndDate)
	s.Nil(res.Data.AuditMeta.DeletedAt)
}

func (s *PeriodApiSuite) Test_GetByID() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods", ws.ID), fiber.MethodPost, model.CreatePeriodRequest{
		Name:       "January 2026",
		StartDate:  parseApiDate("2026-01-01"),
		EndDate:    parseApiDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Created()
	created := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.OK()

	res := s.GetResponse()
	s.Equal(created.ID, res.Data.ID)
	s.Equal("January 2026", res.Data.Name)
}

func (s *PeriodApiSuite) Test_Search() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods", ws.ID), fiber.MethodPost, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseApiDate("2026-01-01"),
		EndDate:    parseApiDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Created()

	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods", ws.ID), fiber.MethodPost, model.CreatePeriodRequest{
		Name:       "Feb 2026",
		StartDate:  parseApiDate("2026-02-01"),
		EndDate:    parseApiDate("2026-02-28"),
		FiscalYear: 2026,
	})
	s.Created()

	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods/search", ws.ID), fiber.MethodPost, model.SearchPeriodRequest{})
	s.OK()

	res := s.getListResponse()
	s.Len(res.Data, 2)
}

func (s *PeriodApiSuite) Test_Update() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods", ws.ID), fiber.MethodPost, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseApiDate("2026-01-01"),
		EndDate:    parseApiDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Created()
	created := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods/%d", ws.ID, created.ID), fiber.MethodPut, model.UpdatePeriodRequest{
		Name:        "January 2026 Updated",
		StartDate:   parseApiDate("2026-01-01"),
		EndDate:     parseApiDate("2026-01-31"),
		FiscalYear:  2026,
		Description: "Updated memo",
	})
	s.OK()

	res := s.GetResponse()
	s.Equal("January 2026 Updated", res.Data.Name)
	s.Equal("Updated memo", res.Data.Description)
}

func (s *PeriodApiSuite) Test_Close_And_Reopen() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods", ws.ID), fiber.MethodPost, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseApiDate("2026-01-01"),
		EndDate:    parseApiDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Created()
	created := s.GetResponse().Data

	// Close
	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods/%d/close", ws.ID, created.ID), fiber.MethodPost, nil)
	s.OK()
	res := s.GetResponse()
	s.Equal(domain.PeriodStatusClosed, res.Data.Status)

	// Updating closed period must fail with 409
	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods/%d", ws.ID, created.ID), fiber.MethodPut, model.UpdatePeriodRequest{
		Name:       "Attempt update closed",
		StartDate:  parseApiDate("2026-01-01"),
		EndDate:    parseApiDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.AssertStatus(fiber.StatusConflict)

	// Reopen
	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods/%d/reopen", ws.ID, created.ID), fiber.MethodPost, nil)
	s.OK()
	res = s.GetResponse()
	s.Equal(domain.PeriodStatusOpen, res.Data.Status)
}

func (s *PeriodApiSuite) Test_Delete() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods", ws.ID), fiber.MethodPost, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseApiDate("2026-01-01"),
		EndDate:    parseApiDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Created()
	created := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods/%d", ws.ID, created.ID), fiber.MethodDelete, nil)
	s.OK()

	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *PeriodApiSuite) Test_Overlap_Conflict() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods", ws.ID), fiber.MethodPost, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseApiDate("2026-01-01"),
		EndDate:    parseApiDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Created()

	// Overlapping date
	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods", ws.ID), fiber.MethodPost, model.CreatePeriodRequest{
		Name:       "Jan 2026 Overlap",
		StartDate:  parseApiDate("2026-01-15"),
		EndDate:    parseApiDate("2026-02-15"),
		FiscalYear: 2026,
	})
	s.AssertStatus(fiber.StatusConflict)
}

func (s *PeriodApiSuite) Test_RBAC_Viewer_Accountant_Admin() {
	owner := s.User
	ws := s.ensureWorkspace("Finance RBAC WS", owner.ID)

	// 1. Viewer member
	viewer := s.AsUser("viewer@example.com", "Viewer User")
	s.AddMember(ws.ID, viewer.ID, domain.RoleViewer)

	// Viewer cannot create -> 403 Forbidden
	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods", ws.ID), fiber.MethodPost, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseApiDate("2026-01-01"),
		EndDate:    parseApiDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.AssertStatus(fiber.StatusForbidden)

	// 2. Accountant member
	accountant := s.AsUser("accountant@example.com", "Accountant User")
	s.AddMember(ws.ID, accountant.ID, domain.RoleAccountant)

	// Accountant can create -> 201 Created
	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods", ws.ID), fiber.MethodPost, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseApiDate("2026-01-01"),
		EndDate:    parseApiDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Created()
	createdPeriod := s.GetResponse().Data

	// Viewer can read -> 200 OK
	s.AsUser("viewer@example.com", "Viewer User")
	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods/%d", ws.ID, createdPeriod.ID), fiber.MethodGet, nil)
	s.OK()

	// Accountant can close -> 200 OK
	s.AsUser("accountant@example.com", "Accountant User")
	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods/%d/close", ws.ID, createdPeriod.ID), fiber.MethodPost, nil)
	s.OK()

	// Accountant cannot delete -> 403 Forbidden
	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods/%d", ws.ID, createdPeriod.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusForbidden)

	// 3. Admin member
	admin := s.AsUser("admin@example.com", "Admin User")
	s.AddMember(ws.ID, admin.ID, domain.RoleAdmin)

	// Admin reopens first (since closed period cannot be deleted)
	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods/%d/reopen", ws.ID, createdPeriod.ID), fiber.MethodPost, nil)
	s.OK()

	// Admin can delete -> 200 OK
	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods/%d", ws.ID, createdPeriod.ID), fiber.MethodDelete, nil)
	s.OK()
}

func (s *PeriodApiSuite) Test_Revalue_Endpoints() {
	ws := s.ensureWorkspace("Finance Reval WS", s.User.ID)
	client := testutil.GetEntCli()

	// Seed COA template via API
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/seed", ws.ID), fiber.MethodPost, model.SeedAccountsRequest{
		Template: "freelancer",
		Lang:     "en",
	})
	s.Created()

	// Find parent 1110 and 3010
	ctx := base.WithWorkspaceID(s.Context, ws.ID)
	accountRepo := repository.NewAccountRepository(client)
	bankParent, err := accountRepo.FindByCode(ctx, "1110")
	s.Require().NoError(err)

	capitalAcc, err := accountRepo.FindByCode(ctx, "3100")
	s.Require().NoError(err)

	// Create USD Bank account via API
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts", ws.ID), fiber.MethodPost, model.CreateAccountRequest{
		Code:     "1115",
		Name:     "Wise USD",
		Type:     "ASSET",
		ParentID: bankParent.ID,
		Currency: "USD",
	})
	s.Created()

	usdAcc, err := accountRepo.FindByCode(ctx, "1115")
	s.Require().NoError(err)

	// Create open period
	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods", ws.ID), fiber.MethodPost, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseApiDate("2026-01-01"),
		EndDate:    parseApiDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Created()
	period := s.GetResponse().Data

	// Exchange rates
	rateRepo := repository.NewExchangeRateRepository(client)
	_, err = rateRepo.Create(ctx, model.CreateExchangeRateRequest{
		FromCurrency:  "USD",
		ToCurrency:    "IDR",
		Rate:          dec(15000),
		EffectiveDate: parseApiDate("2026-01-01"),
	})
	s.Require().NoError(err)

	_, err = rateRepo.Create(ctx, model.CreateExchangeRateRequest{
		FromCurrency:  "USD",
		ToCurrency:    "IDR",
		Rate:          dec(16000),
		EffectiveDate: parseApiDate("2026-01-31"),
	})
	s.Require().NoError(err)

	// Initial posted entry: 100 USD @ 15,000 IDR
	trx, err := client.Transaction.Create().
		SetWorkspaceID(ws.ID).
		SetNumber("TRX-2026-000088").
		SetTransactionDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetCurrency("USD").
		SetExchangeRate(dec(15000)).
		SetSource("MANUAL").
		Save(ctx)
	s.Require().NoError(err)

	e, err := client.Entry.Create().
		SetWorkspaceID(ws.ID).
		SetTransactionID(trx.ID).
		SetPeriodID(period.ID).
		SetNumber("JE-2026-000088").
		SetEntryDate(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)).
		SetStatus("POSTED").
		SetPostedAt(time.Now()).
		SetPostedBy(s.User.ID).
		Save(ctx)
	s.Require().NoError(err)

	_, err = client.Posting.Create().
		SetWorkspaceID(ws.ID).
		SetEntryID(e.ID).
		SetAccountID(usdAcc.ID).
		SetDirection("DEBIT").
		SetAmount(dec(100)).
		SetBaseAmount(dec(1500000)).
		SetPosition(0).
		Save(ctx)
	s.Require().NoError(err)

	_, err = client.Posting.Create().
		SetWorkspaceID(ws.ID).
		SetEntryID(e.ID).
		SetAccountID(capitalAcc.ID).
		SetDirection("CREDIT").
		SetAmount(dec(100)).
		SetBaseAmount(dec(1500000)).
		SetPosition(1).
		Save(ctx)
	s.Require().NoError(err)

	// Trigger revaluation via API
	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods/%d/revalue", ws.ID, period.ID), fiber.MethodPost, nil)
	s.OK()

	var revalRes httpx.WebResponse[model.EntryResponse]
	err = json.NewDecoder(s.HttpResponse.Body).Decode(&revalRes)
	s.Require().NoError(err)
	s.Equal(domain.EntryStatusPosted, revalRes.Data.Status)
	s.Len(revalRes.Data.Postings, 2)

	// Second revaluation should return 409 Conflict
	s.Request(fmt.Sprintf("/workspaces/%d/fin/periods/%d/revalue", ws.ID, period.ID), fiber.MethodPost, nil)
	s.AssertStatus(fiber.StatusConflict)
}

func TestPeriodApiSuite(t *testing.T) {
	suite.Run(t, new(PeriodApiSuite))
}

