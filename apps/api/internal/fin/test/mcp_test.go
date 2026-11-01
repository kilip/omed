package test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/account"
	"github.com/kilip/omed/api/ent/personalaccesstoken"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/fin/domain"
	"github.com/kilip/omed/api/internal/fin/model"
	"github.com/kilip/omed/api/internal/fin/repository"
	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

func init() {
	authz.AddRole(authz.Role{
		ID:          "fin:trx_reader",
		Name:        "Transaction Reader",
		Description: "Can read transactions but not accounts",
		Permissions: []authz.Permission{
			{Object: domain.ObjTransaction, Action: "read"},
		},
	})
}

type mcpRpcResponse struct {
	JsonRpc string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type mcpErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type McpFinApiSuite struct {
	testutil.ApiTestSuite[any]
}

func (s *McpFinApiSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *McpFinApiSuite) ensurePeriod(wsID int64, name string, start, end time.Time, year int) int64 {
	client := testutil.GetEntCli()
	p, err := client.Period.Create().
		SetWorkspaceID(wsID).
		SetName(name).
		SetStartDate(start).
		SetEndDate(end).
		SetFiscalYear(year).
		SetStatus("OPEN").
		Save(base.WithWorkspaceID(s.Context, wsID))
	s.Require().NoError(err)
	return p.ID
}

func (s *McpFinApiSuite) callTool(pat string, name string, args map[string]any) mcpRpcResponse {
	s.SetHeader("Accept", "application/json, text/event-stream")
	s.RequestWithToken("/mcp", fiber.MethodPost, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": args,
		},
	}, pat)
	s.OK()

	var rpcResp mcpRpcResponse
	err := json.NewDecoder(s.HttpResponse.Body).Decode(&rpcResp)
	s.Require().NoError(err)
	return rpcResp
}

func (s *McpFinApiSuite) parseErrorPayload(res mcpRpcResponse) mcpErrorPayload {
	s.Require().True(res.Result.IsError)
	s.Require().NotEmpty(res.Result.Content)
	var p mcpErrorPayload
	err := json.Unmarshal([]byte(res.Result.Content[0].Text), &p)
	s.Require().NoError(err)
	return p
}

func (s *McpFinApiSuite) Test_Mcp_PeriodOverview_Success() {
	ws := s.ensureWorkspace("MCP Fin Overview WS", s.User.ID)
	pat := s.EnsurePAT(s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/seed", ws.ID), fiber.MethodPost, model.SeedAccountsRequest{
		Template: "freelancer",
		Lang:     "en",
	})
	s.Created()

	now := time.Now()
	s.ensurePeriod(ws.ID, "Current Year", time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC), time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.UTC), now.Year())

	// Call fin_period_overview
	res := s.callTool(pat, "fin_period_overview", map[string]any{
		"workspace_id": ws.ID,
	})
	s.Require().False(res.Result.IsError)
	s.Require().NotEmpty(res.Result.Content)

	rawJSON := res.Result.Content[0].Text
	var overview model.PeriodOverviewResponse
	err := json.Unmarshal([]byte(rawJSON), &overview)
	s.Require().NoError(err)
	s.Equal("IDR", overview.BaseCurrency)
	s.NotEmpty(overview.From)
	s.NotEmpty(overview.To)

	// Verify monetary fields are strings in the raw JSON
	var rawMap map[string]any
	s.Require().NoError(json.Unmarshal([]byte(rawJSON), &rawMap))
	_, isRevString := rawMap["revenue"].(string)
	s.True(isRevString, "revenue should be serialized as string")
	_, isExpString := rawMap["expense"].(string)
	s.True(isExpString, "expense should be serialized as string")
	_, isNetString := rawMap["net"].(string)
	s.True(isNetString, "net should be serialized as string")
}

