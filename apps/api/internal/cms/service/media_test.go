package service_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	"github.com/kilip/omed/api/internal/cms/service"
	"github.com/kilip/omed/api/internal/cms/service/mock"
	coredomain "github.com/kilip/omed/api/internal/core/domain"
	coremodel "github.com/kilip/omed/api/internal/core/model"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

//go:generate go run go.uber.org/mock/mockgen -source=media.go -destination=mock/mock_media.go -package=mock

type MediaServiceSuite struct {
	suite.Suite
	ctrl        *gomock.Controller
	mockRepo    *mock.MockMediaRepository
	mockFileSvc *mock.MockFileServiceContract
	svc         service.MediaService
	ctx         context.Context
}

func TestMediaServiceSuite(t *testing.T) {
	suite.Run(t, new(MediaServiceSuite))
}

func (s *MediaServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockRepo = mock.NewMockMediaRepository(s.ctrl)
	s.mockFileSvc = mock.NewMockFileServiceContract(s.ctrl)
	s.svc = service.NewMediaService(s.mockRepo, s.mockFileSvc, service.MediaConfig{
		MaxSizeMB:    10,
		AllowedTypes: []string{".png", ".jpg", ".jpeg", ".webp", ".gif", ".pdf"},
	})
	s.ctx = context.Background()
}

func (s *MediaServiceSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *MediaServiceSuite) Test_Upload_HappyPath() {
	buf := bytes.NewReader([]byte("sample-data"))
	file := coredomain.File{
		ID:          100,
		WorkspaceID: 1,
		Name:        "photo.jpg",
		MimeType:    "image/jpeg",
		Size:        11,
	}
	media := domain.Media{
		ID:          10,
		WorkspaceID: 1,
		FileID:      100,
		Title:       "My Photo",
		AltText:     "Photo alt",
		Caption:     "Photo caption",
		IsPublic:    true,
	}

	s.mockFileSvc.EXPECT().
		Upload(s.ctx, coremodel.UploadFileRequest{
			WorkspaceID: 1,
			Name:        "photo.jpg",
			MimeType:    "image/jpeg",
			Size:        11,
			EntityType:  domain.MediaEntityType,
			EntityID:    0,
			Reader:      buf,
		}).
		Return(file, nil)

	s.mockRepo.EXPECT().
		Create(s.ctx, model.CreateMediaRecordRequest{
			WorkspaceID: 1,
			FileID:      100,
			Title:       "My Photo",
			AltText:     "Photo alt",
			Caption:     "Photo caption",
			IsPublic:    true,
		}).
		Return(media, nil)

	s.mockFileSvc.EXPECT().
		Update(s.ctx, int64(100), coremodel.UpdateFileRequest{
			Name:       "photo.jpg",
			EntityType: domain.MediaEntityType,
			EntityID:   int64(10),
		}).
		Return(file, nil)

	s.mockFileSvc.EXPECT().
		PresignDownload(s.ctx, int64(100)).
		Return("https://s3.example.com/photo.jpg?signed", nil)

	res, err := s.svc.Upload(s.ctx, model.UploadMediaRequest{
		WorkspaceID: 1,
		Title:       "My Photo",
		AltText:     "Photo alt",
		Caption:     "Photo caption",
		Name:        "photo.jpg",
		MimeType:    "image/jpeg",
		Size:        11,
		Reader:      buf,
	})

	s.Require().NoError(err)
	s.Equal(int64(10), res.ID)
	s.Equal("https://s3.example.com/photo.jpg?signed", res.URL)
	s.Equal("My Photo", res.Title)
}

func (s *MediaServiceSuite) Test_Upload_DisallowedType() {
	buf := bytes.NewReader([]byte("<svg></svg>"))
	_, err := s.svc.Upload(s.ctx, model.UploadMediaRequest{
		WorkspaceID: 1,
		Name:        "malicious.svg",
		Size:        15,
		Reader:      buf,
	})
	s.ErrorIs(err, domain.ErrMediaTypeNotAllowed)

	_, err = s.svc.Upload(s.ctx, model.UploadMediaRequest{
		WorkspaceID: 1,
		Name:        "script.sh",
		Size:        15,
		Reader:      buf,
	})
	s.ErrorIs(err, domain.ErrMediaTypeNotAllowed)
}

func (s *MediaServiceSuite) Test_Upload_Oversize() {
	buf := bytes.NewReader([]byte("large"))
	_, err := s.svc.Upload(s.ctx, model.UploadMediaRequest{
		WorkspaceID: 1,
		Name:        "huge.png",
		Size:        11 * 1024 * 1024, // 11MB exceeds 10MB limit
		Reader:      buf,
	})
	s.ErrorIs(err, domain.ErrMediaTooLarge)
}

func (s *MediaServiceSuite) Test_Upload_EmptyFile_Or_NilReader() {
	_, err := s.svc.Upload(s.ctx, model.UploadMediaRequest{
		WorkspaceID: 1,
		Name:        "empty.png",
		Size:        10,
		Reader:      nil,
	})
	s.ErrorIs(err, coredomain.ErrFileRequired)

	buf := bytes.NewReader([]byte(""))
	_, err = s.svc.Upload(s.ctx, model.UploadMediaRequest{
		WorkspaceID: 1,
		Name:        "empty.png",
		Size:        0,
		Reader:      buf,
	})
	s.ErrorIs(err, coredomain.ErrFileEmpty)
}

