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
	"github.com/kilip/omed/api/ent/period"
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
