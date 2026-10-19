package test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type ReportApiSuite struct {
	testutil.ApiTestSuite[model.TrialBalanceResponse]
}

func (s *ReportApiSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *ReportApiSuite) Test_TrialBalance_Endpoint() {
	ws := s.ensureWorkspace("Finance TB WS", s.User.ID)
	client := testutil.GetEntCli()

	// Seed COA template via API
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/seed", ws.ID), fiber.MethodPost, model.SeedAccountsRequest{
		Template: "freelancer",
		Lang:     "en",
	})
	s.Created()

	ctx := base.WithWorkspaceID(s.Context, ws.ID)
	accountRepo := repository.NewAccountRepository(client)
	bankAcc, err := accountRepo.FindByCode(ctx, "1111")
	s.Require().NoError(err)

	capAcc, err := accountRepo.FindByCode(ctx, "3100")
	s.Require().NoError(err)

	// Create period
	periodRepo := repository.NewPeriodRepository(client)
	p, err := periodRepo.Create(ctx, model.CreatePeriodRequest{
		Name:       "Jan 2026",
		StartDate:  parseApiDate("2026-01-01"),
		EndDate:    parseApiDate("2026-01-31"),
		FiscalYear: 2026,
	})
	s.Require().NoError(err)

	// Posted entry
	trx, err := client.Transaction.Create().
		SetWorkspaceID(ws.ID).
		SetNumber("TRX-2026-000099").
		SetTransactionDate(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)).
		SetCurrency("IDR").
		SetExchangeRate(1.0).
		SetSource("MANUAL").
		Save(ctx)
	s.Require().NoError(err)

	e, err := client.Entry.Create().
		SetWorkspaceID(ws.ID).
		SetTransactionID(trx.ID).
		SetPeriodID(p.ID).
		SetNumber("JE-2026-000099").
		SetEntryDate(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)).
		SetStatus("POSTED").
		SetPostedAt(time.Now()).
		SetPostedBy(s.User.ID).
		Save(ctx)
	s.Require().NoError(err)

	amt := 5000000.0
	_, err = client.Posting.Create().
		SetWorkspaceID(ws.ID).
		SetEntryID(e.ID).
		SetAccountID(bankAcc.ID).
		SetDirection("DEBIT").
		SetAmount(amt).
		SetBaseAmount(amt).
		SetPosition(0).
		Save(ctx)
	s.Require().NoError(err)

	_, err = client.Posting.Create().
		SetWorkspaceID(ws.ID).
		SetEntryID(e.ID).
		SetAccountID(capAcc.ID).
		SetDirection("CREDIT").
		SetAmount(amt).
		SetBaseAmount(amt).
		SetPosition(1).
		Save(ctx)
	s.Require().NoError(err)

	// Call POST /fin/reports/trial-balance
	includeCurrencies := true
	s.Request(fmt.Sprintf("/workspaces/%d/fin/reports/trial-balance", ws.ID), fiber.MethodPost, model.TrialBalanceRequest{
		AsOfDate:                   parseApiDate("2026-01-31"),
		IncludePerCurrencyBreakdown: &includeCurrencies,
	})
	s.OK()

	var tbRes httpx.WebResponse[model.TrialBalanceResponse]
	err = json.NewDecoder(s.HttpResponse.Body).Decode(&tbRes)
	s.Require().NoError(err)
	s.True(tbRes.Data.IsBalanced)
	s.Equal(5000000.0, tbRes.Data.TotalDebit)
	s.Equal(5000000.0, tbRes.Data.TotalCredit)
	s.Equal("IDR", tbRes.Data.BaseCurrency)
}

func TestReportApiSuite(t *testing.T) {
	suite.Run(t, new(ReportApiSuite))
}
