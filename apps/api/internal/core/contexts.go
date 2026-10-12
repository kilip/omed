package core

const USER_CONTEXT_KEY = "user"

type UserRole string

const (
	UserRoleAdmin UserRole = "superadmin"
	UserRoleUser  UserRole = "user"
)

// UserContext current logged in user context
type UserContext struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	EmailVerified bool       `json:"emailVerified"`
	Roles         []UserRole `json:"roles"`
}
