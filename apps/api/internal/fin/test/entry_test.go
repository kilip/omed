package test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/account"
	"github.com/kilip/omed/api/ent/period"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

func dec(v int64) decimal.Decimal {
	return decimal.NewFromInt(v)
}

type EntryApiSuite struct {
	testutil.ApiTestSuite[model.EntryResponse]
}

func (s *EntryApiSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *EntryApiSuite) ensureAccount(wsID int64, code, name, accType, normalBalance string) int64 {
	client := testutil.GetEntCli()
	acc, err := client.Account.Create().
		SetWorkspaceID(wsID).
		SetCode(code).
		SetName(name).
		SetType(account.Type(accType)).
		SetNormalBalance(account.NormalBalance(normalBalance)).
		SetIsActive(true).
		SetIsPostable(true).
		Save(base.WithWorkspaceID(s.Context, wsID))
	s.Require().NoError(err)
	return acc.ID
}

func (s *EntryApiSuite) ensurePeriod(wsID int64, name string, start, end time.Time, year int) int64 {
	client := testutil.GetEntCli()
	p, err := client.Period.Create().
		SetWorkspaceID(wsID).
		SetName(name).
		SetStartDate(start).
		SetEndDate(end).
		SetFiscalYear(year).
		SetStatus(period.Status(domain.PeriodStatusOpen)).
		Save(base.WithWorkspaceID(s.Context, wsID))
	s.Require().NoError(err)
	return p.ID
}

func (s *EntryApiSuite) ensureTransaction(wsID int64, desc, ref string, tDate time.Time) int64 {
	client := testutil.GetEntCli()
	tx, err := client.Transaction.Create().
		SetWorkspaceID(wsID).
		SetNumber(fmt.Sprintf("TRX-%d-%d", tDate.Year(), time.Now().UnixNano()%100000)).
		SetTransactionDate(tDate).
		SetDescription(desc).
		SetReference(ref).
		SetCurrency("IDR").
		SetExchangeRate(dec(1)).
		SetSource(domain.TransactionSourceManual).
		Save(base.WithWorkspaceID(s.Context, wsID))
	s.Require().NoError(err)
	return tx.ID
}

func (s *EntryApiSuite) getListResponse() httpx.WebResponse[[]model.EntryResponse] {
	var res httpx.WebResponse[[]model.EntryResponse]
	s.T().Helper()
	defer s.HttpResponse.Body.Close()
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&res))
	return res
}

func (s *EntryApiSuite) Test_Create_DraftEntry_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	acc1 := s.ensureAccount(ws.ID, "1000", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit)
	acc2 := s.ensureAccount(ws.ID, "4000", "Revenue", domain.AccountTypeRevenue, domain.NormalBalanceCredit)
	pID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)
	tDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	txID := s.ensureTransaction(ws.ID, "Service revenue", "INV-001", tDate)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries", ws.ID), fiber.MethodPost, model.CreateEntryRequest{
		TransactionID: txID,
		PeriodID:      pID,
		EntryDate:     model.Date(tDate),
		Description:   "Consulting fees",
		Postings: []model.CreatePostingRequest{
			{AccountID: acc1, Direction: domain.PostingDirectionDebit, Amount: dec(1500000)},
			{AccountID: acc2, Direction: domain.PostingDirectionCredit, Amount: dec(1500000)},
		},
	})
	s.Created()

	res := s.GetResponse()
	s.Equal("Consulting fees", res.Data.Description)
	s.Equal(domain.EntryStatusDraft, res.Data.Status)
	s.Require().Len(res.Data.Postings, 2)
}

func (s *EntryApiSuite) Test_GetByID_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	acc1 := s.ensureAccount(ws.ID, "1000", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit)
	acc2 := s.ensureAccount(ws.ID, "4000", "Revenue", domain.AccountTypeRevenue, domain.NormalBalanceCredit)
	pID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)
	tDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	txID := s.ensureTransaction(ws.ID, "Service revenue", "INV-001", tDate)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries", ws.ID), fiber.MethodPost, model.CreateEntryRequest{
		TransactionID: txID,
		PeriodID:      pID,
		EntryDate:     model.Date(tDate),
		Description:   "Consulting fees",
		Postings: []model.CreatePostingRequest{
			{AccountID: acc1, Direction: domain.PostingDirectionDebit, Amount: dec(2000000)},
			{AccountID: acc2, Direction: domain.PostingDirectionCredit, Amount: dec(2000000)},
		},
	})
	s.Created()
	created := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.OK()

	res := s.GetResponse()
	s.Equal(created.ID, res.Data.ID)
	s.Equal("Consulting fees", res.Data.Description)
	s.Require().Len(res.Data.Postings, 2)
}

