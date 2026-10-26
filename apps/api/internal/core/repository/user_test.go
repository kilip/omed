package repository_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/workspace"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/core/repository"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type UserRepoSuite struct {
	suite.Suite
	client *ent.Client
	repo   repository.UserRepository
}

func (s *UserRepoSuite) SetupSuite() {
	s.client = testutil.CreateTestDB("file:user_repo_test?mode=memory&cache=shared&_fk=1&_busy_timeout=5000")
	s.repo = repository.NewUserRepository(s.client)
}

func (s *UserRepoSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *UserRepoSuite) SetupTest() {
	ctx := base.WithSkipAuditMutator(context.Background())
	_, _ = s.client.WorkspaceMember.Delete().Exec(ctx)
	_, _ = s.client.Workspace.Delete().Exec(ctx)
	_, err := s.client.User.Delete().Exec(ctx)
	s.Require().NoError(err)
}

func (s *UserRepoSuite) createTestUser(name, email, avatar string) *ent.User {
	ctx := context.Background()
	builder := s.client.User.Create().
		SetName(name).
		SetEmail(email)
	if avatar != "" {
		builder.SetAvatar(avatar)
	}
	u, err := builder.Save(ctx)
	s.Require().NoError(err)
	return u
}

func (s *UserRepoSuite) Test_GetByID_Success() {
	ctx := context.Background()
	created := s.createTestUser("John Doe", "john@example.com", "https://example.com/avatar.png")

	usr, err := s.repo.GetByID(ctx, created.ID)
	s.Require().NoError(err)
	s.Equal(created.ID, usr.ID)
	s.Equal("John Doe", usr.Name)
	s.Equal("john@example.com", usr.Email)
	s.Equal("https://example.com/avatar.png", usr.Avatar)
	s.True(usr.EmailVerified)
	s.False(usr.CreatedAt.IsZero())
	s.False(usr.UpdatedAt.IsZero())
}

func (s *UserRepoSuite) Test_GetByID_UnverifiedEmail() {
	ctx := context.Background()
	created, err := s.client.User.Create().
		SetName("Unverified").
		SetEmail("unverified@example.com").
		SetEmailVerified(false).
		Save(ctx)
	s.Require().NoError(err)

	usr, err := s.repo.GetByID(ctx, created.ID)
	s.Require().NoError(err)
	s.False(usr.EmailVerified)
}

func (s *UserRepoSuite) Test_GetByID_NotFound() {
	ctx := context.Background()
	usr, err := s.repo.GetByID(ctx, 99999)
	s.ErrorIs(err, base.ErrNotFound)
	s.Equal(domain.User{}, usr)
}

func (s *UserRepoSuite) Test_Create_Success() {
	ctx := context.Background()
	req := model.CreateUserRequest{
		Name:   "Jane Doe",
		Email:  "jane@example.com",
		Avatar: "https://example.com/jane.png",
	}

	dom, err := s.repo.Create(ctx, req)
	s.Require().NoError(err)
	s.Positive(dom.ID)
	s.Equal(req.Name, dom.Name)
	s.Equal(req.Email, dom.Email)
	s.Equal(req.Avatar, dom.Avatar)
	s.False(dom.CreatedAt.IsZero())
	s.False(dom.UpdatedAt.IsZero())

	// Verify in database
	dbUser, err := s.client.User.Get(ctx, dom.ID)
	s.Require().NoError(err)
	s.Equal(req.Name, dbUser.Name)
	s.Equal(req.Email, dbUser.Email)
	s.Equal(req.Avatar, dbUser.Avatar)
	s.Equal("en", dbUser.Locale)
	s.Equal("USD", dbUser.Currency)

	// Verify personal workspace has matching base currency
	ws, err := s.client.Workspace.Query().Where(workspace.OwnerIDEQ(dom.ID)).Only(ctx)
	s.Require().NoError(err)
	s.Equal("USD", ws.BaseCurrency)
}

