// Package domain defines the domain models and business rules for the finance module.
package domain

import (
	"fmt"
	"time"

	"github.com/kilip/omed/api/internal/base"
)

// Standard accounting period statuses.
const (
	PeriodStatusOpen   = "OPEN"
	PeriodStatusClosed = "CLOSED"
)

// ValidPeriodStatuses lists all valid period statuses.
var ValidPeriodStatuses = map[string]bool{
	PeriodStatusOpen:   true,
	PeriodStatusClosed: true,
}

// Period represents an Accounting Period entity in the domain.
type Period struct {
	ID          int64          `json:"id"`
	WorkspaceID int64          `json:"workspaceId"`
	Name        string         `json:"name"`
	StartDate   time.Time      `json:"startDate"`
	EndDate     time.Time      `json:"endDate"`
	FiscalYear  int            `json:"fiscalYear"`
	Status      string         `json:"status"`
	Description string         `json:"description,omitempty"`
	AuditMeta   base.AuditMeta `json:"auditMeta"`
}

// PeriodRevaluationReference returns standard reference string for a period's revaluation transaction.
func PeriodRevaluationReference(periodID int64) string {
	return fmt.Sprintf("REVAL-PERIOD-%d", periodID)
}


