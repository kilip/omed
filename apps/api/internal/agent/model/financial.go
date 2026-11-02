package model

// FinancialStatus represents the availability of the financial analysis agent.
type FinancialStatus struct {
	Enabled   bool    `json:"enabled" example:"true"`
	Available bool    `json:"available" example:"true"`
	Reason    *string `json:"reason,omitempty" example:"NO_PROVIDER"`
}

// AnalyzeRequest holds the raw file data and metadata for financial analysis.
type AnalyzeRequest struct {
	Flow     string `json:"flow" example:"EXPENSE"`
	FileName string `json:"fileName" example:"receipt.pdf"`
	MIME     string `json:"mime" example:"application/pdf"`
	Data     []byte `json:"data"`
}

// FinancialWarning represents a warning produced during financial analysis validation.
type FinancialWarning struct {
	Code      string `json:"code" example:"CATEGORY_UNRESOLVED"`
	ItemIndex *int   `json:"itemIndex,omitempty" example:"0"`
}

// SuggestedItem represents a single suggested transaction line item.
type SuggestedItem struct {
	AccountID   *int64 `json:"accountId,omitempty" example:"40130"`
	Description string `json:"description" example:"Office supplies"`
	Quantity    string `json:"quantity" example:"1"`
	UnitPrice   string `json:"unitPrice" example:"100000.00"`
}

// FinancialSuggestion represents the validated suggestion returned from financial document analysis.
type FinancialSuggestion struct {
	DocumentType     string             `json:"documentType" example:"receipt"`
	TransactionDate  *string            `json:"transactionDate,omitempty" example:"2026-01-01"`
	Currency         *string            `json:"currency,omitempty" example:"IDR"`
	ExchangeRate     *string            `json:"exchangeRate,omitempty" example:"1.00000000"`
	Reference        *string            `json:"reference,omitempty" example:"INV-001"`
	Description      *string            `json:"description,omitempty" example:"Office supplies purchase"`
	DocumentTotal    *string            `json:"documentTotal,omitempty" example:"100000.00"`
	PaymentAccountID *int64             `json:"paymentAccountId,omitempty" example:"1"`
	Items            []SuggestedItem    `json:"items"`
	Warnings         []FinancialWarning `json:"warnings"`
}
