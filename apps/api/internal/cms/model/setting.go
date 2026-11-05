package model

// CmsSettingResponse represents workspace CMS settings.
type CmsSettingResponse struct {
	DefaultLocale    string   `json:"defaultLocale"`
	SupportedLocales []string `json:"supportedLocales"`
}

// UpdateCmsSettingRequest defines the payload for updating workspace CMS settings.
type UpdateCmsSettingRequest struct {
	DefaultLocale    *string  `json:"defaultLocale,omitempty"`
	SupportedLocales []string `json:"supportedLocales,omitempty"`
}
