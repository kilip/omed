package repository_test

import (
	"context"
	"testing"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/repository"
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type SettingRepoSuite struct {
	suite.Suite
	client      *ent.Client
	repo        repository.SettingRepository
	workspaceID int64
	userID      int64
	ctx         context.Context
}

func TestSettingRepoSuite(t *testing.T) {
	suite.Run(t, new(SettingRepoSuite))
}

func (s *SettingRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:cms_setting_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	core.InitEntCli(s.client)
	s.repo = repository.NewSettingRepository(s.client)
}

func (s *SettingRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *SettingRepoSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.CmsSetting.Delete().Exec(ctx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)

	u, err := s.client.User.Create().
		SetName("Alice").
		SetEmail("alice@example.com").
		Save(ctx)
	s.Require().NoError(err)
	s.userID = u.ID

	userCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:   u.ID,
		Name: u.Name,
	})

	ws, err := s.client.Workspace.Create().
		SetName("Main WS").
		SetOwnerID(u.ID).
		Save(userCtx)
	s.Require().NoError(err)
	s.workspaceID = ws.ID

	s.ctx = base.WithWorkspaceID(userCtx, ws.ID)
}

func (s *SettingRepoSuite) TestGetDefaultWhenNotExists() {
	setting, err := s.repo.Get(s.ctx, s.workspaceID)
	s.Require().NoError(err)
	s.Equal("en", setting.DefaultLocale)
	s.Equal([]string{"en", "id"}, setting.SupportedLocales)
}

func (s *SettingRepoSuite) TestUpsertAndGet() {
	updated, err := s.repo.Upsert(s.ctx, s.workspaceID, "id", []string{"id", "en", "ja"})
	s.Require().NoError(err)
	s.Equal("id", updated.DefaultLocale)
	s.Equal([]string{"id", "en", "ja"}, updated.SupportedLocales)

	fetched, err := s.repo.Get(s.ctx, s.workspaceID)
	s.Require().NoError(err)
	s.Equal("id", fetched.DefaultLocale)
	s.Equal([]string{"id", "en", "ja"}, fetched.SupportedLocales)
}