func (s *McpFinApiSuite) Test_Mcp_AccountBalance_Success() {
	ws := s.ensureWorkspace("MCP Fin Balance WS", s.User.ID)
	pat := s.EnsurePAT(s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/seed", ws.ID), fiber.MethodPost, model.SeedAccountsRequest{
		Template: "freelancer",
		Lang:     "en",
	})
	s.Created()

	res := s.callTool(pat, "fin_account_balance", map[string]any{
		"workspace_id": ws.ID,
		"account":      "11",
	})
	s.Require().False(res.Result.IsError)
	s.Require().NotEmpty(res.Result.Content)

	rawJSON := res.Result.Content[0].Text
	var bal model.AccountBalanceResponse
	err := json.Unmarshal([]byte(rawJSON), &bal)
	s.Require().NoError(err)
	s.Equal("IDR", bal.BaseCurrency)
	s.True(bal.IsBalanced)
	s.NotEmpty(bal.Accounts)
	for _, acc := range bal.Accounts {
		s.True(strings.HasPrefix(acc.Code, "11"), "expected account code starting with 11, got %s", acc.Code)
	}

	// Verify monetary strings in raw JSON
	var rawMap map[string]any
	s.Require().NoError(json.Unmarshal([]byte(rawJSON), &rawMap))
	totals, ok := rawMap["totals"].(map[string]any)
	s.Require().True(ok)
	_, isDebitString := totals["debit"].(string)
	s.True(isDebitString)
}

func (s *McpFinApiSuite) Test_Mcp_FindCashflow_Success() {
	ws := s.ensureWorkspace("MCP Fin Cashflow WS", s.User.ID)
	pat := s.EnsurePAT(s.User.ID)

	client := testutil.GetEntCli()
	cashAcc, err := client.Account.Create().
		SetWorkspaceID(ws.ID).
		SetCode("1101").
		SetName("Cash on Hand").
		SetType(account.TypeASSET).
		SetNormalBalance(account.NormalBalanceDEBIT).
		SetCurrency("IDR").
		SetIsActive(true).
		SetIsPostable(true).
		Save(base.WithWorkspaceID(s.Context, ws.ID))
	s.Require().NoError(err)

	revAcc, err := client.Account.Create().
		SetWorkspaceID(ws.ID).
		SetCode("4101").
		SetName("Consulting Income").
		SetType(account.TypeREVENUE).
		SetNormalBalance(account.NormalBalanceCREDIT).
		SetCurrency("IDR").
		SetIsActive(true).
		SetIsPostable(true).
		Save(base.WithWorkspaceID(s.Context, ws.ID))
	s.Require().NoError(err)

	now := time.Now()
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC), time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.UTC), now.Year())

	// Create income via REST endpoint
	s.Request(fmt.Sprintf("/workspaces/%d/fin/income", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
		PaymentAccountID: cashAcc.ID,
		TransactionDate:  model.Date(now),
		Description:      "Consulting invoice payment",
		Reference:        "INV-2026-001",
		Items: []model.CashFlowItemRequest{
			{
				AccountID:   revAcc.ID,
				Description: "Consulting",
				Quantity:    decimal.NewFromInt(1),
				UnitPrice:   decimal.NewFromInt(5000000),
			},
		},
	})
	s.Created()

	// Call fin_find_cashflow
	res := s.callTool(pat, "fin_find_cashflow", map[string]any{
		"workspace_id": ws.ID,
		"type":         "income",
	})
	s.Require().False(res.Result.IsError)
	s.Require().NotEmpty(res.Result.Content)

	rawJSON := res.Result.Content[0].Text
	var cfRes model.FindCashflowResponse
	err = json.Unmarshal([]byte(rawJSON), &cfRes)
	s.Require().NoError(err)
	s.Require().Len(cfRes.Items, 1)

	doc := cfRes.Items[0]
	s.Equal("INV-2026-001", doc.Reference)
	s.Equal("income", doc.Type)
	s.Equal("posted", doc.Status)
	s.Equal("1101", doc.PaymentAccount.Code)
	s.Equal("IDR", doc.Currency)
	s.True(doc.Amount.Equal(decimal.NewFromInt(5000000)))

	// Acceptance 7: verify internal fields are NOT present in output
	s.NotContains(rawJSON, "attachments")
	s.NotContains(rawJSON, "auditMeta")
	s.NotContains(rawJSON, "entryId")
	s.NotContains(rawJSON, "entryNumber")
	s.NotContains(rawJSON, "workspaceId")
}

