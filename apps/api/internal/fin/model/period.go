// Package model provides Data Transfer Objects (DTOs) and request/response payloads for financial endpoints.
package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// Date represents a calendar date formatted as YYYY-MM-DD in JSON.
type Date time.Time

// Time converts Date to standard time.Time.
func (d Date) Time() time.Time {
	return time.Time(d)
}

// IsZero reports whether Date represents the zero time instant.
func (d Date) IsZero() bool {
	return time.Time(d).IsZero()
}

// UnmarshalJSON unmarshals a date string formatted as YYYY-MM-DD or RFC3339.
func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	if s == "" || s == "null" {
		return nil
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02T15:04:05Z07:00", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			*d = Date(t)
			return nil
		}
	}
	return fmt.Errorf("invalid date format: %s", s)
}

// MarshalJSON marshals Date to JSON as YYYY-MM-DD.
func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(time.Time(d).Format("2006-01-02"))
}

// PeriodResponse represents Accounting Period entity metadata returned by API endpoints.
type PeriodResponse struct {
	ID          int64          `json:"id"`
	WorkspaceID int64          `json:"workspaceId"`
	Name        string         `json:"name"`
	StartDate   Date           `json:"startDate" swaggertype:"string" example:"2026-01-01"`
	EndDate     Date           `json:"endDate" swaggertype:"string" example:"2026-01-31"`
	FiscalYear  int            `json:"fiscalYear" example:"2026"`
	Status      string         `json:"status" example:"OPEN"`
	Description string         `json:"description,omitempty"`
	AuditMeta   base.AuditMeta `json:"auditMeta"`
}

// CreatePeriodRequest defines the payload required to create a new accounting period.
type CreatePeriodRequest struct {
	Name        string `json:"name" validate:"required"`
	StartDate   Date   `json:"startDate" validate:"required" swaggertype:"string" example:"2026-01-01"`
	EndDate     Date   `json:"endDate" validate:"required" swaggertype:"string" example:"2026-01-31"`
	FiscalYear  int    `json:"fiscalYear" validate:"required,gt=0" example:"2026"`
	Description string `json:"description,omitempty"`
}

// UpdatePeriodRequest defines the payload required to update an existing accounting period.
type UpdatePeriodRequest struct {
	Name        string `json:"name" validate:"required"`
	StartDate   Date   `json:"startDate" validate:"required" swaggertype:"string" example:"2026-01-01"`
	EndDate     Date   `json:"endDate" validate:"required" swaggertype:"string" example:"2026-01-31"`
	FiscalYear  int    `json:"fiscalYear" validate:"required,gt=0" example:"2026"`
	Description string `json:"description,omitempty"`
}

// SearchPeriodRequest defines request payload for searching accounting periods in a workspace.
type SearchPeriodRequest struct {
	FiscalYear *int    `json:"fiscalYear,omitempty"`
	Status     *string `json:"status,omitempty"`
}
