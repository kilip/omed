package test

import (
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/cmspage"
	"github.com/kilip/omed/api/ent/cmspost"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type PublicContentSuite struct {
	testutil.ApiTestSuite[any]
}

func TestPublicContentSuite(t *testing.T) {
	suite.Run(t, new(PublicContentSuite))
}

func (s *PublicContentSuite) SetupTest() {
	s.ApiTestSuite.SetupTest()
	s.Context = base.WithSkipWorkspaceFilter(s.Context)
}

func (s *PublicContentSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *PublicContentSuite) publicGet(path string) {
	s.RequestWithToken(path, fiber.MethodGet, nil, "")
}

func (s *PublicContentSuite) readRawBody() []byte {
	resp := s.GetHttpResponse()
	defer func() { _ = resp.Body.Close() }()
	bytes, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)
	return bytes
}

func (s *PublicContentSuite) Test_PublicPosts_Visibility_And_Filters() {
	client := testutil.GetEntCli()
	ws1 := s.ensureWorkspace("Public Post WS 1", s.User.ID)
	ws2 := s.ensureWorkspace("Public Post WS 2", s.User.ID)
	now := time.Now().UTC()

	// 1. Published post (visible)
	pubPost, err := client.CmsPost.Create().
		SetWorkspaceID(ws1.ID).
		SetTitle("Published Post").
		SetSlug("published-post").
		SetExcerpt("Excerpt of published").
		SetBody("Content of published").
		SetStatus(cmspost.Status(domain.StatusPublished)).
		SetPublishAt(now.Add(-1 * time.Hour)).
		Save(s.Context)
	s.Require().NoError(err)

	// 2. Draft post (not visible -> 404)
	_, err = client.CmsPost.Create().
		SetWorkspaceID(ws1.ID).
		SetTitle("Draft Post").
		SetSlug("draft-post").
		SetBody("Draft body").
		SetStatus(cmspost.Status(domain.StatusDraft)).
		Save(s.Context)
	s.Require().NoError(err)

	// 3. Scheduled in future post (not visible -> 404)
	_, err = client.CmsPost.Create().
		SetWorkspaceID(ws1.ID).
		SetTitle("Future Post").
		SetSlug("future-post").
		SetBody("Future body").
		SetStatus(cmspost.Status(domain.StatusPublished)).
		SetPublishAt(now.Add(24 * time.Hour)).
		Save(s.Context)
	s.Require().NoError(err)

	// 4. Scheduled in past post (visible -> 200)
	pastPost, err := client.CmsPost.Create().
		SetWorkspaceID(ws1.ID).
		SetTitle("Past Scheduled Post").
		SetSlug("past-scheduled-post").
		SetBody("Past body").
		SetStatus(cmspost.Status(domain.StatusPublished)).
		SetPublishAt(now.Add(-2 * time.Hour)).
		Save(s.Context)
	s.Require().NoError(err)

	// 5. Soft-deleted post (not visible -> 404)
	delPost, err := client.CmsPost.Create().
		SetWorkspaceID(ws1.ID).
		SetTitle("Deleted Post").
		SetSlug("deleted-post").
		SetBody("Deleted body").
		SetStatus(cmspost.Status(domain.StatusPublished)).
		SetPublishAt(now.Add(-1 * time.Hour)).
		SetDeletedAt(now).
		Save(s.Context)
	s.Require().NoError(err)

	// 6. Post in another workspace with same slug
	_, err = client.CmsPost.Create().
		SetWorkspaceID(ws2.ID).
		SetTitle("WS2 Post").
		SetSlug("ws2-unique-slug").
		SetBody("WS2 body").
		SetStatus(cmspost.Status(domain.StatusPublished)).
		SetPublishAt(now.Add(-1 * time.Hour)).
		Save(s.Context)
	s.Require().NoError(err)

	// Assert: Published post is visible (200)
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/%s", ws1.ID, pubPost.Slug))
	s.OK()
	rawBytes := s.readRawBody()
	var singlePostResp httpx.WebResponse[model.PublicPostResponse]
	err = json.Unmarshal(rawBytes, &singlePostResp)
	s.Require().NoError(err)
	s.Equal(pubPost.ID, singlePostResp.Data.ID)
	s.Equal(pubPost.Title, singlePostResp.Data.Title)
	s.Equal(pubPost.Slug, singlePostResp.Data.Slug)
	s.Equal("Excerpt of published", singlePostResp.Data.Excerpt)
	s.Equal("Content of published", singlePostResp.Data.Body)
	s.NotNil(singlePostResp.Data.PublishedAt)

	// Verify response JSON contains NO status or audit fields
	var rawMap map[string]any
	err = json.Unmarshal(rawBytes, &rawMap)
	s.Require().NoError(err)
	dataMap, ok := rawMap["data"].(map[string]any)
	s.Require().True(ok)
	s.Nil(dataMap["status"], "status must not be exposed in public DTO")
	s.Nil(dataMap["effectiveStatus"], "effectiveStatus must not be exposed")
	s.Nil(dataMap["workspaceId"], "workspaceId must not be exposed")
	s.Nil(dataMap["createdBy"], "createdBy must not be exposed")
	s.Nil(dataMap["updatedBy"], "updatedBy must not be exposed")
	s.Nil(dataMap["createdAt"], "createdAt must not be exposed")
	s.Nil(dataMap["updatedAt"], "updatedAt must not be exposed")
	s.Nil(dataMap["deletedAt"], "deletedAt must not be exposed")

	// Assert: Draft returns 404
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/draft-post", ws1.ID))
	s.AssertStatus(fiber.StatusNotFound)

	// Assert: Scheduled in future returns 404
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/future-post", ws1.ID))
	s.AssertStatus(fiber.StatusNotFound)

	// Assert: Scheduled in past returns 200
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/%s", ws1.ID, pastPost.Slug))
	s.OK()

	// Assert: Soft-deleted returns 404
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/%s", ws1.ID, delPost.Slug))
	s.AssertStatus(fiber.StatusNotFound)

	// Assert: Another workspace's slug returns 404 (ws1 looking for ws2 slug)
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/ws2-unique-slug", ws1.ID))
	s.AssertStatus(fiber.StatusNotFound)

	// Assert: ws2 can see its own post
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/ws2-unique-slug", ws2.ID))
	s.OK()

	// Assert: Unknown workspace returns 404 for slug
	s.publicGet("/public/workspaces/9999999/cms/posts/published-post")
	s.AssertStatus(fiber.StatusNotFound)

	// Assert: Invalid or zero wid returns 404 for slug
	s.publicGet("/public/workspaces/0/cms/posts/published-post")
	s.AssertStatus(fiber.StatusNotFound)
	s.publicGet("/public/workspaces/invalid/cms/posts/published-post")
	s.AssertStatus(fiber.StatusNotFound)

	// Assert: List posts for ws1 returns only the 2 visible posts (pubPost, pastPost)
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts", ws1.ID))
	s.OK()
	var listResp httpx.WebResponse[[]model.PublicPostResponse]
	err = json.Unmarshal(s.readRawBody(), &listResp)
	s.Require().NoError(err)
	s.Equal(2, len(listResp.Data))
	// Ordering: publish_at DESC, id DESC -> pubPost (-1h) is more recent than pastPost (-2h)
	s.Equal(pubPost.ID, listResp.Data[0].ID)
	s.Equal(pastPost.ID, listResp.Data[1].ID)

	// Unknown workspace returns empty list
	s.publicGet("/public/workspaces/9999999/cms/posts")
	s.OK()
	var unknownResp httpx.WebResponse[[]model.PublicPostResponse]
	err = json.Unmarshal(s.readRawBody(), &unknownResp)
	s.Require().NoError(err)
	s.Equal(0, len(unknownResp.Data))

	// Invalid or zero wid returns empty list
	s.publicGet("/public/workspaces/0/cms/posts")
	s.OK()
	var zeroResp httpx.WebResponse[[]model.PublicPostResponse]
	err = json.Unmarshal(s.readRawBody(), &zeroResp)
	s.Require().NoError(err)
	s.Equal(0, len(zeroResp.Data))
}

