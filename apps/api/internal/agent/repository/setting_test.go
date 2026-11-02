package repository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/agentsetting"
	"github.com/kilip/omed/api/internal/agent/repository"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/internal/platform/llmx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSettingRepo(t *testing.T) (*ent.Client, repository.SettingRepository, repository.ProviderRepository, context.Context, context.Context, int64, int64) {
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

	repo := repository.NewSettingRepository(cli)
	providerRepo := repository.NewProviderRepository(cli)
	return cli, repo, providerRepo, ctxWs1, ctxWs2, ws1.ID, ws2.ID
}

func TestSetting_GetNotFound(t *testing.T) {
	_, repo, _, ctxWs1, _, _, _ := setupSettingRepo(t)

	setting, err := repo.Get(ctxWs1)
	require.Error(t, err)
	assert.True(t, ent.IsNotFound(err), "expected ent.IsNotFound to be true, got %v", err)
	assert.Nil(t, setting)
}

func TestSetting_UpsertCreatesThenUpdates(t *testing.T) {
	cli, repo, providerRepo, ctxWs1, _, ws1ID, _ := setupSettingRepo(t)

	prov1, err := providerRepo.Create(ctxWs1, repository.NewProvider{
		Name:      "Provider 1",
		Kind:      llmx.KindOpenAICompat,
		Model:     "gpt-4o",
		APIKeyEnc: "v1.enc",
	})
	require.NoError(t, err)

	prov2, err := providerRepo.Create(ctxWs1, repository.NewProvider{
		Name:      "Provider 2",
		Kind:      llmx.KindAnthropic,
		Model:     "claude-3-5-sonnet",
		APIKeyEnc: "v1.enc",
	})
	require.NoError(t, err)

	// First upsert: creates setting
	created, err := repo.Upsert(ctxWs1, true, &prov1.ID)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.True(t, created.FinancialEnabled)
	require.NotNil(t, created.FinancialProviderID)
	assert.Equal(t, prov1.ID, *created.FinancialProviderID)

	// Second upsert: updates setting in same workspace
	updated, err := repo.Upsert(ctxWs1, false, &prov2.ID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.False(t, updated.FinancialEnabled)
	require.NotNil(t, updated.FinancialProviderID)
	assert.Equal(t, prov2.ID, *updated.FinancialProviderID)
	assert.Equal(t, created.ID, updated.ID)

	// Verify exactly one row exists in the workspace
	count, err := cli.AgentSetting.Query().
		Where(agentsetting.WorkspaceIDEQ(ws1ID)).
		Count(ctxWs1)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "two upserts in one workspace must leave exactly one row")
}

func TestSetting_WorkspaceIsolation(t *testing.T) {
	_, repo, _, ctxWs1, ctxWs2, _, _ := setupSettingRepo(t)

	_, err := repo.Upsert(ctxWs1, true, nil)
	require.NoError(t, err)

	// Query from workspace 2 must not see workspace 1 row
	s2, err := repo.Get(ctxWs2)
	require.Error(t, err)
	assert.True(t, ent.IsNotFound(err), "workspace B should not see workspace A setting")
	assert.Nil(t, s2)
}

func TestSetting_ProviderDeleteSetsNull(t *testing.T) {
	_, repo, providerRepo, ctxWs1, _, _, _ := setupSettingRepo(t)

	prov, err := providerRepo.Create(ctxWs1, repository.NewProvider{
		Name:      "Provider To Delete",
		Kind:      llmx.KindOpenAICompat,
		Model:     "gpt-4o",
		APIKeyEnc: "v1.enc",
	})
	require.NoError(t, err)

	_, err = repo.Upsert(ctxWs1, true, &prov.ID)
	require.NoError(t, err)

	setting, err := repo.Get(ctxWs1)
	require.NoError(t, err)
	require.NotNil(t, setting.FinancialProviderID)
	assert.Equal(t, prov.ID, *setting.FinancialProviderID)

	// Hard delete the provider
	err = providerRepo.Delete(ctxWs1, prov.ID)
	require.NoError(t, err)

	// The setting's financial_provider_id should now be null via ON DELETE SET NULL
	reloaded, err := repo.Get(ctxWs1)
	require.NoError(t, err)
	require.NotNil(t, reloaded)
	assert.True(t, reloaded.FinancialEnabled)
	assert.Nil(t, reloaded.FinancialProviderID, "deleting referenced provider must set financial_provider_id to null")
}
