package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
	"github.com/kilip/omed/api/internal/cms/service"
	"github.com/kilip/omed/api/internal/cms/service/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type PageServiceSuite struct {
	suite.Suite
	ctrl      *gomock.Controller
	mockRepo  *mock.MockPageRepository
	svc       service.PageService
	ctx       context.Context
	fixedTime time.Time
}

func TestPageServiceSuite(t *testing.T) {
	suite.Run(t, new(PageServiceSuite))
}

func (s *PageServiceSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockRepo = mock.NewMockPageRepository(s.ctrl)
	s.fixedTime = time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
	s.svc = service.NewPageService(
		s.mockRepo,
		service.WithPageClock(func() time.Time { return s.fixedTime }),
	)
	s.ctx = context.Background()
}

func (s *PageServiceSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *PageServiceSuite) Test_Create_AutoSlug_NoConflict() {
	s.mockRepo.EXPECT().
		SlugExists(s.ctx, "terms-of-service", nil).
		Return(false, nil)

	s.mockRepo.EXPECT().
		Create(s.ctx, model.CreatePageRecord{
			Title:  "Terms of Service",
			Slug:   "terms-of-service",
			Body:   "",
			Status: domain.StatusDraft,
		}, s.fixedTime).
		Return(domain.Page{
			ID:    1,
			Title: "Terms of Service",
			Slug:  "terms-of-service",
		}, nil)

	p, err := s.svc.Create(s.ctx, model.CreatePageRequest{
		Title: "Terms of Service",
	})
	s.Require().NoError(err)
	s.Equal(int64(1), p.ID)
	s.Equal("terms-of-service", p.Slug)
}

func (s *PageServiceSuite) Test_Create_AutoSlug_ConflictResolution() {
	s.mockRepo.EXPECT().
		SlugExists(s.ctx, "terms-of-service", nil).
		Return(true, nil)
	s.mockRepo.EXPECT().
		SlugExists(s.ctx, "terms-of-service-2", nil).
		Return(false, nil)

	s.mockRepo.EXPECT().
		Create(s.ctx, model.CreatePageRecord{
			Title:  "Terms of Service",
			Slug:   "terms-of-service-2",
			Body:   "",
			Status: domain.StatusDraft,
		}, s.fixedTime).
		Return(domain.Page{
			ID:    2,
			Title: "Terms of Service",
			Slug:  "terms-of-service-2",
		}, nil)

	p, err := s.svc.Create(s.ctx, model.CreatePageRequest{
		Title: "Terms of Service",
	})
	s.Require().NoError(err)
	s.Equal("terms-of-service-2", p.Slug)
}

func (s *PageServiceSuite) Test_Create_ExplicitSlug_ConflictReturnsError() {
	explicit := "custom-terms"
	s.mockRepo.EXPECT().
		SlugExists(s.ctx, "custom-terms", nil).
		Return(true, nil)

	_, err := s.svc.Create(s.ctx, model.CreatePageRequest{
		Title: "Terms of Service",
		Slug:  &explicit,
	})
	s.Require().ErrorIs(err, domain.ErrSlugTaken)
}

func (s *PageServiceSuite) Test_Update_SlugConflict() {
	id := int64(10)
	newSlug := "taken-slug"

	s.mockRepo.EXPECT().
		SlugExists(s.ctx, "taken-slug", &id).
		Return(true, nil)

	_, err := s.svc.Update(s.ctx, id, model.UpdatePageRequest{
		Slug: &newSlug,
	})
	s.Require().ErrorIs(err, domain.ErrSlugTaken)
}

func (s *PageServiceSuite) Test_Publish_FromDraft() {
	id := int64(5)
	draftPage := domain.Page{
		ID:        id,
		Status:    domain.StatusDraft,
		PublishAt: nil,
	}

	s.mockRepo.EXPECT().
		GetByID(s.ctx, id, s.fixedTime).
		Return(draftPage, nil)

	newStatus := domain.StatusPublished
	expectedPublishAt := s.fixedTime
	s.mockRepo.EXPECT().
		Update(s.ctx, id, model.UpdatePageRecord{
			Status:    &newStatus,
			PublishAt: &expectedPublishAt,
		}, s.fixedTime).
		Return(domain.Page{
			ID:              id,
			Status:          domain.StatusPublished,
			EffectiveStatus: domain.EffectiveStatusPublished,
			PublishAt:       &expectedPublishAt,
		}, nil)

	res, err := s.svc.Publish(s.ctx, id)
	s.Require().NoError(err)
	s.Equal(domain.EffectiveStatusPublished, res.EffectiveStatus)
}

