package config

import (
	"context"
	"errors"
	"log"

	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/kilip/omed/finance/ent"
	"github.com/kilip/omed/finance/ent/hook"
	"github.com/kilip/omed/finance/internal/model"
	_ "github.com/lib/pq"
)

func userFromCtx(ctx context.Context) *model.AuthenticatedUser {
	u, _ := ctx.Value("user").(*model.AuthenticatedUser)
	return u
}

func WorkspaceInterceptor() ent.Interceptor {
	return ent.InterceptFunc(func(next ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
			user := userFromCtx(ctx)
			if user == nil {
				return nil, errors.New("missing authenticated user in context")
			}
			if f, ok := q.(interface{ WhereP(...func(*sql.Selector)) }); ok {
				f.WhereP(sql.FieldEQ("workspace_id", user.WorkspaceID))
			}
			return next.Query(ctx, q)
		})
	})
}

func WorkspaceHook() ent.Hook {
	hk := func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			user := userFromCtx(ctx)
			if user == nil {
				return nil, errors.New("missing authenticated user in context")
			}

			switch m.Op() {
			case ent.OpCreate:
				if setter, ok := m.(interface{ SetWorkspaceId(uuid.UUID) }); ok {
					setter.SetWorkspaceId(user.WorkspaceID)
				}
				if setter, ok := m.(interface{ SetCreatedBy(uuid.UUID) }); ok {
					setter.SetCreatedBy(user.ID)
				}
				if setter, ok := m.(interface{ SetUpdatedBy(uuid.UUID) }); ok {
					setter.SetUpdatedBy(user.ID)
				}
			case ent.OpUpdate, ent.OpUpdateOne:
				if f, ok := m.(interface{ WhereP(...func(*sql.Selector)) }); ok {
					f.WhereP(sql.FieldEQ("workspace_id", user.WorkspaceID))
				}
				if setter, ok := m.(interface{ SetUpdatedBy(uuid.UUID) }); ok {
					setter.SetUpdatedBy(user.ID)
				}
			case ent.OpDelete, ent.OpDeleteOne:
				if f, ok := m.(interface{ WhereP(...func(*sql.Selector)) }); ok {
					f.WhereP(sql.FieldEQ("workspace_id", user.WorkspaceID))
				}
			}

			return next.Mutate(ctx, m)
		})
	}
	return hook.On(hk, ent.OpCreate|ent.OpUpdate|ent.OpUpdateOne|ent.OpDelete|ent.OpDeleteOne)
}

func ConfigureDBClient(client *ent.Client) {
	client.Use(WorkspaceHook())
	client.Intercept(WorkspaceInterceptor())
}

func GetDB(config Config) *ent.Client {
	// Define the custom schema mapping
    schemaConfig := ent.SchemaConfig{
        // Replace "User" with the exact name of the Ent struct you annotated
        // Account: "finance",
    }

	client, err := ent.Open(dialect.Postgres, config.DB.URL, ent.AlternateSchema(schemaConfig))
	if err != nil {
		log.Fatalf("Error while connecting to database: %v", err)
	}

	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed to creating schema resources: %v", err)
	}

	ConfigureDBClient(client)
	return client
}
