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
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type TransactionApiSuite struct {
	testutil.ApiTestSuite[model.TransactionResponse]
}

func (s *TransactionApiSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *TransactionApiSuite) ensureAccount(wsID int64, code, name, accType, normalBalance string) int64 {
	client := testutil.GetEntCli()
	acc, err := client.Account.Create().
		SetWorkspaceID(wsID).
		SetCode(code).
		SetName(name).
		SetType(account.Type(accType)).
		SetNormalBalance(account.NormalBalance(normalBalance)).
		SetIsActive(true).
		SetIsPostable(true).
		Save(s.Context)
	s.Require().NoError(err)
	return acc.ID
}

func (s *TransactionApiSuite) ensurePeriod(wsID int64, name string, start, end time.Time, year int) int64 {
	client := testutil.GetEntCli()
	p, err := client.Period.Create().
		SetWorkspaceID(wsID).
		SetName(name).
		SetStartDate(start).
		SetEndDate(end).
		SetFiscalYear(year).
		SetStatus(period.Status(domain.PeriodStatusOpen)).
		Save(s.Context)
	s.Require().NoError(err)
	return p.ID
}

func (s *TransactionApiSuite) getListResponse() httpx.WebResponse[[]model.TransactionResponse] {
	var res httpx.WebResponse[[]model.TransactionResponse]
	s.T().Helper()
	defer s.HttpResponse.Body.Close()
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&res))
	return res
}

func (s *TransactionApiSuite) Test_Create_InlineEntry_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	acc1 := s.ensureAccount(ws.ID, "1000", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit)
	acc2 := s.ensureAccount(ws.ID, "3000", "Owner Equity", domain.AccountTypeEquity, domain.NormalBalanceCredit)
	periodID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)

	tDate := model.Date(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Initial capital injection",
		Reference:       "REF-001",
		Entry: &model.CreateEntryInlineRequest{
			PeriodID:    periodID,
			EntryDate:   tDate,
			Description: "Capital deposit",
			Postings: []model.CreatePostingRequest{
				{AccountID: acc1, Direction: domain.PostingDirectionDebit, Amount: 5000000},
				{AccountID: acc2, Direction: domain.PostingDirectionCredit, Amount: 5000000},
			},
		},
	})
	s.Created()

	res := s.GetResponse()
	s.Equal("Initial capital injection", res.Data.Description)
	s.Equal("REF-001", res.Data.Reference)
	s.Equal("IDR", res.Data.Currency)
	s.Equal("MANUAL", res.Data.Source)
	s.Require().Len(res.Data.Entries, 1)
	s.Require().Len(res.Data.Entries[0].Postings, 2)
}

func (s *TransactionApiSuite) Test_Create_HeaderOnly_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	tDate := model.Date(time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC))

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Header only transaction",
		Reference:       "REF-HEADER",
	})
	s.Created()

	res := s.GetResponse()
	s.Equal("Header only transaction", res.Data.Description)
	s.Empty(res.Data.Entries)
}

func (s *TransactionApiSuite) Test_GetByID_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Transaction to get",
		Reference:       "REF-GET",
	})
	s.Created()
	created := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.OK()

	res := s.GetResponse()
	s.Equal(created.ID, res.Data.ID)
	s.Equal("REF-GET", res.Data.Reference)
}

func (s *TransactionApiSuite) Test_Search_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "First tx",
		Reference:       "REF-SEARCH-1",
	})
	s.Created()

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Second tx",
		Reference:       "REF-SEARCH-2",
	})
	s.Created()

	// Search all in workspace
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/search", ws.ID), fiber.MethodPost, model.SearchTransactionRequest{})
	s.OK()

	listRes := s.getListResponse()
	s.Len(listRes.Data, 2)

	// Search by exact reference
	refQuery := "REF-SEARCH-1"
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/search", ws.ID), fiber.MethodPost, model.SearchTransactionRequest{
		Reference: &refQuery,
	})
	s.OK()

	filteredRes := s.getListResponse()
	s.Len(filteredRes.Data, 1)
	s.Equal("REF-SEARCH-1", filteredRes.Data[0].Reference)
}

func (s *TransactionApiSuite) Test_Update_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Before update",
		Reference:       "OLD-REF",
	})
	s.Created()
	created := s.GetResponse().Data

	newDesc := "After update"
	newRef := "NEW-REF"
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, created.ID), fiber.MethodPut, model.UpdateTransactionRequest{
		Description: &newDesc,
		Reference:   &newRef,
	})
	s.OK()

	res := s.GetResponse()
	s.Equal("After update", res.Data.Description)
	s.Equal("NEW-REF", res.Data.Reference)
}

func (s *TransactionApiSuite) Test_Delete_Successfully() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "To delete",
	})
	s.Created()
	created := s.GetResponse().Data

	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, created.ID), fiber.MethodDelete, nil)
	s.OK()

	// GET after delete returns 404
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *TransactionApiSuite) Test_Update_BlockedIfPosted() {
	ws := s.ensureWorkspace("Finance WS", s.User.ID)
	acc1 := s.ensureAccount(ws.ID, "1000", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit)
	acc2 := s.ensureAccount(ws.ID, "3000", "Owner Equity", domain.AccountTypeEquity, domain.NormalBalanceCredit)
	periodID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 2026)

	tDate := model.Date(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions", ws.ID), fiber.MethodPost, model.CreateTransactionRequest{
		TransactionDate: tDate,
		Description:     "Capital injection",
		Entry: &model.CreateEntryInlineRequest{
			PeriodID:    periodID,
			EntryDate:   tDate,
			Description: "Capital deposit",
			Postings: []model.CreatePostingRequest{
				{AccountID: acc1, Direction: domain.PostingDirectionDebit, Amount: 100000},
				{AccountID: acc2, Direction: domain.PostingDirectionCredit, Amount: 100000},
			},
		},
	})
	s.Created()
	tx := s.GetResponse().Data
	entryID := tx.Entries[0].ID

	// Post the entry
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d/post", ws.ID, entryID), fiber.MethodPost, nil)
	s.OK()

	// Try to update transaction header -> 409 Conflict
	newDesc := "Should fail"
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, tx.ID), fiber.MethodPut, model.UpdateTransactionRequest{
		Description: &newDesc,
	})
	s.AssertStatus(fiber.StatusConflict)

	// Try to delete transaction -> 409 Conflict
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, tx.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusConflict)
}

func TestTransactionApiSuite(t *testing.T) {
	suite.Run(t, new(TransactionApiSuite))
}