func (s *PageServiceSuite) Test_Publish_AlreadyPublishedFails() {
	id := int64(5)
	past := s.fixedTime.Add(-1 * time.Hour)
	livePage := domain.Page{
		ID:        id,
		Status:    domain.StatusPublished,
		PublishAt: &past,
	}

	s.mockRepo.EXPECT().
		GetByID(s.ctx, id, s.fixedTime).
		Return(livePage, nil)

	_, err := s.svc.Publish(s.ctx, id)
	s.Require().ErrorIs(err, domain.ErrInvalidTransition)
}

func (s *PageServiceSuite) Test_Schedule_InPastFails() {
	id := int64(5)
	draftPage := domain.Page{
		ID:     id,
		Status: domain.StatusDraft,
	}

	s.mockRepo.EXPECT().
		GetByID(s.ctx, id, s.fixedTime).
		Return(draftPage, nil)

	past := s.fixedTime.Add(-10 * time.Minute)
	_, err := s.svc.Schedule(s.ctx, id, model.SchedulePageRequest{
		PublishAt: past,
	})
	s.Require().ErrorIs(err, domain.ErrPublishAtInPast)
}

func (s *PageServiceSuite) Test_Schedule_FutureSuccess() {
	id := int64(5)
	draftPage := domain.Page{
		ID:     id,
		Status: domain.StatusDraft,
	}

	s.mockRepo.EXPECT().
		GetByID(s.ctx, id, s.fixedTime).
		Return(draftPage, nil)

	future := s.fixedTime.Add(2 * time.Hour)
	newStatus := domain.StatusPublished
	s.mockRepo.EXPECT().
		Update(s.ctx, id, model.UpdatePageRecord{
			Status:    &newStatus,
			PublishAt: &future,
		}, s.fixedTime).
		Return(domain.Page{
			ID:              id,
			Status:          domain.StatusPublished,
			EffectiveStatus: domain.EffectiveStatusScheduled,
			PublishAt:       &future,
		}, nil)

	res, err := s.svc.Schedule(s.ctx, id, model.SchedulePageRequest{
		PublishAt: future,
	})
	s.Require().NoError(err)
	s.Equal(domain.EffectiveStatusScheduled, res.EffectiveStatus)
}

func (s *PageServiceSuite) Test_Unpublish_Success() {
	id := int64(5)
	future := s.fixedTime.Add(2 * time.Hour)
	scheduledPage := domain.Page{
		ID:        id,
		Status:    domain.StatusPublished,
		PublishAt: &future,
	}

	s.mockRepo.EXPECT().
		GetByID(s.ctx, id, s.fixedTime).
		Return(scheduledPage, nil)

	newStatus := domain.StatusDraft
	s.mockRepo.EXPECT().
		Update(s.ctx, id, model.UpdatePageRecord{
			Status:         &newStatus,
			ClearPublishAt: true,
		}, s.fixedTime).
		Return(domain.Page{
			ID:              id,
			Status:          domain.StatusDraft,
			EffectiveStatus: domain.EffectiveStatusDraft,
			PublishAt:       nil,
		}, nil)

	res, err := s.svc.Unpublish(s.ctx, id)
	s.Require().NoError(err)
	s.Equal(domain.EffectiveStatusDraft, res.EffectiveStatus)
}

func (s *PageServiceSuite) Test_Unpublish_DraftFails() {
	id := int64(5)
	draftPage := domain.Page{
		ID:     id,
		Status: domain.StatusDraft,
	}

	s.mockRepo.EXPECT().
		GetByID(s.ctx, id, s.fixedTime).
		Return(draftPage, nil)

	_, err := s.svc.Unpublish(s.ctx, id)
	s.Require().ErrorIs(err, domain.ErrInvalidTransition)
}

func (s *PageServiceSuite) Test_GetByID_Search_Delete() {
	s.mockRepo.EXPECT().
		GetByID(s.ctx, int64(1), s.fixedTime).
		Return(domain.Page{ID: 1}, nil)

	p, err := s.svc.GetByID(s.ctx, 1)
	s.Require().NoError(err)
	s.Equal(int64(1), p.ID)

	s.mockRepo.EXPECT().
		Search(s.ctx, model.SearchPageRequest{}, s.fixedTime).
		Return([]domain.Page{{ID: 1}}, "cursor-1", nil)

	pages, cursor, err := s.svc.Search(s.ctx, model.SearchPageRequest{})
	s.Require().NoError(err)
	s.Len(pages, 1)
	s.Equal("cursor-1", cursor)

	s.mockRepo.EXPECT().
		Delete(s.ctx, int64(1)).
		Return(nil)

	err = s.svc.Delete(s.ctx, 1)
	s.Require().NoError(err)
}
