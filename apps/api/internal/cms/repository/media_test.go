package repository_test

import (
	"context"
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

type MediaRepoSuite struct {
	suite.Suite
	client      *ent.Client
	repo        repository.MediaRepository
	workspaceID int64
	userID      int64
	fileID      int64
	ctx         context.Context
}

func TestMediaRepoSuite(t *testing.T) {
	suite.Run(t, new(MediaRepoSuite))
}

func (s *MediaRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:cms_media_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	core.InitEntCli(s.client)
	s.repo = repository.NewMediaRepository(s.client)
}

func (s *MediaRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *MediaRepoSuite) SetupTest() {
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
		Name: "Alice",
	})

	w, err := s.client.Workspace.Create().
		SetName("CMS Workspace").
		SetOwnerID(u.ID).
		Save(userCtx)
	s.Require().NoError(err)
	s.workspaceID = w.ID

	s.ctx = base.WithWorkspaceID(userCtx, s.workspaceID)

	f, err := s.client.File.Create().
		SetWorkspaceID(s.workspaceID).
		SetName("banner.png").
		SetBucket("test-bucket").
		SetKey("files/banner.png").
		SetMimeType("image/png").
		SetSize(2048).
		Save(s.ctx)
	s.Require().NoError(err)
	s.fileID = f.ID
}

func (s *MediaRepoSuite) Test_Create_And_GetByID() {
	created, err := s.repo.Create(s.ctx, model.CreateMediaRecordRequest{
		WorkspaceID: s.workspaceID,
		FileID:      s.fileID,
		Title:       "Header Banner",
		AltText:     "Site Banner Alt",
		Caption:     "A nice banner",
		IsPublic:    true,
	})
	s.Require().NoError(err)
	s.Equal("Header Banner", created.Title)
	s.Equal("Site Banner Alt", created.AltText)
	s.Equal("A nice banner", created.Caption)
	s.True(created.IsPublic)
	s.Require().NotNil(created.File)
	s.Equal("banner.png", created.File.Name)
	s.Equal("image/png", created.File.MimeType)

	got, err := s.repo.GetByID(s.ctx, created.ID)
	s.Require().NoError(err)
	s.Equal(created.ID, got.ID)
	s.Equal("Header Banner", got.Title)
	s.Require().NotNil(got.File)
	s.Equal(s.fileID, got.File.ID)
}

func (s *MediaRepoSuite) Test_GetByID_NotFound() {
	_, err := s.repo.GetByID(s.ctx, 99999)
	s.ErrorIs(err, domain.ErrMediaNotFound)
}

func (s *MediaRepoSuite) Test_Update() {
	created, err := s.repo.Create(s.ctx, model.CreateMediaRecordRequest{
		WorkspaceID: s.workspaceID,
		FileID:      s.fileID,
		Title:       "Old Title",
		AltText:     "Old Alt",
		Caption:     "Old Caption",
		IsPublic:    true,
	})
	s.Require().NoError(err)

	newTitle := "New Title"
	newAlt := "New Alt"
	newCaption := "New Caption"
	newPublic := false

	updated, err := s.repo.Update(s.ctx, created.ID, model.UpdateMediaRequest{
		Title:    &newTitle,
		AltText:  &newAlt,
		Caption:  &newCaption,
		IsPublic: &newPublic,
	})
	s.Require().NoError(err)
	s.Equal("New Title", updated.Title)
	s.Equal("New Alt", updated.AltText)
	s.Equal("New Caption", updated.Caption)
	s.False(updated.IsPublic)

	_, err = s.repo.Update(s.ctx, 99999, model.UpdateMediaRequest{Title: &newTitle})
	s.ErrorIs(err, domain.ErrMediaNotFound)
}

func (s *MediaRepoSuite) Test_Delete_SoftDelete() {
	created, err := s.repo.Create(s.ctx, model.CreateMediaRecordRequest{
		WorkspaceID: s.workspaceID,
		FileID:      s.fileID,
		Title:       "To Delete",
		IsPublic:    true,
	})
	s.Require().NoError(err)

	err = s.repo.Delete(s.ctx, created.ID)
	s.NoError(err)

	_, err = s.repo.GetByID(s.ctx, created.ID)
	s.ErrorIs(err, domain.ErrMediaNotFound)

	// Deleting again should return not found
	err = s.repo.Delete(s.ctx, created.ID)
	s.ErrorIs(err, domain.ErrMediaNotFound)
}

