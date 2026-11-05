package domain

// Setting represents CMS configuration for a workspace.
type Setting struct {
	WorkspaceID      int64    `json:"workspaceId"`
	DefaultLocale    string   `json:"defaultLocale"`
	SupportedLocales []string `json:"supportedLocales"`
}
