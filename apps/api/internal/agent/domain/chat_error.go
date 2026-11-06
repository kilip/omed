package domain

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
)

var (
	// ErrConversationNotFound indicates that the requested chat conversation was not found or has been deleted.
	ErrConversationNotFound = errors.New("CONVERSATION_NOT_FOUND")
	// ErrMessageNotFound indicates that the requested chat message was not found.
	ErrMessageNotFound = errors.New("MESSAGE_NOT_FOUND")
	// ErrDraftAlreadyConfirmed indicates that the proposed draft has already been confirmed.
	ErrDraftAlreadyConfirmed = errors.New("DRAFT_ALREADY_CONFIRMED")
	// ErrDraftAlreadyDiscarded indicates that the proposed draft has already been discarded.
	ErrDraftAlreadyDiscarded = errors.New("DRAFT_ALREADY_DISCARDED")
	// ErrDraftNotProposed indicates that the message does not have a proposed draft.
	ErrDraftNotProposed = errors.New("DRAFT_NOT_PROPOSED")
	// ErrInvalidDraftProposal indicates that the draft proposal is malformed or invalid.
	ErrInvalidDraftProposal = errors.New("INVALID_DRAFT_PROPOSAL")
	// ErrAttachmentNotFound indicates that the requested chat attachment was not found.
	ErrAttachmentNotFound = errors.New("ATTACHMENT_NOT_FOUND")
)

func init() {
	base.AddToErrorMap(ErrConversationNotFound, fiber.StatusNotFound, "conversation not found")
	base.AddToErrorMap(ErrMessageNotFound, fiber.StatusNotFound, "message not found")
	base.AddToErrorMap(ErrDraftAlreadyConfirmed, fiber.StatusConflict, "draft proposal has already been confirmed")
	base.AddToErrorMap(ErrDraftAlreadyDiscarded, fiber.StatusConflict, "draft proposal has already been discarded")
	base.AddToErrorMap(ErrDraftNotProposed, fiber.StatusUnprocessableEntity, "message does not contain a proposed draft")
	base.AddToErrorMap(ErrInvalidDraftProposal, fiber.StatusUnprocessableEntity, "invalid draft proposal")
	base.AddToErrorMap(ErrAttachmentNotFound, fiber.StatusNotFound, "attachment not found")
}
