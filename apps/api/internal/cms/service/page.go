//go:generate go run go.uber.org/mock/mockgen -source=page.go -destination=mock/mock_page.go -package=mock

package service

import (
	"context"
	"time"

	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
)

// PageRepository defines persistence operations required by PageService.
type PageRepository interface {
	GetByID(c context.Context, id int64, now time.Time) (domain.Page, error)
	Create(c context.Context, req model.CreatePageRecord, now time.Time) (domain.Page, error)
	Update(c context.Context, id int64, req model.UpdatePageRecord, now time.Time) (domain.Page, error)
	Delete(c context.Context, id int64) error
	Search(c context.Context, req model.SearchPageRequest, now time.Time) ([]domain.Page, string, error)
	SlugExists(c context.Context, slug string, excludeID *int64) (bool, error)
}

// PageServiceOption provides configuration options for PageService.
type PageServiceOption func(*PageService)

// WithPageClock overrides the current time provider in PageService (useful for unit testing).
func WithPageClock(now func() time.Time) PageServiceOption {
	return func(s *PageService) {
		s.now = now
	}
}

// PageService handles business logic and workflow orchestration for CMS pages.
type PageService struct {
	repo PageRepository
	now  func() time.Time
}

// NewPageService constructs a new PageService backed by the given PageRepository and options.
func NewPageService(repo PageRepository, opts ...PageServiceOption) PageService {
	s := PageService{
		repo: repo,
		now:  time.Now,
	}
	for _, opt := range opts {
		opt(&s)
	}
	return s
}

// Create validates and creates a new CMS page in DRAFT status.
func (s PageService) Create(c context.Context, req model.CreatePageRequest) (domain.Page, error) {
	slug, err := s.resolveUniqueSlug(c, req.Title, req.Slug)
	if err != nil {
		return domain.Page{}, err
	}

	body := ""
	if req.Body != nil {
		body = *req.Body
	}

	return s.repo.Create(c, model.CreatePageRecord{
		Title:  req.Title,
		Slug:   slug,
		Body:   body,
		Status: domain.StatusDraft,
	}, s.now())
}

// GetByID retrieves a page by ID.
func (s PageService) GetByID(c context.Context, id int64) (domain.Page, error) {
	return s.repo.GetByID(c, id, s.now())
}

// Search searches pages matching search and pagination parameters.
func (s PageService) Search(c context.Context, req model.SearchPageRequest) ([]domain.Page, string, error) {
	return s.repo.Search(c, req, s.now())
}

// Update modifies metadata and content of an existing page.
func (s PageService) Update(c context.Context, id int64, req model.UpdatePageRequest) (domain.Page, error) {
	if req.Slug != nil && *req.Slug != "" {
		exists, err := s.repo.SlugExists(c, *req.Slug, &id)
		if err != nil {
			return domain.Page{}, err
		}
		if exists {
			return domain.Page{}, domain.ErrSlugTaken
		}
	}

	return s.repo.Update(c, id, model.UpdatePageRecord{
		Title: req.Title,
		Slug:  req.Slug,
		Body:  req.Body,
	}, s.now())
}

// Delete soft-deletes a page by ID.
func (s PageService) Delete(c context.Context, id int64) error {
	return s.repo.Delete(c, id)
}

// Publish publishes a draft or scheduled page, setting status to PUBLISHED and publish_at to now.
func (s PageService) Publish(c context.Context, id int64) (domain.Page, error) {
	now := s.now()
	p, err := s.repo.GetByID(c, id, now)
	if err != nil {
		return domain.Page{}, err
	}

	newStatus, newPublishAt, err := domain.Publish(p.Status, p.PublishAt, now)
	if err != nil {
		return domain.Page{}, err
	}

	return s.repo.Update(c, id, model.UpdatePageRecord{
		Status:    &newStatus,
		PublishAt: newPublishAt,
	}, now)
}

// Schedule schedules a draft or scheduled page to go live at publishAt.
func (s PageService) Schedule(c context.Context, id int64, req model.SchedulePageRequest) (domain.Page, error) {
	now := s.now()
	p, err := s.repo.GetByID(c, id, now)
	if err != nil {
		return domain.Page{}, err
	}

	newStatus, newPublishAt, err := domain.Schedule(p.Status, p.PublishAt, req.PublishAt, now)
	if err != nil {
		return domain.Page{}, err
	}

	return s.repo.Update(c, id, model.UpdatePageRecord{
		Status:    &newStatus,
		PublishAt: newPublishAt,
	}, now)
}

// Unpublish reverts a published or scheduled page to DRAFT status and clears publish_at.
func (s PageService) Unpublish(c context.Context, id int64) (domain.Page, error) {
	now := s.now()
	p, err := s.repo.GetByID(c, id, now)
	if err != nil {
		return domain.Page{}, err
	}

	newStatus, _, err := domain.Unpublish(p.Status, p.PublishAt, now)
	if err != nil {
		return domain.Page{}, err
	}

	return s.repo.Update(c, id, model.UpdatePageRecord{
		Status:         &newStatus,
		ClearPublishAt: true,
	}, now)
}

func (s PageService) resolveUniqueSlug(c context.Context, title string, explicitSlug *string) (string, error) {
	if explicitSlug != nil && *explicitSlug != "" {
		exists, err := s.repo.SlugExists(c, *explicitSlug, nil)
		if err != nil {
			return "", err
		}
		if exists {
			return "", domain.ErrSlugTaken
		}
		return *explicitSlug, nil
	}

	baseSlug := domain.Slugify(title)
	slug := baseSlug
	suffix := 1

	for {
		exists, err := s.repo.SlugExists(c, slug, nil)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		suffix++
		slug = domain.AppendSlugSuffix(baseSlug, suffix)
	}
}
