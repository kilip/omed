package model

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type PeriodStatus string

const (
	PeriodStatusOpen   PeriodStatus = "open"
	PeriodStatusClosed PeriodStatus = "closed"
	PeriodStatusLocked PeriodStatus = "locked"
)

// Period is the response DTO for ent Period.
type Period struct {
	ID            uuid.UUID    `json:"id"`
	WorkspaceID   uuid.UUID    `json:"workspaceId"`
	StartDate     time.Time    `json:"startDate"`
	EndDate       time.Time    `json:"endDate"`
	Status        PeriodStatus `json:"status" enums:"open,closed,locked"`
	CreatedBy     uuid.UUID    `json:"createdBy"`
	CreatedByName string       `json:"createdByName"`
	CreatedAt     time.Time    `json:"createdAt"`
	UpdatedBy     uuid.UUID    `json:"updatedBy"`
	UpdatedByName string       `json:"updatedByName"`
	UpdatedAt     time.Time    `json:"updatedAt"`
}

func (r *Period) UnmarshalJSON(data []byte) error {
	type Alias Period
	var raw struct {
		Alias
		StartDateAlt *time.Time `json:"start_date"`
		EndDateAlt   *time.Time `json:"end_date"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*r = Period(raw.Alias)
	if r.StartDate.IsZero() && raw.StartDateAlt != nil {
		r.StartDate = *raw.StartDateAlt
	}
	if r.EndDate.IsZero() && raw.EndDateAlt != nil {
		r.EndDate = *raw.EndDateAlt
	}
	return nil
}

type CreatePeriodRequest struct {
	StartDate time.Time    `json:"startDate" validate:"required"`
	EndDate   time.Time    `json:"endDate" validate:"required"`
	Status    PeriodStatus `json:"status,omitempty" validate:"omitempty,oneof=open closed locked" enums:"open,closed,locked"`
}

func parseFlexibleTime(v any) (time.Time, error) {
	if v == nil {
		return time.Time{}, nil
	}
	switch val := v.(type) {
	case string:
		if val == "" {
			return time.Time{}, nil
		}
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			return t, nil
		}
		if t, err := time.Parse(time.RFC3339Nano, val); err == nil {
			return t, nil
		}
		if t, err := time.Parse("2006-01-02", val); err == nil {
			return t, nil
		}
		return time.Time{}, fmt.Errorf("invalid time format: %s", val)
	case time.Time:
		return val, nil
	default:
		return time.Time{}, fmt.Errorf("unexpected type for time: %T", v)
	}
}

func (r *CreatePeriodRequest) UnmarshalJSON(data []byte) error {
	var raw struct {
		StartDate    any          `json:"startDate"`
		StartDateAlt any          `json:"start_date"`
		EndDate      any          `json:"endDate"`
		EndDateAlt   any          `json:"end_date"`
		Status       PeriodStatus `json:"status"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.Status = raw.Status

	startVal := raw.StartDate
	if startVal == nil {
		startVal = raw.StartDateAlt
	}
	if startVal != nil {
		t, err := parseFlexibleTime(startVal)
		if err != nil {
			return err
		}
		r.StartDate = t
	}

	endVal := raw.EndDate
	if endVal == nil {
		endVal = raw.EndDateAlt
	}
	if endVal != nil {
		t, err := parseFlexibleTime(endVal)
		if err != nil {
			return err
		}
		r.EndDate = t
	}

	return nil
}

type UpdatePeriodRequest struct {
	Status PeriodStatus `json:"status" validate:"required,oneof=open closed locked" enums:"open,closed,locked"`
}

type ListPeriodRequest struct {
	Status PeriodStatus `query:"status" validate:"omitempty,oneof=open closed locked"`
	Date   string       `query:"date"`
}
