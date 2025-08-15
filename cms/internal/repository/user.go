package repository

import (
	"github.com/kilip/omed/cms/internal/entity"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UserRepository struct {
	Repository[entity.User]
	Log *logrus.Logger
}

func NewUserRepository(log *logrus.Logger) *UserRepository{
	return &UserRepository{
		Log: log,
	}
}

func (r UserRepository) CountByEmail(db *gorm.DB, email string) (int64, error){
	var total int64
	err := db.Model(new(entity.User)).Where("email = ?", email).Count(&total).Error
	return total, err
}