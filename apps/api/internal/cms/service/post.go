//go:generate go run go.uber.org/mock/mockgen -source=post.go -destination=mock/mock_post.go -package=mock

package service

import (
	"context"
	"time"

	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
)

// PostRepository defines persistence operations required by PostService.
type PostRepository interface {
	GetByID(c context.Context, id int64, now time.Time) (domain.Post, error)
	Create(c context.Context, req model.CreatePostRecord, now time.Time) (domain.Post, error)
	Update(c context.Context, id int64, req model.UpdatePostRecord, now time.Time) (domain.Post, error)
	Delete(c context.Context, id int64) error
	Search(c context.Context, req model.SearchPostRequest, now time.Time) ([]domain.Post, string, error)
	SlugExists(c context.Context, slug string, excludeID *int64) (bool, error)
}

// MediaChecker verifies media existence and accessibility for cover media attachment.
type MediaChecker interface {
	GetByID(c context.Context, id int64) (domain.Media, error)
}

// PostServiceOption provides configuration options for PostService.
type PostServiceOption func(*PostService)

// WithPostClock overrides the current time provider in PostService (useful for unit testing).
func WithPostClock(now func() time.Time) PostServiceOption {
	return func(s *PostService) {
		s.now = now
	}
}

// WithPostMediaChecker wires MediaChecker into PostService for validating cover media references.
func WithPostMediaChecker(checker MediaChecker) PostServiceOption {
	return func(s *PostService) {
		s.mediaChecker = checker
	}
}

// PostService handles business logic and workflow orchestration for CMS posts.
type PostService struct {
	repo         PostRepository
	mediaChecker MediaChecker
	now          func() time.Time
}

// NewPostService constructs a new PostService backed by the given PostRepository and options.
func NewPostService(repo PostRepository, opts ...PostServiceOption) PostService {
	s := PostService{
		repo: repo,
		now:  time.Now,
	}
	for _, opt := range opts {
		opt(&s)
	}
	return s
}

// Create validates and creates a new CMS post in DRAFT status.
func (s PostService) Create(c context.Context, req model.CreatePostRequest) (domain.Post, error) {
	slug, err := s.resolveUniqueSlug(c, req.Title, req.Slug)
	if err != nil {
		return domain.Post{}, err
	}

	if req.CoverMediaID != nil {
		if err := s.validateCoverMedia(c, *req.CoverMediaID); err != nil {
			return domain.Post{}, err
		}
	}

	body := ""
	if req.Body != nil {
		body = *req.Body
	}
	excerpt := ""
	if req.Excerpt != nil {
		excerpt = *req.Excerpt
	}

	return s.repo.Create(c, model.CreatePostRecord{
		Title:        req.Title,
		Slug:         slug,
		Excerpt:      excerpt,
		Body:         body,
		CoverMediaID: req.CoverMediaID,
		Status:       domain.StatusDraft,
	}, s.now())
}

// GetByID retrieves a post by ID.
func (s PostService) GetByID(c context.Context, id int64) (domain.Post, error) {
	return s.repo.GetByID(c, id, s.now())
}

// Search searches posts matching search and pagination parameters.
func (s PostService) Search(c context.Context, req model.SearchPostRequest) ([]domain.Post, string, error) {
	return s.repo.Search(c, req, s.now())
}

// Update modifies metadata and content of an existing post.
func (s PostService) Update(c context.Context, id int64, req model.UpdatePostRequest) (domain.Post, error) {
	if req.Slug != nil && *req.Slug != "" {
		exists, err := s.repo.SlugExists(c, *req.Slug, &id)
		if err != nil {
			return domain.Post{}, err
		}
		if exists {
			return domain.Post{}, domain.ErrSlugTaken
		}
	}

	if req.CoverMediaID != nil {
		if *req.CoverMediaID > 0 {
			if err := s.validateCoverMedia(c, *req.CoverMediaID); err != nil {
				return domain.Post{}, err
			}
		}
	}

	record := model.UpdatePostRecord{
		Title:   req.Title,
		Slug:    req.Slug,
		Excerpt: req.Excerpt,
		Body:    req.Body,
	}

	if req.CoverMediaID != nil {
		if *req.CoverMediaID <= 0 {
			record.ClearCoverMedia = true
		} else {
			record.CoverMediaID = req.CoverMediaID
		}
	}

	return s.repo.Update(c, id, record, s.now())
}

// Delete soft-deletes a post by ID.
func (s PostService) Delete(c context.Context, id int64) error {
	return s.repo.Delete(c, id)
}

// Publish publishes a draft or scheduled post, setting status to PUBLISHED and publish_at to now.
func (s PostService) Publish(c context.Context, id int64) (domain.Post, error) {
	now := s.now()
	p, err := s.repo.GetByID(c, id, now)
	if err != nil {
		return domain.Post{}, err
	}

	newStatus, newPublishAt, err := domain.Publish(p.Status, p.PublishAt, now)
	if err != nil {
		return domain.Post{}, err
	}

	return s.repo.Update(c, id, model.UpdatePostRecord{
		Status:    &newStatus,
		PublishAt: newPublishAt,
	}, now)
}

// Schedule schedules a draft or scheduled post to go live at publishAt.
func (s PostService) Schedule(c context.Context, id int64, req model.SchedulePostRequest) (domain.Post, error) {
	now := s.now()
	p, err := s.repo.GetByID(c, id, now)
	if err != nil {
		return domain.Post{}, err
	}

	newStatus, newPublishAt, err := domain.Schedule(p.Status, p.PublishAt, req.PublishAt, now)
	if err != nil {
		return domain.Post{}, err
	}

	return s.repo.Update(c, id, model.UpdatePostRecord{
		Status:    &newStatus,
		PublishAt: newPublishAt,
	}, now)
}

// Unpublish reverts a published or scheduled post to DRAFT status and clears publish_at.
func (s PostService) Unpublish(c context.Context, id int64) (domain.Post, error) {
	now := s.now()
	p, err := s.repo.GetByID(c, id, now)
	if err != nil {
		return domain.Post{}, err
	}

	newStatus, _, err := domain.Unpublish(p.Status, p.PublishAt, now)
	if err != nil {
		return domain.Post{}, err
	}

	return s.repo.Update(c, id, model.UpdatePostRecord{
		Status:         &newStatus,
		ClearPublishAt: true,
	}, now)
}

func (s PostService) resolveUniqueSlug(c context.Context, title string, explicitSlug *string) (string, error) {
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

func (s PostService) validateCoverMedia(c context.Context, mediaID int64) error {
	if s.mediaChecker == nil {
		return nil
	}
	_, err := s.mediaChecker.GetByID(c, mediaID)
	if err != nil {
		return domain.ErrMediaNotFound
	}
	return nil
}
