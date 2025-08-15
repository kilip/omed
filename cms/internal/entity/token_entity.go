package entity

import "gorm.io/gorm"

type UserToken struct {
	gorm.Model
	ID uint64 `gorm:"column:id;primaryKey"`
	Token string `gorm:"column:token"`
	UserID uint64 `gorm:"column:user_id"`
}

func (u *UserToken) TableName() string {
	return "cms_user_tokens"
}
