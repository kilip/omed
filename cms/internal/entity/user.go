package entity

type User struct {
	ID	uint64	`gorm:"column:id;primaryKey"`
	Name string `gorm:"column:name"`
	Email string `gorm:"column:email"`
	Password string `gorm:"column:password"`
	Token     string    `gorm:"column:token"`
	CreatedAt int64 `gorm:"column:created_at;autoCreateTime:milli"`
	UpdatedAt int64 `gorm:"column:updated_at;autoCreateTime:milli;autoUpdateTime:milli"`
}

func (u *User) TableName() string {
	return "cms_users"
}
