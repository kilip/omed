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
		Title:       "My First Post",
		Slug:        "my-first-post",
		Excerpt:     "Short summary",
		Body:        "Hello world content",
		Status:      domain.StatusDraft,
	}, now)
	s.Require().NoError(err)
	s.Equal("My First Post", post.Title)
	s.Equal("my-first-post", post.Slug)
	s.Equal(domain.EffectiveStatusDraft, post.EffectiveStatus)

	// Get
	found, err := s.repo.GetByID(s.ctx, post.ID, now)
	s.Require().NoError(err)
	s.Equal(post.ID, found.ID)
	s.Equal("my-first-post", found.Slug)

	// Update
	newTitle := "Updated Title"
	newSlug := "updated-title"
	updated, err := s.repo.Update(s.ctx, post.ID, model.UpdatePostRecord{
		Title: &newTitle,
		Slug:  &newSlug,
	}, now)
	s.Require().NoError(err)
	s.Equal("Updated Title", updated.Title)
	s.Equal("updated-title", updated.Slug)

	// Delete
	err = s.repo.Delete(s.ctx, post.ID)
	s.Require().NoError(err)

	_, err = s.repo.GetByID(s.ctx, post.ID, now)
	s.Require().ErrorIs(err, domain.ErrPostNotFound)
}

func (s *PostRepoSuite) Test_SlugUniqueness_And_ReuseAfterDelete() {
	now := time.Now()

	// Create post 1
	p1, err := s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Unique Post",
		Slug:        "unique-slug",
		Status:      domain.StatusDraft,
	}, now)
	s.Require().NoError(err)

	// Duplicate slug should fail with ErrSlugTaken
	_, err = s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Another Post",
		Slug:        "unique-slug",
		Status:      domain.StatusDraft,
	}, now)
	s.Require().ErrorIs(err, domain.ErrSlugTaken)

	// SlugExists check
	exists, err := s.repo.SlugExists(s.ctx, "unique-slug", nil)
	s.Require().NoError(err)
	s.True(exists)

	// Exclude p1.ID
	exists, err = s.repo.SlugExists(s.ctx, "unique-slug", &p1.ID)
	s.Require().NoError(err)
	s.False(exists)

	// Soft delete p1
	err = s.repo.Delete(s.ctx, p1.ID)
	s.Require().NoError(err)

	// Slug can now be reused because p1 is deleted
	exists, err = s.repo.SlugExists(s.ctx, "unique-slug", nil)
	s.Require().NoError(err)
	s.False(exists)

	p2, err := s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Reused Post",
		Slug:        "unique-slug",
		Status:      domain.StatusDraft,
	}, now)
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
		Title:       "Golang Tutorial Draft",
		Slug:        "golang-draft",
		Status:      domain.StatusDraft,
	}, now)
	s.Require().NoError(err)

	// 2. Scheduled Future
	_, err = s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Future Release Notes",
		Slug:        "future-release",
		Status:      domain.StatusPublished,
		PublishAt:   &future,
	}, now)
	s.Require().NoError(err)

	// 3. Published Past
	_, err = s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Live Blog Post",
		Slug:        "live-blog",
		Status:      domain.StatusPublished,
		PublishAt:   &past,
	}, now)
	s.Require().NoError(err)

	// Filter by query
	q := "Golang"
	items, _, err := s.repo.Search(s.ctx, model.SearchPostRequest{Query: &q}, now)
	s.Require().NoError(err)
	s.Len(items, 1)
	s.Equal("golang-draft", items[0].Slug)

	// Filter by EffectiveStatus Draft
	draftStatus := domain.EffectiveStatusDraft
	items, _, err = s.repo.Search(s.ctx, model.SearchPostRequest{Status: &draftStatus}, now)
	s.Require().NoError(err)
	s.Len(items, 1)
	s.Equal(domain.EffectiveStatusDraft, items[0].EffectiveStatus)

	// Filter by EffectiveStatus Scheduled
	schedStatus := domain.EffectiveStatusScheduled
	items, _, err = s.repo.Search(s.ctx, model.SearchPostRequest{Status: &schedStatus}, now)
	s.Require().NoError(err)
	s.Len(items, 1)
	s.Equal(domain.EffectiveStatusScheduled, items[0].EffectiveStatus)

	// Filter by EffectiveStatus Published
	pubStatus := domain.EffectiveStatusPublished
	items, _, err = s.repo.Search(s.ctx, model.SearchPostRequest{Status: &pubStatus}, now)
	s.Require().NoError(err)
	s.Len(items, 1)
	s.Equal(domain.EffectiveStatusPublished, items[0].EffectiveStatus)
}

