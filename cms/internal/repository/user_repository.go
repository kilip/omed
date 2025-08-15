package repository

import (
	"github.com/google/uuid"
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

func (r UserRepository) FindByEmail(db *gorm.DB, user *entity.User, email string) error {
	err := db.Model(user).Where("email=?", email).First(user).Error
	return err
}

func (r UserRepository) CreateToken(db *gorm.DB, user *entity.User) (string, error) {
	token := uuid.New().String()

	userToken := &entity.UserToken{
		UserID: user.ID,
		Token: token,
	}

	err := db.Model(userToken).Create(userToken).Error

	return token,err
}
