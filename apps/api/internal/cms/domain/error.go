// Package domain defines the domain models and business errors for the CMS module.
package domain

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
)

var (
	// ErrSlugTaken indicates the slug is already in use in the workspace.
	ErrSlugTaken = errors.New("SLUG_TAKEN")
	// ErrInvalidTransition indicates an invalid publish status transition.
	ErrInvalidTransition = errors.New("INVALID_TRANSITION")
	// ErrPublishAtInPast indicates the scheduled publish time is in the past.
	ErrPublishAtInPast = errors.New("PUBLISH_AT_IN_PAST")
	// ErrMediaInUse indicates the media cannot be deleted because it is referenced.
	ErrMediaInUse = errors.New("MEDIA_IN_USE")
	// ErrMediaTypeNotAllowed indicates the media file type is not allowed.
	ErrMediaTypeNotAllowed = errors.New("MEDIA_TYPE_NOT_ALLOWED")
	// ErrMediaTooLarge indicates the media file exceeds the size limit.
	ErrMediaTooLarge = errors.New("MEDIA_TOO_LARGE")

	// ErrPostNotFound indicates the requested post was not found.
	ErrPostNotFound = errors.New("POST_NOT_FOUND")
	// ErrPageNotFound indicates the requested page was not found.
	ErrPageNotFound = errors.New("PAGE_NOT_FOUND")
	// ErrMediaNotFound indicates the requested media was not found.
	ErrMediaNotFound = errors.New("MEDIA_NOT_FOUND")

	// ErrTitleRequired indicates the title field is missing or empty.
	ErrTitleRequired = errors.New("TITLE_REQUIRED")
	// ErrPublishAtRequired indicates the publishAt field is missing or zero.
	ErrPublishAtRequired = errors.New("PUBLISH_AT_REQUIRED")

	// ErrInvalidLocale indicates the locale is not a valid BCP 47 language tag.
	ErrInvalidLocale = errors.New("INVALID_LOCALE")
	// ErrLocaleNotSupported indicates the requested locale is not configured in workspace supported locales.
	ErrLocaleNotSupported = errors.New("LOCALE_NOT_SUPPORTED")
	// ErrDefaultTranslationRequired indicates the default locale translation cannot be deleted while others exist.
	ErrDefaultTranslationRequired = errors.New("DEFAULT_TRANSLATION_REQUIRED")
	// ErrDefaultLocaleChangeBlocked indicates default locale change is rejected because existing items are missing translations.
	ErrDefaultLocaleChangeBlocked = errors.New("DEFAULT_LOCALE_CHANGE_BLOCKED")
	// ErrLocaleInUse indicates a locale cannot be removed from supported locales because translations exist for it.
	ErrLocaleInUse = errors.New("LOCALE_IN_USE")
	// ErrTranslationNotFound indicates the translation for the specified locale was not found.
	ErrTranslationNotFound = errors.New("TRANSLATION_NOT_FOUND")
)

func init() {
	base.AddToErrorMap(ErrSlugTaken, fiber.StatusConflict, "slug already in use in this workspace")
	base.AddToErrorMap(ErrInvalidTransition, fiber.StatusConflict, "invalid publication status transition")
	base.AddToErrorMap(ErrPublishAtInPast, fiber.StatusUnprocessableEntity, "publish_at must be in the future")
	base.AddToErrorMap(ErrMediaInUse, fiber.StatusConflict, "media cannot be deleted because it is referenced")
	base.AddToErrorMap(ErrMediaTypeNotAllowed, fiber.StatusUnprocessableEntity, "media file type not allowed")
	base.AddToErrorMap(ErrMediaTooLarge, fiber.StatusUnprocessableEntity, "media file exceeds maximum allowed size")

	base.AddToErrorMap(ErrTitleRequired, fiber.StatusUnprocessableEntity, "title is required")
	base.AddToErrorMap(ErrPublishAtRequired, fiber.StatusUnprocessableEntity, "publish_at is required")

	base.AddToErrorMap(ErrPostNotFound, fiber.StatusNotFound, "post not found")
	base.AddToErrorMap(ErrPageNotFound, fiber.StatusNotFound, "page not found")
	base.AddToErrorMap(ErrMediaNotFound, fiber.StatusNotFound, "media not found")

	base.AddToErrorMap(ErrInvalidLocale, fiber.StatusBadRequest, "invalid BCP 47 locale tag")
	base.AddToErrorMap(ErrLocaleNotSupported, fiber.StatusBadRequest, "locale is not supported in this workspace")
	base.AddToErrorMap(ErrDefaultTranslationRequired, fiber.StatusUnprocessableEntity, "default locale translation is required and cannot be deleted while other translations exist")
	base.AddToErrorMap(ErrDefaultLocaleChangeBlocked, fiber.StatusConflict, "cannot change default locale; existing content is missing translations in new default locale")
	base.AddToErrorMap(ErrLocaleInUse, fiber.StatusConflict, "cannot remove locale while translations exist for it")
	base.AddToErrorMap(ErrTranslationNotFound, fiber.StatusNotFound, "translation not found")
}