func (s *EntryApiSuite) Test_Update_DraftEntry_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	acc1 := s.ensureAccount(ws.ID, "1000", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit)
	acc2 := s.ensureAccount(ws.ID, "4000", "Revenue", domain.AccountTypeRevenue, domain.NormalBalanceCredit)
	pID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)
	tDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	txID := s.ensureTransaction(ws.ID, "Service revenue", "INV-001", tDate)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries", ws.ID), fiber.MethodPost, model.CreateEntryRequest{
		TransactionID: txID,
		PeriodID:      pID,
		EntryDate:     model.Date(tDate),
		Description:   "Before update",
		Postings: []model.CreatePostingRequest{
			{AccountID: acc1, Direction: domain.PostingDirectionDebit, Amount: dec(1000000)},
			{AccountID: acc2, Direction: domain.PostingDirectionCredit, Amount: dec(1000000)},
		},
	})
	s.Created()
	created := s.GetResponse().Data

	newDesc := "After update"
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d", ws.ID, created.ID), fiber.MethodPut, model.UpdateEntryRequest{
		Description: &newDesc,
		Postings: []model.CreatePostingRequest{
			{AccountID: acc1, Direction: domain.PostingDirectionDebit, Amount: dec(3000000)},
			{AccountID: acc2, Direction: domain.PostingDirectionCredit, Amount: dec(3000000)},
		},
	})
	s.OK()

	res := s.GetResponse()
	s.Equal("After update", res.Data.Description)
	s.True(res.Data.Postings[0].Amount.Equal(dec(3000000)))
}

func (s *EntryApiSuite) Test_Delete_DraftEntry_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	acc1 := s.ensureAccount(ws.ID, "1000", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit)
	acc2 := s.ensureAccount(ws.ID, "4000", "Revenue", domain.AccountTypeRevenue, domain.NormalBalanceCredit)
	pID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)
	tDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	txID := s.ensureTransaction(ws.ID, "Service revenue", "INV-001", tDate)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries", ws.ID), fiber.MethodPost, model.CreateEntryRequest{
		TransactionID: txID,
		PeriodID:      pID,
		EntryDate:     model.Date(tDate),
		Postings: []model.CreatePostingRequest{
			{AccountID: acc1, Direction: domain.PostingDirectionDebit, Amount: dec(1000000)},
			{AccountID: acc2, Direction: domain.PostingDirectionCredit, Amount: dec(1000000)},
		},
	})
	s.Created()
	created := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d", ws.ID, created.ID), fiber.MethodDelete, nil)
	s.OK()

	// GET after delete -> 404
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *EntryApiSuite) Test_Post_Entry_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	acc1 := s.ensureAccount(ws.ID, "1000", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit)
	acc2 := s.ensureAccount(ws.ID, "4000", "Revenue", domain.AccountTypeRevenue, domain.NormalBalanceCredit)
	pID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)
	tDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	txID := s.ensureTransaction(ws.ID, "Service revenue", "INV-001", tDate)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries", ws.ID), fiber.MethodPost, model.CreateEntryRequest{
		TransactionID: txID,
		PeriodID:      pID,
		EntryDate:     model.Date(tDate),
		Postings: []model.CreatePostingRequest{
			{AccountID: acc1, Direction: domain.PostingDirectionDebit, Amount: dec(500000)},
			{AccountID: acc2, Direction: domain.PostingDirectionCredit, Amount: dec(500000)},
		},
	})
	s.Created()
	entryID := s.GetResponse().Data.ID

	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d/post", ws.ID, entryID), fiber.MethodPost, nil)
	s.OK()

	res := s.GetResponse()
	s.Equal(domain.EntryStatusPosted, res.Data.Status)
	s.NotNil(res.Data.PostedAt)
	s.NotNil(res.Data.PostedBy)

	// Attempting to delete a POSTED entry -> 409 Conflict
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d", ws.ID, entryID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusConflict)
}

