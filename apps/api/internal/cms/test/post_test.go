package test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type PostSuite struct {
	testutil.ApiTestSuite[model.PostResponse]
}

func TestPostSuite(t *testing.T) {
	suite.Run(t, new(PostSuite))
}

func (s *PostSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *PostSuite) Test_Create_Get_Update_Delete_Lifecycle() {
	ws := s.ensureWorkspace("Post Lifecycle WS", s.User.ID)

	// 1. Create
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts", ws.ID), fiber.MethodPost, model.CreatePostRequest{
		Title:   "First Post",
		Excerpt: ptr("Summary"),
		Body:    ptr("Markdown body content"),
	})
	s.Created()
	created := s.GetResponse().Data
	s.Positive(created.ID)
	s.Equal("First Post", created.Title)
	s.Equal("first-post", created.Slug)
	s.Equal(domain.EffectiveStatusDraft, created.EffectiveStatus)

	// 2. GetByID
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.OK()
	fetched := s.GetResponse().Data
	s.Equal(created.ID, fetched.ID)
	s.Equal("first-post", fetched.Slug)

	// 3. Update
	newTitle := "Updated First Post"
	newSlug := "updated-first-post"
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d", ws.ID, created.ID), fiber.MethodPut, model.UpdatePostRequest{
		Title: &newTitle,
		Slug:  &newSlug,
	})
	s.OK()
	updated := s.GetResponse().Data
	s.Equal("Updated First Post", updated.Title)
	s.Equal("updated-first-post", updated.Slug)

	// 4. Delete
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d", ws.ID, created.ID), fiber.MethodDelete, nil)
	s.OK()

	// 5. Get after delete -> 404
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *PostSuite) Test_Search_And_Pagination() {
	ws := s.ensureWorkspace("Post Search WS", s.User.ID)

	// Create 3 posts
	for i := 1; i <= 3; i++ {
		s.Request(fmt.Sprintf("/workspaces/%d/cms/posts", ws.ID), fiber.MethodPost, model.CreatePostRequest{
			Title: fmt.Sprintf("Post %d", i),
		})
		s.Created()
	}

	// Search all
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/search", ws.ID), fiber.MethodPost, model.SearchPostRequest{})
	s.OK()
	res := s.PagedResponse()
	s.Len(res.Data, 3)

	// Search by query
	q := "Post 2"
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/search", ws.ID), fiber.MethodPost, model.SearchPostRequest{
		Query: &q,
	})
	s.OK()
	res = s.PagedResponse()
	s.Len(res.Data, 1)
	s.Equal("post-2", res.Data[0].Slug)

	// Pagination limit 2
	limit := 2
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/search", ws.ID), fiber.MethodPost, model.SearchPostRequest{
		Limit: &limit,
	})
	s.OK()
	p1 := s.PagedResponse()
	s.Len(p1.Data, 2)
	s.NotEmpty(p1.Meta.Cursor)

	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/search", ws.ID), fiber.MethodPost, model.SearchPostRequest{
		Limit:  &limit,
		Cursor: &p1.Meta.Cursor,
	})
	s.OK()
	p2 := s.PagedResponse()
	s.Len(p2.Data, 1)
}

func (s *PostSuite) Test_Publish_Schedule_Unpublish_Flows() {
	ws := s.ensureWorkspace("Post Publishing WS", s.User.ID)

	// Create Draft
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts", ws.ID), fiber.MethodPost, model.CreatePostRequest{
		Title: "Publishable Post",
	})
	s.Created()
	post := s.GetResponse().Data
	s.Equal(domain.EffectiveStatusDraft, post.EffectiveStatus)

	// 1. Schedule for future
	future := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d/schedule", ws.ID, post.ID), fiber.MethodPost, model.SchedulePostRequest{
		PublishAt: future,
	})
	s.OK()
	sched := s.GetResponse().Data
	s.Equal(domain.EffectiveStatusScheduled, sched.EffectiveStatus)
	s.NotNil(sched.PublishAt)

	// 2. Publish now (from SCHEDULED)
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d/publish", ws.ID, post.ID), fiber.MethodPost, nil)
	s.OK()
	pub := s.GetResponse().Data
	s.Equal(domain.EffectiveStatusPublished, pub.EffectiveStatus)

	// 3. Publishing an already published post should fail with 409
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d/publish", ws.ID, post.ID), fiber.MethodPost, nil)
	s.AssertStatus(fiber.StatusConflict)

	// 4. Unpublish (back to DRAFT)
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d/unpublish", ws.ID, post.ID), fiber.MethodPost, nil)
	s.OK()
	unpub := s.GetResponse().Data
	s.Equal(domain.EffectiveStatusDraft, unpub.EffectiveStatus)
	s.Nil(unpub.PublishAt)

	// 5. Unpublishing a draft post should fail with 409
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d/unpublish", ws.ID, post.ID), fiber.MethodPost, nil)
	s.AssertStatus(fiber.StatusConflict)
}

