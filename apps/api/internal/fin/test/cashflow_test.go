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
	coremodel "github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type CashFlowApiSuite struct {
	testutil.ApiTestSuite[model.CashFlowItemResponse]
}

func (s *CashFlowApiSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *CashFlowApiSuite) ensureAccount(wsID int64, code, name, accType, normalBalance string, isActive, isPostable bool) int64 {
	client := testutil.GetEntCli()
	acc, err := client.Account.Create().
		SetWorkspaceID(wsID).
		SetCode(code).
		SetName(name).
		SetType(account.Type(accType)).
		SetNormalBalance(account.NormalBalance(normalBalance)).
		SetIsActive(isActive).
		SetIsPostable(isPostable).
		Save(base.WithWorkspaceID(s.Context, wsID))
	s.Require().NoError(err)
	return acc.ID
}

func (s *CashFlowApiSuite) ensurePeriod(wsID int64, name string, start, end time.Time, year int, status string) int64 {
	client := testutil.GetEntCli()
	p, err := client.Period.Create().
		SetWorkspaceID(wsID).
		SetName(name).
		SetStartDate(start).
		SetEndDate(end).
		SetFiscalYear(year).
		SetStatus(period.Status(status)).
		Save(base.WithWorkspaceID(s.Context, wsID))
	s.Require().NoError(err)
	return p.ID
}

func (s *CashFlowApiSuite) getListResponse() httpx.WebResponse[[]model.CashFlowItemResponse] {
	var res httpx.WebResponse[[]model.CashFlowItemResponse]
	s.T().Helper()
	defer func() { _ = s.HttpResponse.Body.Close() }()
	s.Require().NoError(json.NewDecoder(s.HttpResponse.Body).Decode(&res))
	return res
}

func (s *CashFlowApiSuite) uploadFile(wsID int64, filename string, content []byte) coremodel.FileResponse {
	s.RequestMultipart(fmt.Sprintf("/workspaces/%d/files", wsID), fiber.MethodPost, "file", filename, content)
	s.Created()
	return testutil.DecodeOther[coremodel.FileResponse](s).Data
}

func (s *CashFlowApiSuite) Test_CreateExpense_ValidWithAttachments_Success() {
	ws := s.ensureWorkspace("Finance CF WS", s.User.ID)
	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit, true, true)
	expenseAcc := s.ensureAccount(ws.ID, "5001", "Office Supplies", domain.AccountTypeExpense, domain.NormalBalanceDebit, true, true)
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusOpen)

	uploaded := s.uploadFile(ws.ID, "receipt.pdf", []byte("%PDF-1.4 expense test receipt"))

	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID:  cashAcc,
		TransactionDate:   tDate,
		Description:       "Office Supplies Purchase",
		Reference:         "EXP-2026-001",
		AttachmentFileIDs: []int64{uploaded.ID},
		Items: []model.CashFlowItemRequest{
			{
				AccountID:   expenseAcc,
				Description: "Supplies",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(250000),
			},
		},
	})
	s.Created()

	res := s.GetResponse().Data
	s.Equal("EXPENSE", res.Type)
	s.Equal("POSTED", res.EntryStatus)
	s.True(res.Amount.Equal(decimal.NewFromInt(250000)))
	s.Equal(cashAcc, res.PaymentAccount.ID)
	s.Require().Len(res.Items, 1)
	s.Equal(expenseAcc, res.Items[0].Account.ID)
	s.Require().Len(res.Attachments, 1)
	s.Equal(uploaded.ID, res.Attachments[0].ID)

	// Verify SearchExpenses finds this expense with attachment
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense/search", ws.ID), fiber.MethodPost, model.SearchCashFlowRequest{})
	s.OK()
	list := s.getListResponse().Data
	s.Require().Len(list, 1)
	s.Equal(res.ID, list[0].ID)
	s.Require().Len(list[0].Attachments, 1)
	s.Equal(uploaded.ID, list[0].Attachments[0].ID)
}

func (s *CashFlowApiSuite) Test_CreateIncome_Valid_Success() {
	ws := s.ensureWorkspace("Finance CF Income WS", s.User.ID)
	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit, true, true)
	revenueAcc := s.ensureAccount(ws.ID, "4001", "Consulting", domain.AccountTypeRevenue, domain.NormalBalanceCredit, true, true)
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusOpen)

	tDate := model.Date(time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  tDate,
		Description:      "Consulting Retainer",
		Reference:        "INC-2026-001",
		Items: []model.CashFlowItemRequest{
			{
				AccountID:   revenueAcc,
				Description: "Retainer",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(750000),
			},
		},
	})
	s.Created()

	res := s.GetResponse().Data
	s.Equal("INCOME", res.Type)
	s.Equal("POSTED", res.EntryStatus)
	s.True(res.Amount.Equal(decimal.NewFromInt(750000)))
	s.Equal(cashAcc, res.PaymentAccount.ID)
	s.Require().Len(res.Items, 1)
	s.Equal(revenueAcc, res.Items[0].Account.ID)

	// Verify SearchIncomes finds this income
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income/search", ws.ID), fiber.MethodPost, model.SearchCashFlowRequest{})
	s.OK()
	list := s.getListResponse().Data
	s.Require().Len(list, 1)
	s.Equal(res.ID, list[0].ID)
}

