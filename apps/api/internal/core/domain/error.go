// Package domain defines core domain entities and business errors for the core module.
package domain

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/kilip/omed/api/internal/base"
)

// ErrEmailTaken indicates that the specified email address is already in use by another account.
var ErrEmailTaken = errors.New("USER_EMAIL_TAKEN")

var (
	// ErrInvalidRole indicates that a role being assigned is not registered by any module.
	ErrInvalidRole = errors.New("MEMBER_INVALID_ROLE")
	// ErrOwnerImmutable indicates an attempt to add, modify, or remove the workspace owner as a member.
	ErrOwnerImmutable = errors.New("MEMBER_OWNER_IMMUTABLE")
	// ErrAlreadyMember indicates that the user is already a member of the workspace.
	ErrAlreadyMember = errors.New("MEMBER_ALREADY_EXISTS")
	// ErrPersonalWorkspace indicates a forbidden operation on a personal workspace, such as deleting it.
	ErrPersonalWorkspace = errors.New("WORKSPACE_PERSONAL")
	// ErrPersonalWorkspaceMember indicates that members cannot be added to or invited to a personal workspace.
	ErrPersonalWorkspaceMember = errors.New("WORKSPACE_PERSONAL_MEMBER")
	// ErrInvitationNotFound indicates that the invitation does not exist.
	ErrInvitationNotFound = errors.New("INVITATION_NOT_FOUND")
	// ErrInvitationExpired indicates that the invitation has passed its expiration time.
	ErrInvitationExpired = errors.New("INVITATION_EXPIRED")
	// ErrInvitationMaxUsesReached indicates that the invitation has reached its maximum allowed uses.
	ErrInvitationMaxUsesReached = errors.New("INVITATION_MAX_USES_REACHED")
	// ErrInvitationRevoked indicates that the invitation has been revoked.
	ErrInvitationRevoked = errors.New("INVITATION_REVOKED")
	// ErrFileRequired indicates that no file was provided in an upload request.
	ErrFileRequired = errors.New("FILE_REQUIRED")
	// ErrFileEmpty indicates that the uploaded file has zero bytes.
	ErrFileEmpty = errors.New("FILE_EMPTY")
)

func init() {
	base.AddToErrorMap(ErrEmailTaken, fiber.StatusConflict, "email already used")
	base.AddToErrorMap(ErrInvalidRole, fiber.StatusUnprocessableEntity, "role is not registered")
	base.AddToErrorMap(ErrOwnerImmutable, fiber.StatusConflict, "workspace owner cannot be managed as a member")
	base.AddToErrorMap(ErrAlreadyMember, fiber.StatusConflict, "user is already a member of this workspace")
	base.AddToErrorMap(ErrPersonalWorkspace, fiber.StatusConflict, "personal workspace cannot be deleted")
	base.AddToErrorMap(ErrPersonalWorkspaceMember, fiber.StatusConflict, "cannot add members to a personal workspace")
	base.AddToErrorMap(ErrInvitationNotFound, fiber.StatusNotFound, "invitation not found")
	base.AddToErrorMap(ErrInvitationExpired, fiber.StatusUnprocessableEntity, "invitation has expired")
	base.AddToErrorMap(ErrInvitationMaxUsesReached, fiber.StatusUnprocessableEntity, "invitation usage limit reached")
	base.AddToErrorMap(ErrInvitationRevoked, fiber.StatusUnprocessableEntity, "invitation has been revoked")
	base.AddToErrorMap(ErrFileRequired, fiber.StatusBadRequest, "file is required")
	base.AddToErrorMap(ErrFileEmpty, fiber.StatusUnprocessableEntity, "uploaded file cannot be empty")
}
