package domain

import "time"

type User struct {
	ID        int64
	Name      string
	Email     string
	Avatar    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
