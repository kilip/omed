package repository_test

import (
	"context"
	"testing"

	user2 "github.com/kilip/omed/internal/domain/user"
	"github.com/kilip/omed/internal/infra/database/repository"
	"github.com/kilip/omed/test"
	"github.com/stretchr/testify/assert"
)

var helper = test.NewUserHelper()
var users = repository.NewUserRepository(helper.Q)
var ctx = context.Background()

var user = &user2.User{
	Name:   "Test User",
	Email:  "test.user@example.com",
	Avatar: "http://example.com/avatar",
}

func TestCreate(t *testing.T) {
	helper.IDonTHaveUser(ctx, user.Email)
	err := users.Create(ctx, user)

	assert.Nil(t, err)
}

func TestUpdate(t *testing.T) {
	helper.IHaveUser(ctx, user)
	user.Name = "John Doe"
	err := users.Update(ctx, user)

	assert.Nil(t, err)
	assert.Equal(t, "John Doe", user.Name)
}

func TestDelete(t *testing.T) {
	helper.IHaveUser(ctx, user)
	err := users.Delete(ctx, user.ID.String())

	assert.Nil(t, err)
}
