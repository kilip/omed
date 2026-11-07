package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/account"
	"github.com/kilip/omed/api/ent/entry"
	"github.com/kilip/omed/api/ent/exchangerate"
	"github.com/kilip/omed/api/ent/period"
	"github.com/kilip/omed/api/ent/posting"
	"github.com/kilip/omed/api/ent/transaction"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/model"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/testutil"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

type ChatIntegrationSuite struct {
	testutil.ApiTestSuite[model.ConversationResponse]
	fakeServer *httptest.Server
	mu         sync.Mutex
	chatSteps  []func(w http.ResponseWriter, r *http.Request)
	stepIndex  int
}

func TestChatIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ChatIntegrationSuite))
}

func (s *ChatIntegrationSuite) SetupSuite() {
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

		s.mu.Lock()
		defer s.mu.Unlock()

		if s.stepIndex < len(s.chatSteps) {
			handler := s.chatSteps[s.stepIndex]
			s.stepIndex++
			handler(w, r)
			return
		}

		// Default fallback assistant reply
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-default",
			"object":  "chat.completion",
			"created": 1700000000,
			"model":   "gpt-4o",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "Hello! I am your bookkeeping assistant.",
					},
					"finish_reason": "stop",
				},
			},
		})
	}))

	s.ApiTestSuite.SetupSuite()
}

func (s *ChatIntegrationSuite) TearDownSuite() {
	if s.fakeServer != nil {
		s.fakeServer.Close()
	}
}

func (s *ChatIntegrationSuite) SetupTest() {
	s.ApiTestSuite.SetupTest()
	s.mu.Lock()
	s.chatSteps = nil
	s.stepIndex = 0
	s.mu.Unlock()
}

func (s *ChatIntegrationSuite) setChatSteps(steps ...func(w http.ResponseWriter, r *http.Request)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chatSteps = steps
	s.stepIndex = 0
}

