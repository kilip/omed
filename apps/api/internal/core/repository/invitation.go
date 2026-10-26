package repository

import (
	"context"
	"strconv"
	"strings"
	"time"

	"entgo.io/ent/dialect"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/predicate"
	"github.com/kilip/omed/api/ent/workspaceinvitation"
	"github.com/kilip/omed/api/ent/workspacemember"
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

// IncrementUses increments the uses_count counter of an invitation if max_uses has not been reached.
func (r InvitationRepository) IncrementUses(ctx context.Context, id int64) error {
	ctx = base.WithSkipWorkspaceFilter(ctx)

	const maxRetries = 10
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := entx.WithTx(ctx, r.cli, func(tx *ent.Tx) error {
			inv, err := forUpdate(tx.WorkspaceInvitation.Query(), tx).
				Where(workspaceinvitation.IDEQ(id)).
				Only(ctx)
			if err != nil {
				if ent.IsNotFound(err) {
					return domain.ErrInvitationNotFound
				}
				return err
			}

			if inv.MaxUses > 0 && inv.UsesCount >= inv.MaxUses {
				return domain.ErrInvitationMaxUsesReached
			}

			predicates := []predicate.WorkspaceInvitation{
				workspaceinvitation.IDEQ(id),
			}
			if inv.MaxUses > 0 {
				predicates = append(predicates, workspaceinvitation.UsesCountLT(inv.MaxUses))
			}
			affected, err := tx.WorkspaceInvitation.Update().
				Where(predicates...).
				AddUsesCount(1).
				Save(ctx)
			if err != nil {
				return err
			}
			if affected == 0 {
				return domain.ErrInvitationMaxUsesReached
			}
			return nil
		})

		if err == nil {
			return nil
		}

		if isLockOrConflictError(err) {
			if attempt < maxRetries-1 {
				time.Sleep(10 * time.Millisecond)
				continue
			}
		}

		return err
	}

	return domain.ErrInvitationMaxUsesReached
}

// Accept creates a workspace member and increments uses_count conditionally within a single transaction.
func (r InvitationRepository) Accept(ctx context.Context, id int64, userID int64) (domain.Member, error) {
	var dom domain.Member
	ctx = base.WithSkipWorkspaceFilter(ctx)

	const maxRetries = 10
	for attempt := 0; attempt < maxRetries; attempt++ {
		var wid int64
		err := entx.WithTx(ctx, r.cli, func(tx *ent.Tx) error {
			inv, err := forUpdate(tx.WorkspaceInvitation.Query(), tx).
				Where(workspaceinvitation.IDEQ(id)).
				Only(ctx)
			if err != nil {
				if ent.IsNotFound(err) {
					return domain.ErrInvitationNotFound
				}
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

			predicates := []predicate.WorkspaceInvitation{
				workspaceinvitation.IDEQ(id),
			}
			if inv.MaxUses > 0 {
				predicates = append(predicates, workspaceinvitation.UsesCountLT(inv.MaxUses))
			}
			affected, err := tx.WorkspaceInvitation.Update().
				Where(predicates...).
				AddUsesCount(1).
				Save(ctx)
			if err != nil {
				return err
			}
			if affected == 0 {
				return domain.ErrInvitationMaxUsesReached
			}

			exists, err := tx.WorkspaceMember.Query().
				Where(
					workspacemember.WorkspaceIDEQ(inv.WorkspaceID),
					workspacemember.UserIDEQ(userID),
				).
				Exist(ctx)
			if err != nil {
				return err
			}
			if exists {
				return domain.ErrAlreadyMember
			}

			_, err = tx.WorkspaceMember.Create().
				SetWorkspaceID(inv.WorkspaceID).
				SetUserID(userID).
				SetRoles(inv.Roles).
				Save(ctx)
			if err != nil {
				if ent.IsConstraintError(err) {
					return domain.ErrAlreadyMember
				}
				return err
			}

			wid = inv.WorkspaceID
			return nil
		})

		if err == nil {
			m, findErr := r.cli.WorkspaceMember.Query().
				Where(
					workspacemember.WorkspaceIDEQ(wid),
					workspacemember.UserIDEQ(userID),
				).
				WithUser().
				Only(ctx)
			if findErr != nil {
				return dom, findErr
			}
			return toMemberDomain(m), nil
		}

		if isLockOrConflictError(err) {
			if attempt < maxRetries-1 {
				time.Sleep(10 * time.Millisecond)
				continue
			}
		}

		return dom, err
	}

	return dom, domain.ErrInvitationMaxUsesReached
}

func forUpdate(q *ent.WorkspaceInvitationQuery, tx *ent.Tx) *ent.WorkspaceInvitationQuery {
	if tx != nil && tx.Dialect() != dialect.SQLite {
		return q.ForUpdate()
	}
	return q
}

func isLockOrConflictError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "locked") || strings.Contains(s, "busy")
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
