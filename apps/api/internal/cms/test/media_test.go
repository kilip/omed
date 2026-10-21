package test

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type MediaSuite struct {
	testutil.ApiTestSuite[model.MediaResponse]
}

func TestMediaSuite(t *testing.T) {
	suite.Run(t, new(MediaSuite))
}

func (s *MediaSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *MediaSuite) uploadMedia(wid int64, filename string, content []byte, fields map[string]string) {
	s.T().Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	if filename != "" {
		part, err := w.CreateFormFile("file", filename)
		s.Require().NoError(err)
		_, err = part.Write(content)
		s.Require().NoError(err)
	}
	for k, v := range fields {
		s.Require().NoError(w.WriteField(k, v))
	}
	s.Require().NoError(w.Close())

	req := httptest.NewRequest(fiber.MethodPost, fmt.Sprintf("/workspaces/%d/cms/media", wid), &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if s.SessionToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.SessionToken)
	}

	resp, err := testutil.GetState().FiberApp.Test(req)
	s.Require().NoError(err)
	s.HttpResponse = resp
}

func (s *MediaSuite) Test_Upload_Get_Update_Delete_Lifecycle() {
	ws := s.ensureWorkspace("Media Lifecycle WS", s.User.ID)
	payload := []byte("fake-image-bytes-for-png")

	// 1. Upload
	s.uploadMedia(ws.ID, "cover.png", payload, map[string]string{
		"title":    "Homepage Cover",
		"altText":  "Beautiful scenery",
		"caption":  "Photo of mountains",
		"isPublic": "true",
	})
	s.Created()

	uploaded := s.GetResponse().Data
	s.Positive(uploaded.ID)
	s.Equal("Homepage Cover", uploaded.Title)
	s.Equal("Beautiful scenery", uploaded.AltText)
	s.Equal("Photo of mountains", uploaded.Caption)
	s.True(uploaded.IsPublic)
	s.Equal("cover.png", uploaded.Name)
	s.Equal("image/png", uploaded.MimeType)
	s.Equal(int64(len(payload)), uploaded.Size)
	s.NotEmpty(uploaded.URL)

	// 2. GetByID
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws.ID, uploaded.ID), fiber.MethodGet, nil)
	s.OK()
	fetched := s.GetResponse().Data
	s.Equal(uploaded.ID, fetched.ID)
	s.Equal("Homepage Cover", fetched.Title)
	s.NotEmpty(fetched.URL)

	// 3. Update
	newTitle := "Updated Cover Title"
	newAlt := "Updated Alt"
	newCaption := "Updated Caption"
	newPublic := false
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws.ID, uploaded.ID), fiber.MethodPut, model.UpdateMediaRequest{
		Title:    &newTitle,
		AltText:  &newAlt,
		Caption:  &newCaption,
		IsPublic: &newPublic,
	})
	s.OK()
	updated := s.GetResponse().Data
	s.Equal("Updated Cover Title", updated.Title)
	s.Equal("Updated Alt", updated.AltText)
	s.Equal("Updated Caption", updated.Caption)
	s.False(updated.IsPublic)

	// 4. Delete
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws.ID, uploaded.ID), fiber.MethodDelete, nil)
	s.OK()

	// 5. Verify 404 after delete
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws.ID, uploaded.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *MediaSuite) Test_Search_And_Pagination() {
	ws := s.ensureWorkspace("Media Search WS", s.User.ID)

	s.uploadMedia(ws.ID, "alpha.png", []byte("alpha"), map[string]string{
		"title": "Alpha Banner",
	})
	s.Created()
	m1 := s.GetResponse().Data

	s.uploadMedia(ws.ID, "report.pdf", []byte("pdf-content"), map[string]string{
		"title": "Quarterly Report",
	})
	s.Created()
	m2 := s.GetResponse().Data

	s.uploadMedia(ws.ID, "gamma.jpg", []byte("gamma"), map[string]string{
		"title": "Gamma Photo",
	})
	s.Created()
	m3 := s.GetResponse().Data

	// Search by query
	q := "Report"
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/search", ws.ID), fiber.MethodPost, model.SearchMediaRequest{
		Query: &q,
	})
	s.OK()
	searchRes := s.PagedResponse()
	s.Require().Len(searchRes.Data, 1)
	s.Equal(m2.ID, searchRes.Data[0].ID)

	// Search by mimeType
	mime := "image"
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/search", ws.ID), fiber.MethodPost, model.SearchMediaRequest{
		MimeType: &mime,
	})
	s.OK()
	mimeRes := s.PagedResponse()
	s.Require().Len(mimeRes.Data, 2)

	// Pagination: limit 2
	limit := 2
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/search", ws.ID), fiber.MethodPost, model.SearchMediaRequest{
		Limit: &limit,
	})
	s.OK()
	page1 := s.PagedResponse()
	s.Require().Len(page1.Data, 2)
	s.Equal(m3.ID, page1.Data[0].ID)
	s.Equal(m2.ID, page1.Data[1].ID)
	s.NotEmpty(page1.Meta.Cursor)

	// Page 2
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/search", ws.ID), fiber.MethodPost, model.SearchMediaRequest{
		Limit:  &limit,
		Cursor: &page1.Meta.Cursor,
	})
	s.OK()
	page2 := s.PagedResponse()
	s.Require().Len(page2.Data, 1)
	s.Equal(m1.ID, page2.Data[0].ID)
}

