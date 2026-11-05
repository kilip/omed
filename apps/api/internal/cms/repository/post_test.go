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

type PostRepoSuite struct {
	suite.Suite
	client      *ent.Client
	repo        repository.PostRepository
	workspaceID int64
	userID      int64
	ctx         context.Context
}

func TestPostRepoSuite(t *testing.T) {
	suite.Run(t, new(PostRepoSuite))
}

func (s *PostRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:cms_post_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	core.InitEntCli(s.client)
	s.repo = repository.NewPostRepository(s.client)
}

func (s *PostRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *PostRepoSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.CmsPostTranslation.Delete().Exec(ctx)
	_, _ = s.client.CmsPost.Delete().Exec(ctx)
	_, _ = s.client.CmsMedia.Delete().Exec(ctx)
	_, _ = s.client.File.Delete().Exec(ctx)
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

func (s *PostRepoSuite) Test_Create_Get_Update_Delete() {
	now := time.Now()

	// Create
	post, err := s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Locale:      "en",
		Title:       "My First Post",
		Slug:        "my-first-post",
		Excerpt:     "Short summary",
		Body:        "Hello world content",
		Status:      domain.StatusDraft,
	}, "en", now)
	s.Require().NoError(err)
	s.Equal("My First Post", post.Title)
	s.Equal("my-first-post", post.Slug)
	s.Equal(domain.EffectiveStatusDraft, post.EffectiveStatus)

	// Get
	found, err := s.repo.GetByID(s.ctx, post.ID, "en", "en", now)
	s.Require().NoError(err)
	s.Equal(post.ID, found.ID)
	s.Equal("my-first-post", found.Slug)

	// Update
	newTitle := "Updated Title"
	newSlug := "updated-title"
	updated, err := s.repo.Update(s.ctx, post.ID, model.UpdatePostRecord{
		Locale: "en",
		Title:  &newTitle,
		Slug:   &newSlug,
	}, "en", now)
	s.Require().NoError(err)
	s.Equal("Updated Title", updated.Title)
	s.Equal("updated-title", updated.Slug)

	// Add translation (ID)
	trans, err := s.repo.SaveTranslation(s.ctx, model.SavePostTranslationRecord{
		PostID:      post.ID,
		WorkspaceID: s.workspaceID,
		Locale:      "id",
		Title:       "Judul Baru",
		Slug:        "judul-baru",
		Body:        "Konten Indonesia",
		Status:      domain.StatusDraft,
	}, "en", now)
	s.Require().NoError(err)
	s.Len(trans.Translations, 2)

	// Delete
	err = s.repo.Delete(s.ctx, post.ID)
	s.Require().NoError(err)

	_, err = s.repo.GetByID(s.ctx, post.ID, "en", "en", now)
	s.Require().ErrorIs(err, domain.ErrPostNotFound)
}

func (s *PostRepoSuite) Test_SlugUniqueness_And_ReuseAfterDelete() {
	now := time.Now()

	// Create post 1
	p1, err := s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Locale:      "en",
		Title:       "Unique Post",
		Slug:        "unique-slug",
		Status:      domain.StatusDraft,
	}, "en", now)
	s.Require().NoError(err)

	// Duplicate slug in same locale should fail with ErrSlugTaken
	_, err = s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Locale:      "en",
		Title:       "Another Post",
		Slug:        "unique-slug",
		Status:      domain.StatusDraft,
	}, "en", now)
	s.Require().ErrorIs(err, domain.ErrSlugTaken)

	// Same slug in different locale should succeed
	_, err = s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Locale:      "id",
		Title:       "Post Unik",
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

	p2, err := s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Locale:      "en",
		Title:       "Reused Post",
		Slug:        "unique-slug",
		Status:      domain.StatusDraft,
	}, "en", now)
	s.Require().NoError(err)
	s.Equal("unique-slug", p2.Slug)
}

