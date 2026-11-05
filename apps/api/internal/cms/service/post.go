//go:generate go run go.uber.org/mock/mockgen -source=post.go -destination=mock/mock_post.go -package=mock

package service

import (
	"context"
	"strings"
	"time"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
)

// PostRepository defines persistence operations required by PostService.
type PostRepository interface {
	GetByID(c context.Context, id int64, requestedLocale, defaultLocale string, now time.Time) (domain.Post, error)
	Create(c context.Context, req model.CreatePostRecord, defaultLocale string, now time.Time) (domain.Post, error)
	Update(c context.Context, id int64, req model.UpdatePostRecord, defaultLocale string, now time.Time) (domain.Post, error)
	SaveTranslation(c context.Context, req model.SavePostTranslationRecord, defaultLocale string, now time.Time) (domain.Post, error)
	DeleteTranslation(c context.Context, postID int64, locale string) error
	Delete(c context.Context, id int64) error
	Search(c context.Context, req model.SearchPostRequest, defaultLocale string, now time.Time) ([]domain.Post, string, error)
	SlugExistsInLocale(c context.Context, workspaceID int64, locale, slug string, excludePostID *int64) (bool, error)
	PublicList(c context.Context, workspaceID int64, locale, defaultLocale string, cursor *string, limit int, now time.Time) ([]domain.Post, string, error)
	PublicGetBySlug(c context.Context, workspaceID int64, locale, defaultLocale, slug string, now time.Time) (domain.Post, bool, []model.PublicAlternate, error)
}

// MediaChecker verifies media existence and accessibility for cover media attachment.
type MediaChecker interface {
	GetByID(c context.Context, id int64) (domain.Media, error)
}

// SettingProvider retrieves CMS settings for a workspace.
type SettingProvider interface {
	Get(c context.Context, workspaceID int64) (domain.Setting, error)
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

// WithPostSettingProvider wires SettingProvider into PostService.
func WithPostSettingProvider(provider SettingProvider) PostServiceOption {
	return func(s *PostService) {
		s.settingProvider = provider
	}
}

// PostService handles business logic and workflow orchestration for CMS posts.
type PostService struct {
	repo            PostRepository
	mediaChecker    MediaChecker
	settingProvider SettingProvider
	now             func() time.Time
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

func (s PostService) getSetting(c context.Context, workspaceID int64) domain.Setting {
	if s.settingProvider != nil && workspaceID > 0 {
		if st, err := s.settingProvider.Get(c, workspaceID); err == nil {
			return st
		}
	}
	return domain.Setting{
		WorkspaceID:      workspaceID,
		DefaultLocale:    "en",
		SupportedLocales: []string{"en", "id"},
	}
}

// Create validates and creates a new CMS post and its initial translation in DRAFT status.
func (s PostService) Create(c context.Context, req model.CreatePostRequest) (domain.Post, error) {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	targetLocale := setting.DefaultLocale
	if req.Locale != nil && *req.Locale != "" {
		norm, err := domain.NormalizeLocale(*req.Locale)
		if err != nil {
			return domain.Post{}, domain.ErrInvalidLocale
		}
		if !domain.IsLocaleSupported(norm, setting.SupportedLocales) {
			return domain.Post{}, domain.ErrLocaleNotSupported
		}
		targetLocale = norm
	}

	slug, err := s.resolveUniqueSlug(c, wsID, targetLocale, req.Title, req.Slug, nil)
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
		WorkspaceID:  wsID,
		Locale:       targetLocale,
		Title:        req.Title,
		Slug:         slug,
		Excerpt:      excerpt,
		Body:         body,
		CoverMediaID: req.CoverMediaID,
		Status:       domain.StatusDraft,
	}, setting.DefaultLocale, s.now())
}

// GetByID retrieves a post by ID for the requested locale (or default locale).
func (s PostService) GetByID(c context.Context, id int64, requestedLocale ...string) (domain.Post, error) {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	targetLocale := setting.DefaultLocale
	if len(requestedLocale) > 0 && requestedLocale[0] != "" {
		if norm, err := domain.NormalizeLocale(requestedLocale[0]); err == nil {
			targetLocale = norm
		}
	}

	return s.repo.GetByID(c, id, targetLocale, setting.DefaultLocale, s.now())
}

// Search searches posts matching search and pagination parameters.
func (s PostService) Search(c context.Context, req model.SearchPostRequest) ([]domain.Post, string, error) {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	targetLocale := setting.DefaultLocale
	if req.Locale != nil && *req.Locale != "" {
		if norm, err := domain.NormalizeLocale(*req.Locale); err == nil {
			targetLocale = norm
		}
	}

	return s.repo.Search(c, req, targetLocale, s.now())
}