func (s *ChatIntegrationSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *ChatIntegrationSuite) ensureAccount(wsID int64, code, name, accType, normalBalance string, isActive, isPostable bool) int64 {
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

func (s *ChatIntegrationSuite) ensurePeriod(wsID int64, name string, start, end time.Time, year int, status string) int64 {
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

func (s *ChatIntegrationSuite) ensureExchangeRate(wsID int64, from, to string, rate decimal.Decimal, effDate time.Time) int64 {
	client := testutil.GetEntCli()
	xr, err := client.ExchangeRate.Create().
		SetWorkspaceID(wsID).
		SetFromCurrency(from).
		SetToCurrency(to).
		SetRate(rate).
		SetEffectiveDate(effDate).
		SetSource(exchangerate.SourceMANUAL).
		Save(base.WithWorkspaceID(s.Context, wsID))
	s.Require().NoError(err)
	return xr.ID
}

func (s *ChatIntegrationSuite) ensurePostedTransaction(
	wsID int64,
	periodID int64,
	trxNum, entryNum string,
	trxDate time.Time,
	desc, ref, curr string,
	postings ...struct {
		accountID int64
		direction string
		amount    decimal.Decimal
		quantity  *decimal.Decimal
		unitPrice *decimal.Decimal
		desc      string
	},
) int64 {
	client := testutil.GetEntCli()
	wsCtx := base.WithWorkspaceID(s.Context, wsID)

	tx, err := client.Transaction.Create().
		SetWorkspaceID(wsID).
		SetNumber(trxNum).
		SetTransactionDate(trxDate).
		SetDescription(desc).
		SetReference(ref).
		SetCurrency(curr).
		SetExchangeRate(decimal.NewFromInt(1)).
		SetSource(transaction.SourceMANUAL).
		Save(wsCtx)
	s.Require().NoError(err)

	e, err := client.Entry.Create().
		SetWorkspaceID(wsID).
		SetTransactionID(tx.ID).
		SetPeriodID(periodID).
		SetNumber(entryNum).
		SetEntryDate(trxDate).
		SetDescription(desc).
		SetStatus(entry.StatusPOSTED).
		Save(wsCtx)
	s.Require().NoError(err)

	for i, p := range postings {
		pb := client.Posting.Create().
			SetWorkspaceID(wsID).
			SetEntryID(e.ID).
			SetAccountID(p.accountID).
			SetDirection(posting.Direction(p.direction)).
			SetAmount(p.amount).
			SetBaseAmount(p.amount).
			SetPosition(i).
			SetDescription(p.desc)
		if p.quantity != nil {
			pb.SetQuantity(*p.quantity)
		}
		if p.unitPrice != nil {
			pb.SetUnitPrice(*p.unitPrice)
		}
		_, err = pb.Save(wsCtx)
		s.Require().NoError(err)
	}

	return tx.ID
}

type financialChatSeedFixture struct {
	wsID              int64
	periodID          int64
	kasAccID          int64
	bankKaltimIDR     int64
	bankKaltimUSD     int64
	makananMinumanAcc int64
	pendapatanAcc     int64
}

func (s *ChatIntegrationSuite) seedFinancialWorkspace(name string) financialChatSeedFixture {
	ws := s.ensureWorkspace(name, s.User.ID)
	s.createProviderAndEnable(ws.ID)

	// Seed period covering entire year 2026
	periodID := s.ensurePeriod(
		ws.ID,
		"FY2026",
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC),
		2026,
		"OPEN",
	)

	// Seed accounts
	kasAcc := s.ensureAccount(ws.ID, "1001", "Kas", "ASSET", "DEBIT", true, true)
	bankKaltimIDR := s.ensureAccount(ws.ID, "1002", "Bank Kaltim IDR", "ASSET", "DEBIT", true, true)
	bankKaltimUSD := s.ensureAccount(ws.ID, "1003", "Bank Kaltim USD", "ASSET", "DEBIT", true, true)
	makananMinumanAcc := s.ensureAccount(ws.ID, "5001", "Beban Makanan & Minuman", "EXPENSE", "DEBIT", true, true)
	pendapatanAcc := s.ensureAccount(ws.ID, "4001", "Pendapatan", "REVENUE", "CREDIT", true, true)

	// Seed exchange rate USD -> IDR (rate: 16250)
	s.ensureExchangeRate(
		ws.ID,
		"USD",
		"IDR",
		decimal.NewFromInt(16250),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	)

	// 1. Initial capital / sales income into Kas & Bank Kaltim:
	// Kas += 5,000,000, Pendapatan += 5,000,000
	s.ensurePostedTransaction(
		ws.ID, periodID,
		"TRX-001", "JE-001",
		time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
		"Pendapatan jasa tunai", "REF-001", "IDR",
		struct {
			accountID int64
			direction string
			amount    decimal.Decimal
			quantity  *decimal.Decimal
			unitPrice *decimal.Decimal
			desc      string
		}{accountID: kasAcc, direction: "DEBIT", amount: decimal.NewFromInt(5000000), desc: "Kas masuk"},
		struct {
			accountID int64
			direction string
			amount    decimal.Decimal
			quantity  *decimal.Decimal
			unitPrice *decimal.Decimal
			desc      string
		}{accountID: pendapatanAcc, direction: "CREDIT", amount: decimal.NewFromInt(5000000), desc: "Pendapatan jasa"},
	)

	// Bank Kaltim IDR += 10,000,000, Pendapatan += 10,000,000
	s.ensurePostedTransaction(
		ws.ID, periodID,
		"TRX-002", "JE-002",
		time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC),
		"Pendapatan transfer Bank Kaltim IDR", "REF-002", "IDR",
		struct {
			accountID int64
			direction string
			amount    decimal.Decimal
			quantity  *decimal.Decimal
			unitPrice *decimal.Decimal
			desc      string
		}{accountID: bankKaltimIDR, direction: "DEBIT", amount: decimal.NewFromInt(10000000), desc: "Bank Kaltim masuk"},
		struct {
			accountID int64
			direction string
			amount    decimal.Decimal
			quantity  *decimal.Decimal
			unitPrice *decimal.Decimal
			desc      string
		}{accountID: pendapatanAcc, direction: "CREDIT", amount: decimal.NewFromInt(10000000), desc: "Pendapatan jasa transfer"},
	)

	// 2. Expense: Belanja makanan dan minuman paid from Kas
	// Kas -= 250,000, Beban Makanan & Minuman += 250,000 (Expense cashflow structure)
	qty1 := decimal.NewFromInt(5)
	unitPrice1 := decimal.NewFromInt(50000)
	s.ensurePostedTransaction(
		ws.ID, periodID,
		"TRX-003", "JE-003",
		time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		"Belanja makan siang rapat", "REF-MEAL-01", "IDR",
		struct {
			accountID int64
			direction string
			amount    decimal.Decimal
			quantity  *decimal.Decimal
			unitPrice *decimal.Decimal
			desc      string
		}{accountID: kasAcc, direction: "CREDIT", amount: decimal.NewFromInt(250000), desc: "Pembayaran kas"},
		struct {
			accountID int64
			direction string
			amount    decimal.Decimal
			quantity  *decimal.Decimal
			unitPrice *decimal.Decimal
			desc      string
		}{accountID: makananMinumanAcc, direction: "DEBIT", amount: decimal.NewFromInt(250000), quantity: &qty1, unitPrice: &unitPrice1, desc: "Nasi kotak rapat"},
	)

	// Another expense: Snack sore from Kas -= 150,000
	qty2 := decimal.NewFromInt(10)
	unitPrice2 := decimal.NewFromInt(15000)
	s.ensurePostedTransaction(
		ws.ID, periodID,
		"TRX-004", "JE-004",
		time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC),
		"Beli snack dan kopi kantor", "REF-MEAL-02", "IDR",
		struct {
			accountID int64
			direction string
			amount    decimal.Decimal
			quantity  *decimal.Decimal
			unitPrice *decimal.Decimal
			desc      string
		}{accountID: kasAcc, direction: "CREDIT", amount: decimal.NewFromInt(150000), desc: "Pembayaran kas"},
		struct {
			accountID int64
			direction string
			amount    decimal.Decimal
			quantity  *decimal.Decimal
			unitPrice *decimal.Decimal
			desc      string
		}{accountID: makananMinumanAcc, direction: "DEBIT", amount: decimal.NewFromInt(150000), quantity: &qty2, unitPrice: &unitPrice2, desc: "Kopi dan kue"},
	)

	return financialChatSeedFixture{
		wsID:              ws.ID,
		periodID:          periodID,
		kasAccID:          kasAcc,
		bankKaltimIDR:     bankKaltimIDR,
		bankKaltimUSD:     bankKaltimUSD,
		makananMinumanAcc: makananMinumanAcc,
		pendapatanAcc:     pendapatanAcc,
	}
}