func (s *PostRepoSuite) Test_Search_Filters_And_EffectiveStatus() {
	now := time.Now().Truncate(time.Second)
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	// 1. Draft
	_, err := s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Locale:      "en",
		Title:       "Golang Tutorial Draft",
		Slug:        "golang-draft",
		Status:      domain.StatusDraft,
	}, "en", now)
	s.Require().NoError(err)

	// 2. Scheduled Future
	_, err = s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Locale:      "en",
		Title:       "Future Release Notes",
		Slug:        "future-release",
		Status:      domain.StatusPublished,
		PublishAt:   &future,
	}, "en", now)
	s.Require().NoError(err)

	// 3. Published Past
	_, err = s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Locale:      "en",
		Title:       "Live Blog Post",
		Slug:        "live-blog",
		Status:      domain.StatusPublished,
		PublishAt:   &past,
	}, "en", now)
	s.Require().NoError(err)

	// Filter by query
	q := "Golang"
	items, _, err := s.repo.Search(s.ctx, model.SearchPostRequest{Query: &q}, "en", now)
	s.Require().NoError(err)
	s.Len(items, 1)
	s.Equal("golang-draft", items[0].Slug)

	// Filter by EffectiveStatus Draft
	draftStatus := domain.EffectiveStatusDraft
	items, _, err = s.repo.Search(s.ctx, model.SearchPostRequest{Status: &draftStatus}, "en", now)
	s.Require().NoError(err)
	s.Len(items, 1)
	s.Equal(domain.EffectiveStatusDraft, items[0].EffectiveStatus)

	// Filter by EffectiveStatus Scheduled
	schedStatus := domain.EffectiveStatusScheduled
	items, _, err = s.repo.Search(s.ctx, model.SearchPostRequest{Status: &schedStatus}, "en", now)
	s.Require().NoError(err)
	s.Len(items, 1)
	s.Equal(domain.EffectiveStatusScheduled, items[0].EffectiveStatus)

	// Filter by EffectiveStatus Published
	pubStatus := domain.EffectiveStatusPublished
	items, _, err = s.repo.Search(s.ctx, model.SearchPostRequest{Status: &pubStatus}, "en", now)
	s.Require().NoError(err)
	s.Len(items, 1)
	s.Equal(domain.EffectiveStatusPublished, items[0].EffectiveStatus)
}

func (s *PostRepoSuite) Test_Admin_Cursor_Pagination() {
	now := time.Now()
	for i := 1; i <= 5; i++ {
		_, err := s.repo.Create(s.ctx, model.CreatePostRecord{
			WorkspaceID: s.workspaceID,
			Locale:      "en",
			Title:       fmt.Sprintf("Post %d", i),
			Slug:        fmt.Sprintf("post-%d", i),
			Status:      domain.StatusDraft,
		}, "en", now)
		s.Require().NoError(err)
	}

	limit := 2
	p1, cursor1, err := s.repo.Search(s.ctx, model.SearchPostRequest{Limit: &limit}, "en", now)
	s.Require().NoError(err)
	s.Len(p1, 2)
	s.NotEmpty(cursor1)

	p2, cursor2, err := s.repo.Search(s.ctx, model.SearchPostRequest{Limit: &limit, Cursor: &cursor1}, "en", now)
	s.Require().NoError(err)
	s.Len(p2, 2)
	s.NotEmpty(cursor2)

	p3, cursor3, err := s.repo.Search(s.ctx, model.SearchPostRequest{Limit: &limit, Cursor: &cursor2}, "en", now)
	s.Require().NoError(err)
	s.Len(p3, 1)
	s.Empty(cursor3)
}

func (s *PostRepoSuite) Test_Public_List_Fallback_And_Alternates() {
	now := time.Now().Truncate(time.Second)
	past := now.Add(-1 * time.Hour)

	// 1. Post with published EN and published ID translation
	p1, err := s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Locale:      "en",
		Title:       "Post 1 EN",
		Slug:        "post-1-en",
		Status:      domain.StatusPublished,
		PublishAt:   &past,
	}, "en", now)
	s.Require().NoError(err)

	_, err = s.repo.SaveTranslation(s.ctx, model.SavePostTranslationRecord{
		PostID:      p1.ID,
		WorkspaceID: s.workspaceID,
		Locale:      "id",
		Title:       "Post 1 ID",
		Slug:        "post-1-id",
		Status:      domain.StatusPublished,
		PublishAt:   &past,
	}, "en", now)
	s.Require().NoError(err)

	// 2. Post with published EN only
	_, err = s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Locale:      "en",
		Title:       "Post 2 EN Only",
		Slug:        "post-2-en",
		Status:      domain.StatusPublished,
		PublishAt:   &past,
	}, "en", now)
	s.Require().NoError(err)

	// Query ID locale: post 1 served in ID, post 2 falls back to EN
	list, _, err := s.repo.PublicList(context.Background(), s.workspaceID, "id", "en", nil, 10, now)
	s.Require().NoError(err)
	s.Len(list, 2)

	// Get Post 1 in ID directly
	item, fallback, alts, err := s.repo.PublicGetBySlug(context.Background(), s.workspaceID, "id", "en", "post-1-id", now)
	s.Require().NoError(err)
	s.False(fallback)
	s.Equal("post-1-id", item.Slug)
	s.Equal("id", item.Locale)
	s.Len(alts, 2)

	// Get Post 2 with slug "post-2-en" requested in ID -> falls back to EN!
	itemFallback, fallback, alts, err := s.repo.PublicGetBySlug(context.Background(), s.workspaceID, "id", "en", "post-2-en", now)
	s.Require().NoError(err)
	s.True(fallback)
	s.Equal("post-2-en", itemFallback.Slug)
	s.Equal("en", itemFallback.Locale)
	s.Len(alts, 1)
}
