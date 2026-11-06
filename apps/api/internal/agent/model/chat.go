package model

import (
	"encoding/json"
	"io"

	"github.com/kilip/omed/api/ent/agentchatmessage"
	"github.com/kilip/omed/api/internal/base"
	"github.com/shopspring/decimal"
)

// DraftKind defines the type of draft proposal.
type DraftKind string

const (
	DraftKindExpense DraftKind = "expense"
	DraftKindIncome  DraftKind = "income"
	DraftKindJournal DraftKind = "journal"
)

// DraftItem represents an individual line item in a cash flow draft proposal (expense or income).
type DraftItem struct {
	AccountID   int64           `json:"accountId" example:"40130"`
	Description string          `json:"description,omitempty" example:"Office supplies"`
	Quantity    decimal.Decimal `json:"quantity" swaggertype:"string" example:"1"`
	UnitPrice   decimal.Decimal `json:"unitPrice" swaggertype:"string" example:"70000.00"`
	Amount      decimal.Decimal `json:"amount" swaggertype:"string" example:"70000.00"`
}

// DraftJournalEntry represents a single debit or credit line in a journal draft proposal.
type DraftJournalEntry struct {
	AccountID   int64           `json:"accountId" example:"1101"`
	Direction   string          `json:"direction" example:"DEBIT"` // "DEBIT" or "CREDIT"
	Amount      decimal.Decimal `json:"amount" swaggertype:"string" example:"2000000.00"`
	Description string          `json:"description,omitempty" example:"Cash on Hand"`
}

// DraftProposal represents a unified proposal for a financial draft (expense, income, or journal).
type DraftProposal struct {
	Kind             DraftKind           `json:"kind" example:"expense"` // "expense", "income", "journal"
	TransactionDate  string              `json:"transactionDate" example:"2026-01-01"`
	Description      string              `json:"description" example:"Office supplies purchase"`
	Reference        string              `json:"reference,omitempty" example:"INV-001"`
	Currency         string              `json:"currency" example:"IDR"`
	ExchangeRate     *decimal.Decimal    `json:"exchangeRate,omitempty" swaggertype:"string" example:"1.00000000"`
	PaymentAccountID *int64              `json:"paymentAccountId,omitempty" example:"1"`
	PeriodID         *int64              `json:"periodId,omitempty" example:"1"`
	TotalAmount      decimal.Decimal     `json:"totalAmount" swaggertype:"string" example:"70000.00"`
	Items            []DraftItem         `json:"items,omitempty"`
	Entries          []DraftJournalEntry `json:"entries,omitempty"`
}

// ExpenseDraftProposal represents an expense draft with multiple line items.
type ExpenseDraftProposal struct {
	PaymentAccountID int64            `json:"paymentAccountId" example:"1"`
	TransactionDate  string           `json:"transactionDate" example:"2026-01-01"`
	Reference        string           `json:"reference,omitempty" example:"INV-001"`
	Description      string           `json:"description" example:"Office supplies"`
	Currency         string           `json:"currency" example:"IDR"`
	ExchangeRate     *decimal.Decimal `json:"exchangeRate,omitempty" swaggertype:"string" example:"1.00000000"`
	TotalAmount      decimal.Decimal  `json:"totalAmount" swaggertype:"string" example:"70000.00"`
	Items            []DraftItem      `json:"items"`
}

// IncomeDraftProposal represents an income draft with multiple line items.
type IncomeDraftProposal struct {
	PaymentAccountID int64            `json:"paymentAccountId" example:"1"`
	TransactionDate  string           `json:"transactionDate" example:"2026-01-01"`
	Reference        string           `json:"reference,omitempty" example:"INV-002"`
	Description      string           `json:"description" example:"Consulting revenue"`
	Currency         string           `json:"currency" example:"IDR"`
	ExchangeRate     *decimal.Decimal `json:"exchangeRate,omitempty" swaggertype:"string" example:"1.00000000"`
	TotalAmount      decimal.Decimal  `json:"totalAmount" swaggertype:"string" example:"1000000.00"`
	Items            []DraftItem      `json:"items"`
}

// JournalDraftProposal represents a balanced journal draft with Dr and Cr entries.
type JournalDraftProposal struct {
	PeriodID        *int64              `json:"periodId,omitempty" example:"1"`
	TransactionDate string              `json:"transactionDate" example:"2026-01-01"`
	Reference       string              `json:"reference,omitempty" example:"ATM-001"`
	Description     string              `json:"description" example:"ATM cash withdrawal"`
	Currency        string              `json:"currency" example:"IDR"`
	ExchangeRate    *decimal.Decimal    `json:"exchangeRate,omitempty" swaggertype:"string" example:"1.00000000"`
	TotalAmount     decimal.Decimal     `json:"totalAmount" swaggertype:"string" example:"2000000.00"`
	Entries         []DraftJournalEntry `json:"entries"`
}

