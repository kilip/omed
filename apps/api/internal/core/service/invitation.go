package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/authz"
	"github.com/kilip/omed/api/internal/platform/mailx"
)

// InvitationRepository defines persistence requirements for workspace invitations.
type InvitationRepository interface {
	Create(ctx context.Context, wid int64, req model.CreateInvitationRequest, token string) (domain.Invitation, error)
	FindByID(ctx context.Context, wid, id int64) (domain.Invitation, error)
	FindByToken(ctx context.Context, token string) (domain.Invitation, error)
	Search(ctx context.Context, wid int64, req model.SearchInvitationRequest) ([]domain.Invitation, string, error)
	IncrementUses(ctx context.Context, id int64) error
	Revoke(ctx context.Context, wid, id int64) error
}

// InvitationService coordinates business logic for workspace invitations.
type InvitationService struct {
	invitations InvitationRepository
	workspaces  WorkspaceService
	members     MemberRepository
	users       UserService
	az          *authz.Authorizer
	mailer      mailx.Mailer
	webURL      string
}

// NewInvitationService constructs a new InvitationService.
func NewInvitationService(
	invitations InvitationRepository,
	workspaces WorkspaceService,
	members MemberRepository,
	users UserService,
	mailer mailx.Mailer,
	webURL string,
) InvitationService {
	return InvitationService{
		invitations: invitations,
		workspaces:  workspaces,
		members:     members,
		users:       users,
		mailer:      mailer,
		webURL:      webURL,
	}
}

// SetMailer configures the mailer and web base URL.
func (s *InvitationService) SetMailer(mailer mailx.Mailer, webURL string) {
	s.mailer = mailer
	s.webURL = webURL
}

// SetAuthorizer configures the authorizer used for role validation.
func (s *InvitationService) SetAuthorizer(az *authz.Authorizer) {
	s.az = az
}

// Create generates a new workspace invitation link.
func (s InvitationService) Create(ctx context.Context, wid int64, req model.CreateInvitationRequest) (domain.Invitation, error) {
	ws, err := s.workspaces.GetByID(ctx, wid)
	if err != nil {
		return domain.Invitation{}, err
	}

	if err := s.validateRoles(req.Roles); err != nil {
		return domain.Invitation{}, err
	}

	token, err := generateInvitationToken()
	if err != nil {
		return domain.Invitation{}, err
	}

	inv, err := s.invitations.Create(ctx, wid, req, token)
	if err != nil {
		return domain.Invitation{}, err
	}

	if req.Email != "" {
		s.sendInvitationEmail(ctx, inv, ws, req.Email)
	}

	return inv, nil
}

// Send triggers sending or resending an invitation email for an existing workspace invitation.
func (s InvitationService) Send(ctx context.Context, wid, id int64, req model.SendInvitationRequest) error {
	ws, err := s.workspaces.GetByID(ctx, wid)
	if err != nil {
		return err
	}

	inv, err := s.invitations.FindByID(ctx, wid, id)
	if err != nil {
		return err
	}

	if inv.IsRevoked {
		return domain.ErrInvitationRevoked
	}

	if inv.ExpiresAt != nil && time.Now().After(*inv.ExpiresAt) {
		return domain.ErrInvitationExpired
	}

	if inv.MaxUses > 0 && inv.UsesCount >= inv.MaxUses {
		return domain.ErrInvitationMaxUsesReached
	}

	s.sendInvitationEmail(ctx, inv, ws, req.Email)
	return nil
}

func (s InvitationService) sendInvitationEmail(ctx context.Context, inv domain.Invitation, ws domain.Workspace, recipientEmail string) {
	if s.mailer == nil {
		return
	}

	inviterName := "Workspace Admin"
	if inv.CreatedBy > 0 {
		u, err := s.users.GetByID(ctx, inv.CreatedBy)
		if err == nil && u.Name != "" {
			inviterName = u.Name
		}
	}

	webURL := s.webURL
	if webURL == "" {
		webURL = "http://localhost:3000"
	}
	inviteURL := fmt.Sprintf("%s/invite/%s", strings.TrimRight(webURL, "/"), inv.Token)

	data := map[string]any{
		"WorkspaceName": ws.Name,
		"InviterName":   inviterName,
		"InviteURL":     inviteURL,
	}

	msg := mailx.NewMessage()
	msg.AddTo(recipientEmail, "")
	msg.Subject = fmt.Sprintf("You have been invited to join %s", ws.Name)

	_ = s.mailer.SendTemplateAsync(ctx, "invitation", data, msg)
}