func (s *UserRepoSuite) Test_Create_IndonesianLocale_DefaultsToIDR() {
	ctx := context.Background()
	req := model.CreateUserRequest{
		Name:   "Budi",
		Email:  "budi@example.com",
		Locale: "id",
	}

	dom, err := s.repo.Create(ctx, req)
	s.Require().NoError(err)
	s.Equal("id", dom.Locale)
	s.Equal("IDR", dom.Currency)

	ws, err := s.client.Workspace.Query().Where(workspace.OwnerIDEQ(dom.ID)).Only(ctx)
	s.Require().NoError(err)
	s.Equal("IDR", ws.BaseCurrency)
}

func (s *UserRepoSuite) Test_Update_LocaleAndCurrency() {
	ctx := context.Background()
	created := s.createTestUser("Toni", "toni@example.com", "")

	updateReq := model.UpdateUserRequest{
		Locale:   "id",
		Currency: "IDR",
	}

	dom, err := s.repo.Update(ctx, created.ID, updateReq)
	s.Require().NoError(err)
	s.Equal("id", dom.Locale)
	s.Equal("IDR", dom.Currency)

	dbUser, err := s.client.User.Get(ctx, created.ID)
	s.Require().NoError(err)
	s.Equal("id", dbUser.Locale)
	s.Equal("IDR", dbUser.Currency)
}

func (s *UserRepoSuite) Test_Create_DuplicateEmail() {
	ctx := context.Background()
	s.createTestUser("Original", "duplicate@example.com", "")

	req := model.CreateUserRequest{
		Name:  "Duplicate",
		Email: "duplicate@example.com",
	}

	dom, err := s.repo.Create(ctx, req)
	s.Error(err)
	s.Equal(domain.User{}, dom)
}

func (s *UserRepoSuite) Test_Update_Success() {
	ctx := context.Background()
	created := s.createTestUser("Original Name", "user@example.com", "https://example.com/original.png")

	updateReq := model.UpdateUserRequest{
		Name:   "Updated Name",
		Avatar: "https://example.com/updated.png",
	}

	dom, err := s.repo.Update(ctx, created.ID, updateReq)
	s.Require().NoError(err)
	s.Equal(created.ID, dom.ID)
	s.Equal("Updated Name", dom.Name)
	s.Equal("https://example.com/updated.png", dom.Avatar)
	s.Equal("user@example.com", dom.Email)

	// Verify persisted state
	dbUser, err := s.client.User.Get(ctx, created.ID)
	s.Require().NoError(err)
	s.Equal("Updated Name", dbUser.Name)
	s.Equal("https://example.com/updated.png", dbUser.Avatar)
	s.Equal("user@example.com", dbUser.Email)
}

func (s *UserRepoSuite) Test_Update_NotFound() {
	ctx := context.Background()
	updateReq := model.UpdateUserRequest{
		Name: "Non-existent",
	}

	dom, err := s.repo.Update(ctx, 99999, updateReq)
	s.Error(err)
	s.Equal(domain.User{}, dom)
}

func (s *UserRepoSuite) Test_Delete_Success() {
	ctx := base.WithSkipAuditMutator(context.Background())
	created := s.createTestUser("To Delete", "delete@example.com", "")

	err := s.repo.Delete(ctx, created.ID)
	s.Require().NoError(err)

	// Verify deleted
	_, err = s.repo.GetByID(ctx, created.ID)
	s.ErrorIs(err, base.ErrNotFound)
}

func (s *UserRepoSuite) Test_Delete_NotFound() {
	ctx := base.WithSkipAuditMutator(context.Background())
	err := s.repo.Delete(ctx, 99999)
	s.Error(err)
}

func (s *UserRepoSuite) Test_Search_DefaultLimit() {
	ctx := context.Background()
	for i := 1; i <= 25; i++ {
		s.createTestUser(fmt.Sprintf("User %02d", i), fmt.Sprintf("user%02d@example.com", i), "")
	}

	users, cursor, err := s.repo.Search(ctx, model.SearchRequest{
		Limit: 0, // Should default to 20
	})
	s.Require().NoError(err)
	s.Len(users, 20)
	s.NotEmpty(cursor)
}

