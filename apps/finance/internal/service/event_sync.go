package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	TopicUser = "auth.user.v1"
	TopicTeam = "auth.team.v1"
)

type EventSync struct {
	users UserRepository      // interface yang sudah ada + Upsert/Delete
	works WorkspaceRepository // + Upsert
}

func NewEventSync(users UserRepository, works WorkspaceRepository) EventSync {
	return EventSync{
		users,
		works,
	}
}

func (s EventSync) HandleUser(ctx context.Context, rec *kgo.Record) error {
	id, err := uuid.Parse(string(rec.Key))
	if err != nil {
		return nil // pesan rusak: skip, jangan bikin stuck
	}
	if rec.Value == nil { // tombstone
		return s.users.Delete(ctx, id)
	}
	var p struct {
		Name  string  `json:"name"`
		Image *string `json:"image"`
	}
	if err := json.Unmarshal(rec.Value, &p); err != nil {
		return nil
	}
	avatar := ""
	if p.Image != nil {
		avatar = *p.Image
	}
	return s.users.Upsert(ctx, model.UserSnapshot{ID: id, Name: p.Name, Avatar: avatar})
}

func (s EventSync) HandleTeam(ctx context.Context, rec *kgo.Record) error {
	id, err := uuid.Parse(string(rec.Key))
	if err != nil {
		return nil
	}

	// delete workspace
	if rec.Value == nil {
		return s.works.Delete(ctx, id)
	}

	var p struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(rec.Value, &p); err != nil {
		return nil
	}
	return s.works.Upsert(ctx, model.WorkspaceSnapshot{ID: id, Name: p.Name})
}