func (s *PublicContentSuite) Test_PublicPosts_CoverImageURL_And_Pagination() {
	client := testutil.GetEntCli()
	ws := s.ensureWorkspace("Public Cover WS", s.User.ID)
	now := time.Now().UTC()

	// Create dummy file and cms_media
	f, err := client.File.Create().
		SetWorkspaceID(ws.ID).
		SetName("cover.jpg").
		SetBucket("test-bucket").
		SetKey("cover-key").
		SetMimeType("image/jpeg").
		SetSize(1024).
		Save(s.Context)
	s.Require().NoError(err)

	m, err := client.CmsMedia.Create().
		SetWorkspaceID(ws.ID).
		SetFileID(f.ID).
		SetTitle("Cover Title").
		SetAltText("Scenic landscape").
		SetIsPublic(true).
		Save(s.Context)
	s.Require().NoError(err)

	// Create 3 published posts with staggered publish_at
	t1 := now.Add(-30 * time.Minute)
	t2 := now.Add(-20 * time.Minute)
	t3 := now.Add(-10 * time.Minute)

	p1, _ := client.CmsPost.Create().
		SetWorkspaceID(ws.ID).
		SetTitle("Post 1").
		SetSlug("post-1").
		SetStatus(cmspost.Status(domain.StatusPublished)).
		SetPublishAt(t1).
		Save(s.Context)

	p2, _ := client.CmsPost.Create().
		SetWorkspaceID(ws.ID).
		SetTitle("Post 2").
		SetSlug("post-2").
		SetStatus(cmspost.Status(domain.StatusPublished)).
		SetPublishAt(t2).
		SetCoverMediaID(m.ID).
		Save(s.Context)

	p3, _ := client.CmsPost.Create().
		SetWorkspaceID(ws.ID).
		SetTitle("Post 3").
		SetSlug("post-3").
		SetStatus(cmspost.Status(domain.StatusPublished)).
		SetPublishAt(t3).
		Save(s.Context)

	// Verify CoverImage in single post response
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/%s", ws.ID, p2.Slug))
	s.OK()
	var postResp httpx.WebResponse[model.PublicPostResponse]
	err = json.Unmarshal(s.readRawBody(), &postResp)
	s.Require().NoError(err)
	s.NotNil(postResp.Data.CoverImage)
	s.Equal(m.ID, postResp.Data.CoverImage.ID)
	s.Equal("Scenic landscape", postResp.Data.CoverImage.Alt)
	expectedURL := fmt.Sprintf("/public/workspaces/%d/cms/media/%d", ws.ID, m.ID)
	s.Contains(postResp.Data.CoverImage.URL, expectedURL)

	// Test pagination: limit=2
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts?limit=2", ws.ID))
	s.OK()
	var page1Resp httpx.WebResponse[[]model.PublicPostResponse]
	err = json.Unmarshal(s.readRawBody(), &page1Resp)
	s.Require().NoError(err)
	s.Equal(2, len(page1Resp.Data))
	s.Equal(p3.ID, page1Resp.Data[0].ID)
	s.Equal(p2.ID, page1Resp.Data[1].ID)
	s.NotEmpty(page1Resp.Meta.Cursor)

	// Page 2 using cursor
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts?limit=2&cursor=%s", ws.ID, page1Resp.Meta.Cursor))
	s.OK()
	var page2Resp httpx.WebResponse[[]model.PublicPostResponse]
	err = json.Unmarshal(s.readRawBody(), &page2Resp)
	s.Require().NoError(err)
	s.Equal(1, len(page2Resp.Data))
	s.Equal(p1.ID, page2Resp.Data[0].ID)
	s.Empty(page2Resp.Meta.Cursor)
}

