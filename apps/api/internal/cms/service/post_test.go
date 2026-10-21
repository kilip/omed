package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	"github.com/kilip/omed/api/internal/cms/service"
	"github.com/kilip/omed/api/internal/cms/service/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type PostServiceSuite struct {
	suite.Suite
	ctrl             *gomock.Controller
	mockRepo         *mock.MockPostRepository
	mockMediaChecker *mock.MockMediaChecker
	svc              service.PostService
	ctx              context.Context
	fixedTime        time.Time
}

func TestPostServiceSuite(t *testing.T) {
	suite.Run(t, new(PostServiceSuite))
}

func (s *PostServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockRepo = mock.NewMockPostRepository(s.ctrl)
	s.mockMediaChecker = mock.NewMockMediaChecker(s.ctrl)
	s.fixedTime = time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
	s.svc = service.NewPostService(
		s.mockRepo,
		service.WithPostClock(func() time.Time { return s.fixedTime }),
		service.WithPostMediaChecker(s.mockMediaChecker),
	)
	s.ctx = context.Background()
}

func (s *PostServiceSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *PostServiceSuite) Test_Create_AutoSlug_NoConflict() {
	s.mockRepo.EXPECT().
		SlugExists(s.ctx, "hello-world", nil).
		Return(false, nil)

	s.mockRepo.EXPECT().
		Create(s.ctx, model.CreatePostRecord{
			Title:   "Hello World",
			Slug:    "hello-world",
			Excerpt: "",
			Body:    "",
			Status:  domain.StatusDraft,
		}, s.fixedTime).
		Return(domain.Post{
			ID:    1,
			Title: "Hello World",
			Slug:  "hello-world",
		}, nil)

	p, err := s.svc.Create(s.ctx, model.CreatePostRequest{
		Title: "Hello World",
	})
	s.Require().NoError(err)
	s.Equal(int64(1), p.ID)
	s.Equal("hello-world", p.Slug)
}

func (s *PostServiceSuite) Test_Create_AutoSlug_ConflictResolution() {
	// First slug exists
	s.mockRepo.EXPECT().
		SlugExists(s.ctx, "hello-world", nil).
		Return(true, nil)
	// Second slug (suffix 2) does not exist
	s.mockRepo.EXPECT().
		SlugExists(s.ctx, "hello-world-2", nil).
		Return(false, nil)

	s.mockRepo.EXPECT().
		Create(s.ctx, model.CreatePostRecord{
			Title:   "Hello World",
			Slug:    "hello-world-2",
			Excerpt: "",
			Body:    "",
			Status:  domain.StatusDraft,
		}, s.fixedTime).
		Return(domain.Post{
			ID:    2,
			Title: "Hello World",
			Slug:  "hello-world-2",
		}, nil)

	p, err := s.svc.Create(s.ctx, model.CreatePostRequest{
		Title: "Hello World",
	})
	s.Require().NoError(err)
	s.Equal("hello-world-2", p.Slug)
}

func (s *PostServiceSuite) Test_Create_ExplicitSlug_ConflictReturnsError() {
	explicit := "custom-slug"
	s.mockRepo.EXPECT().
		SlugExists(s.ctx, "custom-slug", nil).
		Return(true, nil)

	_, err := s.svc.Create(s.ctx, model.CreatePostRequest{
		Title: "Hello World",
		Slug:  &explicit,
	})
	s.Require().ErrorIs(err, domain.ErrSlugTaken)
}

func (s *PostServiceSuite) Test_Create_BadCoverMedia() {
	coverID := int64(999)
	s.mockRepo.EXPECT().
		SlugExists(s.ctx, "post-title", nil).
		Return(false, nil)

	s.mockMediaChecker.EXPECT().
		GetByID(s.ctx, coverID).
		Return(domain.Media{}, errors.New("not found"))

	_, err := s.svc.Create(s.ctx, model.CreatePostRequest{
		Title:        "Post Title",
		CoverMediaID: &coverID,
	})
	s.Require().ErrorIs(err, domain.ErrMediaNotFound)
}

func (s *PostServiceSuite) Test_Update_SlugConflict() {
	id := int64(10)
	newSlug := "taken-slug"

	s.mockRepo.EXPECT().
		SlugExists(s.ctx, "taken-slug", &id).
		Return(true, nil)

	_, err := s.svc.Update(s.ctx, id, model.UpdatePostRequest{
		Slug: &newSlug,
	})
	s.Require().ErrorIs(err, domain.ErrSlugTaken)
}

func (s *PostServiceSuite) Test_Update_BadCoverMedia() {
	id := int64(10)
	coverID := int64(55)

	s.mockMediaChecker.EXPECT().
		GetByID(s.ctx, coverID).
		Return(domain.Media{}, errors.New("not found"))

	_, err := s.svc.Update(s.ctx, id, model.UpdatePostRequest{
		CoverMediaID: &coverID,
	})
	s.Require().ErrorIs(err, domain.ErrMediaNotFound)
}