func (s *ChatIntegrationSuite) createProviderAndEnable(wsID int64) int64 {
	s.Request(fmt.Sprintf("/workspaces/%d/agent/providers", wsID), fiber.MethodPost, model.CreateProviderRequest{
		Name:    "OpenAI Test Provider",
		Kind:    "openai_compat",
		BaseURL: s.fakeServer.URL,
		Model:   "gpt-4o",
		APIKey:  "sk-test-secret-key-chat",
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

func (s *ChatIntegrationSuite) sendMessageMultipart(wsID, convID int64, content string, files ...struct {
	filename string
	data     []byte
}) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	if content != "" {
		s.Require().NoError(w.WriteField("content", content))
	}

	for _, f := range files {
		part, err := w.CreateFormFile("file", f.filename)
		s.Require().NoError(err)
		_, err = part.Write(f.data)
		s.Require().NoError(err)
	}
	s.Require().NoError(w.Close())

	req := httptest.NewRequest(fiber.MethodPost, fmt.Sprintf("/workspaces/%d/agent/chat/conversations/%d/messages", wsID, convID), &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if s.SessionToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.SessionToken)
	}
	for _, c := range s.Cookies {
		req.AddCookie(c)
	}
	for k, v := range s.Headers {
		req.Header[k] = v
	}

	targetApp := testutil.GetState().FiberApp
	resp, err := targetApp.Test(req)
	s.Require().NoError(err)
	s.HttpResponse = resp
}

func (s *ChatIntegrationSuite) TestChat_CreateAndListConversation() {
	ws := s.ensureWorkspace("Chat Create List WS", s.User.ID)
	s.createProviderAndEnable(ws.ID)

	// Create conversation
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", ws.ID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "Test Receipt Conversation",
	})
	s.Created()
	conv := testutil.DecodeOther[model.ConversationResponse](s).Data
	s.Require().Positive(conv.ID)
	s.Equal("Test Receipt Conversation", conv.Title)

	// List conversations
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", ws.ID), fiber.MethodGet, nil)
	s.OK()
	list := testutil.DecodeOther[model.ListConversationsResponse](s).Data
	s.True(list.Total >= 1)
	found := false
	for _, c := range list.Items {
		if c.ID == conv.ID {
			found = true
			break
		}
	}
	s.True(found, "created conversation should be in list items")
}

