package core

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/account"
	"github.com/kilip/omed/api/ent/intercept"
	"github.com/kilip/omed/api/internal/base"
)

type creatorMutator interface {
	SetCreatedBy(int64)
}

type updaterMutator interface {
	SetUpdatedBy(int64)
}

type deleterMutator interface {
	SetDeletedBy(int64)
}

type deleterIDMutator interface {
	SetDeletedByID(int64)
}

type workspaceMutator interface {
	SetWorkspaceID(int64)
}

func hasAuditField(m ent.Mutation, snake, camel string) bool {
	if _, exists := m.Field(snake); exists {
		return true
	}
	if _, exists := m.Field(camel); exists {
		return true
	}
	return false
}

// InitEntCli registers the audit mutator, workspace mutator, and query interceptors on the given Ent client.
func InitEntCli(cli *ent.Client) {
	setupAuditMutator(cli)
	setupWorkspaceMutator(cli)
	setupWorkspaceInterceptor(cli)
}

func setupWorkspaceMutator(cli *ent.Client) {
	cli.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if !m.Op().Is(ent.OpCreate) {
				return next.Mutate(ctx, m)
			}
			w, isWorkspaceMutator := m.(workspaceMutator)
			if !isWorkspaceMutator {
				return next.Mutate(ctx, m)
			}

			wsID := base.WorkspaceIDFromContext(ctx)
			if wsID > 0 {
				w.SetWorkspaceID(wsID)
			}
			return next.Mutate(ctx, m)
		})
	})
}

type workspaceInterceptor struct{}

func (workspaceInterceptor) filter(ctx context.Context, q intercept.Query) error {
	if base.SkipWorkspaceFilterFromContext(ctx) {
		return nil
	}
	wsID := base.WorkspaceIDFromContext(ctx)
	if wsID <= 0 {
		return nil
	}
	switch q.Type() {
	case ent.TypeAccount:
		q.WhereP(sql.FieldEQ(account.FieldWorkspaceID, wsID))
	}
	return nil
}

func (w workspaceInterceptor) Intercept(next ent.Querier) ent.Querier {
	return ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
		query, err := intercept.NewQuery(q)
		if err != nil {
			return nil, err
		}
		if err := w.filter(ctx, query); err != nil {
			return nil, err
		}
		return next.Query(ctx, q)
	})
}

func (w workspaceInterceptor) Traverse(ctx context.Context, q ent.Query) error {
	query, err := intercept.NewQuery(q)
	if err != nil {
		return err
	}
	return w.filter(ctx, query)
}

func setupWorkspaceInterceptor(cli *ent.Client) {
	cli.Intercept(workspaceInterceptor{})
}

func setupAuditMutator(cli *ent.Client) {
	cli.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			if base.SkipAuditMutatorFromContext(ctx) {
				return next.Mutate(ctx, m)
			}
			c, isCreator := m.(creatorMutator)
			u, isUpdater := m.(updaterMutator)
			d, isDeleter := m.(deleterMutator)
			dID, isDeleterID := m.(deleterIDMutator)
			if !isCreator && !isUpdater && !isDeleter && !isDeleterID {
				return next.Mutate(ctx, m)
			}
			if !m.Op().Is(ent.OpCreate | ent.OpUpdate | ent.OpUpdateOne | ent.OpDelete | ent.OpDeleteOne) {
				return next.Mutate(ctx, m)
			}

			userID := base.UserFromContext(ctx).ID
			if userID > 0 {
				setDeleter := func() {
					if isDeleter {
						if !hasAuditField(m, "deleted_by", "deletedBy") {
							d.SetDeletedBy(userID)
						}
					} else if !hasAuditField(m, "deleted_by", "deletedBy") {
						_ = m.SetField("deleted_by", userID)
					}

					if isDeleterID {
						if !hasAuditField(m, "deleted_by_id", "deletedByID") {
							dID.SetDeletedByID(userID)
						}
					} else if !hasAuditField(m, "deleted_by_id", "deletedByID") {
						_ = m.SetField("deleted_by_id", userID)
					}
				}

				if m.Op().Is(ent.OpCreate) {
					if isCreator {
						if !hasAuditField(m, "created_by", "createdBy") {
							c.SetCreatedBy(userID)
						}
					} else if !hasAuditField(m, "created_by", "createdBy") {
						_ = m.SetField("created_by", userID)
					}

					if isUpdater {
						if !hasAuditField(m, "updated_by", "updatedBy") {
							u.SetUpdatedBy(userID)
						}
					} else if !hasAuditField(m, "updated_by", "updatedBy") {
						_ = m.SetField("updated_by", userID)
					}
				} else if m.Op().Is(ent.OpUpdate | ent.OpUpdateOne) {
					if hasAuditField(m, "deleted_at", "deletedAt") || hasAuditField(m, "deleted_by", "deletedBy") || hasAuditField(m, "deleted_by_id", "deletedByID") {
						setDeleter()
					}

					if isUpdater {
						u.SetUpdatedBy(userID)
					} else {
						_ = m.SetField("updated_by", userID)
					}
				} else if m.Op().Is(ent.OpDelete | ent.OpDeleteOne) {
					setDeleter()
				}
			}
			return next.Mutate(ctx, m)
		})
	})
}
