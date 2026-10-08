package config

import (
	"context"
	"log"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/kilip/omed/finance/ent/account"

	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/ent/hook"
	"github.com/kilip/omed/finance/internal/core"
)

func workspaceHook() ent.Hook {
	hk := func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			user := core.UserFromContext(ctx)
			switch m.Op() {
			case ent.OpCreate:
				if setter, ok := m.(interface{ SetWorkspaceID(uuid.UUID) }); ok {
					setter.SetWorkspaceID(user.WorkspaceID)
				}
				if setter, ok := m.(interface{ SetCreatedBy(uuid.UUID) }); ok {
					setter.SetCreatedBy(user.ID)
				}
				if setter, ok := m.(interface{ SetUpdatedBy(uuid.UUID) }); ok {
					setter.SetUpdatedBy(user.ID)
				}
			case ent.OpUpdate, ent.OpUpdateOne:
				if f, ok := m.(interface {
					WhereP(...func(*entsql.Selector))
				}); ok {
					f.WhereP(entsql.FieldEQ("workspace_id", user.WorkspaceID))
				}
				if setter, ok := m.(interface{ SetUpdatedBy(uuid.UUID) }); ok {
					setter.SetUpdatedBy(user.ID)
				}
			case ent.OpDelete, ent.OpDeleteOne:
				if f, ok := m.(interface {
					WhereP(...func(*entsql.Selector))
				}); ok {
					f.WhereP(entsql.FieldEQ("workspace_id", user.WorkspaceID))
				}
			}

			return next.Mutate(ctx, m)
		})

	}
	return hook.On(hk, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne|ent.OpDelete|ent.OpDeleteOne)
}

func workspaceInterceptor() ent.Interceptor {
	return ent.InterceptFunc(func(next ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
			user := core.UserFromContext(ctx)
			if user.WorkspaceID != uuid.Nil {
				switch query := q.(type) {
				case *ent.AccountQuery:
					query.Where(account.WorkspaceIDEQ(user.WorkspaceID))
				}
			}
			return next.Query(ctx, q)
		})
	})
}

func ConfigureDBClient(client *ent.Client) {
	client.Use(workspaceHook())
	client.Intercept(workspaceInterceptor())
}

func GetEntClient(cfg Config) *ent.Client {
	cl, err := ent.Open("postgres", cfg.DatabaseUrl)
	if err != nil {
		log.Fatalf("DB failed connection %s", err)
	}

	return cl
}
