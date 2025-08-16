package test

import (
	"context"
	"errors"

	"github.com/kilip/omed/cms/internal/entity"
	"github.com/kilip/omed/cms/internal/utils"
	"gorm.io/gorm"
)

var TestUser = &entity.User{
	Name: "Test User",
	Email: "test@example.com",
	Avatar: "https://example.com/avatar",
}

func iDonTHaveUser(email string) {
	ctx := context.Background()
	_, err := gorm.G[entity.User](db).Where("email = ?", email).Delete(ctx)
	if err != nil {
		panic(err)
	}
}

func iHaveUser(user *entity.User){
	ctx := context.Background()
	ex, err := gorm.G[entity.User](db).Where("email = ?", user.Email).First(ctx);

	if errors.Is(err, gorm.ErrRecordNotFound){
		password, err := utils.HashPassword("secret")
		if err != nil {
			panic(err)
		}

		user.Password = password
		if err := gorm.G[entity.User](db).Create(ctx, user); err != nil {
			panic(err)
		}
	}

	user.ID = ex.ID
	
}

func iAmNotLogin(user *entity.User){
	ctx := context.Background()
	_, err := gorm.G[entity.UserToken](db).Where("user_id = ?", user.ID).Delete(ctx);
	if err != nil {
		panic(err)
	}
}