func (s *PostRepoSuite) Test_Admin_Cursor_Pagination() {
	now := time.Now()
	for i := 1; i <= 5; i++ {
		_, err := s.repo.Create(s.ctx, model.CreatePostRecord{
			WorkspaceID: s.workspaceID,
			Title:       fmt.Sprintf("Post %d", i),
			Slug:        fmt.Sprintf("post-%d", i),
			Status:      domain.StatusDraft,
		}, now)
		s.Require().NoError(err)
	}

	limit := 2
	p1, cursor1, err := s.repo.Search(s.ctx, model.SearchPostRequest{Limit: &limit}, now)
	s.Require().NoError(err)
	s.Len(p1, 2)
	s.NotEmpty(cursor1)

	p2, cursor2, err := s.repo.Search(s.ctx, model.SearchPostRequest{Limit: &limit, Cursor: &cursor1}, now)
	s.Require().NoError(err)
	s.Len(p2, 2)
	s.NotEmpty(cursor2)

	p3, cursor3, err := s.repo.Search(s.ctx, model.SearchPostRequest{Limit: &limit, Cursor: &cursor2}, now)
	s.Require().NoError(err)
	s.Len(p3, 1)
	s.Empty(cursor3)
}

func (s *PostRepoSuite) Test_Public_List_And_Composite_Cursor() {
	now := time.Now().Truncate(time.Second)
	t1 := now.Add(-3 * time.Hour)
	t2 := now.Add(-2 * time.Hour)
	tSame := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	// 1. Visible post t1
	_, err := s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Post 1",
		Slug:        "post-1",
		Status:      domain.StatusPublished,
		PublishAt:   &t1,
	}, now)
	s.Require().NoError(err)

	// 2. Visible post t2
	_, err = s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Post 2",
		Slug:        "post-2",
		Status:      domain.StatusPublished,
		PublishAt:   &t2,
	}, now)
	s.Require().NoError(err)

	// 3 & 4. Visible posts with identical publish_at (tests composite tie-breaker publish_at DESC, id DESC)
	_, err = s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Post 3 Same",
		Slug:        "post-3-same",
		Status:      domain.StatusPublished,
		PublishAt:   &tSame,
	}, now)
	s.Require().NoError(err)

	_, err = s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Post 4 Same",
		Slug:        "post-4-same",
		Status:      domain.StatusPublished,
		PublishAt:   &tSame,
	}, now)
	s.Require().NoError(err)

	// 5. Scheduled in future (should NOT appear in public list)
	_, err = s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Post Future",
		Slug:        "post-future",
		Status:      domain.StatusPublished,
		PublishAt:   &future,
	}, now)
	s.Require().NoError(err)

	// 6. Draft (should NOT appear in public list)
	_, err = s.repo.Create(s.ctx, model.CreatePostRecord{
		WorkspaceID: s.workspaceID,
		Title:       "Post Draft",
		Slug:        "post-draft",
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
	s.Empty(cursor2) // Exactly 4 visible items total

	// Verify all returned items are unique
	seen := make(map[int64]bool)
	for _, it := range append(page1, page2...) {
		s.False(seen[it.ID], "duplicate post ID in pagination")
		seen[it.ID] = true
	}
	s.Len(seen, 4)

	// Test PublicGetBySlug
	post1, err := s.repo.PublicGetBySlug(context.Background(), s.workspaceID, "post-1", now)
	s.Require().NoError(err)
	s.Equal("post-1", post1.Slug)

	// Future post via PublicGetBySlug should return ErrPostNotFound
	_, err = s.repo.PublicGetBySlug(context.Background(), s.workspaceID, "post-future", now)
	s.Require().ErrorIs(err, domain.ErrPostNotFound)

	// Draft post via PublicGetBySlug should return ErrPostNotFound
	_, err = s.repo.PublicGetBySlug(context.Background(), s.workspaceID, "post-draft", now)
	s.Require().ErrorIs(err, domain.ErrPostNotFound)
}
