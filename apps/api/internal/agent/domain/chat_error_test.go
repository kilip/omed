package domain_test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/agent/domain"
	"github.com/kilip/omed/api/internal/base"
	"github.com/stretchr/testify/assert"
)

func TestChatErrorsRegistered(t *testing.T) {
	testCases := []struct {
		err            error
		expectedCode   string
		expectedStatus int
		expectedMsg    string
	}{
		{
			err:            domain.ErrConversationNotFound,
			expectedCode:   "CONVERSATION_NOT_FOUND",
			expectedStatus: fiber.StatusNotFound,
			expectedMsg:    "conversation not found",
		},
		{
			err:            domain.ErrMessageNotFound,
			expectedCode:   "MESSAGE_NOT_FOUND",
			expectedStatus: fiber.StatusNotFound,
			expectedMsg:    "message not found",
		},
		{
			err:            domain.ErrDraftAlreadyConfirmed,
			expectedCode:   "DRAFT_ALREADY_CONFIRMED",
			expectedStatus: fiber.StatusConflict,
			expectedMsg:    "draft proposal has already been confirmed",
		},
		{
			err:            domain.ErrDraftAlreadyDiscarded,
			expectedCode:   "DRAFT_ALREADY_DISCARDED",
			expectedStatus: fiber.StatusConflict,
			expectedMsg:    "draft proposal has already been discarded",
		},
		{
			err:            domain.ErrDraftNotProposed,
			expectedCode:   "DRAFT_NOT_PROPOSED",
			expectedStatus: fiber.StatusUnprocessableEntity,
			expectedMsg:    "message does not contain a proposed draft",
		},
		{
			err:            domain.ErrInvalidDraftProposal,
			expectedCode:   "INVALID_DRAFT_PROPOSAL",
			expectedStatus: fiber.StatusUnprocessableEntity,
			expectedMsg:    "invalid draft proposal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.expectedCode, func(t *testing.T) {
			item := base.MapError(tc.err)
			assert.Equal(t, tc.expectedCode, item.Code)
			assert.Equal(t, tc.expectedStatus, item.Status)
			assert.Equal(t, tc.expectedMsg, item.Message)
		})
	}
}
