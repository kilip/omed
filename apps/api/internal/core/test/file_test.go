package test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/ent"
	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/platform/s3x"
	"github.com/kilip/omed/api/testutil"
	"github.com/stretchr/testify/suite"
)

type FileSuite struct {
	testutil.ApiTestSuite[model.FileResponse]
}

func (s *FileSuite) ensureWorkspace(name string, ownerID int64) *ent.Workspace {
	client := testutil.GetEntCli()
	w, err := client.Workspace.Create().
		SetName(name).
		SetOwnerID(ownerID).
		Save(s.Context)
	s.Require().NoError(err)
	return w
}

func (s *FileSuite) uploadWithFields(wid int64, filename string, content []byte, fields map[string]string) {
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

	req := httptest.NewRequest(fiber.MethodPost, fmt.Sprintf("/workspaces/%d/files", wid), &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if s.SessionToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.SessionToken)
	}

	resp, err := testutil.GetState().FiberApp.Test(req)
	s.Require().NoError(err)
	s.HttpResponse = resp
}

func (s *FileSuite) Test_FullLifecycle_UploadGetSearchDownloadPresignUpdateDelete() {
	ws := s.ensureWorkspace("Files Workspace", s.User.ID)
	payload := []byte("hello workspace s3 attachment content")

	// 1. Upload with custom name and entity metadata
	s.uploadWithFields(ws.ID, "original.txt", payload, map[string]string{
		"name":       "invoice-attachment.txt",
		"entityType": "fin:transaction",
		"entityId":   "42",
	})
	s.Created()

	uploaded := s.GetResponse().Data
	s.Positive(uploaded.ID)
	s.Equal(ws.ID, uploaded.WorkspaceID)
	s.Equal("invoice-attachment.txt", uploaded.Name)
	s.Equal(fmt.Sprintf("omed-ws-%d", ws.ID), uploaded.Bucket)
	s.Contains(uploaded.Key, "fin/transaction/")
	s.Contains(uploaded.Key, ".txt")
	s.Equal(int64(len(payload)), uploaded.Size)
	s.Equal("fin:transaction", uploaded.EntityType)
	s.Equal(int64(42), uploaded.EntityID)

	// Verify bucket and object exist in storage
	st := testutil.GetStorage()
	exists, err := st.BucketExists(context.Background(), uploaded.Bucket)
	s.Require().NoError(err)
	s.True(exists)

	// 2. GetByID
	s.Request(fmt.Sprintf("/workspaces/%d/files/%d", ws.ID, uploaded.ID), fiber.MethodGet, nil)
	s.OK()
	fetched := s.GetResponse().Data
	s.Equal(uploaded.ID, fetched.ID)
	s.Equal("invoice-attachment.txt", fetched.Name)

	// 3. Search
	s.Request(fmt.Sprintf("/workspaces/%d/files/search", ws.ID), fiber.MethodPost, model.SearchFileRequest{
		EntityType: "fin:transaction",
		EntityID:   42,
	})
	s.OK()
	searchRes := s.PagedResponse()
	s.Require().Len(searchRes.Data, 1)
	s.Equal(uploaded.ID, searchRes.Data[0].ID)

	// 4. Download binary stream
	s.Request(fmt.Sprintf("/workspaces/%d/files/%d/download", ws.ID, uploaded.ID), fiber.MethodGet, nil)
	s.OK()
	s.Contains(s.HttpResponse.Header.Get(fiber.HeaderContentDisposition), "invoice-attachment.txt")
	downloadedBytes, err := io.ReadAll(s.HttpResponse.Body)
	_ = s.HttpResponse.Body.Close()
	s.Require().NoError(err)
	s.Equal(payload, downloadedBytes)

	// 5. Presign download URL
	s.Request(fmt.Sprintf("/workspaces/%d/files/%d/presign", ws.ID, uploaded.ID), fiber.MethodGet, nil)
	s.OK()
	presignRes := testutil.DecodeOther[model.FileDownloadURLResponse](s)
	s.Contains(presignRes.Data.URL, uploaded.Bucket)

	// 6. Update file metadata
	s.Request(fmt.Sprintf("/workspaces/%d/files/%d", ws.ID, uploaded.ID), fiber.MethodPut, model.UpdateFileRequest{
		Name:       "renamed-attachment.txt",
		EntityType: "fin:entry",
		EntityID:   100,
	})
	s.OK()
	updated := s.GetResponse().Data
	s.Equal("renamed-attachment.txt", updated.Name)
	s.Equal("fin:entry", updated.EntityType)
	s.Equal(int64(100), updated.EntityID)

	// 7. Delete file -> removes object in S3 and soft-deletes in DB
	s.Request(fmt.Sprintf("/workspaces/%d/files/%d", ws.ID, uploaded.ID), fiber.MethodDelete, nil)
	s.OK()

	// Verify 404 via API
	s.Request(fmt.Sprintf("/workspaces/%d/files/%d", ws.ID, uploaded.ID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)

	// Verify S3 object is removed
	_, err = st.StatObject(context.Background(), uploaded.Bucket, uploaded.Key)
	s.ErrorIs(err, s3x.ErrObjectNotFound)

	// Verify DB record still exists with deleted_at set
	dbFile, err := testutil.GetEntCli().File.Get(base.WithSkipWorkspaceFilter(s.Context), uploaded.ID)
	s.Require().NoError(err)
	s.NotNil(dbFile.DeletedAt)
	s.Require().NotNil(dbFile.DeletedByID)
	s.Equal(s.User.ID, *dbFile.DeletedByID)
}

func (s *FileSuite) Test_WorkspaceCreateAndDelete_ManagesWorkspaceBucket() {
	// Creating workspace via API provisions S3 bucket
	s.Request("/workspaces", fiber.MethodPost, model.CreateWorkspaceRequest{
		Name: "Bucket Lifecycle WS",
	})
	s.Created()
	wsRes := testutil.DecodeOther[model.WorkspaceResponse](s)
	wid := wsRes.Data.ID
	s.Positive(wid)

	st := testutil.GetStorage()
	bucket := st.WorkspaceBucket(wid)
	exists, err := st.BucketExists(context.Background(), bucket)
	s.Require().NoError(err)
	s.True(exists)

	// Upload a file into the workspace bucket
	s.RequestMultipart(fmt.Sprintf("/workspaces/%d/files", wid), fiber.MethodPost, "file", "doc.txt", []byte("workspace data"))
	s.Created()

	// Deleting workspace via API deletes the workspace bucket and its objects
	s.Request(fmt.Sprintf("/workspaces/%d", wid), fiber.MethodDelete, nil)
	s.OK()

	exists, err = st.BucketExists(context.Background(), bucket)
	s.Require().NoError(err)
	s.False(exists)
}

func (s *FileSuite) Test_RBAC_CoreAndFinRoles() {
	ws := s.ensureWorkspace("RBAC Files WS", s.User.ID)

	// Owner uploads initial file
	s.RequestMultipart(fmt.Sprintf("/workspaces/%d/files", ws.ID), fiber.MethodPost, "file", "report.pdf", []byte("%PDF-1.4 content"))
	s.Created()
	fileID := s.GetResponse().Data.ID

	// 1. core:member can read, cannot write
	member := s.AsUser("file-member@example.com", "Member User")
	s.AddMember(ws.ID, member.ID, "core:member")

	s.Request(fmt.Sprintf("/workspaces/%d/files/%d", ws.ID, fileID), fiber.MethodGet, nil)
	s.OK()

	s.RequestMultipart(fmt.Sprintf("/workspaces/%d/files", ws.ID), fiber.MethodPost, "file", "blocked.txt", []byte("nope"))
	s.AssertStatus(fiber.StatusForbidden)

	s.Request(fmt.Sprintf("/workspaces/%d/files/%d", ws.ID, fileID), fiber.MethodDelete, nil)
	s.AssertStatus(fiber.StatusForbidden)

	// 2. fin:viewer can read, cannot write
	viewer := s.AsUser("file-fin-viewer@example.com", "Fin Viewer")
	s.AddMember(ws.ID, viewer.ID, "fin:viewer")

	s.Request(fmt.Sprintf("/workspaces/%d/files/%d", ws.ID, fileID), fiber.MethodGet, nil)
	s.OK()

	s.RequestMultipart(fmt.Sprintf("/workspaces/%d/files", ws.ID), fiber.MethodPost, "file", "blocked2.txt", []byte("nope"))
	s.AssertStatus(fiber.StatusForbidden)

	// 3. fin:accountant can read and write
	accountant := s.AsUser("file-fin-accountant@example.com", "Fin Accountant")
	s.AddMember(ws.ID, accountant.ID, "fin:accountant")

	s.RequestMultipart(fmt.Sprintf("/workspaces/%d/files", ws.ID), fiber.MethodPost, "file", "allowed.txt", []byte("allowed by accountant"))
	s.Created()
	accFileID := s.GetResponse().Data.ID

	s.Request(fmt.Sprintf("/workspaces/%d/files/%d", ws.ID, accFileID), fiber.MethodDelete, nil)
	s.OK()

	// 4. Non-member gets 404
	s.AsUser("file-stranger@example.com", "Stranger")
	s.Request(fmt.Sprintf("/workspaces/%d/files/%d", ws.ID, fileID), fiber.MethodGet, nil)
	s.AssertStatus(fiber.StatusNotFound)
}

func (s *FileSuite) Test_Upload_ValidationErrors() {
	ws := s.ensureWorkspace("Validation WS", s.User.ID)

	// Missing file part
	s.uploadWithFields(ws.ID, "", nil, map[string]string{"name": "no-file"})
	s.AssertStatus(fiber.StatusBadRequest)

	// Invalid entityId
	s.uploadWithFields(ws.ID, "test.txt", []byte("abc"), map[string]string{"entityId": "not-an-int"})
	s.AssertStatus(fiber.StatusBadRequest)
}

func TestFileSuite(t *testing.T) {
	suite.Run(t, new(FileSuite))
}
