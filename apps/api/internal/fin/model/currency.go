// Package model provides Data Transfer Objects (DTOs) and request/response payloads for finance endpoints.
package model

// CurrencyResponse represents a currency returned by currency endpoints.
type CurrencyResponse struct {
	ID       int64  `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

// UpdateBaseCurrencyRequest defines the payload required to update workspace base currency.
type UpdateBaseCurrencyRequest struct {
	BaseCurrency string `json:"baseCurrency" validate:"required,len=3"`
}

// WorkspaceBaseCurrencyResponse represents workspace response with base currency setting.
type WorkspaceBaseCurrencyResponse struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	BaseCurrency string `json:"baseCurrency"`
}