func (s *CashFlowApiSuite) Test_CreateIncome_TwoItemsPosted_Success() {
	ws := s.ensureWorkspace("Finance Income 2 Items WS", s.User.ID)
	cashAcc := s.ensureAccount(ws.ID, "1120", "Bank Kaltim", domain.AccountTypeAsset, domain.NormalBalanceDebit, true, true)
	rev1 := s.ensureAccount(ws.ID, "4130", "Consulting & Advisory Income", domain.AccountTypeRevenue, domain.NormalBalanceCredit, true, true)
	rev2 := s.ensureAccount(ws.ID, "4110", "Development Income", domain.AccountTypeRevenue, domain.NormalBalanceCredit, true, true)
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusOpen)

	tDate := model.Date(time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  tDate,
		Reference:        "inv-123456",
		Description:      "Consulting and scaffolding",
		Currency:         "IDR",
		Status:           "POSTED",
		Items: []model.CashFlowItemRequest{
			{
				AccountID:   rev1,
				Description: "Konsultasi Improvement",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(1000000),
			},
			{
				AccountID:   rev2,
				Description: "Scaffold base project",
				Quantity:    decimal.NewFromInt(6),
				UnitPrice:   decimal.NewFromInt(500000),
			},
		},
	})
	s.Created()

	res := s.GetResponse().Data
	s.Equal("INCOME", res.Type)
	s.Equal("POSTED", res.EntryStatus)
	s.True(res.Amount.Equal(decimal.NewFromInt(4000000)))
	s.Require().Len(res.Items, 2)

	// GET /entries/{entryId} shows 3 postings
	s.Request(fmt.Sprintf("/workspaces/%d/fin/entries/%d", ws.ID, res.EntryID), fiber.MethodGet, nil)
	s.OK()
	entryRes := testutil.DecodeOther[model.EntryResponse](s).Data
	s.Require().Len(entryRes.Postings, 3)

	// POST /fin/reports/trial-balance reflects 4000000 on the payment account
	asOf := model.Date(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/reports/trial-balance", ws.ID), fiber.MethodPost, model.TrialBalanceRequest{
		AsOfDate: asOf,
	})
	s.OK()
	tbRes := testutil.DecodeOther[model.TrialBalanceResponse](s).Data
	var foundPayment bool
	for _, acc := range tbRes.Accounts {
		if acc.AccountID == cashAcc {
			foundPayment = true
			s.True(acc.Debit.Equal(decimal.NewFromInt(4000000)))
			s.True(acc.NetBalance.Equal(decimal.NewFromInt(4000000)))
		}
	}
	s.True(foundPayment)
}

func (s *CashFlowApiSuite) Test_CreateExpense_Draft_LeavesTrialBalanceUnchanged() {
	ws := s.ensureWorkspace("Finance Draft Expense WS", s.User.ID)
	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit, true, true)
	expenseAcc := s.ensureAccount(ws.ID, "5001", "Office Supplies", domain.AccountTypeExpense, domain.NormalBalanceDebit, true, true)
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusOpen)

	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  tDate,
		Status:           "DRAFT",
		Items: []model.CashFlowItemRequest{
			{
				AccountID: expenseAcc,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(500000),
			},
		},
	})
	s.Created()
	res := s.GetResponse().Data
	s.Equal("DRAFT", res.EntryStatus)

	// Trial balance should be empty / total debit is zero
	asOf := model.Date(time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/reports/trial-balance", ws.ID), fiber.MethodPost, model.TrialBalanceRequest{
		AsOfDate: asOf,
	})
	s.OK()
	tbRes := testutil.DecodeOther[model.TrialBalanceResponse](s).Data
	s.True(tbRes.TotalDebit.IsZero())
	s.True(tbRes.TotalCredit.IsZero())
}

