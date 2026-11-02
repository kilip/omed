package model

// UpdateSettingRequest defines the payload to update workspace agent settings.
type UpdateSettingRequest struct {
	FinancialEnabled    *bool  `json:"financialEnabled" validate:"required" example:"true"`
	FinancialProviderID *int64 `json:"financialProviderId" example:"1"`
}

// SettingResponse represents the API response view of workspace agent settings.
type SettingResponse struct {
	FinancialEnabled    bool   `json:"financialEnabled" example:"true"`
	FinancialProviderID *int64 `json:"financialProviderId" example:"1"`
}
