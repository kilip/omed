package service

import (
	"context"
	"strings"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/authz"
)

// MemberRepository defines persistence requirements for workspace members.
type MemberRepository interface {
	Find(ctx context.Context, wid, uid int64) (domain.Member, error)
	Search(ctx context.Context, wid int64, req model.SearchMemberRequest) ([]domain.Member, string, error)
	Create(ctx context.Context, wid, uid int64, roles []string) (domain.Member, error)
	UpdateRoles(ctx context.Context, wid, uid int64, roles []string) (domain.Member, error)
	Delete(ctx context.Context, wid, uid int64) error
	WorkspaceIDsForUser(ctx context.Context, uid int64) ([]int64, error)
}

// MemberService coordinates business logic for workspace members and implements authz.MembershipResolver.
type MemberService struct {
	members    MemberRepository
	workspaces WorkspaceService
	users      UserService
	az         *authz.Authorizer
}

// NewMemberService constructs a new MemberService.
func NewMemberService(
	members MemberRepository,
	workspaces WorkspaceService,
	users UserService,
) MemberService {
	return MemberService{
		members:    members,
		workspaces: workspaces,
		users:      users,
	}
}

// SetAuthorizer configures the authorizer used for role validation.
func (s *MemberService) SetAuthorizer(az *authz.Authorizer) {
	s.az = az
}

// ResolveMembership implements authz.MembershipResolver.
func (s MemberService) ResolveMembership(ctx context.Context, userID, workspaceID int64) (authz.Membership, error) {
	ws, err := s.workspaces.GetByID(ctx, workspaceID)
	if err != nil {
		return authz.Membership{}, err
	}

	if ws.OwnerID == userID {
		return authz.Membership{
			WorkspaceID: workspaceID,
			UserID:      userID,
			IsOwner:     true,
			Roles:       []string{domain.RoleAdmin},
		}, nil
	}

	m, err := s.members.Find(ctx, workspaceID, userID)
	if err != nil {
		return authz.Membership{}, err
	}

	return authz.Membership{
		WorkspaceID: workspaceID,
		UserID:      userID,
		IsOwner:     false,
		Roles:       m.Roles,
	}, nil
}

// Search searches members of a workspace.
func (s MemberService) Search(ctx context.Context, wid int64, req model.SearchMemberRequest) ([]domain.Member, string, error) {
	return s.members.Search(ctx, wid, req)
}

// Invite adds a member to a workspace by email, creating the user account if needed.
func (s MemberService) Invite(ctx context.Context, wid int64, req model.InviteMemberRequest) (domain.Member, error) {
	ws, err := s.workspaces.GetByID(ctx, wid)
	if err != nil {
		return domain.Member{}, err
	}

	if err := s.validateRoles(req.Roles); err != nil {
		return domain.Member{}, err
	}

	u, err := s.users.FindByEmail(ctx, req.Email)
	if err != nil {
		if !base.IsNotFound(err) {
			return domain.Member{}, err
		}
		// User does not exist yet; auto-register with default name
		name := strings.Split(req.Email, "@")[0]
		u, err = s.users.Create(ctx, model.CreateUserRequest{
			Name:  name,
			Email: req.Email,
		})
		if err != nil {
			return domain.Member{}, err
		}
	}

	if u.ID == ws.OwnerID {
		return domain.Member{}, domain.ErrOwnerImmutable
	}

	return s.members.Create(ctx, wid, u.ID, req.Roles)
}

// UpdateRoles updates the roles of a member.
func (s MemberService) UpdateRoles(ctx context.Context, wid, uid int64, req model.UpdateMemberRequest) (domain.Member, error) {
	ws, err := s.workspaces.GetByID(ctx, wid)
	if err != nil {
		return domain.Member{}, err
	}

	if uid == ws.OwnerID {
		return domain.Member{}, domain.ErrOwnerImmutable
	}

	if err := s.validateRoles(req.Roles); err != nil {
		return domain.Member{}, err
	}

	return s.members.UpdateRoles(ctx, wid, uid, req.Roles)
}

// Remove removes a member from a workspace.
func (s MemberService) Remove(ctx context.Context, wid, uid int64) error {
	ws, err := s.workspaces.GetByID(ctx, wid)
	if err != nil {
		return err
	}

	if uid == ws.OwnerID {
		return domain.ErrOwnerImmutable
	}

	return s.members.Delete(ctx, wid, uid)
}

func (s MemberService) validateRoles(roles []string) error {
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
