package core

import "time"

type ByUser struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
}

type AuditMeta struct {
	CreatedBy ByUser    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedBy ByUser    `json:"updatedBy"`
	UpdatedAt time.Time `json:"updatedAt"`
}
