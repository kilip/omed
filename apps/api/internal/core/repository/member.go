package repository

import (
	"context"
	"strconv"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/predicate"
	"github.com/kilip/omed/api/ent/workspacemember"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/entx"
)

// MemberRepository handles persistence for workspace members.
type MemberRepository struct {
	cli *ent.Client
}

// NewMemberRepository constructs a MemberRepository.
func NewMemberRepository(cli *ent.Client) MemberRepository {
	return MemberRepository{cli: cli}
}

// Find retrieves a specific member of a workspace by user ID.
func (r MemberRepository) Find(ctx context.Context, wid, uid int64) (domain.Member, error) {
	var dom domain.Member
	m, err := r.cli.WorkspaceMember.Query().
		Where(
			workspacemember.WorkspaceIDEQ(wid),
			workspacemember.UserIDEQ(uid),
		).
		WithUser().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return dom, base.ErrNotFound
		}
		return dom, err
	}
	return toMemberDomain(m), nil
}

// Search searches members of a workspace with cursor pagination.
func (r MemberRepository) Search(ctx context.Context, wid int64, req model.SearchMemberRequest) ([]domain.Member, string, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	predicates := []predicate.WorkspaceMember{
		workspacemember.WorkspaceIDEQ(wid),
	}

	if req.Cursor != "" {
		cursorID, err := strconv.ParseInt(req.Cursor, 10, 64)
		if err != nil || cursorID < 0 {
			return nil, "", base.ErrInvalidCursor
		}
		predicates = append(predicates, workspacemember.IDLT(cursorID))
	}

	members, err := r.cli.WorkspaceMember.Query().
		Where(predicates...).
		WithUser().
		Order(ent.Desc(workspacemember.FieldID)).
		Limit(limit + 1).
		All(ctx)
	if err != nil {
		return nil, "", err
	}

	var cursor string
	if len(members) > limit {
		cursor = strconv.FormatInt(members[limit-1].ID, 10)
		members = members[:limit]
	}

	res := make([]domain.Member, len(members))
	for i, m := range members {
		res[i] = toMemberDomain(m)
	}

	return res, cursor, nil
}

// Create persists a new workspace member within a transaction.
func (r MemberRepository) Create(ctx context.Context, wid, uid int64, roles []string) (domain.Member, error) {
	var dom domain.Member
	var createdByID *int64
	if u := base.UserFromContext(ctx); u.ID != 0 {
		createdByID = &u.ID
	}

	err := entx.WithTx(ctx, r.cli, func(tx *ent.Tx) error {
		builder := tx.WorkspaceMember.Create().
			SetWorkspaceID(wid).
			SetUserID(uid).
			SetRoles(roles)
		if createdByID != nil {
			builder.SetCreatedByID(*createdByID).SetUpdatedByID(*createdByID)
		}
		_, err := builder.Save(ctx)
		return err
	})
	if err != nil {
		if ent.IsConstraintError(err) {
			return dom, domain.ErrAlreadyMember
		}
		return dom, err
	}

	return r.Find(ctx, wid, uid)
}

// UpdateRoles replaces the roles of a member.
func (r MemberRepository) UpdateRoles(ctx context.Context, wid, uid int64, roles []string) (domain.Member, error) {
	var dom domain.Member
	var updatedByID *int64
	if u := base.UserFromContext(ctx); u.ID != 0 {
		updatedByID = &u.ID
	}

	err := entx.WithTx(ctx, r.cli, func(tx *ent.Tx) error {
		query := tx.WorkspaceMember.Update().
			Where(
				workspacemember.WorkspaceIDEQ(wid),
				workspacemember.UserIDEQ(uid),
			).
			SetRoles(roles)
		if updatedByID != nil {
			query.SetUpdatedByID(*updatedByID)
		}
		count, err := query.Save(ctx)
		if err != nil {
			return err
		}
		if count == 0 {
			return base.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return dom, err
	}

	return r.Find(ctx, wid, uid)
}

// Delete removes a member from a workspace.
func (r MemberRepository) Delete(ctx context.Context, wid, uid int64) error {
	return entx.WithTx(ctx, r.cli, func(tx *ent.Tx) error {
		count, err := tx.WorkspaceMember.Delete().
			Where(
				workspacemember.WorkspaceIDEQ(wid),
				workspacemember.UserIDEQ(uid),
			).
			Exec(ctx)
		if err != nil {
			return err
		}
		if count == 0 {
			return base.ErrNotFound
		}
		return nil
	})
}

// WorkspaceIDsForUser returns all workspace IDs where the user is a member.
func (r MemberRepository) WorkspaceIDsForUser(ctx context.Context, uid int64) ([]int64, error) {
	var ids []int64
	err := r.cli.WorkspaceMember.Query().
		Where(workspacemember.UserIDEQ(uid)).
		Select(workspacemember.FieldWorkspaceID).
		Scan(ctx, &ids)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func toMemberDomain(m *ent.WorkspaceMember) domain.Member {
	dom := domain.Member{
		ID:          m.ID,
		WorkspaceID: m.WorkspaceID,
		Roles:       m.Roles,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
	if m.Edges.User != nil {
		dom.User = domain.User{
			ID:        m.Edges.User.ID,
			Name:      m.Edges.User.Name,
			Email:     m.Edges.User.Email,
			Avatar:    m.Edges.User.Avatar,
			CreatedAt: m.Edges.User.CreatedAt,
			UpdatedAt: m.Edges.User.UpdatedAt,
		}
	}
	return dom
}
