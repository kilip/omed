package model

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
