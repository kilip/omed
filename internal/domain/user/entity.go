package user

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	ID           uuid.UUID `gorm:"primaryKey;default:uuidv7()" json:"id"`
	Name         string    `gorm:"index" json:"name"`
	Email        string    `gorm:"index;unique" json:"email"`
	Avatar       string    `gorm:"default:null" json:"avatar"`
	PasswordHash string    `gorm:"default:null;type:text" json:"-"`
}