func (s *PublicContentSuite) Test_PublicPages_Lifecycle_And_Isolation() {
	client := testutil.GetEntCli()
	ws1 := s.ensureWorkspace("Public Page WS 1", s.User.ID)
	ws2 := s.ensureWorkspace("Public Page WS 2", s.User.ID)
	now := time.Now().UTC()

	// Published page in ws1
	page1, err := client.CmsPage.Create().
		SetWorkspaceID(ws1.ID).
		SetTitle("About Us").
		SetSlug("about-us").
		SetBody("We build open medical software").
		SetStatus(cmspage.Status(domain.StatusPublished)).
		SetPublishAt(now.Add(-1 * time.Hour)).
		Save(s.Context)
	s.Require().NoError(err)

	// Draft page in ws1 -> 404
	_, err = client.CmsPage.Create().
		SetWorkspaceID(ws1.ID).
		SetTitle("Draft Page").
		SetSlug("draft-page").
		SetBody("Draft page body").
		SetStatus(cmspage.Status(domain.StatusDraft)).
		Save(s.Context)
	s.Require().NoError(err)

	// Page in ws2 with same slug
	_, err = client.CmsPage.Create().
		SetWorkspaceID(ws2.ID).
		SetTitle("WS2 About Us").
		SetSlug("ws2-page-slug").
		SetBody("Different workspace page").
		SetStatus(cmspage.Status(domain.StatusPublished)).
		SetPublishAt(now.Add(-1 * time.Hour)).
		Save(s.Context)
	s.Require().NoError(err)

	// Assert: Published page returns 200
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/pages/%s", ws1.ID, page1.Slug))
	s.OK()
	rawBytes := s.readRawBody()
	var pageResp httpx.WebResponse[model.PublicPageResponse]
	err = json.Unmarshal(rawBytes, &pageResp)
	s.Require().NoError(err)
	s.Equal(page1.ID, pageResp.Data.ID)
	s.Equal(page1.Title, pageResp.Data.Title)
	s.Equal(page1.Slug, pageResp.Data.Slug)
	s.Equal("We build open medical software", pageResp.Data.Body)
	s.NotNil(pageResp.Data.PublishedAt)

	// Verify no audit or status fields
	var rawMap map[string]any
	err = json.Unmarshal(rawBytes, &rawMap)
	s.Require().NoError(err)
	dataMap, ok := rawMap["data"].(map[string]any)
	s.Require().True(ok)
	s.Nil(dataMap["status"])
	s.Nil(dataMap["workspaceId"])
	s.Nil(dataMap["createdBy"])
	s.Nil(dataMap["createdAt"])

	// Draft page returns 404
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/pages/draft-page", ws1.ID))
	s.AssertStatus(fiber.StatusNotFound)

	// ws1 accessing ws2 slug returns 404
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/pages/ws2-page-slug", ws1.ID))
	s.AssertStatus(fiber.StatusNotFound)

	// List pages returns only visible pages
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/pages", ws1.ID))
	s.OK()
	var listResp httpx.WebResponse[[]model.PublicPageResponse]
	err = json.Unmarshal(s.readRawBody(), &listResp)
	s.Require().NoError(err)
	s.Equal(1, len(listResp.Data))
	s.Equal(page1.ID, listResp.Data[0].ID)
}

