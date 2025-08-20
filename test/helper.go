package test

import (
	"context"
	"errors"
	"sync"

	"github.com/kilip/omed/internal/domain/user"
	"github.com/kilip/omed/internal/infra/database"
	"github.com/kilip/omed/internal/infra/database/dal"
	"github.com/kilip/omed/internal/utils"
	"gorm.io/gorm"
)

type UserHelper struct {
	Q *dal.Query
}

var (
	hinstance *UserHelper
	once      sync.Once
)

func NewUserHelper() *UserHelper {
	once.Do(func() {
		conf := utils.NewConfig()
		dab := database.NewGormDB(conf)
		err := dab.AutoMigrate(&user.User{})
		if err != nil {
			panic(err)
		}
		query := dal.Use(dab)
		hinstance = &UserHelper{query}
	})

	return hinstance
}

func (h UserHelper) IDonTHaveUser(ctx context.Context, email string) {
	q := h.Q.User
	exists, err := q.WithContext(ctx).Unscoped().Where(q.Email.Eq(email)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}
		panic(err)
	}

	err1 := h.Q.Transaction(func(tx *dal.Query) error {
		qu := tx.User

		if _, err := qu.WithContext(ctx).Unscoped().Where(qu.ID.Eq(exists.ID)).Delete(); err != nil {
			panic(err)
		}
		return nil
	})

	if err1 != nil {
		panic(err1)
	}

}

func (h UserHelper) IHaveUser(ctx context.Context, user *user.User) {
	err := h.Q.Transaction(func(tx *dal.Query) error {
		q := tx.User

		exists, err := q.WithContext(ctx).Where(q.Email.Eq(user.Email)).First()

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				q.WithContext(ctx).Create(user)
				return nil
			}

			return err
		}

		// user already exists, let's update
		user.ID = exists.ID
		q.WithContext(ctx).Where(q.ID.Eq(exists.ID)).Updates(user)
		return nil
	})
	if err != nil {
		panic(err)
	}
}
