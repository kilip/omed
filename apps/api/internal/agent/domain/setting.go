package domain

// Setting represents workspace agent configuration in the agent domain.
type Setting struct {
	WorkspaceID         int64  `json:"workspaceId"`
	FinancialEnabled    bool   `json:"financialEnabled"`
	FinancialProviderID *int64 `json:"financialProviderId,omitempty"`
}
