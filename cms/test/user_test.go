package test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/kilip/omed/cms/internal/entity"
	"github.com/kilip/omed/cms/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T){
	iDonTHaveUser("john.doe@example.com")
	requestBody := model.RegisterUserRequest {
		Name: "John Doe",
		Email: "john.doe@example.com",
		PlainPassword: "secret",
	}
	jsonBody, err := json.Marshal(requestBody)
	assert.Nil(t, err)

	req, err := http.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(string(jsonBody)),
	)
	assert.Nil(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := app.Test(req)
	assert.Nil(t, err)

	bytes, err := io.ReadAll(res.Body)
	assert.Nil(t, err)

	responseBody := new(model.Resource[model.UserResponse])
	err = json.Unmarshal(bytes, responseBody)
	assert.Nil(t, err)

	assert.Equal(t, 201, res.StatusCode)
	assert.NotEmpty(t, responseBody.Data)
	assert.Equal(t, requestBody.Name, responseBody.Data.Name)
}

func TestLogin(t *testing.T){
	iHaveUser(TestUser)
	iAmNotLogin(TestUser)

	assert.NotNil(t, TestUser.ID, "ensure test user is exists")

	requestBody := model.LoginRequest{
		Email: TestUser.Email,
		Password: "secret",
	}

	bodyJson, err := json.Marshal(requestBody)
	assert.Nil(t, err)

	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(string(bodyJson)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := app.Test(request)
	assert.Nil(t, err)

	bytes, err := io.ReadAll(response.Body)
	assert.Nil(t, err)

	responseBody := new(model.Resource[model.LoginResponse])
	err = json.Unmarshal(bytes, responseBody)
	assert.Nil(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.NotNil(t, responseBody.Data.Token)


	user := new(entity.User)
	err = db.Preload("Tokens").Where("email = ?", requestBody.Email).First(user).Error
	assert.Nil(t, err)
	assert.Len(t, user.Tokens, 1)
	assert.Equal(t, responseBody.Data.Token,user.Tokens[0].Token)
}


func TestUserProfile(t *testing.T){
	TestLogin(t)

	token := new(entity.UserToken)
	err := db.Where("user_id = ?", TestUser.ID).First(token).Error
	assert.Nil(t, err)
	assert.NotNil(t, token.Token)

	request := httptest.NewRequest(
		http.MethodGet,
		"/me",
		nil,
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", token.Token)

	response, err := app.Test(request)
	assert.Nil(t, err)

	bytes, err := io.ReadAll(response.Body)
	assert.Nil(t, err)

	responseBody := new(model.Resource[model.AuthResponse])
	err = json.Unmarshal(bytes, responseBody)
	assert.Nil(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, TestUser.ID, responseBody.Data.UserID)
	assert.Equal(t, TestUser.Name, responseBody.Data.Name)
	assert.Equal(t, TestUser.Avatar, responseBody.Data.Avatar)
}

func TestAdminUserLogin(t *testing.T){
  requestBody := model.LoginRequest{
		Email: "admin@example.com",
		Password: "admin",
	}

	bodyJson, err := json.Marshal(requestBody)
	assert.Nil(t, err)

	request := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(string(bodyJson)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := app.Test(request)
	assert.Nil(t, err)

	bytes, err := io.ReadAll(response.Body)
	assert.Nil(t, err)

	responseBody := new(model.Resource[model.LoginResponse])
	err = json.Unmarshal(bytes, responseBody)
	assert.Nil(t, err)

	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.NotNil(t, responseBody.Data.Token)
}
