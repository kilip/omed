package repository

import (
	"github.com/google/uuid"
	"github.com/kilip/omed/cms/internal/entity"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)


type TokenRepository struct {
	Repository[entity.UserToken]
	Log *logrus.Logger
}

func NewTokenRepository(log *logrus.Logger) *TokenRepository {
	return &TokenRepository{
		Log: log,
	}
}

func (r TokenRepository) CreateToken(db *gorm.DB, user *entity.User) (string, error) {
	token := uuid.New().String()

	userToken := &entity.UserToken{
		UserID: user.ID,
		Token: token,
	}

	err := db.Model(userToken).Create(userToken).Error

	return token,err
}

func (r TokenRepository) FindByToken(db *gorm.DB, userToken *entity.UserToken, token string) error {
	return db.Where("token = ?", token).First(userToken).Error
}
