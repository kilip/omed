package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/core"
	"github.com/kilip/omed/finance/internal/model"
)

type UserRepository interface {
	GetById(ctx context.Context, id uuid.UUID) (*model.UserSnapshot, error)
	Create(ctx context.Context, user model.UserSnapshot) error
	Update(ctx context.Context, user model.UserSnapshot) error
	Delete(ctx context.Context, id uuid.UUID) error
	Upsert(ctx context.Context, user model.UserSnapshot) error
}

type WorkspaceRepository interface {
	GetById(ctx context.Context, id uuid.UUID) (*model.WorkspaceSnapshot, error)
	Create(ctx context.Context, ws model.WorkspaceSnapshot) error
	Update(ctx context.Context, ws model.WorkspaceSnapshot) error
	Delete(ctx context.Context, id uuid.UUID) error
	Upsert(ctx context.Context, ws model.WorkspaceSnapshot) error
}

type UserSnapshotService struct {
	users     UserRepository
	works     WorkspaceRepository
	logger    *slog.Logger
	expiredAt time.Time
}

func NewUserSnapshotService(users UserRepository, works WorkspaceRepository, logger *slog.Logger) UserSnapshotService {
	expiredAt := time.Now().AddDate(0, 0, -7)

	return UserSnapshotService{
		users,
		works,
		logger,
		expiredAt,
	}
}

func (s UserSnapshotService) CheckSnapshot(ctx context.Context, user core.AuthenticatedUser) error {
	var errs error

	if err := s.checkUser(ctx, user); err != nil {
		errs = errors.Join(core.ErrUserSnapshot, err)
	}

	if err := s.checkWorkspace(ctx, user); err != nil {
		errs = errors.Join(core.ErrUserSnapshot, err, errs)
	}

	return errs
}

func (s UserSnapshotService) checkUser(ctx context.Context, user core.AuthenticatedUser) error {
	snapshot, err := s.users.GetById(ctx, user.ID)
	if err != nil {
		if core.IsItemNotFound(err) {
			return s.users.Create(ctx, model.UserSnapshot{
				ID:     user.ID,
				Name:   user.Name,
				Avatar: user.Avatar,
			})
		}
		return err
	}

	if snapshot.SyncedAt.Before(s.expiredAt) {
		snapshot.SyncedAt = time.Now()
		return s.users.Update(ctx, *snapshot)
	}

	return nil
}

func (s UserSnapshotService) checkWorkspace(ctx context.Context, user core.AuthenticatedUser) error {
	snapshot := model.WorkspaceSnapshot{
		ID:   user.WorkspaceID,
		Name: user.WorkspaceName,
	}

	ws, err := s.works.GetById(ctx, snapshot.ID)
	if err != nil {
		if core.IsItemNotFound(err) {
			return s.works.Create(ctx, snapshot)
		}

		return err
	}

	if ws.SyncedAt.Before(s.expiredAt) {
		snapshot.SyncedAt = time.Now()
		return s.works.Update(ctx, snapshot)
	}

	return nil
}
