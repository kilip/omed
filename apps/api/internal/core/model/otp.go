// Package model provides Data Transfer Objects (DTOs) and request/response payloads for the core module.
package model

// OTPRequest represents the request payload for requesting a login OTP code.
type OTPRequest struct {
	Email  string `json:"email" validate:"required,email"`
	Locale string `json:"locale,omitempty"`
}

// OTPRequestResponse represents the response returned after an OTP code is requested.
type OTPRequestResponse struct {
	Success            bool `json:"success"`
	CodeLength         int  `json:"codeLength"`
	TTLSeconds         int  `json:"ttlSeconds"`
	ResendAfterSeconds int  `json:"resendAfterSeconds"`
}

// OTPVerifyRequest represents the request payload for verifying a login OTP code.
type OTPVerifyRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required"`
}
