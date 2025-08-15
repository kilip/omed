package entity

import "gorm.io/gorm"

type User struct {
	gorm.Model
	ID	uint64	`gorm:"column:id;primaryKey"`
	Name string `gorm:"column:name"`
	Email string `gorm:"column:email"`
	Password string `gorm:"column:password"`
	CreatedAt int64 `gorm:"column:create_at;autoCreateTime:milli"`
	UpdatedAt int64 `gorm:"column:updated_at;autoCreateTime:milli;autoUpdateTime:milli"`
	Tokens []UserToken
}

func (u *User) TableName() string {
	return "cms_users"
}
