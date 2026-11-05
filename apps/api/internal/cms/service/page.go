//go:generate go run go.uber.org/mock/mockgen -source=page.go -destination=mock/mock_page.go -package=mock

package service

import (
	"context"
	"strings"
	"time"

	"github.com/kilip/omed/api/internal/base"
	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/kilip/omed/api/internal/cms/model"
)

// PageRepository defines persistence operations required by PageService.
type PageRepository interface {
	GetByID(c context.Context, id int64, requestedLocale, defaultLocale string, now time.Time) (domain.Page, error)
	Create(c context.Context, req model.CreatePageRecord, defaultLocale string, now time.Time) (domain.Page, error)
	Update(c context.Context, id int64, req model.UpdatePageRecord, defaultLocale string, now time.Time) (domain.Page, error)
	SaveTranslation(c context.Context, req model.SavePageTranslationRecord, defaultLocale string, now time.Time) (domain.Page, error)
	DeleteTranslation(c context.Context, pageID int64, locale string) error
	Delete(c context.Context, id int64) error
	Search(c context.Context, req model.SearchPageRequest, defaultLocale string, now time.Time) ([]domain.Page, string, error)
	SlugExistsInLocale(c context.Context, workspaceID int64, locale, slug string, excludePageID *int64) (bool, error)
	PublicList(c context.Context, workspaceID int64, locale, defaultLocale string, cursor *string, limit int, now time.Time) ([]domain.Page, string, error)
	PublicGetBySlug(c context.Context, workspaceID int64, locale, defaultLocale, slug string, now time.Time) (domain.Page, bool, []model.PublicAlternate, error)
}

// PageServiceOption provides configuration options for PageService.
type PageServiceOption func(*PageService)

// WithPageClock overrides the current time provider in PageService (useful for unit testing).
func WithPageClock(now func() time.Time) PageServiceOption {
	return func(s *PageService) {
		s.now = now
	}
}

// WithPageSettingProvider wires SettingProvider into PageService.
func WithPageSettingProvider(provider SettingProvider) PageServiceOption {
	return func(s *PageService) {
		s.settingProvider = provider
	}
}