func (s *PostServiceSuite) Test_Publish_FromDraft() {
	id := int64(5)
	draftPost := domain.Post{
		ID:        id,
		Status:    domain.StatusDraft,
		PublishAt: nil,
	}

	s.mockRepo.EXPECT().
		GetByID(s.ctx, id, s.fixedTime).
		Return(draftPost, nil)

	newStatus := domain.StatusPublished
	expectedPublishAt := s.fixedTime
	s.mockRepo.EXPECT().
		Update(s.ctx, id, model.UpdatePostRecord{
			Status:    &newStatus,
			PublishAt: &expectedPublishAt,
		}, s.fixedTime).
		Return(domain.Post{
			ID:              id,
			Status:          domain.StatusPublished,
			EffectiveStatus: domain.EffectiveStatusPublished,
			PublishAt:       &expectedPublishAt,
		}, nil)

	res, err := s.svc.Publish(s.ctx, id)
	s.Require().NoError(err)
	s.Equal(domain.EffectiveStatusPublished, res.EffectiveStatus)
}

func (s *PostServiceSuite) Test_Publish_AlreadyPublishedFails() {
	id := int64(5)
	past := s.fixedTime.Add(-1 * time.Hour)
	livePost := domain.Post{
		ID:        id,
		Status:    domain.StatusPublished,
		PublishAt: &past,
	}

	s.mockRepo.EXPECT().
		GetByID(s.ctx, id, s.fixedTime).
		Return(livePost, nil)

	_, err := s.svc.Publish(s.ctx, id)
	s.Require().ErrorIs(err, domain.ErrInvalidTransition)
}

func (s *PostServiceSuite) Test_Schedule_InPastFails() {
	id := int64(5)
	draftPost := domain.Post{
		ID:     id,
		Status: domain.StatusDraft,
	}

	s.mockRepo.EXPECT().
		GetByID(s.ctx, id, s.fixedTime).
		Return(draftPost, nil)

	past := s.fixedTime.Add(-10 * time.Minute)
	_, err := s.svc.Schedule(s.ctx, id, model.SchedulePostRequest{
		PublishAt: past,
	})
	s.Require().ErrorIs(err, domain.ErrPublishAtInPast)
}

func (s *PostServiceSuite) Test_Schedule_FutureSuccess() {
	id := int64(5)
	draftPost := domain.Post{
		ID:     id,
		Status: domain.StatusDraft,
	}

	s.mockRepo.EXPECT().
		GetByID(s.ctx, id, s.fixedTime).
		Return(draftPost, nil)

	future := s.fixedTime.Add(2 * time.Hour)
	newStatus := domain.StatusPublished
	s.mockRepo.EXPECT().
		Update(s.ctx, id, model.UpdatePostRecord{
			Status:    &newStatus,
			PublishAt: &future,
		}, s.fixedTime).
		Return(domain.Post{
			ID:              id,
			Status:          domain.StatusPublished,
			EffectiveStatus: domain.EffectiveStatusScheduled,
			PublishAt:       &future,
		}, nil)

	res, err := s.svc.Schedule(s.ctx, id, model.SchedulePostRequest{
		PublishAt: future,
	})
	s.Require().NoError(err)
	s.Equal(domain.EffectiveStatusScheduled, res.EffectiveStatus)
}

func (s *PostServiceSuite) Test_Unpublish_Success() {
	id := int64(5)
	future := s.fixedTime.Add(2 * time.Hour)
	scheduledPost := domain.Post{
		ID:        id,
		Status:    domain.StatusPublished,
		PublishAt: &future,
	}

	s.mockRepo.EXPECT().
		GetByID(s.ctx, id, s.fixedTime).
		Return(scheduledPost, nil)

	newStatus := domain.StatusDraft
	s.mockRepo.EXPECT().
		Update(s.ctx, id, model.UpdatePostRecord{
			Status:         &newStatus,
			ClearPublishAt: true,
		}, s.fixedTime).
		Return(domain.Post{
			ID:              id,
			Status:          domain.StatusDraft,
			EffectiveStatus: domain.EffectiveStatusDraft,
			PublishAt:       nil,
		}, nil)

	res, err := s.svc.Unpublish(s.ctx, id)
	s.Require().NoError(err)
	s.Equal(domain.EffectiveStatusDraft, res.EffectiveStatus)
}

func (s *PostServiceSuite) Test_Unpublish_DraftFails() {
	id := int64(5)
	draftPost := domain.Post{
		ID:     id,
		Status: domain.StatusDraft,
	}

	s.mockRepo.EXPECT().
		GetByID(s.ctx, id, s.fixedTime).
		Return(draftPost, nil)

	_, err := s.svc.Unpublish(s.ctx, id)
	s.Require().ErrorIs(err, domain.ErrInvalidTransition)
}

func (s *PostServiceSuite) Test_GetByID_Search_Delete() {
	s.mockRepo.EXPECT().
		GetByID(s.ctx, int64(1), s.fixedTime).
		Return(domain.Post{ID: 1}, nil)

	p, err := s.svc.GetByID(s.ctx, 1)
	s.Require().NoError(err)
	s.Equal(int64(1), p.ID)

	s.mockRepo.EXPECT().
		Search(s.ctx, model.SearchPostRequest{}, s.fixedTime).
		Return([]domain.Post{{ID: 1}}, "cursor-1", nil)

	posts, cursor, err := s.svc.Search(s.ctx, model.SearchPostRequest{})
	s.Require().NoError(err)
	s.Len(posts, 1)
	s.Equal("cursor-1", cursor)

	s.mockRepo.EXPECT().
		Delete(s.ctx, int64(1)).
		Return(nil)

	err = s.svc.Delete(s.ctx, 1)
	s.Require().NoError(err)
}
