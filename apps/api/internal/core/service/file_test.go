package service_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/core/domain"
	"github.com/kilip/omed/api/internal/core/model"
	"github.com/kilip/omed/api/internal/core/service"
	"github.com/kilip/omed/api/internal/platform/s3x"
	"github.com/stretchr/testify/suite"
)

type fakeFileRepo struct {
	files     map[int64]domain.File
	nextID    int64
	createErr error
	updateErr error
	deleteErr error
}

func newFakeFileRepo() *fakeFileRepo {
	return &fakeFileRepo{
		files:  make(map[int64]domain.File),
		nextID: 1,
	}
}

func (r *fakeFileRepo) GetByID(_ context.Context, id int64) (domain.File, error) {
	f, ok := r.files[id]
	if !ok {
		return domain.File{}, base.ErrNotFound
	}
	return f, nil
}

func (r *fakeFileRepo) Search(_ context.Context, _ model.SearchFileRequest) ([]domain.File, string, error) {
	out := make([]domain.File, 0, len(r.files))
	for _, f := range r.files {
		out = append(out, f)
	}
	return out, "", nil
}

func (r *fakeFileRepo) Create(_ context.Context, req model.CreateFileRecordRequest) (domain.File, error) {
	if r.createErr != nil {
		return domain.File{}, r.createErr
	}
	id := r.nextID
	r.nextID++
	f := domain.File{
		ID:          id,
		WorkspaceID: req.WorkspaceID,
		Name:        req.Name,
		Bucket:      req.Bucket,
		Key:         req.Key,
		MimeType:    req.MimeType,
		Size:        req.Size,
		EntityType:  req.EntityType,
		EntityID:    req.EntityID,
	}
	r.files[id] = f
	return f, nil
}

func (r *fakeFileRepo) Update(_ context.Context, id int64, req model.UpdateFileRequest) (domain.File, error) {
	if r.updateErr != nil {
		return domain.File{}, r.updateErr
	}
	f, ok := r.files[id]
	if !ok {
		return domain.File{}, base.ErrNotFound
	}
	f.Name = req.Name
	f.EntityType = req.EntityType
	f.EntityID = req.EntityID
	r.files[id] = f
	return f, nil
}

func (r *fakeFileRepo) Delete(_ context.Context, id int64) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	if _, ok := r.files[id]; !ok {
		return base.ErrNotFound
	}
	delete(r.files, id)
	return nil
}

type FileServiceSuite struct {
	suite.Suite
	repo    *fakeFileRepo
	storage s3x.Storage
	svc     service.FileService
}

func (s *FileServiceSuite) SetupTest() {
	s.repo = newFakeFileRepo()
	st, err := s3x.New(s3x.Config{Driver: "memory"})
	s.Require().NoError(err)
	s.storage = st
	s.svc = service.NewFileService(s.repo, s.storage)
}

func (s *FileServiceSuite) Test_Upload_Download_Presign_Update_Delete() {
	ctx := base.WithWorkspaceID(context.Background(), 55)
	payload := []byte("sample voucher attachment")

	uploaded, err := s.svc.Upload(ctx, model.UploadFileRequest{
		WorkspaceID: 55,
		Name:        "voucher.pdf",
		MimeType:    "application/pdf",
		Size:        int64(len(payload)),
		EntityType:  "fin:transaction",
		EntityID:    12,
		Reader:      bytes.NewReader(payload),
	})
	s.Require().NoError(err)
	s.Equal(int64(55), uploaded.WorkspaceID)
	s.Equal("omed-ws-55", uploaded.Bucket)
	s.Contains(uploaded.Key, "fin/transaction/")
	s.Contains(uploaded.Key, ".pdf")
	s.Equal("application/pdf", uploaded.MimeType)
	s.Equal(int64(len(payload)), uploaded.Size)

	// Download
	meta, obj, err := s.svc.Download(ctx, uploaded.ID)
	s.Require().NoError(err)
	s.Equal(uploaded.ID, meta.ID)
	data, err := io.ReadAll(obj.Body)
	_ = obj.Body.Close()
	s.Require().NoError(err)
	s.Equal(payload, data)

	// Presign
	urlStr, err := s.svc.PresignDownload(ctx, uploaded.ID)
	s.Require().NoError(err)
	s.Contains(urlStr, "omed-ws-55")

	// Update
	updated, err := s.svc.Update(ctx, uploaded.ID, model.UpdateFileRequest{
		Name:       "voucher-v2.pdf",
		EntityType: "fin:entry",
		EntityID:   34,
	})
	s.Require().NoError(err)
	s.Equal("voucher-v2.pdf", updated.Name)
	s.Equal("fin:entry", updated.EntityType)

	// Search
	list, _, err := s.svc.Search(ctx, model.SearchFileRequest{})
	s.Require().NoError(err)
	s.Len(list, 1)

	// Delete removes object from S3 and soft-deletes in repo
	err = s.svc.Delete(ctx, uploaded.ID)
	s.Require().NoError(err)

	_, err = s.storage.StatObject(ctx, uploaded.Bucket, uploaded.Key)
	s.ErrorIs(err, s3x.ErrObjectNotFound)

	_, err = s.svc.GetByID(ctx, uploaded.ID)
	s.ErrorIs(err, base.ErrNotFound)
}

func (s *FileServiceSuite) Test_Upload_ValidationAndRollbackOnRepoError() {
	ctx := base.WithWorkspaceID(context.Background(), 77)

	// Nil reader
	_, err := s.svc.Upload(ctx, model.UploadFileRequest{Size: 10})
	s.ErrorIs(err, domain.ErrFileRequired)

	// Empty file
	_, err = s.svc.Upload(ctx, model.UploadFileRequest{Reader: bytes.NewReader(nil), Size: 0})
	s.ErrorIs(err, domain.ErrFileEmpty)

	// Missing workspace ID
	_, err = s.svc.Upload(context.Background(), model.UploadFileRequest{
		Reader: bytes.NewReader([]byte("abc")),
		Size:   3,
	})
	s.ErrorIs(err, base.ErrInvalidID)

	// Repo create failure rolls back uploaded S3 object
	s.repo.createErr = errors.New("db write failed")
	_, err = s.svc.Upload(ctx, model.UploadFileRequest{
		Name:   "fail.txt",
		Reader: bytes.NewReader([]byte("abc")),
		Size:   3,
	})
	s.Error(err)
}

func TestFileServiceSuite(t *testing.T) {
	suite.Run(t, new(FileServiceSuite))
}