func (s *PostSuite) Test_ValidationErrors_422() {
	ws := s.ensureWorkspace("Post Validation WS", s.User.ID)

	// Empty title
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts", ws.ID), fiber.MethodPost, model.CreatePostRequest{
		Title: "",
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)

	// Schedule in past
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts", ws.ID), fiber.MethodPost, model.CreatePostRequest{
		Title: "Valid Title",
	})
	s.Created()
	post := s.GetResponse().Data

	past := time.Now().Add(-1 * time.Hour)
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d/schedule", ws.ID, post.ID), fiber.MethodPost, model.SchedulePostRequest{
		PublishAt: past,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *PostSuite) Test_SlugConflict_409() {
	ws := s.ensureWorkspace("Post Slug WS", s.User.ID)

	slug := "custom-slug"
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts", ws.ID), fiber.MethodPost, model.CreatePostRequest{
		Title: "Post 1",
		Slug:  &slug,
	})
	s.Created()

	// Try creating post 2 with same explicit slug -> 409
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts", ws.ID), fiber.MethodPost, model.CreatePostRequest{
		Title: "Post 2",
		Slug:  &slug,
	})
	s.AssertStatus(fiber.StatusConflict)
}

func (s *PostSuite) Test_Permission_Matrix() {
	ws := s.ensureWorkspace("Post Perms WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts", ws.ID), fiber.MethodPost, model.CreatePostRequest{
		Title: "Seed Post",
	})
	s.Created()
	post := s.GetResponse().Data

	// 1. Member with NO role -> 403
	norole := s.AsUser("norole-post@example.com", "No Role User")
	s.AddMember(ws.ID, norole.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d", ws.ID, post.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusForbidden)

	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts", ws.ID), fiber.MethodPost, model.CreatePostRequest{Title: "Fail"})
	s.AssertStatus(fiber.StatusForbidden)

	// 2. Viewer role: can read, cannot create/update/delete
	viewer := s.AsUser("viewer-post@example.com", "Viewer User")
	s.AddMember(ws.ID, viewer.ID, domain.RoleViewer)

	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d", ws.ID, post.ID), fiber.MethodGet, nil)
	s.OK()

	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts", ws.ID), fiber.MethodPost, model.CreatePostRequest{Title: "Fail"})
	s.AssertStatus(fiber.StatusForbidden)

	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d", ws.ID, post.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusForbidden)

	// 3. Editor role: can write
	editor := s.AsUser("editor-post@example.com", "Editor User")
	s.AddMember(ws.ID, editor.ID, domain.RoleEditor)

	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts", ws.ID), fiber.MethodPost, model.CreatePostRequest{Title: "Editor Created"})
	s.Created()

	// Editor can publish
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d/publish", ws.ID, post.ID), fiber.MethodPost, nil)
	s.OK()
}

func (s *PostSuite) Test_Unauthenticated_401() {
	ws := s.ensureWorkspace("Post Unauth WS", s.User.ID)

	s.SessionToken = ""
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts", ws.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusUnauthorized)
}

func (s *PostSuite) Test_CrossWorkspace_Isolation() {
	ws1 := s.ensureWorkspace("Post WS 1", s.User.ID)
	ws2 := s.ensureWorkspace("Post WS 2", s.User.ID)

	// Create post in WS 1
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts", ws1.ID), fiber.MethodPost, model.CreatePostRequest{
		Title: "WS1 Post",
	})
	s.Created()
	post1 := s.GetResponse().Data

	// Access post1 from ws2 -> 404
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d", ws2.ID, post1.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)

	// Update post1 from ws2 -> 404
	newTitle := "Hacked"
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d", ws2.ID, post1.ID), fiber.MethodPut, model.UpdatePostRequest{Title: &newTitle})
	s.AssertStatus(fiber.StatusNotFound)

	// Delete post1 from ws2 -> 404
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/%d", ws2.ID, post1.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusNotFound)

	// Search in ws2 should not return ws1 post
	s.Request(fmt.Sprintf("/workspaces/%d/cms/posts/search", ws2.ID), fiber.MethodPost, model.SearchPostRequest{})
	s.OK()
	s.Empty(s.PagedResponse().Data)
}

func ptr[T any](v T) *T {
	return &v
}
