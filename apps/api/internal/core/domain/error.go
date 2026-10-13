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
)

func init() {
	base.AddToErrorMap(ErrEmailTaken, fiber.StatusConflict, "email already used")
	base.AddToErrorMap(ErrInvalidRole, fiber.StatusUnprocessableEntity, "role is not registered")
	base.AddToErrorMap(ErrOwnerImmutable, fiber.StatusConflict, "workspace owner cannot be managed as a member")
	base.AddToErrorMap(ErrAlreadyMember, fiber.StatusConflict, "user is already a member of this workspace")
	base.AddToErrorMap(ErrPersonalWorkspace, fiber.StatusConflict, "personal workspace cannot be deleted")
}
