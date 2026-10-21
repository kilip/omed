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
		SetName("Page WS").
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
		Title:       "About Us",
		Slug:        "about-us",
		Body:        "About page content",
		Status:      domain.StatusDraft,
	}, now)
	s.Require().NoError(err)
	s.Equal("About Us", page.Title)
	s.Equal("about-us", page.Slug)
	s.Equal(domain.EffectiveStatusDraft, page.EffectiveStatus)

	// Get
	found, err := s.repo.GetByID(s.ctx, page.ID, now)
	s.Require().NoError(err)
	s.Equal(page.ID, found.ID)
	s.Equal("about-us", found.Slug)

	// Update
	newTitle := "About Our Company"
	newSlug := "about-company"
	updated, err := s.repo.Update(s.ctx, page.ID, model.UpdatePageRecord{
		Title: &newTitle,
		Slug:  &newSlug,
	}, now)
	s.Require().NoError(err)
	s.Equal("About Our Company", updated.Title)
	s.Equal("about-company", updated.Slug)

	// Delete
	err = s.repo.Delete(s.ctx, page.ID)
	s.Require().NoError(err)

	_, err = s.repo.GetByID(s.ctx, page.ID, now)
	s.Require().ErrorIs(err, domain.ErrPageNotFound)
}

func (s *PageRepoSuite) Test_SlugUniqueness_And_ReuseAfterDelete() {
	now := time.Now()

	// Create page 1
	p1, err := s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Privacy Policy",
		Slug:        "privacy-policy",
		Status:      domain.StatusDraft,
	}, now)
	s.Require().NoError(err)

	// Duplicate slug should fail with ErrSlugTaken
	_, err = s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Another Privacy",
		Slug:        "privacy-policy",
		Status:      domain.StatusDraft,
	}, now)
	s.Require().ErrorIs(err, domain.ErrSlugTaken)

	// SlugExists check
	exists, err := s.repo.SlugExists(s.ctx, "privacy-policy", nil)
	s.Require().NoError(err)
	s.True(exists)

	// Exclude p1.ID
	exists, err = s.repo.SlugExists(s.ctx, "privacy-policy", &p1.ID)
	s.Require().NoError(err)
	s.False(exists)

	// Soft delete p1
	err = s.repo.Delete(s.ctx, p1.ID)
	s.Require().NoError(err)

	// Slug can now be reused because p1 is deleted
	exists, err = s.repo.SlugExists(s.ctx, "privacy-policy", nil)
	s.Require().NoError(err)
	s.False(exists)

	p2, err := s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Reused Privacy Policy",
		Slug:        "privacy-policy",
		Status:      domain.StatusDraft,
	}, now)
	s.Require().NoError(err)
	s.Equal("privacy-policy", p2.Slug)
}

func (s *PageRepoSuite) Test_Search_Filters_And_EffectiveStatus() {
	now := time.Now().Truncate(time.Second)
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	// 1. Draft
	_, err := s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Contact Us Draft",
		Slug:        "contact-draft",
		Status:      domain.StatusDraft,
	}, now)
	s.Require().NoError(err)

	// 2. Scheduled Future
	_, err = s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Future Terms of Service",
		Slug:        "future-terms",
		Status:      domain.StatusPublished,
		PublishAt:   &future,
	}, now)
	s.Require().NoError(err)

	// 3. Published Past
	_, err = s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Live FAQ",
		Slug:        "live-faq",
		Status:      domain.StatusPublished,
		PublishAt:   &past,
	}, now)
	s.Require().NoError(err)

	// Filter by query
	q := "FAQ"
	items, _, err := s.repo.Search(s.ctx, model.SearchPageRequest{Query: &q}, now)
	s.Require().NoError(err)
	s.Len(items, 1)
	s.Equal("live-faq", items[0].Slug)

	// Filter by EffectiveStatus Draft
	draftStatus := domain.EffectiveStatusDraft
	items, _, err = s.repo.Search(s.ctx, model.SearchPageRequest{Status: &draftStatus}, now)
	s.Require().NoError(err)
	s.Len(items, 1)
	s.Equal(domain.EffectiveStatusDraft, items[0].EffectiveStatus)

	// Filter by EffectiveStatus Scheduled
	schedStatus := domain.EffectiveStatusScheduled
	items, _, err = s.repo.Search(s.ctx, model.SearchPageRequest{Status: &schedStatus}, now)
	s.Require().NoError(err)
	s.Len(items, 1)
	s.Equal(domain.EffectiveStatusScheduled, items[0].EffectiveStatus)

	// Filter by EffectiveStatus Published
	pubStatus := domain.EffectiveStatusPublished
	items, _, err = s.repo.Search(s.ctx, model.SearchPageRequest{Status: &pubStatus}, now)
	s.Require().NoError(err)
	s.Len(items, 1)
	s.Equal(domain.EffectiveStatusPublished, items[0].EffectiveStatus)
}