func (s *ChatIntegrationSuite) TestChat_SendMessageWithAttachment_AndDownload() {
	ws := s.ensureWorkspace("Chat Send Message WS", s.User.ID)
	s.createProviderAndEnable(ws.ID)

	// Create accounts
	bankAcc := s.ensureAccount(ws.ID, "1101", "Cash on Hand", "ASSET", "DEBIT", true, true)
	expAcc := s.ensureAccount(ws.ID, "5101", "Office Supplies Expense", "EXPENSE", "DEBIT", true, true)

	// Create conversation
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", ws.ID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "Receipt Analysis Conversation",
	})
	s.Created()
	conv := testutil.DecodeOther[model.ConversationResponse](s).Data

	// Set LLM mock responses:
	// Turn 1: tool call propose_draft
	// Turn 2: final assistant message
	propJSON := fmt.Sprintf(`{
		"kind": "expense",
		"paymentAccountId": %d,
		"transactionDate": "2026-01-15",
		"description": "Office Supplies Receipt",
		"currency": "IDR",
		"totalAmount": "70000.00",
		"items": [
			{
				"accountId": %d,
				"description": "Paper & Pens",
				"quantity": "1",
				"unitPrice": "70000.00",
				"amount": "70000.00"
			}
		]
	}`, bankAcc, expAcc)

	s.setChatSteps(
		func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-tool",
				"object":  "chat.completion",
				"created": 1700000000,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role": "assistant",
							"tool_calls": []map[string]any{
								{
									"id":   "call_prop_1",
									"type": "function",
									"function": map[string]any{
										"name":      "propose_draft",
										"arguments": propJSON,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
			})
		},
		func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-final",
				"object":  "chat.completion",
				"created": 1700000001,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role":    "assistant",
							"content": "I analyzed the receipt and proposed an expense draft of Rp 70.000.",
						},
						"finish_reason": "stop",
					},
				},
			})
		},
	)

	// Send message with PNG attachment
	fakePNG := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15c4\x00\x00\x00\nIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01\r\n-\xb4\x00\x00\x00\x00IEND\xaeB`\x82")
	s.sendMessageMultipart(ws.ID, conv.ID, "Please analyze this receipt", struct {
		filename string
		data     []byte
	}{filename: "receipt.png", data: fakePNG})
	s.OK()

	msgResp := testutil.DecodeOther[model.SendMessageResponse](s).Data
	s.Equal("proposed", msgResp.AssistantMessage.DraftStatus)
	s.Require().NotNil(msgResp.DraftProposal)
	s.Equal(model.DraftKindExpense, msgResp.DraftProposal.Kind)
	s.True(decimal.RequireFromString("70000").Equal(msgResp.DraftProposal.TotalAmount))
	s.Require().Len(msgResp.UserMessage.Attachments, 1)

	attID := msgResp.UserMessage.Attachments[0].ID

	// Download attachment stream
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/attachments/%d/download", ws.ID, attID), fiber.MethodGet, nil)
	s.OK()
	bodyBytes, err := io.ReadAll(s.HttpResponse.Body)
	s.Require().NoError(err)
	s.Equal(fakePNG, bodyBytes)
}

func (s *ChatIntegrationSuite) TestChat_SendMessageAttachment_SizeLimit() {
	ws := s.ensureWorkspace("Chat Attachment Limit WS", s.User.ID)
	s.createProviderAndEnable(ws.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", ws.ID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "Size Limit Test Conversation",
	})
	s.Created()
	conv := testutil.DecodeOther[model.ConversationResponse](s).Data

	// Test oversized attachment (> 500KB) -> returns 413 FILE_TOO_LARGE
	oversizedBytes := make([]byte, 500*1024+1)
	copy(oversizedBytes, []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15c4\x00\x00\x00\nIDATx\x9cc\x00\x01\x00\x00\x05\x00\x01\r\n-\xb4\x00\x00\x00\x00IEND\xaeB`\x82"))

	s.sendMessageMultipart(ws.ID, conv.ID, "Upload oversized file", struct {
		filename string
		data     []byte
	}{filename: "too_large.png", data: oversizedBytes})

	s.Equal(fiber.StatusRequestEntityTooLarge, s.HttpResponse.StatusCode)
	errResp := s.GetErrorResponse()
	s.Equal("FILE_TOO_LARGE", errResp.Error.Code)
	s.Equal("file size exceeds maximum allowed limit", errResp.Error.Message)
}

func (s *ChatIntegrationSuite) TestChat_ConfirmDraft_AndIdempotency() {
	ws := s.ensureWorkspace("Chat Confirm Draft WS", s.User.ID)
	s.createProviderAndEnable(ws.ID)

	bankAcc := s.ensureAccount(ws.ID, "1101", "Cash on Hand", "ASSET", "DEBIT", true, true)
	expAcc := s.ensureAccount(ws.ID, "5101", "Office Supplies Expense", "EXPENSE", "DEBIT", true, true)
	s.ensurePeriod(ws.ID, "FY2026-Jan", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC), 2026, "OPEN")

	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", ws.ID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "Confirm Draft Conversation",
	})
	s.Created()
	conv := testutil.DecodeOther[model.ConversationResponse](s).Data

	propJSON := fmt.Sprintf(`{
		"kind": "expense",
		"paymentAccountId": %d,
		"transactionDate": "2026-01-15",
		"description": "Office Supplies Receipt",
		"currency": "IDR",
		"totalAmount": "70000.00",
		"items": [
			{
				"accountId": %d,
				"description": "Paper & Pens",
				"quantity": "1",
				"unitPrice": "70000.00",
				"amount": "70000.00"
			}
		]
	}`, bankAcc, expAcc)

	s.setChatSteps(
		func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-tool",
				"object":  "chat.completion",
				"created": 1700000000,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role": "assistant",
							"tool_calls": []map[string]any{
								{
									"id":   "call_prop_1",
									"type": "function",
									"function": map[string]any{
										"name":      "propose_draft",
										"arguments": propJSON,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
			})
		},
		func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-final",
				"object":  "chat.completion",
				"created": 1700000001,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role":    "assistant",
							"content": "Proposed expense draft.",
						},
						"finish_reason": "stop",
					},
				},
			})
		},
	)

	s.sendMessageMultipart(ws.ID, conv.ID, "Record receipt")
	s.OK()
	msgResp := testutil.DecodeOther[model.SendMessageResponse](s).Data
	asstMsgID := msgResp.AssistantMessage.ID

	// 1. First Confirm: returns 200 with transactionId
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/messages/%d/confirm", ws.ID, asstMsgID), fiber.MethodPost, model.ConfirmDraftRequest{})
	s.OK()
	confirmResp := testutil.DecodeOther[model.ConfirmDraftResponse](s).Data
	s.Require().Positive(confirmResp.TransactionID)

	// 2. Second Confirm: conflict 409 DRAFT_ALREADY_CONFIRMED
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/messages/%d/confirm", ws.ID, asstMsgID), fiber.MethodPost, model.ConfirmDraftRequest{})
	s.AssertStatus(fiber.StatusConflict)

	// Check conversation detail: message draftStatus should now be confirmed
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations/%d", ws.ID, conv.ID), fiber.MethodGet, nil)
	s.OK()
	detail := testutil.DecodeOther[model.ConversationDetailResponse](s).Data
	var confirmedMsg *model.MessageResponse
	for _, m := range detail.Messages {
		if m.ID == asstMsgID {
			confirmedMsg = &m
			break
		}
	}
	s.Require().NotNil(confirmedMsg)
	s.Equal("confirmed", confirmedMsg.DraftStatus)
	s.Require().NotNil(confirmedMsg.ConfirmedTransactionID)
	s.Equal(confirmResp.TransactionID, *confirmedMsg.ConfirmedTransactionID)
}