func (s *EntryApiSuite) Test_Reverse_Entry_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	acc1 := s.ensureAccount(ws.ID, "1000", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit)
	acc2 := s.ensureAccount(ws.ID, "4000", "Revenue", domain.AccountTypeRevenue, domain.NormalBalanceCredit)
	pID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)
	tDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	txID := s.ensureTransaction(ws.ID, "Service revenue", "INV-001", tDate)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries", ws.ID), fiber.MethodPost, model.CreateEntryRequest{
		TransactionID: txID,
		PeriodID:      pID,
		EntryDate:     model.Date(tDate),
		Postings: []model.CreatePostingRequest{
			{AccountID: acc1, Direction: domain.PostingDirectionDebit, Amount: dec(750000)},
			{AccountID: acc2, Direction: domain.PostingDirectionCredit, Amount: dec(750000)},
		},
	})
	s.Created()
	entryID := s.GetResponse().Data.ID

	// Post the entry first
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d/post", ws.ID, entryID), fiber.MethodPost, nil)
	s.OK()

	// Reverse
	revDesc := "Correction reversal"
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d/reverse", ws.ID, entryID), fiber.MethodPost, model.ReverseEntryRequest{
		Description: revDesc,
	})
	s.Created()

	rev := s.GetResponse()
	s.Equal(domain.EntryStatusPosted, rev.Data.Status)
	s.Equal(&entryID, rev.Data.ReversalOfID)
	s.Require().Len(rev.Data.Postings, 2)
	// Postings swapped: acc1 should now be CREDIT, acc2 should now be DEBIT
	s.Equal(acc1, rev.Data.Postings[0].AccountID)
	s.Equal(domain.PostingDirectionCredit, rev.Data.Postings[0].Direction)
	s.Equal(acc2, rev.Data.Postings[1].AccountID)
	s.Equal(domain.PostingDirectionDebit, rev.Data.Postings[1].Direction)
}

func (s *EntryApiSuite) Test_Void_Entry_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	acc1 := s.ensureAccount(ws.ID, "1000", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit)
	acc2 := s.ensureAccount(ws.ID, "4000", "Revenue", domain.AccountTypeRevenue, domain.NormalBalanceCredit)
	pID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)
	tDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	txID := s.ensureTransaction(ws.ID, "Service revenue", "INV-001", tDate)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries", ws.ID), fiber.MethodPost, model.CreateEntryRequest{
		TransactionID: txID,
		PeriodID:      pID,
		EntryDate:     model.Date(tDate),
		Postings: []model.CreatePostingRequest{
			{AccountID: acc1, Direction: domain.PostingDirectionDebit, Amount: dec(1200000)},
			{AccountID: acc2, Direction: domain.PostingDirectionCredit, Amount: dec(1200000)},
		},
	})
	s.Created()
	entryID := s.GetResponse().Data.ID

	// Post the entry
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d/post", ws.ID, entryID), fiber.MethodPost, nil)
	s.OK()

	// Void
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d/void", ws.ID, entryID), fiber.MethodPost, nil)
	s.OK()

	voidRes := s.GetResponse()
	s.Equal(domain.EntryStatusVoid, voidRes.Data.Status)
}

func (s *EntryApiSuite) Test_Search_Entries_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	acc1 := s.ensureAccount(ws.ID, "1000", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit)
	acc2 := s.ensureAccount(ws.ID, "4000", "Revenue", domain.AccountTypeRevenue, domain.NormalBalanceCredit)
	pID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)
	tDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	txID := s.ensureTransaction(ws.ID, "Service revenue", "INV-001", tDate)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries", ws.ID), fiber.MethodPost, model.CreateEntryRequest{
		TransactionID: txID,
		PeriodID:      pID,
		EntryDate:     model.Date(tDate),
		Postings: []model.CreatePostingRequest{
			{AccountID: acc1, Direction: domain.PostingDirectionDebit, Amount: dec(100000)},
			{AccountID: acc2, Direction: domain.PostingDirectionCredit, Amount: dec(100000)},
		},
	})
	s.Created()

	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/search", ws.ID), fiber.MethodPost, model.SearchEntryRequest{
		PeriodID: &pID,
	})
	s.OK()

	listRes := s.getListResponse()
	s.Len(listRes.Data, 1)
}