func (s *CashFlowApiSuite) Test_UpdateIncome_DraftToPosted_Success() {
	ws := s.ensureWorkspace("Finance Update Draft Income WS", s.User.ID)
	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit, true, true)
	revAcc := s.ensureAccount(ws.ID, "4001", "Consulting", domain.AccountTypeRevenue, domain.NormalBalanceCredit, true, true)
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusOpen)

	tDate := model.Date(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  tDate,
		Status:           "DRAFT",
		Items: []model.CashFlowItemRequest{
			{
				AccountID: revAcc,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(1000000),
			},
		},
	})
	s.Created()
	draft := s.GetResponse().Data
	s.Equal("DRAFT", draft.EntryStatus)

	// Update draft to posted
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income/%d", ws.ID, draft.ID), fiber.MethodPut, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  tDate,
		Status:           "POSTED",
		Items: []model.CashFlowItemRequest{
			{
				AccountID: revAcc,
				Quantity:  decimal.NewFromInt(2),
				UnitPrice: decimal.NewFromInt(1000000),
			},
		},
	})
	s.OK()
	updated := s.GetResponse().Data
	s.Equal("POSTED", updated.EntryStatus)
	s.True(updated.Amount.Equal(decimal.NewFromInt(2000000)))

	// GetIncome returns the updated document
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income/%d", ws.ID, draft.ID), fiber.MethodGet, nil)
	s.OK()
	getRes := s.GetResponse().Data
	s.Equal(draft.ID, getRes.ID)
	s.Equal("POSTED", getRes.EntryStatus)
	s.True(getRes.Amount.Equal(decimal.NewFromInt(2000000)))
}

func (s *CashFlowApiSuite) Test_UpdateIncome_Posted_Returns409EntryNotDraft() {
	ws := s.ensureWorkspace("Finance Update Posted WS", s.User.ID)
	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit, true, true)
	revAcc := s.ensureAccount(ws.ID, "4001", "Consulting", domain.AccountTypeRevenue, domain.NormalBalanceCredit, true, true)
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusOpen)

	tDate := model.Date(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  tDate,
		Status:           "POSTED",
		Items: []model.CashFlowItemRequest{
			{
				AccountID: revAcc,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(1000000),
			},
		},
	})
	s.Created()
	posted := s.GetResponse().Data
	s.Equal("POSTED", posted.EntryStatus)

	// Attempt PUT on posted document
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income/%d", ws.ID, posted.ID), fiber.MethodPut, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  tDate,
		Items: []model.CashFlowItemRequest{
			{
				AccountID: revAcc,
				Quantity:  decimal.NewFromInt(2),
				UnitPrice: decimal.NewFromInt(1000000),
			},
		},
	})
	s.AssertStatus(fiber.StatusConflict)
}

func (s *CashFlowApiSuite) Test_GetExpense_OnIncomeID_Returns404() {
	ws := s.ensureWorkspace("Finance Mismatched Type WS", s.User.ID)
	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit, true, true)
	revAcc := s.ensureAccount(ws.ID, "4001", "Consulting", domain.AccountTypeRevenue, domain.NormalBalanceCredit, true, true)
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusOpen)

	tDate := model.Date(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  tDate,
		Items: []model.CashFlowItemRequest{
			{
				AccountID: revAcc,
				Quantity:  decimal.NewFromInt(1),
				UnitPrice: decimal.NewFromInt(1000000),
			},
		},
	})
	s.Created()
	income := s.GetResponse().Data

	// GET on /fin/expense/{id} must return 404
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense/%d", ws.ID, income.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *CashFlowApiSuite) Test_Search_StatusFilter() {
	ws := s.ensureWorkspace("Finance Search Status WS", s.User.ID)
	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit, true, true)
	expenseAcc := s.ensureAccount(ws.ID, "5001", "Office Supplies", domain.AccountTypeExpense, domain.NormalBalanceDebit, true, true)
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusOpen)

	tDate := model.Date(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	// Create draft expense
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  tDate,
		Status:           "DRAFT",
		Items: []model.CashFlowItemRequest{
			{AccountID: expenseAcc, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100000)},
		},
	})
	s.Created()
	draftID := s.GetResponse().Data.ID

	// Create posted expense
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  tDate,
		Status:           "POSTED",
		Items: []model.CashFlowItemRequest{
			{AccountID: expenseAcc, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(200000)},
		},
	})
	s.Created()
	postedID := s.GetResponse().Data.ID

	// Filter DRAFT
	statusDraft := "DRAFT"
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense/search", ws.ID), fiber.MethodPost, model.SearchCashFlowRequest{
		Status: &statusDraft,
	})
	s.OK()
	draftList := s.getListResponse().Data
	s.Require().Len(draftList, 1)
	s.Equal(draftID, draftList[0].ID)

	// Filter POSTED
	statusPosted := "POSTED"
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense/search", ws.ID), fiber.MethodPost, model.SearchCashFlowRequest{
		Status: &statusPosted,
	})
	s.OK()
	postedList := s.getListResponse().Data
	s.Require().Len(postedList, 1)
	s.Equal(postedID, postedList[0].ID)
}