func (s *ChatIntegrationSuite) TestChat_DiscardDraft() {
	ws := s.ensureWorkspace("Chat Discard Draft WS", s.User.ID)
	s.createProviderAndEnable(ws.ID)

	bankAcc := s.ensureAccount(ws.ID, "1101", "Cash on Hand", "ASSET", "DEBIT", true, true)
	expAcc := s.ensureAccount(ws.ID, "5101", "Office Supplies Expense", "EXPENSE", "DEBIT", true, true)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", ws.ID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "Discard Draft Conversation",
	})
	s.Created()
	conv := testutil.DecodeOther[model.ConversationResponse](s).Data

	propJSON := fmt.Sprintf(`{
		"kind": "expense",
		"paymentAccountId": %d,
		"transactionDate": "2026-01-15",
		"description": "Discardable Expense",
		"currency": "IDR",
		"totalAmount": "70000.00",
		"items": [
			{
				"accountId": %d,
				"description": "Paper",
				"quantity": "1",
				"unitPrice": "70000.00",
				"amount": "70000.00"
			}
		]
	}`, bankAcc, expAcc)

	s.setChatSteps(
		func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-tool",
				"object":  "chat.completion",
				"created": 1700000000,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role": "assistant",
							"tool_calls": []map[string]any{
								{
									"id":   "call_prop_1",
									"type": "function",
									"function": map[string]any{
										"name":      "propose_draft",
										"arguments": propJSON,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
			})
		},
		func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-final",
				"object":  "chat.completion",
				"created": 1700000001,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role":    "assistant",
							"content": "Proposed draft to discard.",
						},
						"finish_reason": "stop",
					},
				},
			})
		},
	)

	s.sendMessageMultipart(ws.ID, conv.ID, "Record discardable")
	s.OK()
	msgResp := testutil.DecodeOther[model.SendMessageResponse](s).Data
	asstMsgID := msgResp.AssistantMessage.ID

	// 1. Discard draft: returns 200
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/messages/%d/discard", ws.ID, asstMsgID), fiber.MethodPost, nil)
	s.OK()

	// 2. Discarding again -> 409 DRAFT_ALREADY_DISCARDED
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/messages/%d/discard", ws.ID, asstMsgID), fiber.MethodPost, nil)
	s.AssertStatus(fiber.StatusConflict)

	// 3. Confirming after discard -> 409 DRAFT_ALREADY_DISCARDED
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/messages/%d/confirm", ws.ID, asstMsgID), fiber.MethodPost, model.ConfirmDraftRequest{})
	s.AssertStatus(fiber.StatusConflict)
}

