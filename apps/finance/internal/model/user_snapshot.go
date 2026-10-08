package model

import (
	"time"

	"github.com/google/uuid"
)

type UserSnapshot struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Avatar    string    `json:"avatar,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	SyncedAt  time.Time `json:"syncedAt"`
}

type WorkspaceSnapshot struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	SyncedAt  time.Time `json:"syncedAt"`
}
