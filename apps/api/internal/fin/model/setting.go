package model

// FinSettingResponse represents workspace-level financial configuration.
type FinSettingResponse struct {
	Onboarding                bool   `json:"onboarding"`
	DecimalStyle              string `json:"decimalStyle" example:"comma"`
	BaseCurrency              string `json:"baseCurrency" example:"IDR"`
	FXGainAccountID           *int64 `json:"fxGainAccountId,omitempty" example:"7210"`
	FXLossAccountID           *int64 `json:"fxLossAccountId,omitempty" example:"7310"`
	FXUnrealizedGainAccountID *int64 `json:"fxUnrealizedGainAccountId,omitempty" example:"7220"`
	FXUnrealizedLossAccountID *int64 `json:"fxUnrealizedLossAccountId,omitempty" example:"7320"`
	FXRoundingAccountID       *int64 `json:"fxRoundingAccountId,omitempty" example:"7400"`
}

// UpdateFinSettingRequest represents the payload to update workspace finance settings.
type UpdateFinSettingRequest struct {
	Onboarding   *bool  `json:"onboarding"`
	DecimalStyle string `json:"decimalStyle" validate:"required,oneof=dot comma" example:"comma"`
}

// FXAccountsSettingResponse represents the configured foreign exchange accounts for a workspace.
type FXAccountsSettingResponse struct {
	FXGainAccountID           *int64 `json:"fxGainAccountId" example:"7210"`
	FXLossAccountID           *int64 `json:"fxLossAccountId" example:"7310"`
	FXUnrealizedGainAccountID *int64 `json:"fxUnrealizedGainAccountId" example:"7220"`
	FXUnrealizedLossAccountID *int64 `json:"fxUnrealizedLossAccountId" example:"7320"`
	FXRoundingAccountID       *int64 `json:"fxRoundingAccountId" example:"7400"`
}

// UpdateFXAccountsRequest represents the payload to update FX accounts configuration.
type UpdateFXAccountsRequest struct {
	FXGainAccountID           *int64 `json:"fxGainAccountId" validate:"omitempty"`
	FXLossAccountID           *int64 `json:"fxLossAccountId" validate:"omitempty"`
	FXUnrealizedGainAccountID *int64 `json:"fxUnrealizedGainAccountId" validate:"omitempty"`
	FXUnrealizedLossAccountID *int64 `json:"fxUnrealizedLossAccountId" validate:"omitempty"`
	FXRoundingAccountID       *int64 `json:"fxRoundingAccountId" validate:"omitempty"`
}