func (s *UserRepoSuite) Test_Search_LimitCapped() {
	ctx := context.Background()
	for i := 1; i <= 5; i++ {
		s.createTestUser(fmt.Sprintf("User %d", i), fmt.Sprintf("capped%d@example.com", i), "")
	}

	users, _, err := s.repo.Search(ctx, model.SearchRequest{
		Limit: 150, // Should cap at 100
	})
	s.Require().NoError(err)
	s.Len(users, 5)
}

func (s *UserRepoSuite) Test_Search_ByName_Fold() {
	ctx := context.Background()
	s.createTestUser("Alice Wonderland", "alice1@example.com", "")
	s.createTestUser("Bob Builder", "bob@example.com", "")
	s.createTestUser("ALICE Cooper", "alice2@example.com", "")

	users, _, err := s.repo.Search(ctx, model.SearchRequest{
		Name: "alice",
	})
	s.Require().NoError(err)
	s.Len(users, 2)
	for _, u := range users {
		s.Contains([]string{"Alice Wonderland", "ALICE Cooper"}, u.Name)
	}
}

func (s *UserRepoSuite) Test_Search_EmptyName_ReturnsAll() {
	ctx := context.Background()
	s.createTestUser("User A", "a@example.com", "")
	s.createTestUser("User B", "b@example.com", "")
	s.createTestUser("User C", "c@example.com", "")

	users, _, err := s.repo.Search(ctx, model.SearchRequest{})
	s.Require().NoError(err)
	s.Len(users, 3)
	// Verify ordered descending by ID
	s.Greater(users[0].ID, users[1].ID)
	s.Greater(users[1].ID, users[2].ID)
}

func (s *UserRepoSuite) Test_Search_Pagination() {
	ctx := context.Background()
	for i := 1; i <= 5; i++ {
		s.createTestUser(fmt.Sprintf("PaginationUser %d", i), fmt.Sprintf("p%d@example.com", i), "")
	}

	// Page 1: limit 2
	page1, cursor1, err := s.repo.Search(ctx, model.SearchRequest{
		Name:  "PaginationUser",
		Limit: 2,
	})
	s.Require().NoError(err)
	s.Len(page1, 2)
	s.NotEmpty(cursor1)

	// Page 2: limit 2, using cursor1
	page2, cursor2, err := s.repo.Search(ctx, model.SearchRequest{
		Name:   "PaginationUser",
		Limit:  2,
		Cursor: cursor1,
	})
	s.Require().NoError(err)
	s.Len(page2, 2)
	s.NotEmpty(cursor2)

	// All IDs in page 1 should be greater than page 2
	s.Greater(page1[1].ID, page2[0].ID)

	// Page 3: final record
	page3, cursor3, err := s.repo.Search(ctx, model.SearchRequest{
		Name:   "PaginationUser",
		Limit:  2,
		Cursor: cursor2,
	})
	s.Require().NoError(err)
	s.Len(page3, 1)
	s.Empty(cursor3)
}

func (s *UserRepoSuite) Test_Search_InvalidCursor() {
	ctx := context.Background()

	_, _, err := s.repo.Search(ctx, model.SearchRequest{
		Cursor: "invalid-cursor-value",
	})
	s.ErrorIs(err, base.ErrInvalidCursor)

	_, _, err = s.repo.Search(ctx, model.SearchRequest{
		Cursor: "-10",
	})
	s.ErrorIs(err, base.ErrInvalidCursor)
}

func (s *UserRepoSuite) Test_Search_NoResults() {
	ctx := context.Background()

	users, cursor, err := s.repo.Search(ctx, model.SearchRequest{
		Name: "NonExistentUserXYZ",
	})
	s.Require().NoError(err)
	s.Empty(users)
	s.Empty(cursor)
}

func TestUserRepoSuite(t *testing.T) {
	suite.Run(t, new(UserRepoSuite))
}
