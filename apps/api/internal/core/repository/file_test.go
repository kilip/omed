package repository_test

import (
	"context"
	"testing"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/core/repository"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type FileRepoSuite struct {
	suite.Suite
	client *ent.Client
	repo   repository.FileRepository
	user   *ent.User
	ws     *ent.Workspace
	ctx    context.Context
}

func (s *FileRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:file_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.repo = repository.NewFileRepository(s.client)
}

func (s *FileRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *FileRepoSuite) SetupTest() {
	rawCtx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.File.Delete().Exec(rawCtx)
	_, _ = s.client.WorkspaceMember.Delete().Exec(rawCtx)
	_, _ = s.client.Workspace.Delete().Exec(rawCtx)
	_, _ = s.client.User.Delete().Exec(rawCtx)

	u, err := s.client.User.Create().
		SetName("File Owner").
		SetEmail("owner@example.com").
		Save(rawCtx)
	s.Require().NoError(err)
	s.user = u

	userCtx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, base.UserContext{
		ID:            u.ID,
		Name:          u.Name,
		EmailVerified: true,
		Roles:         []base.UserRole{base.UserRoleUser},
	})

	ws, err := s.client.Workspace.Create().
		SetName("Primary Workspace").
		SetOwnerID(u.ID).
		Save(userCtx)
	s.Require().NoError(err)
	s.ws = ws

	s.ctx = base.WithWorkspaceID(userCtx, ws.ID)
}

func (s *FileRepoSuite) Test_CreateGetUpdateAndSoftDelete() {
	created, err := s.repo.Create(s.ctx, model.CreateFileRecordRequest{
		WorkspaceID: s.ws.ID,
		Name:        "receipt.pdf",
		Bucket:      "omed-ws-1",
		Key:         "fin/receipt-1.pdf",
		MimeType:    "application/pdf",
		Size:        2048,
		EntityType:  "fin:transaction",
		EntityID:    99,
	})
	s.Require().NoError(err)
	s.Positive(created.ID)
	s.Equal("receipt.pdf", created.Name)
	s.Equal("omed-ws-1", created.Bucket)
	s.Equal("fin/receipt-1.pdf", created.Key)
	s.Equal("application/pdf", created.MimeType)
	s.Equal(int64(2048), created.Size)
	s.Equal("fin:transaction", created.EntityType)
	s.Equal(int64(99), created.EntityID)
	s.Equal("File Owner", created.CreatedBy.Name)

	// GetByID
	fetched, err := s.repo.GetByID(s.ctx, created.ID)
	s.Require().NoError(err)
	s.Equal(created.ID, fetched.ID)

	// Update
	updated, err := s.repo.Update(s.ctx, created.ID, model.UpdateFileRequest{
		Name:       "renamed-receipt.pdf",
		EntityType: "fin:entry",
		EntityID:   105,
	})
	s.Require().NoError(err)
	s.Equal("renamed-receipt.pdf", updated.Name)
	s.Equal("fin:entry", updated.EntityType)
	s.Equal(int64(105), updated.EntityID)

	// Soft Delete
	err = s.repo.Delete(s.ctx, created.ID)
	s.Require().NoError(err)

	// GetByID after soft delete returns ErrNotFound
	_, err = s.repo.GetByID(s.ctx, created.ID)
	s.ErrorIs(err, base.ErrNotFound)

	// Verify soft-delete fields in DB directly
	dbFile, err := s.client.File.Get(s.ctx, created.ID)
	s.Require().NoError(err)
	s.NotNil(dbFile.DeletedAt)
	s.Require().NotNil(dbFile.DeletedByID)
	s.Equal(s.user.ID, *dbFile.DeletedByID)

	// Same key can be reused after soft-delete
	reused, err := s.repo.Create(s.ctx, model.CreateFileRecordRequest{
		WorkspaceID: s.ws.ID,
		Name:        "receipt-new.pdf",
		Bucket:      "omed-ws-1",
		Key:         "fin/receipt-1.pdf",
		MimeType:    "application/pdf",
		Size:        4096,
	})
	s.Require().NoError(err)
	s.NotEqual(created.ID, reused.ID)
}

func (s *FileRepoSuite) Test_Search_FiltersAndPagination() {
	for i, name := range []string{"alpha.png", "beta.pdf", "gamma.pdf"} {
		mime := "application/pdf"
		if i == 0 {
			mime = "image/png"
		}
		_, err := s.repo.Create(s.ctx, model.CreateFileRecordRequest{
			WorkspaceID: s.ws.ID,
			Name:        name,
			Bucket:      "omed-ws-1",
			Key:         "key-" + name,
			MimeType:    mime,
			Size:        int64(100 * (i + 1)),
			EntityType:  "fin:transaction",
			EntityID:    int64(10 + i),
		})
		s.Require().NoError(err)
	}

	// Filter by mimeType
	pdfs, cursor, err := s.repo.Search(s.ctx, model.SearchFileRequest{
		MimeType: "pdf",
	})
	s.Require().NoError(err)
	s.Empty(cursor)
	s.Len(pdfs, 2)

	// Filter by entityType & entityId
	byEntity, _, err := s.repo.Search(s.ctx, model.SearchFileRequest{
		EntityType: "fin:transaction",
		EntityID:   10,
	})
	s.Require().NoError(err)
	s.Len(byEntity, 1)
	s.Equal("alpha.png", byEntity[0].Name)

	// Cursor pagination
	page1, nextCursor, err := s.repo.Search(s.ctx, model.SearchFileRequest{Limit: 2})
	s.Require().NoError(err)
	s.Len(page1, 2)
	s.NotEmpty(nextCursor)

	page2, nextCursor2, err := s.repo.Search(s.ctx, model.SearchFileRequest{Cursor: nextCursor, Limit: 2})
	s.Require().NoError(err)
	s.Len(page2, 1)
	s.Empty(nextCursor2)

	// Invalid cursor
	_, _, err = s.repo.Search(s.ctx, model.SearchFileRequest{Cursor: "invalid"})
	s.ErrorIs(err, base.ErrInvalidCursor)
}

func TestFileRepoSuite(t *testing.T) {
	suite.Run(t, new(FileRepoSuite))
}
