package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/account"
	"github.com/kilip/omed/api/ent/period"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

var (
	testPNGBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15c4\x00\x00\x00\nIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01\r\n-\xb4\x00\x00\x00\x00IEND\xaeB`\x82")
	testPDFBytes = []byte("%PDF-1.4\n1 0 obj\n<<\n/Type /Catalog\n>>\nendobj\ntrailer\n<<\n/Root 1 0 R\n>>\n%%EOF")
)

type FinancialIntegrationSuite struct {
	testutil.ApiTestSuite[model.FinancialSuggestion]
	fakeServer    *httptest.Server
	fakeCallCount atomic.Int32
	chatResponse  string
}

func TestFinancialIntegrationSuite(t *testing.T) {
	suite.Run(t, new(FinancialIntegrationSuite))
}

func (s *FinancialIntegrationSuite) SetupSuite() {
	s.fakeServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/models" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "gpt-4o"},
				},
			})
			return
		}

		s.fakeCallCount.Add(1)

		respText := s.chatResponse
		if respText == "" {
			respText = `{"documentType":"receipt","transactionDate":"2026-01-01","paymentAccountId":null,"currency":"IDR","exchangeRate":"1.0","reference":"REF-1","description":"Test","documentTotal":"100000","items":[]}`
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1700000000,
			"model":   "gpt-4o",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": respText,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 20,
				"total_tokens":      30,
			},
		})
	}))

	s.ApiTestSuite.SetupSuite()
}

func (s *FinancialIntegrationSuite) TearDownSuite() {
	if s.fakeServer != nil {
		s.fakeServer.Close()
	}
}

func (s *FinancialIntegrationSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *FinancialIntegrationSuite) ensureAccount(wsID int64, code, name, accType, normalBalance string, isActive, isPostable bool) int64 {
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

func (s *FinancialIntegrationSuite) createProviderAndEnable(wsID int64) int64 {
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", wsID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "OpenAI Test Provider",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "gpt-4o",
		APIKey:  "sk-test-secret-key-financial",
	})
	s.Created()
	prov := testutil.DecodeOther[model.ProviderResponse](s).Data
	s.Require().Positive(prov.ID)

	enabled := true
	s.Request(fmt.Sprintf("/workspaces/%d/agent/settings", wsID), fiber.MethodPatch, model.UpdateSettingRequest{
		FinancialEnabled:    &enabled,
		FinancialProviderID: &prov.ID,
	})
	s.OK()
	return prov.ID
}

func (s *FinancialIntegrationSuite) ensurePeriod(wsID int64, name string, start, end time.Time, year int, status string) int64 {
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

func (s *FinancialIntegrationSuite) requestAnalyze(wsID int64, filename string, content []byte, flow string) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if filename != "" {
		part, err := w.CreateFormFile("file", filename)
		s.Require().NoError(err)
		if len(content) > 0 {
			_, err = part.Write(content)
			s.Require().NoError(err)
		}
	}
	if flow != "" {
		s.Require().NoError(w.WriteField("flow", flow))
	}
	s.Require().NoError(w.Close())

	req := httptest.NewRequest(fiber.MethodPost, fmt.Sprintf("/workspaces/%d/agent/financial/analyze", wsID), &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if s.SessionToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.SessionToken)
	}
	for _, c := range s.Cookies {
		req.AddCookie(c)
	}

	resp, err := testutil.GetState().FiberApp.Test(req)
	s.Require().NoError(err)
	s.HttpResponse = resp
}

func (s *FinancialIntegrationSuite) TestFinancial_Status_Disabled() {
	ws := s.ensureWorkspace("Status Disabled WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/financial/status", ws.ID), fiber.MethodGet, nil)
	s.OK()

	st := testutil.DecodeOther[model.FinancialStatus](s).Data
	s.False(st.Enabled)
	s.False(st.Available)
	s.Require().NotNil(st.Reason)
	s.Equal("DISABLED", *st.Reason)
}

func (s *FinancialIntegrationSuite) TestFinancial_Status_Enabled() {
	ws := s.ensureWorkspace("Status Enabled WS", s.User.ID)
	s.createProviderAndEnable(ws.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/financial/status", ws.ID), fiber.MethodGet, nil)
	s.OK()

	st := testutil.DecodeOther[model.FinancialStatus](s).Data
	s.True(st.Enabled)
	s.True(st.Available)
	s.Nil(st.Reason)
}

func (s *FinancialIntegrationSuite) TestFinancial_Analyze_Success() {
	ws := s.ensureWorkspace("Analyze Success WS", s.User.ID)
	s.createProviderAndEnable(ws.ID)

	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", "ASSET", "DEBIT", true, true)
	expenseAcc := s.ensureAccount(ws.ID, "5001", "Office Supplies", "EXPENSE", "DEBIT", true, true)

	s.chatResponse = fmt.Sprintf(`{
		"documentType": "receipt",
		"transactionDate": "2026-01-15",
		"paymentAccountId": %d,
		"currency": "IDR",
		"exchangeRate": "1.0",
		"reference": "INV-2026-001",
		"description": "Office paper purchase",
		"documentTotal": "150000.00",
		"items": [
			{
				"accountId": %d,
				"description": "Paper A4",
				"quantity": "2",
				"unitPrice": "75000.00"
			}
		]
	}`, cashAcc, expenseAcc)

	s.requestAnalyze(ws.ID, "receipt.png", testPNGBytes, "EXPENSE")
	s.OK()

	res := s.GetResponse().Data
	s.Equal("receipt", res.DocumentType)
	s.Require().NotNil(res.PaymentAccountID)
	s.Equal(cashAcc, *res.PaymentAccountID)
	s.Require().Len(res.Items, 1)
	s.Require().NotNil(res.Items[0].AccountID)
	s.Equal(expenseAcc, *res.Items[0].AccountID)
	s.Equal("Paper A4", res.Items[0].Description)
}

func (s *FinancialIntegrationSuite) TestFinancial_Analyze_ReceiptUnitsAndThousandSeparators_Issue55() {
	ws := s.ensureWorkspace("Analyze Issue 55 WS", s.User.ID)
	s.createProviderAndEnable(ws.ID)

	s.ensurePeriod(ws.ID, "Jan 2026", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, "OPEN")

	cashAcc := s.ensureAccount(ws.ID, "1001", "Cash", "ASSET", "DEBIT", true, true)
	expenseAcc := s.ensureAccount(ws.ID, "5001", "Food & Beverage", "EXPENSE", "DEBIT", true, true)

	s.chatResponse = fmt.Sprintf(`{
		"documentType": "receipt",
		"transactionDate": "2026-01-15",
		"paymentAccountId": %d,
		"currency": "IDR",
		"exchangeRate": "1.0",
		"reference": "RCP-55",
		"description": "Grocery purchase",
		"documentTotal": "Rp 70.000",
		"items": [
			{
				"accountId": %d,
				"description": "Indomie Goreng",
				"quantity": "1 lusin",
				"unitPrice": "36,000"
			},
			{
				"accountId": %d,
				"description": "Fruit Tea Apple",
				"quantity": "1 500 ml",
				"unitPrice": "7,000"
			},
			{
				"accountId": %d,
				"description": "Belfood Sosis Bakar",
				"quantity": "1",
				"unitPrice": "27,000"
			}
		]
	}`, cashAcc, expenseAcc, expenseAcc, expenseAcc)

	s.requestAnalyze(ws.ID, "receipt.png", testPNGBytes, "EXPENSE")
	s.OK()

	res := s.GetResponse().Data
	s.Equal("receipt", res.DocumentType)
	s.Require().NotNil(res.PaymentAccountID)
	s.Equal(cashAcc, *res.PaymentAccountID)

	s.Require().Len(res.Warnings, 0, "No ITEM_INVALID warnings expected")
	s.Require().Len(res.Items, 3)

	s.Equal("Indomie Goreng (lusin)", res.Items[0].Description)
	s.Equal("1", res.Items[0].Quantity)
	s.Equal("36000", res.Items[0].UnitPrice)

	s.Equal("Fruit Tea Apple 500 ml", res.Items[1].Description)
	s.Equal("1", res.Items[1].Quantity)
	s.Equal("7000", res.Items[1].UnitPrice)

	s.Equal("Belfood Sosis Bakar", res.Items[2].Description)
	s.Equal("1", res.Items[2].Quantity)
	s.Equal("27000", res.Items[2].UnitPrice)

	s.Require().NotNil(res.DocumentTotal)
	s.Equal("70000", *res.DocumentTotal)
}

func (s *FinancialIntegrationSuite) TestFinancial_Analyze_StripsForeignAccountID() {
	ws1 := s.ensureWorkspace("Analyze WS 1", s.User.ID)
	s.createProviderAndEnable(ws1.ID)

	ws2 := s.ensureWorkspace("Analyze WS 2", s.User.ID)
	foreignAcc := s.ensureAccount(ws2.ID, "9999", "Foreign Account", "ASSET", "DEBIT", true, true)

	// Upstream returns account ID from ws2 (or non-existent account)
	s.chatResponse = fmt.Sprintf(`{
		"documentType": "receipt",
		"transactionDate": "2026-01-15",
		"paymentAccountId": %d,
		"currency": "IDR",
		"exchangeRate": "1.0",
		"reference": "INV-FOREIGN",
		"description": "Test",
		"documentTotal": "50000.00",
		"items": [
			{
				"accountId": %d,
				"description": "Foreign item",
				"quantity": "1",
				"unitPrice": "50000.00"
			}
		]
	}`, foreignAcc, foreignAcc)

	s.requestAnalyze(ws1.ID, "receipt.png", testPNGBytes, "EXPENSE")
	s.OK()

	res := s.GetResponse().Data
	// Foreign account ID must be stripped (nil)
	s.Nil(res.PaymentAccountID)
	s.Require().Len(res.Items, 1)
	s.Nil(res.Items[0].AccountID)

	// Warnings should be emitted
	warningCodes := make([]string, len(res.Warnings))
	for i, w := range res.Warnings {
		warningCodes[i] = w.Code
	}
	s.Contains(warningCodes, "PAYMENT_ACCOUNT_UNRESOLVED")
	s.Contains(warningCodes, "CATEGORY_UNRESOLVED")
}

func (s *FinancialIntegrationSuite) TestFinancial_Analyze_DisabledWorkspace_409() {
	ws := s.ensureWorkspace("Analyze Disabled WS", s.User.ID)

	s.fakeCallCount.Store(0)
	s.requestAnalyze(ws.ID, "receipt.png", testPNGBytes, "EXPENSE")

	s.AssertStatus(fiber.StatusConflict) // domain.ErrFinancialAgentDisabled maps to 409
	s.Equal(int32(0), s.fakeCallCount.Load())
}

func (s *FinancialIntegrationSuite) TestFinancial_Authz_FinancialUserWithoutFinCreate_403() {
	ws := s.ensureWorkspace("Authz FinUser WS", s.User.ID)
	s.createProviderAndEnable(ws.ID)

	// Create member with agent:financial-user only (no fin:accountant)
	member := s.AsUser("finuser@example.com", "Financial User Only")
	s.AddMember(ws.ID, member.ID, domain.RoleFinancialUser)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/financial/status", ws.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusForbidden)

	s.requestAnalyze(ws.ID, "receipt.png", testPNGBytes, "EXPENSE")
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *FinancialIntegrationSuite) TestFinancial_Authz_FinAccountantPlusFinancialUser_200And403OnSettings() {
	ws := s.ensureWorkspace("Authz Accountant FinUser WS", s.User.ID)
	s.createProviderAndEnable(ws.ID)

	// Create member with both fin:accountant and agent:financial-user
	member := s.AsUser("accountant_finuser@example.com", "Accountant & Fin User")
	s.AddMember(ws.ID, member.ID, "fin:accountant", domain.RoleFinancialUser)

	// Status should be allowed (200)
	s.Request(fmt.Sprintf("/workspaces/%d/agent/financial/status", ws.ID), fiber.MethodGet, nil)
	s.OK()

	// Settings should be forbidden (403)
	s.Request(fmt.Sprintf("/workspaces/%d/agent/settings", ws.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *FinancialIntegrationSuite) TestFinancial_Analyze_PDFWithOpenAICompat_422() {
	ws := s.ensureWorkspace("Analyze PDF WS", s.User.ID)
	s.createProviderAndEnable(ws.ID)

	// OpenAI Compat provider doesn't support PDF (Documents=false) -> 422 UNSUPPORTED_DOCUMENT
	s.requestAnalyze(ws.ID, "invoice.pdf", testPDFBytes, "EXPENSE")
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *FinancialIntegrationSuite) TestFinancial_CrossWorkspaceProviderIsolation() {
	ws1 := s.ensureWorkspace("Isolation WS 1", s.User.ID)
	prov1ID := s.createProviderAndEnable(ws1.ID)

	ws2 := s.ensureWorkspace("Isolation WS 2", s.User.ID)

	// WS 2 tries to use WS 1's provider
	enabled := true
	s.Request(fmt.Sprintf("/workspaces/%d/agent/settings", ws2.ID), fiber.MethodPatch, model.UpdateSettingRequest{
		FinancialEnabled:    &enabled,
		FinancialProviderID: &prov1ID,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}
