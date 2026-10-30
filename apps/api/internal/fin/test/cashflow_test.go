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
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.CreateExpenseRequest{
		PaymentAccountID:  cashAcc,
		ExpenseAccountID:  expenseAcc,
		Amount:            decimal.NewFromInt(250000),
		TransactionDate:   tDate,
		Description:       "Office Supplies Purchase",
		Reference:         "EXP-2026-001",
		AttachmentFileIDs: []int64{uploaded.ID},
	})
	s.Created()

	res := s.GetResponse().Data
	s.Equal("EXPENSE", res.Type)
	s.Equal("POSTED", res.EntryStatus)
	s.True(res.Amount.Equal(decimal.NewFromInt(250000)))
	s.Equal(cashAcc, res.PaymentAccount.ID)
	s.Equal(expenseAcc, res.CategoryAccount.ID)
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
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income", ws.ID), fiber.MethodPost, model.CreateIncomeRequest{
		PaymentAccountID: cashAcc,
		RevenueAccountID: revenueAcc,
		Amount:           decimal.NewFromInt(750000),
		TransactionDate:  tDate,
		Description:      "Consulting Retainer",
		Reference:        "INC-2026-001",
	})
	s.Created()

	res := s.GetResponse().Data
	s.Equal("INCOME", res.Type)
	s.Equal("POSTED", res.EntryStatus)
	s.True(res.Amount.Equal(decimal.NewFromInt(750000)))
	s.Equal(cashAcc, res.PaymentAccount.ID)
	s.Equal(revenueAcc, res.CategoryAccount.ID)

	// Verify SearchIncomes finds this income
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income/search", ws.ID), fiber.MethodPost, model.SearchCashFlowRequest{})
	s.OK()
	list := s.getListResponse().Data
	s.Require().Len(list, 1)
	s.Equal(res.ID, list[0].ID)
}

func (s *CashFlowApiSuite) Test_CreateExpense_InvalidAccountType() {
	ws := s.ensureWorkspace("Finance Invalid Type WS", s.User.ID)
	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", domain.AccountTypeAsset, domain.NormalBalanceDebit, true, true)
	expenseAcc := s.ensureAccount(ws.ID, "5001", "Office Supplies", domain.AccountTypeExpense, domain.NormalBalanceDebit, true, true)
	equityAcc := s.ensureAccount(ws.ID, "3001", "Equity", domain.AccountTypeEquity, domain.NormalBalanceCredit, true, true)
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, domain.PeriodStatusOpen)

	tDate := model.Date(time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC))

	// Non-ASSET payment account
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.CreateExpenseRequest{
		PaymentAccountID: equityAcc,
		ExpenseAccountID: expenseAcc,
		Amount:           decimal.NewFromInt(100000),
		TransactionDate:  tDate,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)

	// Non-EXPENSE category account
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.CreateExpenseRequest{
		PaymentAccountID: cashAcc,
		ExpenseAccountID: equityAcc,
		Amount:           decimal.NewFromInt(100000),
		TransactionDate:  tDate,
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
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.CreateExpenseRequest{
		PaymentAccountID: inactiveCash,
		ExpenseAccountID: expenseAcc,
		Amount:           decimal.NewFromInt(100000),
		TransactionDate:  tDate,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)

	// Non-postable expense account
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.CreateExpenseRequest{
		PaymentAccountID: cashAcc,
		ExpenseAccountID: nonPostExpense,
		Amount:           decimal.NewFromInt(100000),
		TransactionDate:  tDate,
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
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.CreateExpenseRequest{
		PaymentAccountID: cashAcc,
		ExpenseAccountID: expenseAcc,
		Amount:           decimal.NewFromInt(100000),
		TransactionDate:  model.Date(time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)),
	})
	s.AssertStatus(fiber.StatusNotFound)

	// Date in closed period (Feb 2026) -> 409
	s.Request(fmt.Sprintf("/workspaces/%d/fin/expense", ws.ID), fiber.MethodPost, model.CreateExpenseRequest{
		PaymentAccountID: cashAcc,
		ExpenseAccountID: expenseAcc,
		Amount:           decimal.NewFromInt(100000),
		TransactionDate:  model.Date(time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)),
	})
	s.AssertStatus(fiber.StatusConflict)
}

func TestCashFlowApiSuite(t *testing.T) {
	suite.Run(t, new(CashFlowApiSuite))
}
