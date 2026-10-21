package domain

import (
	"time"
)

const (
	// StatusDraft indicates a draft content item.
	StatusDraft = "DRAFT"
	// StatusPublished indicates a published content item.
	StatusPublished = "PUBLISHED"

	// EffectiveStatusDraft indicates an item effectively in draft state.
	EffectiveStatusDraft = "DRAFT"
	// EffectiveStatusScheduled indicates an item scheduled to go live in the future.
	EffectiveStatusScheduled = "SCHEDULED"
	// EffectiveStatusPublished indicates an item currently visible to the public.
	EffectiveStatusPublished = "PUBLISHED"
)

// EffectiveStatus computes the publication status based on status, publishAt, and now:
// - DRAFT if status == "DRAFT" (or not PUBLISHED).
// - SCHEDULED if status == "PUBLISHED" and publishAt > now.
// - PUBLISHED if status == "PUBLISHED" and publishAt <= now.
func EffectiveStatus(status string, publishAt *time.Time, now time.Time) string {
	if status != StatusPublished {
		return EffectiveStatusDraft
	}
	if publishAt != nil && publishAt.After(now) {
		return EffectiveStatusScheduled
	}
	return EffectiveStatusPublished
}

// Publish transitions a post/page to PUBLISHED with publish_at = now.
// Transitions from DRAFT or SCHEDULED -> PUBLISHED.
// Transitions from PUBLISHED -> ErrInvalidTransition.
func Publish(currentStatus string, currentPublishAt *time.Time, now time.Time) (string, *time.Time, error) {
	if currentStatus != StatusDraft && currentStatus != StatusPublished {
		return "", nil, ErrInvalidTransition
	}
	eff := EffectiveStatus(currentStatus, currentPublishAt, now)
	if eff == EffectiveStatusPublished {
		return "", nil, ErrInvalidTransition
	}
	t := now
	return StatusPublished, &t, nil
}

// Schedule transitions a post/page to PUBLISHED with publish_at = publishAt.
// publishAt must be strictly in the future (> now), otherwise ErrPublishAtInPast is returned.
// Transitions from DRAFT or SCHEDULED -> PUBLISHED.
// Transitions from PUBLISHED -> ErrInvalidTransition.
func Schedule(currentStatus string, currentPublishAt *time.Time, publishAt time.Time, now time.Time) (string, *time.Time, error) {
	if currentStatus != StatusDraft && currentStatus != StatusPublished {
		return "", nil, ErrInvalidTransition
	}
	eff := EffectiveStatus(currentStatus, currentPublishAt, now)
	if eff == EffectiveStatusPublished {
		return "", nil, ErrInvalidTransition
	}
	if !publishAt.After(now) {
		return "", nil, ErrPublishAtInPast
	}
	t := publishAt
	return StatusPublished, &t, nil
}

// Unpublish transitions a post/page to DRAFT with publish_at = nil.
// Transitions from PUBLISHED or SCHEDULED -> DRAFT.
// Transitions from DRAFT -> ErrInvalidTransition.
func Unpublish(currentStatus string, currentPublishAt *time.Time, now time.Time) (string, *time.Time, error) {
	if currentStatus != StatusDraft && currentStatus != StatusPublished {
		return "", nil, ErrInvalidTransition
	}
	eff := EffectiveStatus(currentStatus, currentPublishAt, now)
	if eff == EffectiveStatusDraft {
		return "", nil, ErrInvalidTransition
	}
	return StatusDraft, nil, nil
}