func (s *MediaRepoSuite) Test_IsReferenced() {
	media, err := s.repo.Create(s.ctx, model.CreateMediaRecordRequest{
		WorkspaceID: s.workspaceID,
		FileID:      s.fileID,
		Title:       "Cover Image",
		IsPublic:    true,
	})
	s.Require().NoError(err)

	ref, err := s.repo.IsReferenced(s.ctx, media.ID)
	s.NoError(err)
	s.False(ref)

	// Create post referencing media as cover_media_id
	post, err := s.client.CmsPost.Create().
		SetWorkspaceID(s.workspaceID).
		SetTitle("My Post").
		SetSlug("my-post").
		SetCoverMediaID(media.ID).
		Save(s.ctx)
	s.Require().NoError(err)

	ref, err = s.repo.IsReferenced(s.ctx, media.ID)
	s.NoError(err)
	s.True(ref)

	// Soft-delete post
	err = s.client.CmsPost.UpdateOneID(post.ID).
		SetDeletedAt(time.Now()).
		Exec(s.ctx)
	s.Require().NoError(err)

	ref, err = s.repo.IsReferenced(s.ctx, media.ID)
	s.NoError(err)
	s.False(ref)
}

func (s *MediaRepoSuite) Test_Search_And_Pagination() {
	// Create another file
	f2, err := s.client.File.Create().
		SetWorkspaceID(s.workspaceID).
		SetName("document.pdf").
		SetBucket("test-bucket").
		SetKey("files/document.pdf").
		SetMimeType("application/pdf").
		SetSize(4096).
		Save(s.ctx)
	s.Require().NoError(err)

	f3, err := s.client.File.Create().
		SetWorkspaceID(s.workspaceID).
		SetName("avatar.jpg").
		SetBucket("test-bucket").
		SetKey("files/avatar.jpg").
		SetMimeType("image/jpeg").
		SetSize(1024).
		Save(s.ctx)
	s.Require().NoError(err)

	m1, err := s.repo.Create(s.ctx, model.CreateMediaRecordRequest{
		WorkspaceID: s.workspaceID,
		FileID:      s.fileID,
		Title:       "Alpha banner",
		IsPublic:    true,
	})
	s.Require().NoError(err)

	m2, err := s.repo.Create(s.ctx, model.CreateMediaRecordRequest{
		WorkspaceID: s.workspaceID,
		FileID:      f2.ID,
		Title:       "Beta PDF document",
		IsPublic:    true,
	})
	s.Require().NoError(err)

	m3, err := s.repo.Create(s.ctx, model.CreateMediaRecordRequest{
		WorkspaceID: s.workspaceID,
		FileID:      f3.ID,
		Title:       "Gamma avatar",
		IsPublic:    true,
	})
	s.Require().NoError(err)

	// Search by query
	q := "pdf"
	items, _, err := s.repo.Search(s.ctx, model.SearchMediaRequest{
		Query: &q,
	})
	s.NoError(err)
	s.Len(items, 1)
	s.Equal(m2.ID, items[0].ID)

	// Search by mimeType
	mime := "image"
	items, _, err = s.repo.Search(s.ctx, model.SearchMediaRequest{
		MimeType: &mime,
	})
	s.NoError(err)
	s.Len(items, 2)

	// Pagination: limit 2
	limit := 2
	itemsPage1, cursor, err := s.repo.Search(s.ctx, model.SearchMediaRequest{
		Limit: &limit,
	})
	s.NoError(err)
	s.Len(itemsPage1, 2)
	s.NotEmpty(cursor)
	s.Equal(m3.ID, itemsPage1[0].ID)
	s.Equal(m2.ID, itemsPage1[1].ID)

	// Page 2
	itemsPage2, cursor2, err := s.repo.Search(s.ctx, model.SearchMediaRequest{
		Limit:  &limit,
		Cursor: &cursor,
	})
	s.NoError(err)
	s.Len(itemsPage2, 1)
	s.Empty(cursor2)
	s.Equal(m1.ID, itemsPage2[0].ID)
}

func (s *MediaRepoSuite) Test_Workspace_Isolation() {
	created, err := s.repo.Create(s.ctx, model.CreateMediaRecordRequest{
		WorkspaceID: s.workspaceID,
		FileID:      s.fileID,
		Title:       "Tenant A media",
		IsPublic:    true,
	})
	s.Require().NoError(err)

	// Switch to Workspace B
	ctxB := base.WithWorkspaceID(s.ctx, s.workspaceID+999)

	_, err = s.repo.GetByID(ctxB, created.ID)
	s.ErrorIs(err, domain.ErrMediaNotFound)

	items, _, err := s.repo.Search(ctxB, model.SearchMediaRequest{})
	s.NoError(err)
	s.Empty(items)
}
