package core

import (
	"context"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
)

type creatorMutator interface {
	SetCreatedBy(int64)
}

type updaterMutator interface {
	SetUpdatedBy(int64)
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

// InitMutator registers the audit mutator hook on the given Ent client.
func InitMutator(cli *ent.Client) {
	cli.Use(func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			c, isCreator := m.(creatorMutator)
			u, isUpdater := m.(updaterMutator)
			if !isCreator && !isUpdater {
				return next.Mutate(ctx, m)
			}
			if !m.Op().Is(ent.OpCreate | ent.OpUpdate | ent.OpUpdateOne) {
				return next.Mutate(ctx, m)
			}
			if ctx.Value(base.USER_CONTEXT_KEY) == nil {
				return next.Mutate(ctx, m)
			}

			userID := base.UserFromContext(ctx).ID
			if userID > 0 {
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
					if isUpdater {
						u.SetUpdatedBy(userID)
					} else {
						_ = m.SetField("updated_by", userID)
					}
				}
			}
			return next.Mutate(ctx, m)
		})
	})
}
