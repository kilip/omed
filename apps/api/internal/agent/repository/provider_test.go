package repository_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/agentprovider"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/agent/repository"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/platform/llmx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRepo(t *testing.T) (*ent.Client, repository.ProviderRepository, context.Context, context.Context) {
	t.Helper()
	cli := testutil.CreateTestDB(fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1&_busy_timeout=5000", t.Name()))
	core.InitEntCli(cli)
	t.Cleanup(func() {
		_ = cli.Close()
	})

	initCtx := base.WithSkipAuditMutator(base.WithSkipWorkspaceFilter(context.Background()))
	u, err := cli.User.Create().
		SetName("Test User").
		SetEmail(fmt.Sprintf("%s@example.com", t.Name())).
		Save(initCtx)
	require.NoError(t, err)

	userCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   u.ID,
		Name: u.Name,
	})

	ws1, err := cli.Workspace.Create().
		SetName("WS 1").
		SetOwnerID(u.ID).
		Save(userCtx)
	require.NoError(t, err)

	ws2, err := cli.Workspace.Create().
		SetName("WS 2").
		SetOwnerID(u.ID).
		Save(userCtx)
	require.NoError(t, err)

	ctxWs1 := base.WithWorkspaceID(userCtx, ws1.ID)
	ctxWs2 := base.WithWorkspaceID(userCtx, ws2.ID)

	repo := repository.NewProviderRepository(cli)
	return cli, repo, ctxWs1, ctxWs2
}

func TestCreateAndGet(t *testing.T) {
	_, repo, ctx, _ := setupRepo(t)

	caps := &llmx.Capabilities{Vision: true, ToolCalling: true}
	created, err := repo.Create(ctx, repository.NewProvider{
		Name:         "OpenAI Prod",
		Kind:         llmx.KindOpenAICompat,
		BaseURL:      "https://api.openai.com/v1",
		Model:        "gpt-4o",
		APIKeyEnc:    "v1.enc",
		KeyHint:      "sk-...1234",
		Capabilities: caps,
		IsDefault:    true,
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "OpenAI Prod", created.Name)
	assert.Equal(t, agentprovider.Kind(llmx.KindOpenAICompat), created.Kind)
	assert.True(t, created.IsDefault)

	got, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "OpenAI Prod", got.Name)
}

func TestNameUniquePerWorkspace(t *testing.T) {
	_, repo, ctxWs1, ctxWs2 := setupRepo(t)

	_, err := repo.Create(ctxWs1, repository.NewProvider{
		Name:      "Shared Name",
		Kind:      llmx.KindOpenAICompat,
		Model:     "gpt-4o",
		APIKeyEnc: "v1.enc",
	})
	require.NoError(t, err)

	// Same name in workspace 2 should succeed
	_, err = repo.Create(ctxWs2, repository.NewProvider{
		Name:      "Shared Name",
		Kind:      llmx.KindOpenAICompat,
		Model:     "gpt-4o",
		APIKeyEnc: "v1.enc",
	})
	require.NoError(t, err)

	// Same name in workspace 1 should return ErrProviderNameTaken
	_, err = repo.Create(ctxWs1, repository.NewProvider{
		Name:      "Shared Name",
		Kind:      llmx.KindOpenAICompat,
		Model:     "gpt-4o",
		APIKeyEnc: "v1.enc",
	})
	require.ErrorIs(t, err, domain.ErrProviderNameTaken)
}

func TestGetForeignWorkspaceIsNotFound(t *testing.T) {
	_, repo, ctxWs1, ctxWs2 := setupRepo(t)

	created, err := repo.Create(ctxWs1, repository.NewProvider{
		Name:      "Ws1 Provider",
		Kind:      llmx.KindOpenAICompat,
		Model:     "gpt-4o",
		APIKeyEnc: "v1.enc",
	})
	require.NoError(t, err)

	_, err = repo.Get(ctxWs2, created.ID)
	require.ErrorIs(t, err, domain.ErrProviderNotFound)
}

func TestSetDefaultMovesFlag(t *testing.T) {
	_, repo, ctx, _ := setupRepo(t)

	p1, err := repo.Create(ctx, repository.NewProvider{
		Name:      "Provider 1",
		Kind:      llmx.KindOpenAICompat,
		Model:     "gpt-4o",
		APIKeyEnc: "v1.enc",
		IsDefault: true,
	})
	require.NoError(t, err)

	p2, err := repo.Create(ctx, repository.NewProvider{
		Name:      "Provider 2",
		Kind:      llmx.KindAnthropic,
		Model:     "gpt-4o",
		APIKeyEnc: "v1.enc",
		IsDefault: false,
	})
	require.NoError(t, err)

	// Move default to p2
	_, err = repo.SetDefault(ctx, p2.ID)
	require.NoError(t, err)

	got1, err := repo.Get(ctx, p1.ID)
	require.NoError(t, err)
	assert.False(t, got1.IsDefault)

	got2, err := repo.Get(ctx, p2.ID)
	require.NoError(t, err)
	assert.True(t, got2.IsDefault)

	def, err := repo.GetDefault(ctx)
	require.NoError(t, err)
	assert.Equal(t, p2.ID, def.ID)
}

func TestConcurrentSetDefaultLeavesOneDefault(t *testing.T) {
	cli, repo, ctx, _ := setupRepo(t)

	p1, err := repo.Create(ctx, repository.NewProvider{
		Name:      "P1",
		Kind:      llmx.KindOpenAICompat,
		Model:     "gpt-4o",
		APIKeyEnc: "v1.enc",
	})
	require.NoError(t, err)

	p2, err := repo.Create(ctx, repository.NewProvider{
		Name:      "P2",
		Kind:      llmx.KindAnthropic,
		Model:     "gpt-4o",
		APIKeyEnc: "v1.enc",
	})
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(2)

	run := func(id int64) {
		defer wg.Done()
		_, _ = repo.SetDefault(ctx, id)
	}

	go run(p1.ID)
	go run(p2.ID)
	wg.Wait()

	count, err := cli.AgentProvider.Query().
		Where(agentprovider.IsDefaultEQ(true)).
		Count(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "exactly one provider must remain default")
}

func TestGetDefaultNone(t *testing.T) {
	_, repo, ctx, _ := setupRepo(t)

	_, err := repo.GetDefault(ctx)
	require.ErrorIs(t, err, domain.ErrNoDefaultProvider)
}

func TestDeleteIsHard(t *testing.T) {
	cli, repo, ctx, _ := setupRepo(t)

	created, err := repo.Create(ctx, repository.NewProvider{
		Name:      "To Delete",
		Kind:      llmx.KindOpenAICompat,
		Model:     "gpt-4o",
		APIKeyEnc: "v1.enc",
	})
	require.NoError(t, err)

	err = repo.Delete(ctx, created.ID)
	require.NoError(t, err)

	// Hard delete verification: row does not exist even bypassing workspace filter
	skipCtx := base.WithSkipWorkspaceFilter(context.Background())
	exists, err := cli.AgentProvider.Query().
		Where(agentprovider.IDEQ(created.ID)).
		Exist(skipCtx)
	require.NoError(t, err)
	assert.False(t, exists, "provider should be permanently deleted")
}