func (s *EntryApiSuite) Test_AccountLedger_Endpoint_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	accCash := s.ensureAccount(ws.ID, "1000", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit)
	accRev := s.ensureAccount(ws.ID, "4000", "Revenue", domain.AccountTypeRevenue, domain.NormalBalanceCredit)
	pID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)
	tDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	txID := s.ensureTransaction(ws.ID, "Ledger test tx", "REF-L", tDate)

	// Create and post 2 entries:
	// Entry 1: Cash DEBIT 500, Rev CREDIT 500 -> Cash balance: 500
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries", ws.ID), fiber.MethodPost, model.CreateEntryRequest{
		TransactionID: txID,
		PeriodID:      pID,
		EntryDate:     model.Date(tDate),
		Postings: []model.CreatePostingRequest{
			{AccountID: accCash, Direction: domain.PostingDirectionDebit, Amount: dec(500)},
			{AccountID: accRev, Direction: domain.PostingDirectionCredit, Amount: dec(500)},
		},
	})
	s.Created()
	e1ID := s.GetResponse().Data.ID
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d/post", ws.ID, e1ID), fiber.MethodPost, nil)
	s.OK()

	// Entry 2: Cash DEBIT 300, Rev CREDIT 300 -> Cash balance: 800
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries", ws.ID), fiber.MethodPost, model.CreateEntryRequest{
		TransactionID: txID,
		PeriodID:      pID,
		EntryDate:     model.Date(tDate),
		Postings: []model.CreatePostingRequest{
			{AccountID: accCash, Direction: domain.PostingDirectionDebit, Amount: dec(300)},
			{AccountID: accRev, Direction: domain.PostingDirectionCredit, Amount: dec(300)},
		},
	})
	s.Created()
	e2ID := s.GetResponse().Data.ID
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d/post", ws.ID, e2ID), fiber.MethodPost, nil)
	s.OK()

	// Query ledger for Cash account
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/%d/ledger?page=1&limit=10", ws.ID, accCash), fiber.MethodGet, nil)
	s.OK()

	var ledgerRes httpx.WebResponse[model.AccountLedgerResponse]
	err := json.NewDecoder(s.HttpResponse.Body).Decode(&ledgerRes)
	s.Require().NoError(err)
	s.Equal(int64(2), ledgerRes.Data.Total)
	s.True(ledgerRes.Data.Balance.Equal(dec(800)))
	s.True(ledgerRes.Data.BaseBalance.Equal(dec(800)))
	s.Require().Len(ledgerRes.Data.Postings, 2)
	s.True(ledgerRes.Data.Postings[0].RunningBalance.Equal(dec(500)))
	s.True(ledgerRes.Data.Postings[0].Balance.Equal(dec(500)))
	s.True(ledgerRes.Data.Postings[0].BaseBalance.Equal(dec(500)))
	s.True(ledgerRes.Data.Postings[1].RunningBalance.Equal(dec(800)))
	s.True(ledgerRes.Data.Postings[1].Balance.Equal(dec(800)))
	s.True(ledgerRes.Data.Postings[1].BaseBalance.Equal(dec(800)))
}