func (s *MediaServiceSuite) Test_Upload_Compensation_On_Insert_Failure() {
	buf := bytes.NewReader([]byte("sample"))
	file := coredomain.File{
		ID:          100,
		WorkspaceID: 1,
		Name:        "photo.jpg",
		Size:        6,
	}

	s.mockFileSvc.EXPECT().
		Upload(s.ctx, gomock.Any()).
		Return(file, nil)

	dbErr := errors.New("database insert error")
	s.mockRepo.EXPECT().
		Create(s.ctx, gomock.Any()).
		Return(domain.Media{}, dbErr)

	// Compensation must delete the uploaded file
	s.mockFileSvc.EXPECT().
		Delete(s.ctx, int64(100)).
		Return(nil)

	_, err := s.svc.Upload(s.ctx, model.UploadMediaRequest{
		WorkspaceID: 1,
		Name:        "photo.jpg",
		Size:        6,
		Reader:      buf,
	})

	s.ErrorIs(err, dbErr)
}

func (s *MediaServiceSuite) Test_Delete_When_Referenced() {
	media := domain.Media{
		ID:     10,
		FileID: 100,
	}

	s.mockRepo.EXPECT().
		GetByID(s.ctx, int64(10)).
		Return(media, nil)

	s.mockRepo.EXPECT().
		IsReferenced(s.ctx, int64(10)).
		Return(true, nil)

	// Neither repo.Delete nor fileSvc.Delete must be called
	err := s.svc.Delete(s.ctx, 10)
	s.ErrorIs(err, domain.ErrMediaInUse)
}

func (s *MediaServiceSuite) Test_Delete_When_Not_Referenced() {
	media := domain.Media{
		ID:     10,
		FileID: 100,
	}

	s.mockRepo.EXPECT().
		GetByID(s.ctx, int64(10)).
		Return(media, nil)

	s.mockRepo.EXPECT().
		IsReferenced(s.ctx, int64(10)).
		Return(false, nil)

	s.mockRepo.EXPECT().
		Delete(s.ctx, int64(10)).
		Return(nil)

	s.mockFileSvc.EXPECT().
		Delete(s.ctx, int64(100)).
		Return(nil)

	err := s.svc.Delete(s.ctx, 10)
	s.NoError(err)
}

func (s *MediaServiceSuite) Test_Delete_NotFound() {
	s.mockRepo.EXPECT().
		GetByID(s.ctx, int64(999)).
		Return(domain.Media{}, domain.ErrMediaNotFound)

	err := s.svc.Delete(s.ctx, 999)
	s.ErrorIs(err, domain.ErrMediaNotFound)
}

func (s *MediaServiceSuite) Test_GetByID_Success() {
	media := domain.Media{
		ID:     10,
		FileID: 100,
		Title:  "Test",
	}

	s.mockRepo.EXPECT().
		GetByID(s.ctx, int64(10)).
		Return(media, nil)

	s.mockFileSvc.EXPECT().
		PresignDownload(s.ctx, int64(100)).
		Return("https://s3.example.com/photo.jpg?signed", nil)

	got, err := s.svc.GetByID(s.ctx, 10)
	s.Require().NoError(err)
	s.Equal("https://s3.example.com/photo.jpg?signed", got.URL)
	s.Equal("Test", got.Title)
}

func (s *MediaServiceSuite) Test_Search_Success() {
	items := []domain.Media{
		{ID: 1, FileID: 101, Title: "Item 1"},
		{ID: 2, FileID: 102, Title: "Item 2"},
	}

	q := "Item"
	s.mockRepo.EXPECT().
		Search(s.ctx, model.SearchMediaRequest{Query: &q}).
		Return(items, "cursor-123", nil)

	s.mockFileSvc.EXPECT().
		PresignDownload(s.ctx, int64(101)).
		Return("https://s3.example.com/1?signed", nil)

	s.mockFileSvc.EXPECT().
		PresignDownload(s.ctx, int64(102)).
		Return("https://s3.example.com/2?signed", nil)

	got, cursor, err := s.svc.Search(s.ctx, model.SearchMediaRequest{Query: &q})
	s.Require().NoError(err)
	s.Equal("cursor-123", cursor)
	s.Len(got, 2)
	s.Equal("https://s3.example.com/1?signed", got[0].URL)
	s.Equal("https://s3.example.com/2?signed", got[1].URL)
}

func (s *MediaServiceSuite) Test_Update_Success() {
	newTitle := "Updated Title"
	media := domain.Media{
		ID:     10,
		FileID: 100,
		Title:  newTitle,
	}

	s.mockRepo.EXPECT().
		Update(s.ctx, int64(10), model.UpdateMediaRequest{Title: &newTitle}).
		Return(media, nil)

	s.mockFileSvc.EXPECT().
		PresignDownload(s.ctx, int64(100)).
		Return("https://s3.example.com/photo.jpg?signed", nil)

	got, err := s.svc.Update(s.ctx, 10, model.UpdateMediaRequest{Title: &newTitle})
	s.Require().NoError(err)
	s.Equal(newTitle, got.Title)
	s.Equal("https://s3.example.com/photo.jpg?signed", got.URL)
}