func (s *ChatIntegrationSuite) TestChat_DeleteConversation() {
	ws := s.ensureWorkspace("Chat Delete WS", s.User.ID)
	s.createProviderAndEnable(ws.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", ws.ID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "Conversation to Delete",
	})
	s.Created()
	conv := testutil.DecodeOther[model.ConversationResponse](s).Data

	// Delete conversation
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations/%d", ws.ID, conv.ID), fiber.MethodDelete, nil)
	s.OK()

	// Subsequent Get returns 404
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations/%d", ws.ID, conv.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *ChatIntegrationSuite) TestChat_UpdateConversation_AndAutoTitle() {
	ws := s.ensureWorkspace("Chat Rename & AutoTitle WS", s.User.ID)
	s.createProviderAndEnable(ws.ID)

	// 1. Create a default-titled conversation
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", ws.ID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "New conversation",
	})
	s.Created()
	conv := testutil.DecodeOther[model.ConversationResponse](s).Data
	s.Require().Positive(conv.ID)

	// 2. Manual rename via PATCH
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations/%d", ws.ID, conv.ID), fiber.MethodPatch, model.UpdateConversationRequest{
		Title: "Renamed by User",
	})
	s.OK()
	updatedConv := testutil.DecodeOther[model.ConversationResponse](s).Data
	s.Equal("Renamed by User", updatedConv.Title)

	// 3. Rename with blank/whitespace returns 422
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations/%d", ws.ID, conv.ID), fiber.MethodPatch, model.UpdateConversationRequest{
		Title: "   ",
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)

	// 4. Create another conversation with default title to test auto-titling in SendMessage
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", ws.ID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "Percakapan baru",
	})
	s.Created()
	autoConv := testutil.DecodeOther[model.ConversationResponse](s).Data

	s.setChatSteps(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"role":    "assistant",
						"content": "I understand your query.",
					},
					"finish_reason": "stop",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})

	msgContent := "Please help analyze our latest expenses"
	expectedSnippet := string([]rune(msgContent)[:35]) + "..."
	s.sendMessageMultipart(ws.ID, autoConv.ID, msgContent)
	s.OK()
	msgResp := testutil.DecodeOther[model.SendMessageResponse](s).Data
	s.Equal(expectedSnippet, msgResp.ConversationTitle)

	// Verify conversation detail reflects auto-generated title
	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations/%d", ws.ID, autoConv.ID), fiber.MethodGet, nil)
	s.OK()
	convDetail := testutil.DecodeOther[model.ConversationDetailResponse](s).Data
	s.Equal(expectedSnippet, convDetail.Title)
}

func (s *ChatIntegrationSuite) TestChat_Authz_PermissionsDenial_403() {
	ws := s.ensureWorkspace("Chat Authz Denial WS", s.User.ID)
	s.createProviderAndEnable(ws.ID)

	// User lacking fin:transaction create permission (agent:financial-user only)
	userFinOnly := s.AsUser("finonly@example.com", "Financial User Only")
	s.AddMember(ws.ID, userFinOnly.ID, domain.RoleFinancialUser)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", ws.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusForbidden)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", ws.ID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "Forbidden Conv",
	})
	s.AssertStatus(fiber.StatusForbidden)

	// User lacking agent:financial permission (fin:accountant only)
	userAcctOnly := s.AsUser("acctonly@example.com", "Accountant Only")
	s.AddMember(ws.ID, userAcctOnly.ID, "fin:accountant")

	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", ws.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusForbidden)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", ws.ID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "Forbidden Conv 2",
	})
	s.AssertStatus(fiber.StatusForbidden)

	// User with BOTH permissions (fin:accountant + agent:financial-user)
	userBoth := s.AsUser("both@example.com", "Both Perms User")
	s.AddMember(ws.ID, userBoth.ID, "fin:accountant", domain.RoleFinancialUser)

	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", ws.ID), fiber.MethodGet, nil)
	s.OK()

	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", ws.ID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "Allowed Conv",
	})
	s.Created()
}