func (s *EntryApiSuite) Test_FXRounding_AutoBalancing_Endpoint() {
	ws := s.ensureWorkspace("Finance FX WS", s.User.ID)
	accCash := s.ensureAccount(ws.ID, "1000", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit)
	accRev1 := s.ensureAccount(ws.ID, "4000", "Revenue 1", domain.AccountTypeRevenue, domain.NormalBalanceCredit)
	accRev2 := s.ensureAccount(ws.ID, "4100", "Revenue 2", domain.AccountTypeRevenue, domain.NormalBalanceCredit)
	fxAcc := s.ensureAccount(ws.ID, "7400", "Foreign Exchange Rounding", domain.AccountTypeExpense, domain.NormalBalanceDebit)

	pID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)
	tDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)

	// Create USD transaction with rate 10000.5
	client := testutil.GetEntCli()
	tx, err := client.Transaction.Create().
		SetWorkspaceID(ws.ID).
		SetNumber(fmt.Sprintf("TRX-USD-%d", time.Now().UnixNano()%100000)).
		SetTransactionDate(tDate).
		SetDescription("USD with rounding diff").
		SetCurrency("USD").
		SetExchangeRate(decimal.RequireFromString("10000.5")).
		SetSource(domain.TransactionSourceManual).
		Save(base.WithWorkspaceID(s.Context, ws.ID))
	s.Require().NoError(err)

	// 1. Unconfigured FX accounts: difference should fail with 422 FX_ROUNDING_ACCOUNT_NOT_FOUND
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries", ws.ID), fiber.MethodPost, model.CreateEntryRequest{
		TransactionID: tx.ID,
		PeriodID:      pID,
		EntryDate:     model.Date(tDate),
		Description:   "USD entry with 1 IDR rounding diff",
		Postings: []model.CreatePostingRequest{
			{AccountID: accCash, Direction: domain.PostingDirectionDebit, Amount: dec(1)},
			{AccountID: accRev1, Direction: domain.PostingDirectionCredit, Amount: decimal.RequireFromString("0.5")},
			{AccountID: accRev2, Direction: domain.PostingDirectionCredit, Amount: decimal.RequireFromString("0.5")},
		},
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)

	// Configure FX Rounding account via settings API
	s.Request(fmt.Sprintf("/workspaces/%d/fin/settings/fx-accounts", ws.ID), fiber.MethodPut, model.UpdateFXAccountsRequest{
		FXRoundingAccountID: &fxAcc,
	})
	s.OK()

	// Verify GET settings API returns configured account
	s.Request(fmt.Sprintf("/workspaces/%d/fin/settings/fx-accounts", ws.ID), fiber.MethodGet, nil)
	s.OK()

	// 2. Difference <= 1.0 IDR should auto-balance by adding an FX Rounding line
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries", ws.ID), fiber.MethodPost, model.CreateEntryRequest{
		TransactionID: tx.ID,
		PeriodID:      pID,
		EntryDate:     model.Date(tDate),
		Description:   "USD entry with 1 IDR rounding diff",
		Postings: []model.CreatePostingRequest{
			{AccountID: accCash, Direction: domain.PostingDirectionDebit, Amount: dec(1)},
			{AccountID: accRev1, Direction: domain.PostingDirectionCredit, Amount: decimal.RequireFromString("0.5")},
			{AccountID: accRev2, Direction: domain.PostingDirectionCredit, Amount: decimal.RequireFromString("0.5")},
		},
	})
	s.Created()

	entryRes := s.GetResponse().Data
	s.Require().Len(entryRes.Postings, 4)
	s.Equal(fxAcc, entryRes.Postings[3].AccountID)
	s.Equal("CREDIT", entryRes.Postings[3].Direction)
	s.True(entryRes.Postings[3].Amount.Equal(decimal.Zero))
	s.True(entryRes.Postings[3].BaseAmount.Equal(dec(1)))

	// Post the entry
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d/post", ws.ID, entryRes.ID), fiber.MethodPost, nil)
	s.OK()

	// 2. Difference > 1.0 IDR should reject with ENTRY_BASE_NOT_BALANCED
	baseAmtDebit := dec(10005)
	baseAmtCredit := dec(10000)
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries", ws.ID), fiber.MethodPost, model.CreateEntryRequest{
		TransactionID: tx.ID,
		PeriodID:      pID,
		EntryDate:     model.Date(tDate),
		Description:   "USD entry with large base imbalance",
		Postings: []model.CreatePostingRequest{
			{AccountID: accCash, Direction: domain.PostingDirectionDebit, Amount: dec(1), BaseAmount: &baseAmtDebit},
			{AccountID: accRev1, Direction: domain.PostingDirectionCredit, Amount: dec(1), BaseAmount: &baseAmtCredit},
		},
	})
	s.Require().Equal(fiber.StatusUnprocessableEntity, s.HttpResponse.StatusCode)
	errRes := s.GetErrorResponse()
	s.Equal(domain.ErrEntryBaseNotBalanced.Error(), errRes.Error.Code)
}