// PageService handles business logic and workflow orchestration for CMS pages.
type PageService struct {
	repo            PageRepository
	settingProvider SettingProvider
	now             func() time.Time
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

func (s PageService) getSetting(c context.Context, workspaceID int64) domain.Setting {
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

// Create validates and creates a new CMS page and its initial translation in DRAFT status.
func (s PageService) Create(c context.Context, req model.CreatePageRequest) (domain.Page, error) {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	targetLocale := setting.DefaultLocale
	if req.Locale != nil && *req.Locale != "" {
		norm, err := domain.NormalizeLocale(*req.Locale)
		if err != nil {
			return domain.Page{}, domain.ErrInvalidLocale
		}
		if !domain.IsLocaleSupported(norm, setting.SupportedLocales) {
			return domain.Page{}, domain.ErrLocaleNotSupported
		}
		targetLocale = norm
	}

	slug, err := s.resolveUniqueSlug(c, wsID, targetLocale, req.Title, req.Slug, nil)
	if err != nil {
		return domain.Page{}, err
	}

	body := ""
	if req.Body != nil {
		body = *req.Body
	}

	return s.repo.Create(c, model.CreatePageRecord{
		WorkspaceID: wsID,
		Locale:      targetLocale,
		Title:       req.Title,
		Slug:        slug,
		Body:        body,
		Status:      domain.StatusDraft,
	}, setting.DefaultLocale, s.now())
}

// GetByID retrieves a page by ID for the requested locale (or default locale).
func (s PageService) GetByID(c context.Context, id int64, requestedLocale ...string) (domain.Page, error) {
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

// Search searches pages matching search and pagination parameters.
func (s PageService) Search(c context.Context, req model.SearchPageRequest) ([]domain.Page, string, error) {
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

// PublicList retrieves publicly visible pages for a workspace with cursor pagination.
func (s PageService) PublicList(c context.Context, workspaceID int64, locale string, cursor *string, limit int) ([]domain.Page, string, error) {
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

// PublicGetBySlug retrieves a publicly visible page by slug for a workspace, returning domain.Page, fallback flag, and alternates.
func (s PageService) PublicGetBySlug(c context.Context, workspaceID int64, locale, slug string) (domain.Page, bool, []model.PublicAlternate, error) {
	setting := s.getSetting(c, workspaceID)

	targetLocale := locale
	if targetLocale == "" {
		targetLocale = setting.DefaultLocale
	} else {
		norm, err := domain.NormalizeLocale(targetLocale)
		if err != nil {
			return domain.Page{}, false, nil, domain.ErrInvalidLocale
		}
		if !domain.IsLocaleSupported(norm, setting.SupportedLocales) {
			return domain.Page{}, false, nil, domain.ErrLocaleNotSupported
		}
		targetLocale = norm
	}

	return s.repo.PublicGetBySlug(c, workspaceID, targetLocale, setting.DefaultLocale, slug, s.now())
}

// Update modifies metadata and content of an existing page.
func (s PageService) Update(c context.Context, id int64, req model.UpdatePageRequest, locale ...string) (domain.Page, error) {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	targetLocale := setting.DefaultLocale
	if len(locale) > 0 && locale[0] != "" {
		norm, err := domain.NormalizeLocale(locale[0])
		if err != nil {
			return domain.Page{}, domain.ErrInvalidLocale
		}
		targetLocale = norm
	} else if req.Locale != nil && *req.Locale != "" {
		norm, err := domain.NormalizeLocale(*req.Locale)
		if err != nil {
			return domain.Page{}, domain.ErrInvalidLocale
		}
		targetLocale = norm
	}

	if !domain.IsLocaleSupported(targetLocale, setting.SupportedLocales) {
		return domain.Page{}, domain.ErrLocaleNotSupported
	}

	if req.Slug != nil && *req.Slug != "" {
		exists, err := s.repo.SlugExistsInLocale(c, wsID, targetLocale, *req.Slug, &id)
		if err != nil {
			return domain.Page{}, err
		}
		if exists {
			return domain.Page{}, domain.ErrSlugTaken
		}
	}

	record := model.UpdatePageRecord{
		Locale: targetLocale,
		Title:  req.Title,
		Slug:   req.Slug,
		Body:   req.Body,
	}

	return s.repo.Update(c, id, record, setting.DefaultLocale, s.now())
}

// SaveTranslation creates or updates a translation for a page.
func (s PageService) SaveTranslation(c context.Context, id int64, locale string, req model.SavePageTranslationRequest) (domain.Page, error) {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	norm, err := domain.NormalizeLocale(locale)
	if err != nil {
		return domain.Page{}, domain.ErrInvalidLocale
	}
	if !domain.IsLocaleSupported(norm, setting.SupportedLocales) {
		return domain.Page{}, domain.ErrLocaleNotSupported
	}

	slug, err := s.resolveUniqueSlug(c, wsID, norm, req.Title, req.Slug, &id)
	if err != nil {
		return domain.Page{}, err
	}

	body := ""
	if req.Body != nil {
		body = *req.Body
	}

	return s.repo.SaveTranslation(c, model.SavePageTranslationRecord{
		PageID:      id,
		WorkspaceID: wsID,
		Locale:      norm,
		Title:       req.Title,
		Slug:        slug,
		Body:        body,
		Status:      domain.StatusDraft,
	}, setting.DefaultLocale, s.now())
}

// DeleteTranslation removes a translation for a page, enforcing that the default translation cannot be deleted if others exist.
func (s PageService) DeleteTranslation(c context.Context, id int64, locale string) error {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	norm, err := domain.NormalizeLocale(locale)
	if err != nil {
		return domain.ErrInvalidLocale
	}

	page, err := s.repo.GetByID(c, id, norm, setting.DefaultLocale, s.now())
	if err != nil {
		return err
	}

	if len(page.Translations) <= 1 {
		return domain.ErrDefaultTranslationRequired
	}

	if norm == setting.DefaultLocale {
		return domain.ErrDefaultTranslationRequired
	}

	return s.repo.DeleteTranslation(c, id, norm)
}

// Delete soft-deletes a page and all its translations by ID.
func (s PageService) Delete(c context.Context, id int64) error {
	return s.repo.Delete(c, id)
}

func hasPageTranslationForLocale(p domain.Page, targetLocale string) bool {
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

// Publish publishes a draft or scheduled page translation, setting status to PUBLISHED and publish_at to now.
func (s PageService) Publish(c context.Context, id int64, locale ...string) (domain.Page, error) {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	targetLocale := setting.DefaultLocale
	if len(locale) > 0 && locale[0] != "" {
		norm, err := domain.NormalizeLocale(locale[0])
		if err != nil {
			return domain.Page{}, domain.ErrInvalidLocale
		}
		targetLocale = norm
	}

	if !domain.IsLocaleSupported(targetLocale, setting.SupportedLocales) {
		return domain.Page{}, domain.ErrLocaleNotSupported
	}

	now := s.now()
	p, err := s.repo.GetByID(c, id, targetLocale, setting.DefaultLocale, now)
	if err != nil {
		return domain.Page{}, err
	}

	if !hasPageTranslationForLocale(p, targetLocale) {
		return domain.Page{}, domain.ErrTranslationNotFound
	}

	newStatus, newPublishAt, err := domain.Publish(p.Status, p.PublishAt, now)
	if err != nil {
		return domain.Page{}, err
	}

	return s.repo.Update(c, id, model.UpdatePageRecord{
		Locale:    targetLocale,
		Status:    &newStatus,
		PublishAt: newPublishAt,
	}, setting.DefaultLocale, now)
}

// Schedule schedules a page translation to go live at publishAt.
func (s PageService) Schedule(c context.Context, id int64, req model.SchedulePageRequest, locale ...string) (domain.Page, error) {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	targetLocale := setting.DefaultLocale
	if len(locale) > 0 && locale[0] != "" {
		norm, err := domain.NormalizeLocale(locale[0])
		if err != nil {
			return domain.Page{}, domain.ErrInvalidLocale
		}
		targetLocale = norm
	}

	if !domain.IsLocaleSupported(targetLocale, setting.SupportedLocales) {
		return domain.Page{}, domain.ErrLocaleNotSupported
	}

	now := s.now()
	p, err := s.repo.GetByID(c, id, targetLocale, setting.DefaultLocale, now)
	if err != nil {
		return domain.Page{}, err
	}

	if !hasPageTranslationForLocale(p, targetLocale) {
		return domain.Page{}, domain.ErrTranslationNotFound
	}

	newStatus, newPublishAt, err := domain.Schedule(p.Status, p.PublishAt, req.PublishAt, now)
	if err != nil {
		return domain.Page{}, err
	}

	return s.repo.Update(c, id, model.UpdatePageRecord{
		Locale:    targetLocale,
		Status:    &newStatus,
		PublishAt: newPublishAt,
	}, setting.DefaultLocale, now)
}

// Unpublish reverts a page translation to DRAFT status and clears publish_at.
func (s PageService) Unpublish(c context.Context, id int64, locale ...string) (domain.Page, error) {
	wsID := base.WorkspaceIDFromContext(c)
	setting := s.getSetting(c, wsID)

	targetLocale := setting.DefaultLocale
	if len(locale) > 0 && locale[0] != "" {
		norm, err := domain.NormalizeLocale(locale[0])
		if err != nil {
			return domain.Page{}, domain.ErrInvalidLocale
		}
		targetLocale = norm
	}

	if !domain.IsLocaleSupported(targetLocale, setting.SupportedLocales) {
		return domain.Page{}, domain.ErrLocaleNotSupported
	}

	now := s.now()
	p, err := s.repo.GetByID(c, id, targetLocale, setting.DefaultLocale, now)
	if err != nil {
		return domain.Page{}, err
	}

	if !hasPageTranslationForLocale(p, targetLocale) {
		return domain.Page{}, domain.ErrTranslationNotFound
	}

	newStatus, _, err := domain.Unpublish(p.Status, p.PublishAt, now)
	if err != nil {
		return domain.Page{}, err
	}

	return s.repo.Update(c, id, model.UpdatePageRecord{
		Locale:         targetLocale,
		Status:         &newStatus,
		ClearPublishAt: true,
	}, setting.DefaultLocale, now)
}

func (s PageService) resolveUniqueSlug(c context.Context, workspaceID int64, locale, title string, explicitSlug *string, excludeID *int64) (string, error) {
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
