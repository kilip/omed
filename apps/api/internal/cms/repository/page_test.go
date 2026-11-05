package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	"github.com/kilip/omed/api/internal/cms/repository"
	"github.com/kilip/omed/api/internal/core"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type PageRepoSuite struct {
	suite.Suite
	client      *ent.Client
	repo        repository.PageRepository
	workspaceID int64
	userID      int64
	ctx         context.Context
}

func TestPageRepoSuite(t *testing.T) {
	suite.Run(t, new(PageRepoSuite))
}

func (s *PageRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:cms_page_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	core.InitEntCli(s.client)
	s.repo = repository.NewPageRepository(s.client)
}

func (s *PageRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *PageRepoSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.CmsPageTranslation.Delete().Exec(ctx)
	_, _ = s.client.CmsPage.Delete().Exec(ctx)
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

func (s *PageRepoSuite) Test_Create_Get_Update_Delete() {
	now := time.Now()

	// Create
	page, err := s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Locale:      "en",
		Title:       "About Us",
		Slug:        "about-us",
		Body:        "About page content",
		Status:      domain.StatusDraft,
	}, "en", now)
	s.Require().NoError(err)
	s.Equal("About Us", page.Title)
	s.Equal("about-us", page.Slug)
	s.Equal(domain.EffectiveStatusDraft, page.EffectiveStatus)

	// Get
	found, err := s.repo.GetByID(s.ctx, page.ID, "en", "en", now)
	s.Require().NoError(err)
	s.Equal(page.ID, found.ID)
	s.Equal("about-us", found.Slug)

	// Update
	newTitle := "Updated About"
	newSlug := "updated-about"
	updated, err := s.repo.Update(s.ctx, page.ID, model.UpdatePageRecord{
		Locale: "en",
		Title:  &newTitle,
		Slug:   &newSlug,
	}, "en", now)
	s.Require().NoError(err)
	s.Equal("Updated About", updated.Title)
	s.Equal("updated-about", updated.Slug)

	// Add translation (ID)
	trans, err := s.repo.SaveTranslation(s.ctx, model.SavePageTranslationRecord{
		PageID:      page.ID,
		WorkspaceID: s.workspaceID,
		Locale:      "id",
		Title:       "Tentang Kami",
		Slug:        "tentang-kami",
		Body:        "Konten Tentang Kami",
		Status:      domain.StatusDraft,
	}, "en", now)
	s.Require().NoError(err)
	s.Len(trans.Translations, 2)

	// Delete
	err = s.repo.Delete(s.ctx, page.ID)
	s.Require().NoError(err)

	_, err = s.repo.GetByID(s.ctx, page.ID, "en", "en", now)
	s.Require().ErrorIs(err, domain.ErrPageNotFound)
}

func (s *PageRepoSuite) Test_SlugUniqueness_And_ReuseAfterDelete() {
	now := time.Now()

	// Create page 1
	p1, err := s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Locale:      "en",
		Title:       "Unique Page",
		Slug:        "unique-slug",
		Status:      domain.StatusDraft,
	}, "en", now)
	s.Require().NoError(err)

	// Duplicate slug in same locale should fail with ErrSlugTaken
	_, err = s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Locale:      "en",
		Title:       "Another Page",
		Slug:        "unique-slug",
		Status:      domain.StatusDraft,
	}, "en", now)
	s.Require().ErrorIs(err, domain.ErrSlugTaken)

	// Same slug in different locale should succeed
	_, err = s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Locale:      "id",
		Title:       "Halaman Unik",
		Slug:        "unique-slug",
		Status:      domain.StatusDraft,
	}, "en", now)
	s.Require().NoError(err)

	// SlugExistsInLocale check
	exists, err := s.repo.SlugExistsInLocale(s.ctx, s.workspaceID, "en", "unique-slug", nil)
	s.Require().NoError(err)
	s.True(exists)

	// Exclude p1.ID
	exists, err = s.repo.SlugExistsInLocale(s.ctx, s.workspaceID, "en", "unique-slug", &p1.ID)
	s.Require().NoError(err)
	s.False(exists)

	// Soft delete p1
	err = s.repo.Delete(s.ctx, p1.ID)
	s.Require().NoError(err)

	// Slug in 'en' can now be reused because p1 is deleted
	exists, err = s.repo.SlugExistsInLocale(s.ctx, s.workspaceID, "en", "unique-slug", nil)
	s.Require().NoError(err)
	s.False(exists)

	p2, err := s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Locale:      "en",
		Title:       "Reused Page",
		Slug:        "unique-slug",
		Status:      domain.StatusDraft,
	}, "en", now)
	s.Require().NoError(err)
	s.Equal("unique-slug", p2.Slug)
}

func (s *PageRepoSuite) Test_Admin_Cursor_Pagination() {
	now := time.Now()
	for i := 1; i <= 5; i++ {
		_, err := s.repo.Create(s.ctx, model.CreatePageRecord{
			WorkspaceID: s.workspaceID,
			Locale:      "en",
			Title:       fmt.Sprintf("Page %d", i),
			Slug:        fmt.Sprintf("page-%d", i),
			Status:      domain.StatusDraft,
		}, "en", now)
		s.Require().NoError(err)
	}

	limit := 2
	p1, cursor1, err := s.repo.Search(s.ctx, model.SearchPageRequest{Limit: &limit}, "en", now)
	s.Require().NoError(err)
	s.Len(p1, 2)
	s.NotEmpty(cursor1)

	p2, cursor2, err := s.repo.Search(s.ctx, model.SearchPageRequest{Limit: &limit, Cursor: &cursor1}, "en", now)
	s.Require().NoError(err)
	s.Len(p2, 2)
	s.NotEmpty(cursor2)
}