// GetPublicInfo retrieves public preview info for an invitation token without requiring authentication.
func (s InvitationService) GetPublicInfo(ctx context.Context, token string) (model.PublicInvitationResponse, error) {
	inv, err := s.invitations.FindByToken(ctx, token)
	if err != nil {
		return model.PublicInvitationResponse{}, err
	}

	ws, err := s.workspaces.GetByID(ctx, inv.WorkspaceID)
	if err != nil {
		return model.PublicInvitationResponse{}, err
	}

	inviterName := "Workspace Admin"
	if inv.CreatedBy > 0 {
		u, err := s.users.GetByID(ctx, inv.CreatedBy)
		if err == nil && u.Name != "" {
			inviterName = u.Name
		}
	}

	isValid := !inv.IsRevoked &&
		(inv.ExpiresAt == nil || time.Now().Before(*inv.ExpiresAt)) &&
		(inv.MaxUses == 0 || inv.UsesCount < inv.MaxUses)

	return model.PublicInvitationResponse{
		WorkspaceID:          ws.ID,
		WorkspaceName:        ws.Name,
		WorkspaceDescription: ws.Description,
		InviterName:          inviterName,
		Roles:                inv.Roles,
		ExpiresAt:            inv.ExpiresAt,
		IsValid:              isValid,
	}, nil
}

// Accept allows an authenticated user to join a workspace using an invitation token.
func (s InvitationService) Accept(ctx context.Context, token string, userID int64) (domain.Member, error) {
	inv, err := s.invitations.FindByToken(ctx, token)
	if err != nil {
		return domain.Member{}, err
	}

	if inv.IsRevoked {
		return domain.Member{}, domain.ErrInvitationRevoked
	}

	if inv.ExpiresAt != nil && time.Now().After(*inv.ExpiresAt) {
		return domain.Member{}, domain.ErrInvitationExpired
	}

	if inv.MaxUses > 0 && inv.UsesCount >= inv.MaxUses {
		return domain.Member{}, domain.ErrInvitationMaxUsesReached
	}

	ws, err := s.workspaces.GetByID(ctx, inv.WorkspaceID)
	if err != nil {
		return domain.Member{}, err
	}

	if ws.OwnerID == userID {
		return domain.Member{}, domain.ErrAlreadyMember
	}

	_, err = s.members.Find(ctx, inv.WorkspaceID, userID)
	if err == nil {
		return domain.Member{}, domain.ErrAlreadyMember
	} else if !base.IsNotFound(err) {
		return domain.Member{}, err
	}

	m, err := s.members.Create(ctx, inv.WorkspaceID, userID, inv.Roles)
	if err != nil {
		return domain.Member{}, err
	}

	_ = s.invitations.IncrementUses(ctx, inv.ID)
	return m, nil
}

// Search searches invitations for a given workspace.
func (s InvitationService) Search(ctx context.Context, wid int64, req model.SearchInvitationRequest) ([]domain.Invitation, string, error) {
	return s.invitations.Search(ctx, wid, req)
}

// Revoke revokes a workspace invitation.
func (s InvitationService) Revoke(ctx context.Context, wid, id int64) error {
	return s.invitations.Revoke(ctx, wid, id)
}

func (s InvitationService) validateRoles(roles []string) error {
	for _, r := range roles {
		if s.az != nil {
			if !s.az.HasRole(r) {
				return domain.ErrInvalidRole
			}
		} else if !authz.HasRole(r) {
			return domain.ErrInvalidRole
		}
	}
	return nil
}

func generateInvitationToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
