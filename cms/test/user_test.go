package test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/goccy/go-json"
	"github.com/kilip/omed/cms/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestRegister(t *testing.T){
	IDonTHaveUser("john.doe@example.com")
	requestBody := model.RegisterUserRequest {
		Name: "John Doe",
		Email: "john.doe@example.com",
		PlainPassword: "secret",
	}
	jsonBody, err := json.Marshal(requestBody)
	assert.Nil(t, err)

	req, err := http.NewRequest(
		http.MethodPost,
		"/users",
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

}