func (s *ChatIntegrationSuite) TestChat_Financial_CashAndBankBalanceQuery() {
	fix := s.seedFinancialWorkspace("Fin Chat Bank Balance WS")

	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", fix.wsID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "Bank Balance Conversation",
	})
	s.Created()
	conv := testutil.DecodeOther[model.ConversationResponse](s).Data

	var capturedToolArgs map[string]any
	var capturedToolCallID string

	s.setChatSteps(
		// Step 1: Model invokes get_account_balances for "Bank Kaltim"
		func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-tool-bal",
				"object":  "chat.completion",
				"created": 1700000000,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role": "assistant",
							"tool_calls": []map[string]any{
								{
									"id":   "call_bal_1",
									"type": "function",
									"function": map[string]any{
										"name":      "get_account_balances",
										"arguments": `{"account":"Bank Kaltim"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
			})
		},
		// Step 2: Model consumes tool result and provides final answer
		func(w http.ResponseWriter, r *http.Request) {
			var reqBody struct {
				Messages []struct {
					Role       string `json:"role"`
					Content    string `json:"content"`
					ToolCallID string `json:"tool_call_id"`
				} `json:"messages"`
			}
			_ = json.NewDecoder(r.Body).Decode(&reqBody)
			for _, m := range reqBody.Messages {
				if m.Role == "tool" && m.ToolCallID == "call_bal_1" {
					capturedToolCallID = m.ToolCallID
					_ = json.Unmarshal([]byte(m.Content), &capturedToolArgs)
				}
			}

			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-final-bal",
				"object":  "chat.completion",
				"created": 1700000001,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role":    "assistant",
							"content": "Saldo Bank Kaltim IDR saat ini adalah Rp 10.000.000 dan Bank Kaltim USD adalah Rp 0.",
						},
						"finish_reason": "stop",
					},
				},
			})
		},
	)

	s.sendMessageMultipart(fix.wsID, conv.ID, "Berapa saldo Bank Kaltim?")
	s.OK()
	msgResp := testutil.DecodeOther[model.SendMessageResponse](s).Data
	s.Equal("call_bal_1", capturedToolCallID)
	s.Require().NotNil(capturedToolArgs)
	s.Equal("10000000", capturedToolArgs["matchedNetTotal"])
	s.Contains(msgResp.AssistantMessage.Content, "10.000.000")
}

func (s *ChatIntegrationSuite) TestChat_Financial_ProfitLossInquiry() {
	fix := s.seedFinancialWorkspace("Fin Chat Profit Loss WS")

	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", fix.wsID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "Profit Loss Conversation",
	})
	s.Created()
	conv := testutil.DecodeOther[model.ConversationResponse](s).Data

	var capturedToolArgs map[string]any

	s.setChatSteps(
		// Step 1: Model invokes get_profit_loss_breakdown
		func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-tool-pl",
				"object":  "chat.completion",
				"created": 1700000000,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role": "assistant",
							"tool_calls": []map[string]any{
								{
									"id":   "call_pl_1",
									"type": "function",
									"function": map[string]any{
										"name":      "get_profit_loss_breakdown",
										"arguments": `{"dateFrom":"2026-01-01","dateTo":"2026-01-31"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
			})
		},
		// Step 2: Model gives grounded answer
		func(w http.ResponseWriter, r *http.Request) {
			var reqBody struct {
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
			}
			_ = json.NewDecoder(r.Body).Decode(&reqBody)
			for _, m := range reqBody.Messages {
				if m.Role == "tool" {
					_ = json.Unmarshal([]byte(m.Content), &capturedToolArgs)
				}
			}

			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-final-pl",
				"object":  "chat.completion",
				"created": 1700000001,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role":    "assistant",
							"content": "Laba bersih bulan Januari 2026 adalah Rp 14.600.000 (Pendapatan Rp 15.000.000 dan Beban Rp 400.000).",
						},
						"finish_reason": "stop",
					},
				},
			})
		},
	)

	s.sendMessageMultipart(fix.wsID, conv.ID, "Bagaimana rugi laba bulan ini?")
	s.OK()
	msgResp := testutil.DecodeOther[model.SendMessageResponse](s).Data
	s.Require().NotNil(capturedToolArgs)
	s.Equal("15000000", capturedToolArgs["totalRevenue"])
	s.Equal("400000", capturedToolArgs["totalExpense"])
	s.Equal("14600000", capturedToolArgs["netProfit"])
	s.Contains(msgResp.AssistantMessage.Content, "14.600.000")
}