// CreateConversationRequest defines payload to create a new chat conversation.
type CreateConversationRequest struct {
	Title string `json:"title,omitempty" example:"Receipt Analysis"`
}

// SendMessageRequest defines payload to post a message in a conversation.
type SendMessageRequest struct {
	Content string `json:"content" example:"Please analyze this receipt"`
}

// ConfirmDraftRequest defines payload to confirm a proposed draft.
type ConfirmDraftRequest struct {
	Proposal *DraftProposal `json:"proposal,omitempty"`
}

// AttachmentResponse represents the API view of a chat attachment.
type AttachmentResponse struct {
	ID             int64          `json:"id" example:"1"`
	ConversationID int64          `json:"conversationId" example:"1"`
	MessageID      *int64         `json:"messageId,omitempty" example:"1"`
	ObjectKey      string         `json:"objectKey" example:"workspaces/1/agent/chat/uuid.jpg"`
	Bucket         string         `json:"bucket" example:"omed-attachments"`
	MIME           string         `json:"mime" example:"image/jpeg"`
	Size           int64          `json:"size" example:"102400"`
	FileName       string         `json:"fileName" example:"receipt.jpg"`
	AuditMeta      base.AuditMeta `json:"auditMeta"`
}

// MessageResponse represents the API view of a chat message.
type MessageResponse struct {
	ID                     int64                `json:"id" example:"1"`
	ConversationID         int64                `json:"conversationId" example:"1"`
	Role                   string               `json:"role" example:"user"`
	Content                string               `json:"content" example:"Here is the receipt"`
	Parts                  json.RawMessage      `json:"parts,omitempty"`
	DraftStatus            string               `json:"draftStatus" example:"proposed"`
	ConfirmedTransactionID *int64               `json:"confirmedTransactionId,omitempty" example:"100"`
	Attachments            []AttachmentResponse `json:"attachments,omitempty"`
	AuditMeta              base.AuditMeta       `json:"auditMeta"`
}

// ConversationResponse represents the API view of a chat conversation.
type ConversationResponse struct {
	ID        int64             `json:"id" example:"1"`
	Title     string            `json:"title" example:"Receipt Analysis"`
	Messages  []MessageResponse `json:"messages,omitempty"`
	AuditMeta base.AuditMeta    `json:"auditMeta"`
}

// AppendMessageParams defines the parameters for repository AppendMessage.
type AppendMessageParams struct {
	ConversationID         int64
	Role                   agentchatmessage.Role
	Content                string
	Parts                  json.RawMessage
	DraftStatus            agentchatmessage.DraftStatus
	ConfirmedTransactionID *int64
}

// CreateAttachmentParams defines the parameters for repository CreateAttachment.
type CreateAttachmentParams struct {
	ConversationID int64
	MessageID      *int64
	ObjectKey      string
	Bucket         string
	MIME           string
	Size           int64
	FileName       string
}

// AttachmentUpload represents a file uploaded as part of a chat message.
type AttachmentUpload struct {
	FileName string    `json:"fileName"`
	MIME     string    `json:"mime"`
	Size     int64     `json:"size"`
	Data     []byte    `json:"-"`
	Reader   io.Reader `json:"-"`
}

// ReadData returns the binary content of the attachment, reading from Reader if Data is empty.
func (a *AttachmentUpload) ReadData() ([]byte, error) {
	if len(a.Data) > 0 {
		return a.Data, nil
	}
	if a.Reader != nil {
		data, err := io.ReadAll(a.Reader)
		if err != nil {
			return nil, err
		}
		a.Data = data
		if a.Size == 0 {
			a.Size = int64(len(data))
		}
		return data, nil
	}
	return nil, nil
}

// AttachmentDownload represents a downloaded attachment stream with metadata.
type AttachmentDownload struct {
	Attachment AttachmentResponse `json:"attachment"`
	FileName   string             `json:"fileName"`
	MIME       string             `json:"mime"`
	Size       int64              `json:"size"`
	Body       io.ReadCloser      `json:"-"`
}

// ListConversationsResponse represents the paginated response for listing conversations.
type ListConversationsResponse struct {
	Conversations []ConversationResponse `json:"conversations"`
	Items         []ConversationResponse `json:"items,omitempty"`
	Total         int                    `json:"total"`
	Page          int                    `json:"page"`
	Limit         int                    `json:"limit"`
}

// ConversationDetailResponse represents the detailed view of a conversation including its messages.
type ConversationDetailResponse struct {
	ConversationResponse
}

// SendMessageResponse represents the result of sending a message in a conversation.
type SendMessageResponse struct {
	UserMessage      MessageResponse `json:"userMessage"`
	AssistantMessage MessageResponse `json:"assistantMessage"`
	DraftProposal    *DraftProposal  `json:"draftProposal,omitempty"`
}