func (s *McpFinApiSuite) Test_Mcp_NonMember_NotFound() {
	ws := s.ensureWorkspace("MCP Fin Private WS", s.User.ID)

	// Another user creates a PAT and tries to query the workspace
	otherUser := s.AsUser("stranger@example.com", "Stranger User")
	strangerPAT := s.EnsurePAT(otherUser.ID)

	res := s.callTool(strangerPAT, "fin_period_overview", map[string]any{
		"workspace_id": ws.ID,
	})
	p := s.parseErrorPayload(res)
	s.Equal("NOT_FOUND", p.Code)
}

func (s *McpFinApiSuite) Test_Mcp_RoleWithoutAccountRead_ForbiddenOnFindCashflow() {
	owner := s.User
	ws := s.ensureWorkspace("MCP Fin Trx Only WS", owner.ID)

	reader := s.AsUser("trxreader@example.com", "Trx Reader")
	s.AddMember(ws.ID, reader.ID, "fin:trx_reader")
	readerPAT := s.EnsurePAT(reader.ID)

	res := s.callTool(readerPAT, "fin_find_cashflow", map[string]any{
		"workspace_id": ws.ID,
		"type":         "income",
	})
	p := s.parseErrorPayload(res)
	s.Equal("FORBIDDEN", p.Code)
}

func (s *McpFinApiSuite) Test_Mcp_FindCashflow_CursorRoundTrip() {
	ws := s.ensureWorkspace("MCP Fin Cursor WS", s.User.ID)
	pat := s.EnsurePAT(s.User.ID)

	client := testutil.GetEntCli()
	cashAcc, err := client.Account.Create().
		SetWorkspaceID(ws.ID).
		SetCode("1102").
		SetName("Bank Cursor").
		SetType(account.TypeASSET).
		SetNormalBalance(account.NormalBalanceDEBIT).
		SetCurrency("IDR").
		SetIsActive(true).
		SetIsPostable(true).
		Save(base.WithWorkspaceID(s.Context, ws.ID))
	s.Require().NoError(err)

	revAcc, err := client.Account.Create().
		SetWorkspaceID(ws.ID).
		SetCode("4102").
		SetName("Revenue Cursor").
		SetType(account.TypeREVENUE).
		SetNormalBalance(account.NormalBalanceCREDIT).
		SetCurrency("IDR").
		SetIsActive(true).
		SetIsPostable(true).
		Save(base.WithWorkspaceID(s.Context, ws.ID))
	s.Require().NoError(err)

	now := time.Now()
	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC), time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.UTC), now.Year())

	createdIDs := make([]int64, 5)
	for i := 0; i < 5; i++ {
		s.Request(fmt.Sprintf("/workspaces/%d/fin/income", ws.ID), fiber.MethodPost, model.SaveCashFlowRequest{
			PaymentAccountID: cashAcc.ID,
			TransactionDate:  model.Date(time.Date(now.Year(), 1, 1+i, 0, 0, 0, 0, time.UTC)),
			Reference:        fmt.Sprintf("REF-%d", i),
			Description:      fmt.Sprintf("Income %d", i),
			Items: []model.CashFlowItemRequest{
				{
					AccountID:   revAcc.ID,
					Description: "Service",
					Quantity:    decimal.NewFromInt(1),
					UnitPrice:   decimal.NewFromInt(100000),
				},
			},
		})
		s.Created()
		var createdRes httpx.WebResponse[model.CashFlowItemResponse]
		err := json.NewDecoder(s.HttpResponse.Body).Decode(&createdRes)
		s.Require().NoError(err)
		createdIDs[i] = createdRes.Data.ID
	}

	seenIDs := make(map[int64]bool)
	cursor := ""
	for {
		args := map[string]any{
			"workspace_id": ws.ID,
			"type":         "income",
			"limit":        2,
		}
		if cursor != "" {
			args["cursor"] = cursor
		}
		res := s.callTool(pat, "fin_find_cashflow", args)
		s.Require().False(res.Result.IsError)

		var cfRes model.FindCashflowResponse
		err := json.Unmarshal([]byte(res.Result.Content[0].Text), &cfRes)
		s.Require().NoError(err)

		for _, item := range cfRes.Items {
			s.False(seenIDs[item.ID], "duplicate document id %d", item.ID)
			seenIDs[item.ID] = true
		}

		if cfRes.NextCursor == "" {
			break
		}
		cursor = cfRes.NextCursor
	}

	s.Len(seenIDs, 5)
	for _, id := range createdIDs {
		s.True(seenIDs[id], "missing document id %d", id)
	}
}

