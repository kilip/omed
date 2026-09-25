package http_test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/kilip/omed/finance/internal/model"
	"github.com/kilip/omed/finance/internal/util"
	"github.com/kilip/omed/finance/testutil"
	"github.com/stretchr/testify/assert"
)

func TestCreate(t *testing.T) {
	token := testutil.SignToken(t, jwt.MapClaims{
		"id":           util.GenerateID().String(),
		"name":         "Test User",
		"activeTeamId": util.GenerateID().String(),
		"role":         []string{"finance", "user"},
		"activeOrganizationRole": []string{"owner"},
	})

	request := &model.Account{
		Code: "1000",
		Name: "Test Account",
		Type: model.AccountTypeExpense,
	}

	resp := testutil.DoRequest(t, testutil.TestRequest{
		Method: "POST",
		Path:   "/accounts",
		Token:  token,
		Body:   request,
	})

	var json model.Envelope[model.Account]

	testutil.AssertStatus(t, resp, fiber.StatusCreated)
	testutil.DecodeJSON(t, resp, &json)

	assert.Equal(t, request.Name, json.Data.Name)
}