func (s *EntryApiSuite) Test_MultiCurrency_Ledger_Endpoint() {
	ws := s.ensureWorkspace("Finance MultiCurrency WS", s.User.ID)
	client := testutil.GetEntCli()

	// Account restricted to USD
	usdAcc, err := client.Account.Create().
		SetWorkspaceID(ws.ID).
		SetCode("1135").
		SetName("Wise USD").
		SetType(account.Type(domain.AccountTypeAsset)).
		SetNormalBalance(account.NormalBalance(domain.NormalBalanceDebit)).
		SetCurrency("USD").
		SetIsActive(true).
		SetIsPostable(true).
		Save(base.WithWorkspaceID(s.Context, ws.ID))
	s.Require().NoError(err)

	accRev := s.ensureAccount(ws.ID, "4000", "Revenue", domain.AccountTypeRevenue, domain.NormalBalanceCredit)
	pID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)
	tDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)

	// 1. Posting an IDR transaction to USD account should fail with POSTING_CURRENCY_MISMATCH
	txIDR := s.ensureTransaction(ws.ID, "IDR Tx", "REF-IDR", tDate)
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries", ws.ID), fiber.MethodPost, model.CreateEntryRequest{
		TransactionID: txIDR,
		PeriodID:      pID,
		EntryDate:     model.Date(tDate),
		Postings: []model.CreatePostingRequest{
			{AccountID: usdAcc.ID, Direction: domain.PostingDirectionDebit, Amount: dec(150000)},
			{AccountID: accRev, Direction: domain.PostingDirectionCredit, Amount: dec(150000)},
		},
	})
	s.Require().Equal(fiber.StatusUnprocessableEntity, s.HttpResponse.StatusCode)
	errRes := s.GetErrorResponse()
	s.Equal(domain.ErrPostingCurrencyMismatch.Error(), errRes.Error.Code)

	// 2. Post a USD transaction to the USD account
	txUSD, err := client.Transaction.Create().
		SetWorkspaceID(ws.ID).
		SetNumber(fmt.Sprintf("TRX-USD-%d", time.Now().UnixNano()%100000)).
		SetTransactionDate(tDate).
		SetDescription("USD Consulting").
		SetCurrency("USD").
		SetExchangeRate(dec(15000)).
		SetSource(domain.TransactionSourceManual).
		Save(base.WithWorkspaceID(s.Context, ws.ID))
	s.Require().NoError(err)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries", ws.ID), fiber.MethodPost, model.CreateEntryRequest{
		TransactionID: txUSD.ID,
		PeriodID:      pID,
		EntryDate:     model.Date(tDate),
		Postings: []model.CreatePostingRequest{
			{AccountID: usdAcc.ID, Direction: domain.PostingDirectionDebit, Amount: dec(100)},
			{AccountID: accRev, Direction: domain.PostingDirectionCredit, Amount: dec(100)},
		},
	})
	s.Created()
	eID := s.GetResponse().Data.ID

	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d/post", ws.ID, eID), fiber.MethodPost, nil)
	s.OK()

	// 3. Query ledger for USD account: verify dual running balances
	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/%d/ledger?page=1&limit=10", ws.ID, usdAcc.ID), fiber.MethodGet, nil)
	s.OK()

	var ledgerRes httpx.WebResponse[model.AccountLedgerResponse]
	err = json.NewDecoder(s.HttpResponse.Body).Decode(&ledgerRes)
	s.Require().NoError(err)
	s.Equal(int64(1), ledgerRes.Data.Total)
	s.True(ledgerRes.Data.Balance.Equal(dec(100)))
	s.True(ledgerRes.Data.BaseBalance.Equal(dec(1500000)))
	s.Require().Len(ledgerRes.Data.Postings, 1)

	p := ledgerRes.Data.Postings[0]
	s.True(p.Amount.Equal(dec(100)))
	s.True(p.BaseAmount.Equal(dec(1500000)))
	s.True(p.Balance.Equal(dec(100)))
	s.True(p.RunningBalance.Equal(dec(100)))
	s.True(p.BaseBalance.Equal(dec(1500000)))
	s.True(p.RunningBaseBalance.Equal(dec(1500000)))
}

func TestEntryApiSuite(t *testing.T) {
	suite.Run(t, new(EntryApiSuite))
}