// PublicList retrieves publicly visible posts for a workspace with cursor pagination.
func (s PostService) PublicList(c context.Context, workspaceID int64, locale string, cursor *string, limit int) ([]domain.Post, string, error) {
	setting := s.getSetting(c, workspaceID)

	targetLocale := locale
	if targetLocale == "" {
		targetLocale = setting.DefaultLocale
	} else {
		norm, err := domain.NormalizeLocale(targetLocale)
		if err != nil {
			return nil, "", domain.ErrInvalidLocale
		}
		if !domain.IsLocaleSupported(norm, setting.SupportedLocales) {
			return nil, "", domain.ErrLocaleNotSupported
		}
		targetLocale = norm
	}

	return s.repo.PublicList(c, workspaceID, targetLocale, setting.DefaultLocale, cursor, limit, s.now())
}

// PublicGetBySlug retrieves a publicly visible post by slug for a workspace, returning domain.Post, fallback flag, and alternates.
func (s PostService) PublicGetBySlug(c context.Context, workspaceID int64, locale, slug string) (domain.Post, bool, []model.PublicAlternate, error) {
	setting := s.getSetting(c, workspaceID)

	targetLocale := locale
	if targetLocale == "" {
		targetLocale = setting.DefaultLocale
	} else {
		norm, err := domain.NormalizeLocale(targetLocale)
		if err != nil {
			return domain.Post{}, false, nil, domain.ErrInvalidLocale
		}
		if !domain.IsLocaleSupported(norm, setting.SupportedLocales) {
			return domain.Post{}, false, nil, domain.ErrLocaleNotSupported
		}
		targetLocale = norm
	}

	return s.repo.PublicGetBySlug(c, workspaceID, targetLocale, setting.DefaultLocale, slug, s.now())
}

