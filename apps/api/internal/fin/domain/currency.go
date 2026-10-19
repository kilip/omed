// Package domain defines the domain models and business errors for the finance module.
package domain

import "time"

// Currency represents global reference currency data conforming to ISO 4217.
type Currency struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Symbol    string    `json:"symbol"`
	Decimals  int       `json:"decimals"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
