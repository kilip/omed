// Package domain defines the domain models and business errors for the finance module.
package domain

import "time"

const (
	// DecimalStyleDot formats decimals with a dot (e.g. 1,234.56).
	DecimalStyleDot = "dot"
	// DecimalStyleComma formats decimals with a comma (e.g. 1.234,56).
	DecimalStyleComma = "comma"
)

// FinSetting represents workspace-level financial configuration.
type FinSetting struct {
	ID                        int64     `json:"id"`
	WorkspaceID               int64     `json:"workspaceId"`
	Onboarding                bool      `json:"onboarding"`
	DecimalStyle              string    `json:"decimalStyle"`
	FXGainAccountID           *int64    `json:"fxGainAccountId,omitempty"`
	FXLossAccountID           *int64    `json:"fxLossAccountId,omitempty"`
	FXUnrealizedGainAccountID *int64    `json:"fxUnrealizedGainAccountId,omitempty"`
	FXUnrealizedLossAccountID *int64    `json:"fxUnrealizedLossAccountId,omitempty"`
	FXRoundingAccountID       *int64    `json:"fxRoundingAccountId,omitempty"`
	CreatedAt                 time.Time `json:"createdAt"`
	UpdatedAt                 time.Time `json:"updatedAt"`
}
