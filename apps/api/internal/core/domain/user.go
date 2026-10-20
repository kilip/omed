package domain

import "time"

// User represents the domain entity for a registered user.
type User struct {
	ID        int64
	Name      string
	Email     string
	Avatar    string
	Locale    string
	Currency  string
	CreatedAt time.Time
	UpdatedAt time.Time
}