func (s *PageRepoSuite) Test_Admin_Cursor_Pagination() {
	now := time.Now()
	for i := 1; i <= 5; i++ {
		_, err := s.repo.Create(s.ctx, model.CreatePageRecord{
			WorkspaceID: s.workspaceID,
			Title:       fmt.Sprintf("Page %d", i),
			Slug:        fmt.Sprintf("page-%d", i),
			Status:      domain.StatusDraft,
		}, now)
		s.Require().NoError(err)
	}

	limit := 2
	p1, cursor1, err := s.repo.Search(s.ctx, model.SearchPageRequest{Limit: &limit}, now)
	s.Require().NoError(err)
	s.Len(p1, 2)
	s.NotEmpty(cursor1)

	p2, cursor2, err := s.repo.Search(s.ctx, model.SearchPageRequest{Limit: &limit, Cursor: &cursor1}, now)
	s.Require().NoError(err)
	s.Len(p2, 2)
	s.NotEmpty(cursor2)

	p3, cursor3, err := s.repo.Search(s.ctx, model.SearchPageRequest{Limit: &limit, Cursor: &cursor2}, now)
	s.Require().NoError(err)
	s.Len(p3, 1)
	s.Empty(cursor3)
}

func (s *PageRepoSuite) Test_Public_List_And_Composite_Cursor() {
	now := time.Now().Truncate(time.Second)
	t1 := now.Add(-3 * time.Hour)
	t2 := now.Add(-2 * time.Hour)
	tSame := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	// 1. Visible page t1
	_, err := s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Page 1",
		Slug:        "page-1",
		Status:      domain.StatusPublished,
		PublishAt:   &t1,
	}, now)
	s.Require().NoError(err)

	// 2. Visible page t2
	_, err = s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Page 2",
		Slug:        "page-2",
		Status:      domain.StatusPublished,
		PublishAt:   &t2,
	}, now)
	s.Require().NoError(err)

	// 3 & 4. Visible pages with identical publish_at
	_, err = s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Page 3 Same",
		Slug:        "page-3-same",
		Status:      domain.StatusPublished,
		PublishAt:   &tSame,
	}, now)
	s.Require().NoError(err)

	_, err = s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Page 4 Same",
		Slug:        "page-4-same",
		Status:      domain.StatusPublished,
		PublishAt:   &tSame,
	}, now)
	s.Require().NoError(err)

	// 5. Future scheduled (not in public list)
	_, err = s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Page Future",
		Slug:        "page-future",
		Status:      domain.StatusPublished,
		PublishAt:   &future,
	}, now)
	s.Require().NoError(err)

	// 6. Draft (not in public list)
	_, err = s.repo.Create(s.ctx, model.CreatePageRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Page Draft",
		Slug:        "page-draft",
		Status:      domain.StatusDraft,
	}, now)
	s.Require().NoError(err)

	// Public list page 1 (limit 2)
	page1, cursor1, err := s.repo.PublicList(context.Background(), s.workspaceID, nil, 2, now)
	s.Require().NoError(err)
	s.Len(page1, 2)
	s.NotEmpty(cursor1)

	// Public list page 2 (limit 2)
	page2, cursor2, err := s.repo.PublicList(context.Background(), s.workspaceID, &cursor1, 2, now)
	s.Require().NoError(err)
	s.Len(page2, 2)
	s.Empty(cursor2)

	// Test PublicGetBySlug
	pageFound, err := s.repo.PublicGetBySlug(context.Background(), s.workspaceID, "page-1", now)
	s.Require().NoError(err)
	s.Equal("page-1", pageFound.Slug)

	// Future page -> ErrPageNotFound
	_, err = s.repo.PublicGetBySlug(context.Background(), s.workspaceID, "page-future", now)
	s.Require().ErrorIs(err, domain.ErrPageNotFound)

	// Draft page -> ErrPageNotFound
	_, err = s.repo.PublicGetBySlug(context.Background(), s.workspaceID, "page-draft", now)
	s.Require().ErrorIs(err, domain.ErrPageNotFound)
}
