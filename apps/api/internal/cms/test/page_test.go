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

type PageSuite struct {
	testutil.ApiTestSuite[model.PageResponse]
}

func TestPageSuite(t *testing.T) {
	suite.Run(t, new(PageSuite))
}

func (s *PageSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *PageSuite) Test_Create_Get_Update_Delete_Lifecycle() {
	ws := s.ensureWorkspace("Page Lifecycle WS", s.User.ID)

	// 1. Create
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages", ws.ID), fiber.MethodPost, model.CreatePageRequest{
		Title: "First Page",
		Body:  ptr("Markdown body content"),
	})
	s.Created()
	created := s.GetResponse().Data
	s.Positive(created.ID)
	s.Equal("First Page", created.Title)
	s.Equal("first-page", created.Slug)
	s.Equal(domain.EffectiveStatusDraft, created.EffectiveStatus)

	// 2. GetByID
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.OK()
	fetched := s.GetResponse().Data
	s.Equal(created.ID, fetched.ID)
	s.Equal("first-page", fetched.Slug)

	// 3. Update
	newTitle := "Updated First Page"
	newSlug := "updated-first-page"
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d", ws.ID, created.ID), fiber.MethodPut, model.UpdatePageRequest{
		Title: &newTitle,
		Slug:  &newSlug,
	})
	s.OK()
	updated := s.GetResponse().Data
	s.Equal("Updated First Page", updated.Title)
	s.Equal("updated-first-page", updated.Slug)

	// 4. Delete
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d", ws.ID, created.ID), fiber.MethodDelete, nil)
	s.OK()

	// 5. Get after delete -> 404
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d", ws.ID, created.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *PageSuite) Test_Search_And_Pagination() {
	ws := s.ensureWorkspace("Page Search WS", s.User.ID)

	// Create 3 pages
	for i := 1; i <= 3; i++ {
		s.Request(fmt.Sprintf("/workspaces/%d/cms/pages", ws.ID), fiber.MethodPost, model.CreatePageRequest{
			Title: fmt.Sprintf("Page %d", i),
		})
		s.Created()
	}

	// Search all
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/search", ws.ID), fiber.MethodPost, model.SearchPageRequest{})
	s.OK()
	res := s.PagedResponse()
	s.Len(res.Data, 3)

	// Search by query
	q := "Page 2"
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/search", ws.ID), fiber.MethodPost, model.SearchPageRequest{
		Query: &q,
	})
	s.OK()
	res = s.PagedResponse()
	s.Len(res.Data, 1)
	s.Equal("page-2", res.Data[0].Slug)

	// Pagination limit 2
	limit := 2
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/search", ws.ID), fiber.MethodPost, model.SearchPageRequest{
		Limit: &limit,
	})
	s.OK()
	p1 := s.PagedResponse()
	s.Len(p1.Data, 2)
	s.NotEmpty(p1.Meta.Cursor)

	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/search", ws.ID), fiber.MethodPost, model.SearchPageRequest{
		Limit:  &limit,
		Cursor: &p1.Meta.Cursor,
	})
	s.OK()
	p2 := s.PagedResponse()
	s.Len(p2.Data, 1)
}

func (s *PageSuite) Test_Publish_Schedule_Unpublish_Flows() {
	ws := s.ensureWorkspace("Page Publishing WS", s.User.ID)

	// Create Draft
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages", ws.ID), fiber.MethodPost, model.CreatePageRequest{
		Title: "Publishable Page",
	})
	s.Created()
	page := s.GetResponse().Data
	s.Equal(domain.EffectiveStatusDraft, page.EffectiveStatus)

	// 1. Schedule for future
	future := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Second)
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d/schedule", ws.ID, page.ID), fiber.MethodPost, model.SchedulePageRequest{
		PublishAt: future,
	})
	s.OK()
	sched := s.GetResponse().Data
	s.Equal(domain.EffectiveStatusScheduled, sched.EffectiveStatus)
	s.NotNil(sched.PublishAt)

	// 2. Publish now (from SCHEDULED)
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d/publish", ws.ID, page.ID), fiber.MethodPost, nil)
	s.OK()
	pub := s.GetResponse().Data
	s.Equal(domain.EffectiveStatusPublished, pub.EffectiveStatus)

	// 3. Publishing an already published page should fail with 409
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d/publish", ws.ID, page.ID), fiber.MethodPost, nil)
	s.AssertStatus(fiber.StatusConflict)

	// 4. Unpublish (back to DRAFT)
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d/unpublish", ws.ID, page.ID), fiber.MethodPost, nil)
	s.OK()
	unpub := s.GetResponse().Data
	s.Equal(domain.EffectiveStatusDraft, unpub.EffectiveStatus)
	s.Nil(unpub.PublishAt)

	// 5. Unpublishing a draft page should fail with 409
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d/unpublish", ws.ID, page.ID), fiber.MethodPost, nil)
	s.AssertStatus(fiber.StatusConflict)
}