func (s *McpFinApiSuite) Test_Mcp_AccountBalance_TruncatedFlag() {
	ws := s.ensureWorkspace("MCP Fin Truncated WS", s.User.ID)
	pat := s.EnsurePAT(s.User.ID)
	client := testutil.GetEntCli()
	ctx := base.WithWorkspaceID(s.Context, ws.ID)

	now := time.Now()
	pID := s.ensurePeriod(ws.ID, "Jan 2026", time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC), time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.UTC), now.Year())

	trx, err := client.Transaction.Create().
		SetWorkspaceID(ws.ID).
		SetNumber("TRX-BATCH-60").
		SetTransactionDate(now).
		SetDescription("Seeding 60 accounts").
		SetSource("MANUAL").
		SetCurrency("IDR").
		SetExchangeRate(dec(1)).
		Save(ctx)
	s.Require().NoError(err)

	je, err := client.Entry.Create().
		SetWorkspaceID(ws.ID).
		SetTransactionID(trx.ID).
		SetPeriodID(pID).
		SetNumber("JE-BATCH-60").
		SetEntryDate(now).
		SetStatus("POSTED").
		Save(ctx)
	s.Require().NoError(err)

	// Create 60 active accounts with balanced postings (debit 100, credit 100)
	for i := 1; i <= 60; i++ {
		code := fmt.Sprintf("80%02d", i)
		acc, err := client.Account.Create().
			SetWorkspaceID(ws.ID).
			SetCode(code).
			SetName(fmt.Sprintf("Expense Account %s", code)).
			SetType(account.TypeEXPENSE).
			SetNormalBalance(account.NormalBalanceDEBIT).
			SetCurrency("IDR").
			SetIsActive(true).
			SetIsPostable(true).
			Save(ctx)
		s.Require().NoError(err)

		_, err = client.Posting.Create().
			SetWorkspaceID(ws.ID).
			SetEntryID(je.ID).
			SetAccountID(acc.ID).
			SetDirection("DEBIT").
			SetAmount(dec(100)).
			SetBaseAmount(dec(100)).
			SetPosition(i * 2).
			Save(ctx)
		s.Require().NoError(err)

		_, err = client.Posting.Create().
			SetWorkspaceID(ws.ID).
			SetEntryID(je.ID).
			SetAccountID(acc.ID).
			SetDirection("CREDIT").
			SetAmount(dec(100)).
			SetBaseAmount(dec(100)).
			SetPosition(i*2 + 1).
			Save(ctx)
		s.Require().NoError(err)
	}

	res := s.callTool(pat, "fin_account_balance", map[string]any{
		"workspace_id": ws.ID,
	})
	s.Require().False(res.Result.IsError)

	var bal model.AccountBalanceResponse
	err = json.Unmarshal([]byte(res.Result.Content[0].Text), &bal)
	s.Require().NoError(err)
	s.True(bal.Truncated)
	s.Len(bal.Accounts, 50)
	s.Contains(bal.Hint, "account")
	s.True(bal.IsBalanced)
	s.True(bal.Totals.Debit.Equal(dec(6000)))
	s.True(bal.Totals.Credit.Equal(dec(6000)))
}

