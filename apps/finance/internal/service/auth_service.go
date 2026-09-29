package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/shared"
)

//go:generate go run go.uber.org/mock/mockgen@latest -source=auth_service.go -destination=mocks/mock_auth_service.go -package=mocks
type UserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.UserSnapshot, error)
	Create(ctx context.Context, user model.UserSnapshot) error
	Update(ctx context.Context, user model.UserSnapshot) error
}

type WorkspaceRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*model.WorkspaceSnapshot, error)
	Create(ctx context.Context, works model.WorkspaceSnapshot) error
	Update(ctx context.Context, works model.WorkspaceSnapshot) error
}

type AuthService struct {
	users     UserRepository
	works     WorkspaceRepository
	log       *slog.Logger
	expiredAt time.Time
}

func NewAuthService(users UserRepository, works WorkspaceRepository, log *slog.Logger) AuthService {
	expiredAt := time.Now().AddDate(0, 0, -7)
	return AuthService{
		users,
		works,
		log,
		expiredAt,
	}
}

func (s AuthService) checkUserSnapshot(ctx context.Context, auth shared.AuthenticatedUser) error {
	var snapshot model.UserSnapshot
	if err := shared.ToValue(auth, &snapshot); err != nil {
		return err
	}

	user, err := s.users.GetByID(ctx, auth.ID)
	if err != nil {
		if errors.Is(err, shared.ErrItemNotFound) {
			return s.users.Create(ctx, snapshot)
		}
	}

	if user.SyncedAt.Before(s.expiredAt) {
		snapshot.SyncedAt = time.Now()
		return s.users.Update(ctx, snapshot)
	}

	return nil
}

func (s AuthService) checkWorkspaceSnapshot(ctx context.Context, auth shared.AuthenticatedUser) error {
	var snapshot model.WorkspaceSnapshot
	if err := shared.ToValue(auth, &snapshot); err != nil {
		return err
	}

	ws, err := s.works.GetByID(ctx, auth.WorkspaceID)
	if err != nil {
		if errors.Is(err, shared.ErrItemNotFound) {
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

func (s AuthService) CheckSnapshot(ctx context.Context, auth shared.AuthenticatedUser) error {
	userError := s.checkUserSnapshot(ctx, auth)
	wsError := s.checkWorkspaceSnapshot(ctx, auth)
	return errors.Join(userError, wsError)
}