func (s *PageSuite) Test_ValidationErrors_422() {
	ws := s.ensureWorkspace("Page Validation WS", s.User.ID)

	// Empty title
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages", ws.ID), fiber.MethodPost, model.CreatePageRequest{
		Title: "",
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)

	// Schedule in past
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages", ws.ID), fiber.MethodPost, model.CreatePageRequest{
		Title: "Valid Title",
	})
	s.Created()
	page := s.GetResponse().Data

	past := time.Now().Add(-1 * time.Hour)
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d/schedule", ws.ID, page.ID), fiber.MethodPost, model.SchedulePageRequest{
		PublishAt: past,
	})
	s.AssertStatus(fiber.StatusUnprocessableEntity)
}

func (s *PageSuite) Test_SlugConflict_409() {
	ws := s.ensureWorkspace("Page Slug WS", s.User.ID)

	slug := "custom-slug"
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages", ws.ID), fiber.MethodPost, model.CreatePageRequest{
		Title: "Page 1",
		Slug:  &slug,
	})
	s.Created()

	// Try creating page 2 with same explicit slug -> 409
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages", ws.ID), fiber.MethodPost, model.CreatePageRequest{
		Title: "Page 2",
		Slug:  &slug,
	})
	s.AssertStatus(fiber.StatusConflict)
}

func (s *PageSuite) Test_Permission_Matrix() {
	ws := s.ensureWorkspace("Page Perms WS", s.User.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages", ws.ID), fiber.MethodPost, model.CreatePageRequest{
		Title: "Seed Page",
	})
	s.Created()
	page := s.GetResponse().Data

	// 1. Member with NO role -> 403
	norole := s.AsUser("norole-page@example.com", "No Role User")
	s.AddMember(ws.ID, norole.ID)

	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d", ws.ID, page.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusForbidden)

	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages", ws.ID), fiber.MethodPost, model.CreatePageRequest{Title: "Fail"})
	s.AssertStatus(fiber.StatusForbidden)

	// 2. Viewer role: can read, cannot create/update/delete
	viewer := s.AsUser("viewer-page@example.com", "Viewer User")
	s.AddMember(ws.ID, viewer.ID, domain.RoleViewer)

	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d", ws.ID, page.ID), fiber.MethodGet, nil)
	s.OK()

	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages", ws.ID), fiber.MethodPost, model.CreatePageRequest{Title: "Fail"})
	s.AssertStatus(fiber.StatusForbidden)

	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d", ws.ID, page.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusForbidden)

	// 3. Editor role: can write
	editor := s.AsUser("editor-page@example.com", "Editor User")
	s.AddMember(ws.ID, editor.ID, domain.RoleEditor)

	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages", ws.ID), fiber.MethodPost, model.CreatePageRequest{Title: "Editor Created"})
	s.Created()

	// Editor can publish
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d/publish", ws.ID, page.ID), fiber.MethodPost, nil)
	s.OK()
}

func (s *PageSuite) Test_Unauthenticated_401() {
	ws := s.ensureWorkspace("Page Unauth WS", s.User.ID)

	s.SessionToken = ""
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages", ws.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusUnauthorized)
}

func (s *PageSuite) Test_CrossWorkspace_Isolation() {
	ws1 := s.ensureWorkspace("Page WS 1", s.User.ID)
	ws2 := s.ensureWorkspace("Page WS 2", s.User.ID)

	// Create page in WS 1
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages", ws1.ID), fiber.MethodPost, model.CreatePageRequest{
		Title: "WS1 Page",
	})
	s.Created()
	page1 := s.GetResponse().Data

	// Access page1 from ws2 -> 404
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d", ws2.ID, page1.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)

	// Update page1 from ws2 -> 404
	newTitle := "Hacked"
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d", ws2.ID, page1.ID), fiber.MethodPut, model.UpdatePageRequest{Title: &newTitle})
	s.AssertStatus(fiber.StatusNotFound)

	// Delete page1 from ws2 -> 404
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/%d", ws2.ID, page1.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusNotFound)

	// Search in ws2 should not return ws1 page
	s.Request(fmt.Sprintf("/workspaces/%d/cms/pages/search", ws2.ID), fiber.MethodPost, model.SearchPageRequest{})
	s.OK()
	s.Empty(s.PagedResponse().Data)
}