func (s *McpFinApiSuite) Test_Mcp_PeriodOverview_EqualsReportSummary() {
	ws := s.ensureWorkspace("MCP Fin Summary WS", s.User.ID)
	pat := s.EnsurePAT(s.User.ID)
	client := testutil.GetEntCli()

	s.Request(fmt.Sprintf("/workspaces/%d/fin/accounts/seed", ws.ID), fiber.MethodPost, model.SeedAccountsRequest{
		Template: "freelancer",
		Lang:     "en",
	})
	s.Created()

	ctx := base.WithWorkspaceID(s.Context, ws.ID)
	accountRepo := repository.NewAccountRepository(client)
	bankAcc, err := accountRepo.FindByCode(ctx, "1111")
	s.Require().NoError(err)
	revAcc, err := accountRepo.FindByCode(ctx, "4140")
	s.Require().NoError(err)

	now := time.Now()
	pID := s.ensurePeriod(ws.ID, "Test Period", time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC), time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.UTC), now.Year())

	trx, err := client.Transaction.Create().
		SetWorkspaceID(ws.ID).
		SetNumber("TRX-TEST-SUMMARY").
		SetTransactionDate(now).
		SetCurrency("IDR").
		SetExchangeRate(dec(1)).
		SetSource("MANUAL").
		Save(ctx)
	s.Require().NoError(err)

	e, err := client.Entry.Create().
		SetWorkspaceID(ws.ID).
		SetTransactionID(trx.ID).
		SetPeriodID(pID).
		SetNumber("JE-TEST-SUMMARY").
		SetEntryDate(now).
		SetStatus("POSTED").
		SetPostedAt(now).
		SetPostedBy(s.User.ID).
		Save(ctx)
	s.Require().NoError(err)

	amt := dec(7500000)
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
		SetAccountID(revAcc.ID).
		SetDirection("CREDIT").
		SetAmount(amt).
		SetBaseAmount(amt).
		SetPosition(1).
		Save(ctx)
	s.Require().NoError(err)

	// Call GET /workspaces/{id}/fin/reports/summary?months=6
	s.Request(fmt.Sprintf("/workspaces/%d/fin/reports/summary?months=6", ws.ID), fiber.MethodGet, nil)
	s.OK()
	var sumRes httpx.WebResponse[model.FinancialSummaryResponse]
	err = json.NewDecoder(s.HttpResponse.Body).Decode(&sumRes)
	s.Require().NoError(err)

	// Find the current month figures from Summary
	curMonthStr := now.Format("2006-01")
	var expectedRev, expectedExp, expectedNet decimal.Decimal
	for _, m := range sumRes.Data.Monthly {
		if m.Month == curMonthStr {
			expectedRev = m.Revenue
			expectedExp = m.Expense
			expectedNet = m.Revenue.Sub(m.Expense)
			break
		}
	}

	// Call fin_period_overview via /mcp
	res := s.callTool(pat, "fin_period_overview", map[string]any{
		"workspace_id": ws.ID,
		"from_month":   curMonthStr,
		"to_month":     curMonthStr,
	})
	s.Require().False(res.Result.IsError)

	var overview model.PeriodOverviewResponse
	err = json.Unmarshal([]byte(res.Result.Content[0].Text), &overview)
	s.Require().NoError(err)
	s.Equal(curMonthStr, overview.From)
	s.Equal(curMonthStr, overview.To)
	s.True(overview.Revenue.Equal(expectedRev), "revenue mismatch: expected %s, got %s", expectedRev, overview.Revenue)
	s.True(overview.Expense.Equal(expectedExp), "expense mismatch: expected %s, got %s", expectedExp, overview.Expense)
	s.True(overview.Net.Equal(expectedNet), "net mismatch: expected %s, got %s", expectedNet, overview.Net)
}

func (s *McpFinApiSuite) Test_Mcp_RevokedToken_Rejected() {
	pat := s.EnsurePAT(s.User.ID)

	// Revoke the token directly in DB
	client := testutil.GetEntCli()
	tokens, err := client.PersonalAccessToken.Query().Where(personalaccesstoken.UserIDEQ(s.User.ID)).All(s.Context)
	s.Require().NoError(err)
	for _, tok := range tokens {
		_, err := client.PersonalAccessToken.UpdateOneID(tok.ID).SetRevokedAt(time.Now()).Save(s.Context)
		s.Require().NoError(err)
	}

	s.SetHeader("Accept", "application/json, text/event-stream")
	s.RequestWithToken("/mcp", fiber.MethodPost, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "fin_period_overview",
			"arguments": map[string]any{
				"workspace_id": 999,
			},
		},
	}, pat)
	s.AssertStatus(fiber.StatusUnauthorized)

	errResp := s.GetErrorResponse()
	s.Equal("UNAUTHORIZED", errResp.Error.Code)
	s.Equal("invalid token", errResp.Error.Message)
}

func TestMcpFinApiSuite(t *testing.T) {
	suite.Run(t, new(McpFinApiSuite))
}