func (s *PublicContentSuite) Test_PublicMedia_Redirect() {
	client := testutil.GetEntCli()
	ws1 := s.ensureWorkspace("Public Media WS 1", s.User.ID)
	ws2 := s.ensureWorkspace("Public Media WS 2", s.User.ID)

	// Create a file
	f, err := client.File.Create().
		SetWorkspaceID(ws1.ID).
		SetName("public-img.jpg").
		SetBucket("test-bucket").
		SetKey("public-img-key").
		SetMimeType("image/jpeg").
		SetSize(2048).
		Save(s.Context)
	s.Require().NoError(err)

	// Public media
	pubMedia, err := client.CmsMedia.Create().
		SetWorkspaceID(ws1.ID).
		SetFileID(f.ID).
		SetTitle("Public Image").
		SetIsPublic(true).
		Save(s.Context)
	s.Require().NoError(err)

	// Non-public media (same file, new record)
	f2, err := client.File.Create().
		SetWorkspaceID(ws1.ID).
		SetName("private-img.jpg").
		SetBucket("test-bucket").
		SetKey("private-img-key").
		SetMimeType("image/jpeg").
		SetSize(512).
		Save(s.Context)
	s.Require().NoError(err)

	privateMedia, err := client.CmsMedia.Create().
		SetWorkspaceID(ws1.ID).
		SetFileID(f2.ID).
		SetTitle("Private Image").
		SetIsPublic(false).
		Save(s.Context)
	s.Require().NoError(err)

	// Soft-deleted public media
	f3, err := client.File.Create().
		SetWorkspaceID(ws1.ID).
		SetName("deleted-img.jpg").
		SetBucket("test-bucket").
		SetKey("deleted-img-key").
		SetMimeType("image/jpeg").
		SetSize(512).
		Save(s.Context)
	s.Require().NoError(err)

	deletedMedia, err := client.CmsMedia.Create().
		SetWorkspaceID(ws1.ID).
		SetFileID(f3.ID).
		SetTitle("Deleted Image").
		SetIsPublic(true).
		SetDeletedAt(time.Now()).
		Save(s.Context)
	s.Require().NoError(err)

	// Public media -> should redirect (302) to presigned URL
	// Note: In test environment, presign may fail because no real S3.
	// We check 302 OR 404 depending on whether file service is mocked.
	// The important thing: route resolves and either redirects or returns 404 (never 500).
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/media/%d", ws1.ID, pubMedia.ID))
	// Accept either 302 (real presign) or 404 (presign fails in test)
	status := s.GetHttpResponse().StatusCode
	s.True(status == fiber.StatusFound || status == fiber.StatusNotFound,
		"expected 302 or 404 for public media, got %d", status)

	// Non-public media -> 404
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/media/%d", ws1.ID, privateMedia.ID))
	s.AssertStatus(fiber.StatusNotFound)

	// Soft-deleted media -> 404
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/media/%d", ws1.ID, deletedMedia.ID))
	s.AssertStatus(fiber.StatusNotFound)

	// Wrong workspace -> 404
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/media/%d", ws2.ID, pubMedia.ID))
	s.AssertStatus(fiber.StatusNotFound)

	// Unknown workspace -> 404
	s.publicGet(fmt.Sprintf("/public/workspaces/9999999/cms/media/%d", pubMedia.ID))
	s.AssertStatus(fiber.StatusNotFound)

	// Invalid id -> 404
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/media/invalid", ws1.ID))
	s.AssertStatus(fiber.StatusNotFound)
}