// Update modifies metadata and content of an existing post.
func (s PostService) Update(c context.Context, id int64, req model.UpdatePostRequest, locale ...string) (domain.Post, error) {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	targetLocale := setting.DefaultLocale
	if len(locale) > 0 && locale[0] != "" {
		norm, err := domain.NormalizeLocale(locale[0])
		if err != nil {
			return domain.Post{}, domain.ErrInvalidLocale
		}
		targetLocale = norm
	} else if req.Locale != nil && *req.Locale != "" {
		norm, err := domain.NormalizeLocale(*req.Locale)
		if err != nil {
			return domain.Post{}, domain.ErrInvalidLocale
		}
		targetLocale = norm
	}

	if !domain.IsLocaleSupported(targetLocale, setting.SupportedLocales) {
		return domain.Post{}, domain.ErrLocaleNotSupported
	}

	if req.Slug != nil && *req.Slug != "" {
		exists, err := s.repo.SlugExistsInLocale(c, wsID, targetLocale, *req.Slug, &id)
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
		Locale:  targetLocale,
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

	return s.repo.Update(c, id, record, setting.DefaultLocale, s.now())
}

// SaveTranslation creates or updates a translation for a post.
func (s PostService) SaveTranslation(c context.Context, id int64, locale string, req model.SavePostTranslationRequest) (domain.Post, error) {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	norm, err := domain.NormalizeLocale(locale)
	if err != nil {
		return domain.Post{}, domain.ErrInvalidLocale
	}
	if !domain.IsLocaleSupported(norm, setting.SupportedLocales) {
		return domain.Post{}, domain.ErrLocaleNotSupported
	}

	slug, err := s.resolveUniqueSlug(c, wsID, norm, req.Title, req.Slug, &id)
	if err != nil {
		return domain.Post{}, err
	}

	excerpt := ""
	if req.Excerpt != nil {
		excerpt = *req.Excerpt
	}
	body := ""
	if req.Body != nil {
		body = *req.Body
	}

	return s.repo.SaveTranslation(c, model.SavePostTranslationRecord{
		PostID:      id,
		WorkspaceID: wsID,
		Locale:      norm,
		Title:       req.Title,
		Slug:        slug,
		Excerpt:     excerpt,
		Body:        body,
		Status:      domain.StatusDraft,
	}, setting.DefaultLocale, s.now())
}

// DeleteTranslation removes a translation for a post, enforcing that the default translation cannot be deleted if others exist.
func (s PostService) DeleteTranslation(c context.Context, id int64, locale string) error {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	norm, err := domain.NormalizeLocale(locale)
	if err != nil {
		return domain.ErrInvalidLocale
	}

	post, err := s.repo.GetByID(c, id, norm, setting.DefaultLocale, s.now())
	if err != nil {
		return err
	}

	if len(post.Translations) <= 1 {
		return domain.ErrDefaultTranslationRequired
	}

	if norm == setting.DefaultLocale {
		return domain.ErrDefaultTranslationRequired
	}

	return s.repo.DeleteTranslation(c, id, norm)
}

// Delete soft-deletes a post and all its translations by ID.
func (s PostService) Delete(c context.Context, id int64) error {
	return s.repo.Delete(c, id)
}

func hasPostTranslationForLocale(p domain.Post, targetLocale string) bool {
	if strings.EqualFold(p.Locale, targetLocale) {
		return true
	}
	for _, tr := range p.Translations {
		if strings.EqualFold(tr.Locale, targetLocale) {
			return true
		}
	}
	return false
}

// Publish publishes a draft or scheduled post translation, setting status to PUBLISHED and publish_at to now.
func (s PostService) Publish(c context.Context, id int64, locale ...string) (domain.Post, error) {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	targetLocale := setting.DefaultLocale
	if len(locale) > 0 && locale[0] != "" {
		norm, err := domain.NormalizeLocale(locale[0])
		if err != nil {
			return domain.Post{}, domain.ErrInvalidLocale
		}
		targetLocale = norm
	}

	if !domain.IsLocaleSupported(targetLocale, setting.SupportedLocales) {
		return domain.Post{}, domain.ErrLocaleNotSupported
	}

	now := s.now()
	p, err := s.repo.GetByID(c, id, targetLocale, setting.DefaultLocale, now)
	if err != nil {
		return domain.Post{}, err
	}

	if !hasPostTranslationForLocale(p, targetLocale) {
		return domain.Post{}, domain.ErrTranslationNotFound
	}

	newStatus, newPublishAt, err := domain.Publish(p.Status, p.PublishAt, now)
	if err != nil {
		return domain.Post{}, err
	}

	return s.repo.Update(c, id, model.UpdatePostRecord{
		Locale:    targetLocale,
		Status:    &newStatus,
		PublishAt: newPublishAt,
	}, setting.DefaultLocale, now)
}

// Schedule schedules a post translation to go live at publishAt.
func (s PostService) Schedule(c context.Context, id int64, req model.SchedulePostRequest, locale ...string) (domain.Post, error) {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	targetLocale := setting.DefaultLocale
	if len(locale) > 0 && locale[0] != "" {
		norm, err := domain.NormalizeLocale(locale[0])
		if err != nil {
			return domain.Post{}, domain.ErrInvalidLocale
		}
		targetLocale = norm
	}

	if !domain.IsLocaleSupported(targetLocale, setting.SupportedLocales) {
		return domain.Post{}, domain.ErrLocaleNotSupported
	}

	now := s.now()
	p, err := s.repo.GetByID(c, id, targetLocale, setting.DefaultLocale, now)
	if err != nil {
		return domain.Post{}, err
	}

	if !hasPostTranslationForLocale(p, targetLocale) {
		return domain.Post{}, domain.ErrTranslationNotFound
	}

	newStatus, newPublishAt, err := domain.Schedule(p.Status, p.PublishAt, req.PublishAt, now)
	if err != nil {
		return domain.Post{}, err
	}

	return s.repo.Update(c, id, model.UpdatePostRecord{
		Locale:    targetLocale,
		Status:    &newStatus,
		PublishAt: newPublishAt,
	}, setting.DefaultLocale, now)
}

// Unpublish reverts a post translation to DRAFT status and clears publish_at.
func (s PostService) Unpublish(c context.Context, id int64, locale ...string) (domain.Post, error) {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	targetLocale := setting.DefaultLocale
	if len(locale) > 0 && locale[0] != "" {
		norm, err := domain.NormalizeLocale(locale[0])
		if err != nil {
			return domain.Post{}, domain.ErrInvalidLocale
		}
		targetLocale = norm
	}

	if !domain.IsLocaleSupported(targetLocale, setting.SupportedLocales) {
		return domain.Post{}, domain.ErrLocaleNotSupported
	}

	now := s.now()
	p, err := s.repo.GetByID(c, id, targetLocale, setting.DefaultLocale, now)
	if err != nil {
		return domain.Post{}, err
	}

	if !hasPostTranslationForLocale(p, targetLocale) {
		return domain.Post{}, domain.ErrTranslationNotFound
	}

	newStatus, _, err := domain.Unpublish(p.Status, p.PublishAt, now)
	if err != nil {
		return domain.Post{}, err
	}

	return s.repo.Update(c, id, model.UpdatePostRecord{
		Locale:         targetLocale,
		Status:         &newStatus,
		ClearPublishAt: true,
	}, setting.DefaultLocale, now)
}

func (s PostService) resolveUniqueSlug(c context.Context, workspaceID int64, locale, title string, explicitSlug *string, excludeID *int64) (string, error) {
	if explicitSlug != nil && *explicitSlug != "" {
		exists, err := s.repo.SlugExistsInLocale(c, workspaceID, locale, *explicitSlug, excludeID)
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
		exists, err := s.repo.SlugExistsInLocale(c, workspaceID, locale, slug, excludeID)
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