func (s *ChatIntegrationSuite) TestChat_Financial_CategoryExpenseCashflowSearch() {
	fix := s.seedFinancialWorkspace("Fin Chat Category Search WS")

	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", fix.wsID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "Category Expense Conversation",
	})
	s.Created()
	conv := testutil.DecodeOther[model.ConversationResponse](s).Data

	var capturedToolArgs map[string]any

	s.setChatSteps(
		// Step 1: Model invokes search_cashflows with categoryAccount
		func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-tool-cat",
				"object":  "chat.completion",
				"created": 1700000000,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role": "assistant",
							"tool_calls": []map[string]any{
								{
									"id":   "call_cat_1",
									"type": "function",
									"function": map[string]any{
										"name":      "search_cashflows",
										"arguments": `{"type":"expense","categoryAccount":"Beban Makanan & Minuman"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
			})
		},
		// Step 2: Final response
		func(w http.ResponseWriter, r *http.Request) {
			var reqBody struct {
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
			}
			_ = json.NewDecoder(r.Body).Decode(&reqBody)
			for _, m := range reqBody.Messages {
				if m.Role == "tool" {
					_ = json.Unmarshal([]byte(m.Content), &capturedToolArgs)
				}
			}

			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-final-cat",
				"object":  "chat.completion",
				"created": 1700000001,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role":    "assistant",
							"content": "Total belanja makanan dan minuman adalah Rp 400.000 dari 2 transaksi.",
						},
						"finish_reason": "stop",
					},
				},
			})
		},
	)

	s.sendMessageMultipart(fix.wsID, conv.ID, "Berapa total belanja makanan dan minuman?")
	s.OK()
	msgResp := testutil.DecodeOther[model.SendMessageResponse](s).Data
	s.Require().NotNil(capturedToolArgs)
	s.Equal("400000", capturedToolArgs["returnedBaseAmountSum"])
	items, ok := capturedToolArgs["items"].([]any)
	s.Require().True(ok)
	s.Len(items, 2)
	s.Contains(msgResp.AssistantMessage.Content, "400.000")
}

func (s *ChatIntegrationSuite) TestChat_Financial_AccountLedgerMutasiInquiry() {
	fix := s.seedFinancialWorkspace("Fin Chat Ledger WS")

	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", fix.wsID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "Account Ledger Conversation",
	})
	s.Created()
	conv := testutil.DecodeOther[model.ConversationResponse](s).Data

	var capturedToolArgs map[string]any

	s.setChatSteps(
		// Step 1: Model invokes get_account_ledger
		func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-tool-ledger",
				"object":  "chat.completion",
				"created": 1700000000,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role": "assistant",
							"tool_calls": []map[string]any{
								{
									"id":   "call_ledger_1",
									"type": "function",
									"function": map[string]any{
										"name":      "get_account_ledger",
										"arguments": `{"account":"Kas"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
			})
		},
		// Step 2: Assistant responds with mutasi entries and running balance
		func(w http.ResponseWriter, r *http.Request) {
			var reqBody struct {
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
			}
			_ = json.NewDecoder(r.Body).Decode(&reqBody)
			for _, m := range reqBody.Messages {
				if m.Role == "tool" {
					_ = json.Unmarshal([]byte(m.Content), &capturedToolArgs)
				}
			}

			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-final-ledger",
				"object":  "chat.completion",
				"created": 1700000001,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role":    "assistant",
							"content": "Berikut mutasi akun Kas: terdapat 3 transaksi dengan saldo akhir Rp 4.600.000.",
						},
						"finish_reason": "stop",
					},
				},
			})
		},
	)

	s.sendMessageMultipart(fix.wsID, conv.ID, "Tampilkan mutasi Kas")
	s.OK()
	msgResp := testutil.DecodeOther[model.SendMessageResponse](s).Data
	s.Require().NotNil(capturedToolArgs)
	s.Equal("Kas", capturedToolArgs["accountName"])
	s.Equal(float64(3), capturedToolArgs["totalCount"])
	postingsList, ok := capturedToolArgs["postings"].([]any)
	s.Require().True(ok)
	s.Len(postingsList, 3)
	lastPosting := postingsList[len(postingsList)-1].(map[string]any)
	s.Equal("4600000", lastPosting["runningBalance"])
	s.Contains(msgResp.AssistantMessage.Content, "4.600.000")
}

func (s *ChatIntegrationSuite) TestChat_Financial_ExchangeRateInquiry() {
	fix := s.seedFinancialWorkspace("Fin Chat Exchange Rate WS")

	s.Request(fmt.Sprintf("/workspaces/%d/agent/chat/conversations", fix.wsID), fiber.MethodPost, model.CreateConversationRequest{
		Title: "Exchange Rate Conversation",
	})
	s.Created()
	conv := testutil.DecodeOther[model.ConversationResponse](s).Data

	var capturedToolArgs map[string]any

	s.setChatSteps(
		// Step 1: Model invokes lookup_exchange_rate
		func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-tool-fx",
				"object":  "chat.completion",
				"created": 1700000000,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role": "assistant",
							"tool_calls": []map[string]any{
								{
									"id":   "call_fx_1",
									"type": "function",
									"function": map[string]any{
										"name":      "lookup_exchange_rate",
										"arguments": `{"fromCurrency":"USD","toCurrency":"IDR"}`,
									},
								},
							},
						},
						"finish_reason": "tool_calls",
					},
				},
			})
		},
		// Step 2: Assistant responds with effective rate
		func(w http.ResponseWriter, r *http.Request) {
			var reqBody struct {
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
			}
			_ = json.NewDecoder(r.Body).Decode(&reqBody)
			for _, m := range reqBody.Messages {
				if m.Role == "tool" {
					_ = json.Unmarshal([]byte(m.Content), &capturedToolArgs)
				}
			}

			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "chatcmpl-final-fx",
				"object":  "chat.completion",
				"created": 1700000001,
				"model":   "gpt-4o",
				"choices": []map[string]any{
					{
						"index": 0,
						"message": map[string]any{
							"role":    "assistant",
							"content": "Kurs efektif USD ke IDR adalah 16.250.",
						},
						"finish_reason": "stop",
					},
				},
			})
		},
	)

	s.sendMessageMultipart(fix.wsID, conv.ID, "Berapa kurs USD ke IDR hari ini?")
	s.OK()
	msgResp := testutil.DecodeOther[model.SendMessageResponse](s).Data
	s.Require().NotNil(capturedToolArgs)
	s.Equal("USD", capturedToolArgs["fromCurrency"])
	s.Equal("IDR", capturedToolArgs["toCurrency"])
	s.Equal("16250", capturedToolArgs["rate"])
	s.Contains(msgResp.AssistantMessage.Content, "16.250")
}
