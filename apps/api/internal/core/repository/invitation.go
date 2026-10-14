package repository

import (
	"context"
	"strconv"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/predicate"
	"github.com/kilip/omed/api/ent/workspaceinvitation"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/entx"
)

// InvitationRepository handles persistence for workspace invitations.
type InvitationRepository struct {
	cli *ent.Client
}

// NewInvitationRepository constructs an InvitationRepository.
func NewInvitationRepository(cli *ent.Client) InvitationRepository {
	return InvitationRepository{cli: cli}
}

// Create persists a new workspace invitation link.
func (r InvitationRepository) Create(ctx context.Context, wid int64, req model.CreateInvitationRequest, token string) (domain.Invitation, error) {
	var dom domain.Invitation
	var created *ent.WorkspaceInvitation

	err := entx.WithTx(ctx, r.cli, func(tx *ent.Tx) error {
		builder := tx.WorkspaceInvitation.Create().
			SetWorkspaceID(wid).
			SetToken(token).
			SetRoles(req.Roles).
			SetMaxUses(req.MaxUses).
			SetNillableExpiresAt(req.ExpiresAt)

		res, err := builder.Save(ctx)
		if err != nil {
			return err
		}
		created = res
		return nil
	})
	if err != nil {
		return dom, err
	}

	return toInvitationDomain(created), nil
}

// FindByID retrieves a workspace invitation by workspace ID and invitation ID.
func (r InvitationRepository) FindByID(ctx context.Context, wid, id int64) (domain.Invitation, error) {
	var dom domain.Invitation
	inv, err := r.cli.WorkspaceInvitation.Query().
		Where(
			workspaceinvitation.WorkspaceIDEQ(wid),
			workspaceinvitation.IDEQ(id),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom, domain.ErrInvitationNotFound
		}
		return dom, err
	}
	return toInvitationDomain(inv), nil
}

// FindByToken retrieves an invitation by its unique token.
func (r InvitationRepository) FindByToken(ctx context.Context, token string) (domain.Invitation, error) {
	var dom domain.Invitation
	// Skip workspace filter as token is globally unique and may be queried before workspace resolution
	ctx = base.WithSkipWorkspaceFilter(ctx)

	inv, err := r.cli.WorkspaceInvitation.Query().
		Where(workspaceinvitation.TokenEQ(token)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom, domain.ErrInvitationNotFound
		}
		return dom, err
	}
	return toInvitationDomain(inv), nil
}

// Search retrieves workspace invitations with cursor pagination.
func (r InvitationRepository) Search(ctx context.Context, wid int64, req model.SearchInvitationRequest) ([]domain.Invitation, string, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	predicates := []predicate.WorkspaceInvitation{
		workspaceinvitation.WorkspaceIDEQ(wid),
	}

	if req.Cursor != "" {
		cursorID, err := strconv.ParseInt(req.Cursor, 10, 64)
		if err != nil || cursorID < 0 {
			return nil, "", base.ErrInvalidCursor
		}
		predicates = append(predicates, workspaceinvitation.IDLT(cursorID))
	}

	invitations, err := r.cli.WorkspaceInvitation.Query().
		Where(predicates...).
		Order(ent.Desc(workspaceinvitation.FieldID)).
		Limit(limit + 1).
		All(ctx)
	if err != nil {
		return nil, "", err
	}

	var cursor string
	if len(invitations) > limit {
		cursor = strconv.FormatInt(invitations[limit-1].ID, 10)
		invitations = invitations[:limit]
	}

	res := make([]domain.Invitation, len(invitations))
	for i, inv := range invitations {
		res[i] = toInvitationDomain(inv)
	}

	return res, cursor, nil
}

// IncrementUses increments the uses_count counter of an invitation.
func (r InvitationRepository) IncrementUses(ctx context.Context, id int64) error {
	ctx = base.WithSkipWorkspaceFilter(ctx)
	return r.cli.WorkspaceInvitation.UpdateOneID(id).
		AddUsesCount(1).
		Exec(ctx)
}

// Revoke marks an invitation as revoked.
func (r InvitationRepository) Revoke(ctx context.Context, wid, id int64) error {
	count, err := r.cli.WorkspaceInvitation.Update().
		Where(
			workspaceinvitation.WorkspaceIDEQ(wid),
			workspaceinvitation.IDEQ(id),
		).
		SetIsRevoked(true).
		Save(ctx)
	if err != nil {
		return err
	}
	if count == 0 {
		return domain.ErrInvitationNotFound
	}
	return nil
}

func toInvitationDomain(inv *ent.WorkspaceInvitation) domain.Invitation {
	return domain.Invitation{
		ID:          inv.ID,
		WorkspaceID: inv.WorkspaceID,
		Token:       inv.Token,
		Roles:       inv.Roles,
		MaxUses:     inv.MaxUses,
		UsesCount:   inv.UsesCount,
		ExpiresAt:   inv.ExpiresAt,
		IsRevoked:   inv.IsRevoked,
		CreatedBy:   inv.CreatedBy,
		CreatedAt:   inv.CreatedAt,
		UpdatedAt:   inv.UpdatedAt,
	}
}
