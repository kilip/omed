package base

import "time"

// ByUser identifies an author or modifier associated with an entity.
type ByUser struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

// AuditMeta encapsulates standard creation and modification audit timestamps and authors.
type AuditMeta struct {
	CreatedBy ByUser    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedBy ByUser    `json:"updatedBy"`
	UpdatedAt time.Time `json:"updatedAt"`
}
