package domain_test

import (
	"testing"
	"time"

	"github.com/kilip/omed/api/internal/cms/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEffectiveStatus(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name      string
		status    string
		publishAt *time.Time
		expected  string
	}{
		{
			name:      "draft with nil publishAt",
			status:    domain.StatusDraft,
			publishAt: nil,
			expected:  domain.EffectiveStatusDraft,
		},
		{
			name:      "draft with past publishAt",
			status:    domain.StatusDraft,
			publishAt: &past,
			expected:  domain.EffectiveStatusDraft,
		},
		{
			name:      "draft with future publishAt",
			status:    domain.StatusDraft,
			publishAt: &future,
			expected:  domain.EffectiveStatusDraft,
		},
		{
			name:      "empty status treated as draft",
			status:    "",
			publishAt: &past,
			expected:  domain.EffectiveStatusDraft,
		},
		{
			name:      "published with nil publishAt",
			status:    domain.StatusPublished,
			publishAt: nil,
			expected:  domain.EffectiveStatusPublished,
		},
		{
			name:      "published with past publishAt",
			status:    domain.StatusPublished,
			publishAt: &past,
			expected:  domain.EffectiveStatusPublished,
		},
		{
			name:      "published with publishAt exactly equal to now",
			status:    domain.StatusPublished,
			publishAt: &now,
			expected:  domain.EffectiveStatusPublished,
		},
		{
			name:      "published with future publishAt is scheduled",
			status:    domain.StatusPublished,
			publishAt: &future,
			expected:  domain.EffectiveStatusScheduled,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := domain.EffectiveStatus(tc.status, tc.publishAt, now)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestPublish(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name        string
		curStatus   string
		curPubAt    *time.Time
		expectedErr error
	}{
		{
			name:        "publish from draft",
			curStatus:   domain.StatusDraft,
			curPubAt:    nil,
			expectedErr: nil,
		},
		{
			name:        "publish from scheduled (status=PUBLISHED, pubAt in future)",
			curStatus:   domain.StatusPublished,
			curPubAt:    &future,
			expectedErr: nil,
		},
		{
			name:        "publish from already published (past pubAt)",
			curStatus:   domain.StatusPublished,
			curPubAt:    &past,
			expectedErr: domain.ErrInvalidTransition,
		},
		{
			name:        "publish from already published (pubAt == now)",
			curStatus:   domain.StatusPublished,
			curPubAt:    &now,
			expectedErr: domain.ErrInvalidTransition,
		},
		{
			name:        "publish from unknown status",
			curStatus:   "UNKNOWN",
			curPubAt:    nil,
			expectedErr: domain.ErrInvalidTransition,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			newStatus, newPubAt, err := domain.Publish(tc.curStatus, tc.curPubAt, now)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
				assert.Empty(t, newStatus)
				assert.Nil(t, newPubAt)
			} else {
				require.NoError(t, err)
				assert.Equal(t, domain.StatusPublished, newStatus)
				require.NotNil(t, newPubAt)
				assert.True(t, newPubAt.Equal(now))
			}
		})
	}
}

func TestSchedule(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)
	future2 := now.Add(2 * time.Hour)

	tests := []struct {
		name        string
		curStatus   string
		curPubAt    *time.Time
		scheduleAt  time.Time
		expectedErr error
	}{
		{
			name:        "schedule from draft with future time",
			curStatus:   domain.StatusDraft,
			curPubAt:    nil,
			scheduleAt:  future,
			expectedErr: nil,
		},
		{
			name:        "reschedule from scheduled with new future time",
			curStatus:   domain.StatusPublished,
			curPubAt:    &future,
			scheduleAt:  future2,
			expectedErr: nil,
		},
		{
			name:        "schedule from draft with past time",
			curStatus:   domain.StatusDraft,
			curPubAt:    nil,
			scheduleAt:  past,
			expectedErr: domain.ErrPublishAtInPast,
		},
		{
			name:        "schedule from draft with time equal to now",
			curStatus:   domain.StatusDraft,
			curPubAt:    nil,
			scheduleAt:  now,
			expectedErr: domain.ErrPublishAtInPast,
		},
		{
			name:        "schedule from scheduled with past time",
			curStatus:   domain.StatusPublished,
			curPubAt:    &future,
			scheduleAt:  past,
			expectedErr: domain.ErrPublishAtInPast,
		},
		{
			name:        "schedule from already published",
			curStatus:   domain.StatusPublished,
			curPubAt:    &past,
			scheduleAt:  future,
			expectedErr: domain.ErrInvalidTransition,
		},
		{
			name:        "schedule from unknown status",
			curStatus:   "UNKNOWN",
			curPubAt:    nil,
			scheduleAt:  future,
			expectedErr: domain.ErrInvalidTransition,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			newStatus, newPubAt, err := domain.Schedule(tc.curStatus, tc.curPubAt, tc.scheduleAt, now)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
				assert.Empty(t, newStatus)
				assert.Nil(t, newPubAt)
			} else {
				require.NoError(t, err)
				assert.Equal(t, domain.StatusPublished, newStatus)
				require.NotNil(t, newPubAt)
				assert.True(t, newPubAt.Equal(tc.scheduleAt))
			}
		})
	}
}

func TestUnpublish(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	past := now.Add(-1 * time.Hour)
	future := now.Add(1 * time.Hour)

	tests := []struct {
		name        string
		curStatus   string
		curPubAt    *time.Time
		expectedErr error
	}{
		{
			name:        "unpublish from published",
			curStatus:   domain.StatusPublished,
			curPubAt:    &past,
			expectedErr: nil,
		},
		{
			name:        "unpublish from scheduled",
			curStatus:   domain.StatusPublished,
			curPubAt:    &future,
			expectedErr: nil,
		},
		{
			name:        "unpublish from draft is invalid",
			curStatus:   domain.StatusDraft,
			curPubAt:    nil,
			expectedErr: domain.ErrInvalidTransition,
		},
		{
			name:        "unpublish from unknown status",
			curStatus:   "UNKNOWN",
			curPubAt:    nil,
			expectedErr: domain.ErrInvalidTransition,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			newStatus, newPubAt, err := domain.Unpublish(tc.curStatus, tc.curPubAt, now)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
				assert.Empty(t, newStatus)
				assert.Nil(t, newPubAt)
			} else {
				require.NoError(t, err)
				assert.Equal(t, domain.StatusDraft, newStatus)
				assert.Nil(t, newPubAt)
			}
		})
	}
}
