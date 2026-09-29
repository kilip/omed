package model

import (
	"time"

	"github.com/google/uuid"
)

type UserSnapshot struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Avatar   string    `json:"avatar,omitempty"`
	SyncedAt time.Time `json:"syncedAt,omitempty"`
}

type WorkspaceSnapshot struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	SyncedAt time.Time `json:"syncedAt,omitempty"`
}
