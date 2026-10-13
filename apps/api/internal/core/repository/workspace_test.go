package repository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/core/repository"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type WorkspaceRepoSuite struct {
	suite.Suite
	client *ent.Client
	repo   repository.WorkspaceRepository
}

func (s *WorkspaceRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:workspace_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.repo = repository.NewWorkspaceRepository(s.client)
}

func (s *WorkspaceRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *WorkspaceRepoSuite) SetupTest() {
	ctx := context.Background()
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, _ = s.client.User.Delete().Exec(ctx)
}

func (s *WorkspaceRepoSuite) createTestUser(name, email string) *ent.User {
	ctx := context.Background()
	u, err := s.client.User.Create().
		SetName(name).
		SetEmail(email).
		Save(ctx)
	s.Require().NoError(err)
	return u
}

func (s *WorkspaceRepoSuite) createTestWorkspace(name, desc string, ownerID int64) *ent.Workspace {
	ctx := context.Background()
	w, err := s.client.Workspace.Create().
		SetName(name).
		SetDescription(desc).
		SetOwnerID(ownerID).
		SetCreatedByID(ownerID).
		SetUpdatedByID(ownerID).
		Save(ctx)
	s.Require().NoError(err)
	return w
}

func (s *WorkspaceRepoSuite) Test_GetByID_Success() {
	ctx := context.Background()
	user := s.createTestUser("Alice", "alice@example.com")
	w := s.createTestWorkspace("Main Workspace", "Main Desc", user.ID)

	dom, err := s.repo.GetByID(ctx, w.ID)
	s.Require().NoError(err)
	s.Equal(w.ID, dom.ID)
	s.Equal("Main Workspace", dom.Name)
	s.Equal("Main Desc", dom.Description)
	s.Equal(user.ID, dom.OwnerID)
	s.Equal(user.Name, dom.Owner.Name)
	s.Equal(user.Name, dom.CreatedBy.Name)
	s.Equal(user.Name, dom.UpdatedBy.Name)
	s.False(dom.CreatedAt.IsZero())
	s.False(dom.UpdatedAt.IsZero())
}

func (s *WorkspaceRepoSuite) Test_GetByID_NotFound() {
	ctx := context.Background()
	dom, err := s.repo.GetByID(ctx, 99999)
	s.ErrorIs(err, base.ErrNotFound)
	s.Equal(domain.Workspace{}, dom)
}

func (s *WorkspaceRepoSuite) Test_Create_Success() {
	ctx := context.Background()
	user := s.createTestUser("Bob", "bob@example.com")

	req := model.CreateWorkspaceRequest{
		Name:        "Bob Workspace",
		Description: "Bob's private space",
		OwnerID:     user.ID,
	}

	dom, err := s.repo.Create(ctx, req)
	s.Require().NoError(err)
	s.Positive(dom.ID)
	s.Equal("Bob Workspace", dom.Name)
	s.Equal("Bob's private space", dom.Description)
	s.Equal(user.ID, dom.Owner.ID)
	s.Equal("Bob", dom.CreatedBy.Name)
	s.False(dom.CreatedAt.IsZero())

	// Verify in database
	dbW, err := s.client.Workspace.Get(ctx, dom.ID)
	s.Require().NoError(err)
	s.Equal(req.Name, dbW.Name)
	s.Equal(req.Description, dbW.Description)
}

func (s *WorkspaceRepoSuite) Test_Create_WithUserContext() {
	user := s.createTestUser("Charlie", "charlie@example.com")
	userCtx := base.UserContext{
		ID:   user.ID,
		Name: "Charlie",
	}
	ctx := context.WithValue(context.Background(), base.USER_CONTEXT_KEY, userCtx)

	req := model.CreateWorkspaceRequest{
		Name: "Charlie Context Workspace",
	}

	dom, err := s.repo.Create(ctx, req)
	s.Require().NoError(err)
	s.Equal(user.ID, dom.Owner.ID)
	s.Equal("Charlie", dom.CreatedBy.Name)
}

func (s *WorkspaceRepoSuite) Test_Create_NoUser_Unauthorized() {
	ctx := context.Background()
	req := model.CreateWorkspaceRequest{
		Name: "Orphan Workspace",
	}

	_, err := s.repo.Create(ctx, req)
	s.ErrorIs(err, base.ErrUnauthorized)
}

func (s *WorkspaceRepoSuite) Test_Update_Success() {
	ctx := context.Background()
	user := s.createTestUser("Diana", "diana@example.com")
	w := s.createTestWorkspace("Original", "Original Desc", user.ID)

	req := model.UpdateWorkspaceRequest{
		Name:        "Updated Workspace",
		Description: "Updated Desc",
	}

	dom, err := s.repo.Update(ctx, w.ID, req)
	s.Require().NoError(err)
	s.Equal("Updated Workspace", dom.Name)
	s.Equal("Updated Desc", dom.Description)
}

func (s *WorkspaceRepoSuite) Test_Update_NotFound() {
	ctx := context.Background()
	req := model.UpdateWorkspaceRequest{
		Name: "Non-existent",
	}

	_, err := s.repo.Update(ctx, 99999, req)
	s.ErrorIs(err, base.ErrNotFound)
}

func (s *WorkspaceRepoSuite) Test_Delete_Success() {
	ctx := context.Background()
	user := s.createTestUser("Eve", "eve@example.com")
	w := s.createTestWorkspace("To Delete", "Desc", user.ID)

	err := s.repo.Delete(ctx, w.ID)
	s.Require().NoError(err)

	_, err = s.repo.GetByID(ctx, w.ID)
	s.ErrorIs(err, base.ErrNotFound)
}

func (s *WorkspaceRepoSuite) Test_Delete_NotFound() {
	ctx := context.Background()
	err := s.repo.Delete(ctx, 99999)
	s.ErrorIs(err, base.ErrNotFound)
}

func (s *WorkspaceRepoSuite) Test_Search_Success_Pagination() {
	ctx := context.Background()
	user := s.createTestUser("Frank", "frank@example.com")

	for i := 1; i <= 5; i++ {
		s.createTestWorkspace(fmt.Sprintf("Team %d", i), "", user.ID)
	}

	// Search limit 2
	res, cursor, err := s.repo.Search(ctx, model.SearchWorkspaceRequest{
		Name:  "Team",
		Limit: 2,
	})
	s.Require().NoError(err)
	s.Len(res, 2)
	s.NotEmpty(cursor)

	// Search next page
	res2, cursor2, err := s.repo.Search(ctx, model.SearchWorkspaceRequest{
		Name:   "Team",
		Cursor: cursor,
		Limit:  2,
	})
	s.Require().NoError(err)
	s.Len(res2, 2)
	s.NotEmpty(cursor2)
	s.Greater(res[0].ID, res2[0].ID)
}

func (s *WorkspaceRepoSuite) Test_Search_InvalidCursor() {
	ctx := context.Background()
	_, _, err := s.repo.Search(ctx, model.SearchWorkspaceRequest{
		Cursor: "invalid-cursor",
	})
	s.ErrorIs(err, base.ErrInvalidCursor)
}

func TestWorkspaceRepoSuite(t *testing.T) {
	suite.Run(t, new(WorkspaceRepoSuite))
}