func (s *CashFlowApiSuite) Test_DeleteDraftViaTransactions_Success() {
	ws := s.ensureWorkspace("Finance Delete Draft WS", s.User.ID)
	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit, true, true)
	revAcc := s.ensureAccount(ws.ID, "4001", "Consulting", domain.AccountTypeRevenue, domain.NormalBalanceCredit, true, true)
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusOpen)

	tDate := model.Date(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  tDate,
		Status:           "DRAFT",
		Items: []model.CashFlowItemRequest{
			{AccountID: revAcc, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(500000)},
		},
	})
	s.Created()
	draft := s.GetResponse().Data

	// Delete via transactions endpoint
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, draft.ID), fiber.MethodDelete, nil)
	s.OK()

	// Verify GetIncome returns 404
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income/%d", ws.ID, draft.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *CashFlowApiSuite) Test_DeletePostedViaTransactions_Returns409TransactionHasPostedEntries() {
	ws := s.ensureWorkspace("Finance Delete Posted WS", s.User.ID)
	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit, true, true)
	revAcc := s.ensureAccount(ws.ID, "4001", "Consulting", domain.AccountTypeRevenue, domain.NormalBalanceCredit, true, true)
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusOpen)

	tDate := model.Date(time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  tDate,
		Status:           "POSTED",
		Items: []model.CashFlowItemRequest{
			{AccountID: revAcc, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(500000)},
		},
	})
	s.Created()
	posted := s.GetResponse().Data

	// Delete via transactions endpoint should return 409
	s.Request(fmt.Sprintf("/workspaces/%d/fin/transactions/%d", ws.ID, posted.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusConflict)
}

func (s *CashFlowApiSuite) Test_CreateIncome_NoItems_ReturnsValidationError() {
	ws := s.ensureWorkspace("Finance Validation Error WS", s.User.ID)
	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit, true, true)
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusOpen)

	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  tDate,
		Items:            []model.CashFlowItemRequest{},
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *CashFlowApiSuite) Test_CreateExpense_InvalidAccountType() {
	ws := s.ensureWorkspace("Finance Invalid Type WS", s.User.ID)
	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit, true, true)
	expenseAcc := s.ensureAccount(ws.ID, "5001", "Office Supplies", domain.AccountTypeExpense, domain.NormalBalanceDebit, true, true)
	equityAcc := s.ensureAccount(ws.ID, "3001", "Equity", domain.AccountTypeEquity, domain.NormalBalanceCredit, true, true)
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusOpen)

	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	// Non-ASSET payment account
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: equityAcc,
		TransactionDate:  tDate,
		Items: []model.CashFlowItemRequest{
			{AccountID: expenseAcc, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100000)},
		},
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)

	// Non-EXPENSE category account
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  tDate,
		Items: []model.CashFlowItemRequest{
			{AccountID: equityAcc, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100000)},
		},
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *CashFlowApiSuite) Test_CreateExpense_InactiveOrNotPostableAccount() {
	ws := s.ensureWorkspace("Finance Inactive WS", s.User.ID)
	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit, true, true)
	inactiveCash := s.ensureAccount(ws.ID, "1002", "Old Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit, false, true)
	expenseAcc := s.ensureAccount(ws.ID, "5001", "Office Supplies", domain.AccountTypeExpense, domain.NormalBalanceDebit, true, true)
	nonPostExpense := s.ensureAccount(ws.ID, "5000", "Total Expenses", domain.AccountTypeExpense, domain.NormalBalanceDebit, true, false)
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusOpen)

	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	// Inactive payment account
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: inactiveCash,
		TransactionDate:  tDate,
		Items: []model.CashFlowItemRequest{
			{AccountID: expenseAcc, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100000)},
		},
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)

	// Non-postable expense account
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  tDate,
		Items: []model.CashFlowItemRequest{
			{AccountID: nonPostExpense, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100000)},
		},
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *CashFlowApiSuite) Test_CreateExpense_PeriodNotFoundOrClosed() {
	ws := s.ensureWorkspace("Finance Period WS", s.User.ID)
	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit, true, true)
	expenseAcc := s.ensureAccount(ws.ID, "5001", "Office Supplies", domain.AccountTypeExpense, domain.NormalBalanceDebit, true, true)
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusOpen)
	s.ensurePeriod(ws.ID, "Feb 2026", time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusClosed)

	// Date outside open periods (March 2026) -> 404
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  model.Date(time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)),
		Items: []model.CashFlowItemRequest{
			{AccountID: expenseAcc, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100000)},
		},
	})
	s.AssertStatus(fiber.StatusNotFound)

	// Date in closed period (Feb 2026) -> 409
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc,
		TransactionDate:  model.Date(time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)),
		Items: []model.CashFlowItemRequest{
			{AccountID: expenseAcc, Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100000)},
		},
	})
	s.AssertStatus(fiber.StatusConflict)
}

func TestCashFlowApiSuite(t *testing.T) {
	suite.Run(t, new(CashFlowApiSuite))
}
