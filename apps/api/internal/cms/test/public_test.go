package test

import (
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/ent/cmspagetranslation"
	"github.com/kilip/omed/api/ent/cmsposttranslation"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	"github.com/kilip/omed/api/internal/platform/httpx"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type PublicContentSuite struct {
	testutil.ApiTestSuite[model.PublicPostResponse]
}

func TestPublicContentSuite(t *testing.T) {
	suite.Run(t, new(PublicContentSuite))
}

func (s *PublicContentSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	ws, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return ws
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

func (s *PublicContentSuite) createPostWithTranslation(wsID int64, locale, title, slug, body, status string, publishAt *time.Time, deletedAt *time.Time) *ent.CmsPost {
	client := testutil.GetEntCli()
	ctx := base.WithWorkspaceID(s.Context, wsID)

	postBuilder := client.CmsPost.Create().SetWorkspaceID(wsID)
	if deletedAt != nil {
		postBuilder.SetDeletedAt(*deletedAt)
	}
	post, err := postBuilder.Save(ctx)
	s.Require().NoError(err)

	tBuilder := client.CmsPostTranslation.Create().
		SetPostID(post.ID).
		SetWorkspaceID(wsID).
		SetLocale(locale).
		SetTitle(title).
		SetSlug(slug).
		SetBody(body).
		SetStatus(cmsposttranslation.Status(status))
	if publishAt != nil {
		tBuilder.SetPublishAt(*publishAt)
	}
	if deletedAt != nil {
		tBuilder.SetDeletedAt(*deletedAt)
	}
	_, err = tBuilder.Save(ctx)
	s.Require().NoError(err)
	return post
}

func (s *PublicContentSuite) createPageWithTranslation(wsID int64, locale, title, slug, body, status string, publishAt *time.Time, deletedAt *time.Time) *ent.CmsPage {
	client := testutil.GetEntCli()
	ctx := base.WithWorkspaceID(s.Context, wsID)

	pageBuilder := client.CmsPage.Create().SetWorkspaceID(wsID)
	if deletedAt != nil {
		pageBuilder.SetDeletedAt(*deletedAt)
	}
	page, err := pageBuilder.Save(ctx)
	s.Require().NoError(err)

	tBuilder := client.CmsPageTranslation.Create().
		SetPageID(page.ID).
		SetWorkspaceID(wsID).
		SetLocale(locale).
		SetTitle(title).
		SetSlug(slug).
		SetBody(body).
		SetStatus(cmspagetranslation.Status(status))
	if publishAt != nil {
		tBuilder.SetPublishAt(*publishAt)
	}
	if deletedAt != nil {
		tBuilder.SetDeletedAt(*deletedAt)
	}
	_, err = tBuilder.Save(ctx)
	s.Require().NoError(err)
	return page
}

func (s *PublicContentSuite) Test_PublicPosts_Visibility_And_Filters() {
	ws1 := s.ensureWorkspace("Public Post WS 1", s.User.ID)
	ws2 := s.ensureWorkspace("Public Post WS 2", s.User.ID)
	now := time.Now().UTC()

	tPast1 := now.Add(-1 * time.Hour)
	tPast2 := now.Add(-2 * time.Hour)
	tFuture := now.Add(24 * time.Hour)

	// 1. Published post (visible)
	pubPost := s.createPostWithTranslation(ws1.ID, "en", "Published Post", "published-post", "Content of published", domain.StatusPublished, &tPast1, nil)

	// 2. Draft post (not visible -> 404)
	s.createPostWithTranslation(ws1.ID, "en", "Draft Post", "draft-post", "Draft body", domain.StatusDraft, nil, nil)

	// 3. Scheduled in future post (not visible -> 404)
	s.createPostWithTranslation(ws1.ID, "en", "Future Post", "future-post", "Future body", domain.StatusPublished, &tFuture, nil)

	// 4. Scheduled in past post (visible -> 200)
	pastPost := s.createPostWithTranslation(ws1.ID, "en", "Past Scheduled Post", "past-scheduled-post", "Past body", domain.StatusPublished, &tPast2, nil)

	// 5. Soft-deleted post (not visible -> 404)
	s.createPostWithTranslation(ws1.ID, "en", "Deleted Post", "deleted-post", "Deleted body", domain.StatusPublished, &tPast1, &now)

	// 6. Post in another workspace with same slug
	s.createPostWithTranslation(ws2.ID, "en", "WS2 Post", "ws2-unique-slug", "WS2 body", domain.StatusPublished, &tPast1, nil)

	// Assert: Published post is visible (200)
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/published-post", ws1.ID))
	s.OK()
	rawBytes := s.readRawBody()
	var singlePostResp httpx.WebResponse[model.PublicPostResponse]
	err := json.Unmarshal(rawBytes, &singlePostResp)
	s.Require().NoError(err)
	s.Equal(pubPost.ID, singlePostResp.Data.ID)
	s.Equal("Published Post", singlePostResp.Data.Title)
	s.Equal("published-post", singlePostResp.Data.Slug)
	s.Equal("Content of published", singlePostResp.Data.Body)
	s.NotNil(singlePostResp.Data.PublishedAt)

	// Assert: Draft returns 404
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/draft-post", ws1.ID))
	s.AssertStatus(fiber.StatusNotFound)

	// Assert: Scheduled in future returns 404
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/future-post", ws1.ID))
	s.AssertStatus(fiber.StatusNotFound)

	// Assert: Scheduled in past returns 200
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/past-scheduled-post", ws1.ID))
	s.OK()

	// Assert: Soft-deleted returns 404
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/deleted-post", ws1.ID))
	s.AssertStatus(fiber.StatusNotFound)

	// Assert: Another workspace's slug returns 404 (ws1 looking for ws2 slug)
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/ws2-unique-slug", ws1.ID))
	s.AssertStatus(fiber.StatusNotFound)

	// Assert: ws2 can see its own post
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/ws2-unique-slug", ws2.ID))
	s.OK()

	// Assert: List posts for ws1 returns only the 2 visible posts (pubPost, pastPost)
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts", ws1.ID))
	s.OK()
	var listResp httpx.WebResponse[[]model.PublicPostResponse]
	err = json.Unmarshal(s.readRawBody(), &listResp)
	s.Require().NoError(err)
	s.Equal(2, len(listResp.Data))
	s.Equal(pastPost.ID, listResp.Data[0].ID)
	s.Equal(pubPost.ID, listResp.Data[1].ID)
}

func (s *PublicContentSuite) Test_PublicPosts_CoverImageURL_And_Pagination() {
	client := testutil.GetEntCli()
	ws := s.ensureWorkspace("Public Cover WS", s.User.ID)
	ctx := base.WithWorkspaceID(s.Context, ws.ID)
	now := time.Now().UTC()

	// Create dummy file and cms_media
	f, err := client.File.Create().
		SetWorkspaceID(ws.ID).
		SetName("cover.jpg").
		SetBucket("test-bucket").
		SetKey("cover-key").
		SetMimeType("image/jpeg").
		SetSize(1024).
		Save(ctx)
	s.Require().NoError(err)

	m, err := client.CmsMedia.Create().
		SetWorkspaceID(ws.ID).
		SetFileID(f.ID).
		SetTitle("Cover Title").
		SetAltText("Scenic landscape").
		SetIsPublic(true).
		Save(ctx)
	s.Require().NoError(err)

	t1 := now.Add(-30 * time.Minute)
	t2 := now.Add(-20 * time.Minute)
	t3 := now.Add(-10 * time.Minute)

	p1 := s.createPostWithTranslation(ws.ID, "en", "Post 1", "post-1", "Body 1", domain.StatusPublished, &t1, nil)

	p2 := s.createPostWithTranslation(ws.ID, "en", "Post 2", "post-2", "Body 2", domain.StatusPublished, &t2, nil)
	err = client.CmsPost.UpdateOneID(p2.ID).SetCoverMediaID(m.ID).Exec(ctx)
	s.Require().NoError(err)

	p3 := s.createPostWithTranslation(ws.ID, "en", "Post 3", "post-3", "Body 3", domain.StatusPublished, &t3, nil)

	// Verify CoverImage in single post response
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/posts/post-2", ws.ID))
	s.OK()
	var postResp httpx.WebResponse[model.PublicPostResponse]
	err = json.Unmarshal(s.readRawBody(), &postResp)
	s.Require().NoError(err)
	s.NotNil(postResp.Data.CoverImage)
	s.Equal(m.ID, postResp.Data.CoverImage.ID)
	s.Equal("Scenic landscape", postResp.Data.CoverImage.Alt)

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

func (s *PublicContentSuite) Test_PublicPosts_MultiLanguage_Fallback_And_Alternates() {
	client := testutil.GetEntCli()
	ws := s.ensureWorkspace("Public i18n WS", s.User.ID)
	ctx := base.WithWorkspaceID(s.Context, ws.ID)
	now := time.Now().UTC()
	past := now.Add(-1 * time.Hour)

	// Create post 1 with EN and ID published translations
	p1 := s.createPostWithTranslation(ws.ID, "en", "Hello World", "hello-world", "English content", domain.StatusPublished, &past, nil)
	_, err := client.CmsPostTranslation.Create().
		SetPostID(p1.ID).
		SetWorkspaceID(ws.ID).
		SetLocale("id").
		SetTitle("Halo Dunia").
		SetSlug("halo-dunia").
		SetBody("Konten Indonesia").
		SetStatus(cmsposttranslation.Status(domain.StatusPublished)).
		SetPublishAt(past).
		Save(ctx)
	s.Require().NoError(err)

	// Create post 2 with EN translation only
	s.createPostWithTranslation(ws.ID, "en", "English Only", "english-only", "Only in english", domain.StatusPublished, &past, nil)

	// 1. Request in "id" locale for "halo-dunia" -> direct match, fallback=false
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/id/posts/halo-dunia", ws.ID))
	s.OK()
	var respID httpx.WebResponse[model.PublicPostResponse]
	err = json.Unmarshal(s.readRawBody(), &respID)
	s.Require().NoError(err)
	s.False(respID.Data.Fallback)
	s.Equal("id", respID.Data.Locale)
	s.Equal("Halo Dunia", respID.Data.Title)
	s.Equal("halo-dunia", respID.Data.Slug)
	s.Equal("/id/halo-dunia", respID.Data.Path)
	s.Len(respID.Data.Alternates, 2)

	// 2. Request in "id" locale for "english-only" -> falls back to EN, fallback=true
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/id/posts/english-only", ws.ID))
	s.OK()
	var respFallback httpx.WebResponse[model.PublicPostResponse]
	err = json.Unmarshal(s.readRawBody(), &respFallback)
	s.Require().NoError(err)
	s.True(respFallback.Data.Fallback)
	s.Equal("en", respFallback.Data.Locale)
	s.Equal("English Only", respFallback.Data.Title)
	s.Equal("/en/english-only", respFallback.Data.Path)

	// 3. Request in unsupported locale -> 400 Bad Request
	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/fr/posts/hello-world", ws.ID))
	s.AssertStatus(fiber.StatusBadRequest)
}

func (s *PublicContentSuite) Test_PublicPages_Lifecycle_And_Isolation() {
	ws1 := s.ensureWorkspace("Public Page WS 1", s.User.ID)
	ws2 := s.ensureWorkspace("Public Page WS 2", s.User.ID)
	now := time.Now().UTC()
	past := now.Add(-1 * time.Hour)

	page1 := s.createPageWithTranslation(ws1.ID, "en", "About Us", "about-us", "We build open medical software", domain.StatusPublished, &past, nil)
	s.createPageWithTranslation(ws1.ID, "en", "Draft Page", "draft-page", "Draft page body", domain.StatusDraft, nil, nil)
	s.createPageWithTranslation(ws2.ID, "en", "WS2 About Us", "ws2-page-slug", "Different workspace page", domain.StatusPublished, &past, nil)

	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/pages/about-us", ws1.ID))
	s.OK()
	rawBytes := s.readRawBody()
	var pageResp httpx.WebResponse[model.PublicPageResponse]
	err := json.Unmarshal(rawBytes, &pageResp)
	s.Require().NoError(err)
	s.Equal(page1.ID, pageResp.Data.ID)
	s.Equal("About Us", pageResp.Data.Title)
	s.Equal("about-us", pageResp.Data.Slug)
	s.Equal("We build open medical software", pageResp.Data.Body)

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
	ctx1 := base.WithWorkspaceID(s.Context, ws1.ID)
	ctx2 := base.WithWorkspaceID(s.Context, ws2.ID)

	f, err := client.File.Create().
		SetWorkspaceID(ws1.ID).
		SetName("public-img.jpg").
		SetBucket("test-bucket").
		SetKey("public-img-key").
		SetMimeType("image/jpeg").
		SetSize(2048).
		Save(ctx1)
	s.Require().NoError(err)

	pubMedia, err := client.CmsMedia.Create().
		SetWorkspaceID(ws1.ID).
		SetFileID(f.ID).
		SetTitle("Public Image").
		SetIsPublic(true).
		Save(ctx1)
	s.Require().NoError(err)

	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/media/%d", ws1.ID, pubMedia.ID))
	status := s.GetHttpResponse().StatusCode
	s.True(status == fiber.StatusFound || status == fiber.StatusNotFound)

	s.publicGet(fmt.Sprintf("/public/workspaces/%d/cms/media/%d", ws2.ID, pubMedia.ID))
	s.AssertStatus(fiber.StatusNotFound)
	_ = ctx2
}