func (s *MediaSuite) Test_Delete_Conflict_WhenInUse() {
	ws := s.ensureWorkspace("Media InUse WS", s.User.ID)

	s.uploadMedia(ws.ID, "cover.png", []byte("image-data"), map[string]string{
		"title": "Cover For Post",
	})
	s.Created()
	media := s.GetResponse().Data

	// Create post referencing media as cover_media_id
	client := testutil.GetEntCli()
	post, err := client.CmsPost.Create().
		SetWorkspaceID(ws.ID).
		SetTitle("Post Using Media").
		SetSlug("post-using-media").
		SetCoverMediaID(media.ID).
		Save(s.Context)
	s.Require().NoError(err)

	// Delete must fail with 409 Conflict
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws.ID, media.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusConflict)

	// Soft-delete the referencing post
	err = client.CmsPost.DeleteOneID(post.ID).Exec(s.Context)
	s.Require().NoError(err)

	// Delete media must now succeed
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws.ID, media.ID), fiber.MethodDelete, nil)
	s.OK()
}

func (s *MediaSuite) Test_Authorization_MemberWithoutRole_403() {
	ws := s.ensureWorkspace("Media Auth WS", s.User.ID)

	s.uploadMedia(ws.ID, "cover.png", []byte("image-data"), map[string]string{
		"title": "Cover",
	})
	s.Created()
	media := s.GetResponse().Data

	// Switch to a member with NO roles
	member := s.AsUser("norole@example.com", "No Role User")
	s.AddMember(ws.ID, member.ID)

	// GET -> 403
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws.ID, media.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusForbidden)

	// Search -> 403
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/search", ws.ID), fiber.MethodPost, model.SearchMediaRequest{})
	s.AssertStatus(fiber.StatusForbidden)

	// Upload -> 403
	s.uploadMedia(ws.ID, "test.png", []byte("data"), nil)
	s.AssertStatus(fiber.StatusForbidden)

	// Update -> 403
	title := "Hacked"
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws.ID, media.ID), fiber.MethodPut, model.UpdateMediaRequest{Title: &title})
	s.AssertStatus(fiber.StatusForbidden)

	// Delete -> 403
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws.ID, media.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusForbidden)
}

func (s *MediaSuite) Test_Authorization_Roles_Viewer_And_Editor() {
	ws := s.ensureWorkspace("Media Roles WS", s.User.ID)

	s.uploadMedia(ws.ID, "cover.png", []byte("image-data"), map[string]string{
		"title": "Cover",
	})
	s.Created()
	media := s.GetResponse().Data

	// 1. Viewer member
	viewer := s.AsUser("viewer@example.com", "Viewer User")
	s.AddMember(ws.ID, viewer.ID, domain.RoleViewer)

	// Viewer can GET
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws.ID, media.ID), fiber.MethodGet, nil)
	s.OK()

	// Viewer can Search
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/search", ws.ID), fiber.MethodPost, model.SearchMediaRequest{})
	s.OK()

	// Viewer cannot upload -> 403
	s.uploadMedia(ws.ID, "test.png", []byte("data"), nil)
	s.AssertStatus(fiber.StatusForbidden)

	// Viewer cannot delete -> 403
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws.ID, media.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusForbidden)

	// 2. Editor member
	editor := s.AsUser("editor@example.com", "Editor User")
	s.AddMember(ws.ID, editor.ID, domain.RoleEditor)

	// Editor can upload
	s.uploadMedia(ws.ID, "editor-upload.png", []byte("data"), map[string]string{"title": "Editor Upload"})
	s.Created()
	editorMedia := s.GetResponse().Data

	// Editor can update
	t := "Updated By Editor"
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws.ID, editorMedia.ID), fiber.MethodPut, model.UpdateMediaRequest{Title: &t})
	s.OK()

	// Editor can delete
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws.ID, editorMedia.ID), fiber.MethodDelete, nil)
	s.OK()
}

func (s *MediaSuite) Test_Unauthenticated_401() {
	ws := s.ensureWorkspace("Media Unauth WS", s.User.ID)

	s.uploadMedia(ws.ID, "cover.png", []byte("image-data"), nil)
	s.Created()
	media := s.GetResponse().Data

	s.SessionToken = ""

	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws.ID, media.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusUnauthorized)

	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/search", ws.ID), fiber.MethodPost, model.SearchMediaRequest{})
	s.AssertStatus(fiber.StatusUnauthorized)
}

func (s *MediaSuite) Test_CrossWorkspace_Isolation() {
	ws1 := s.ensureWorkspace("Workspace 1", s.User.ID)
	ws2 := s.ensureWorkspace("Workspace 2", s.User.ID)

	s.uploadMedia(ws1.ID, "ws1.png", []byte("ws1-bytes"), map[string]string{
		"title": "WS1 Media",
	})
	s.Created()
	media1 := s.GetResponse().Data

	// Access media1 from ws2 -> 404
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws2.ID, media1.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)

	// Update media1 from ws2 -> 404
	newTitle := "Hacked"
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws2.ID, media1.ID), fiber.MethodPut, model.UpdateMediaRequest{Title: &newTitle})
	s.AssertStatus(fiber.StatusNotFound)

	// Delete media1 from ws2 -> 404
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/%d", ws2.ID, media1.ID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusNotFound)

	// Search in ws2 should not return ws1 media
	s.Request(fmt.Sprintf("/workspaces/%d/cms/media/search", ws2.ID), fiber.MethodPost, model.SearchMediaRequest{})
	s.OK()
	s.Empty(s.PagedResponse().Data)
}
